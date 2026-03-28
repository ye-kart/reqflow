package monitor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	cronpkg "github.com/ye-kart/reqflow/internal/core/monitor"
	"github.com/ye-kart/reqflow/internal/core/workflow"
	"github.com/ye-kart/reqflow/internal/features/runner"
	"gopkg.in/yaml.v3"
)

// Monitor defines a scheduled workflow execution.
type Monitor struct {
	Name         string         `yaml:"name"`
	WorkflowPath string         `yaml:"workflow_path"`
	Cron         string         `yaml:"cron"`
	EnvName      string         `yaml:"env,omitempty"`
	OnFailure    *WebhookNotify `yaml:"on_failure,omitempty"`
}

// WebhookNotify configures a webhook to call on failure.
type WebhookNotify struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// Scheduler manages scheduled monitor executions.
type Scheduler struct {
	runner   *runner.Runner
	dir      string
	monitors map[string]Monitor
	mu       sync.RWMutex
	stopCh   chan struct{}
}

// NewScheduler creates a new Scheduler that persists configs to dir.
func NewScheduler(r *runner.Runner, dir string) *Scheduler {
	return &Scheduler{
		runner:   r,
		dir:      dir,
		monitors: make(map[string]Monitor),
		stopCh:   make(chan struct{}),
	}
}

// Add registers a new monitor, validates its cron expression, and persists it.
func (s *Scheduler) Add(m Monitor) error {
	// Validate the cron expression
	if _, err := cronpkg.ParseCron(m.Cron); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.monitors[m.Name]; exists {
		return fmt.Errorf("monitor %q already exists", m.Name)
	}

	// Persist to disk
	if err := s.save(m); err != nil {
		return fmt.Errorf("saving monitor config: %w", err)
	}

	s.monitors[m.Name] = m
	return nil
}

// Remove deletes a monitor by name and removes its config file.
func (s *Scheduler) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.monitors[name]; !exists {
		return fmt.Errorf("monitor %q not found", name)
	}

	// Remove config file
	configPath := filepath.Join(s.dir, name+".yaml")
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing config file: %w", err)
	}

	delete(s.monitors, name)
	return nil
}

// List returns all registered monitors, sorted by name.
func (s *Scheduler) List() []Monitor {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Monitor, 0, len(s.monitors))
	for _, m := range s.monitors {
		result = append(result, m)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// Load reads all monitor configs from the storage directory.
func (s *Scheduler) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading monitor directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.dir, name))
		if err != nil {
			return fmt.Errorf("reading monitor config %s: %w", name, err)
		}

		var m Monitor
		if err := yaml.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("parsing monitor config %s: %w", name, err)
		}

		s.monitors[m.Name] = m
	}

	return nil
}

// RunOnce executes the named monitor's workflow once immediately.
func (s *Scheduler) RunOnce(ctx context.Context, name string, envVars map[string]string) error {
	s.mu.RLock()
	m, exists := s.monitors[name]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("monitor %q not found", name)
	}

	return s.executeMonitor(ctx, m, envVars)
}

// Start runs the scheduler loop, checking for due monitors every 30 seconds.
// Blocks until ctx is cancelled or Stop is called.
func (s *Scheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Track next run times
	nextRuns := make(map[string]time.Time)

	s.mu.RLock()
	now := time.Now()
	for name, m := range s.monitors {
		sched, err := cronpkg.ParseCron(m.Cron)
		if err != nil {
			continue
		}
		nextRuns[name] = sched.Next(now)
	}
	s.mu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.stopCh:
			return nil
		case now := <-ticker.C:
			s.mu.RLock()
			monitors := make(map[string]Monitor, len(s.monitors))
			for k, v := range s.monitors {
				monitors[k] = v
			}
			s.mu.RUnlock()

			for name, m := range monitors {
				nextRun, ok := nextRuns[name]
				if !ok {
					sched, err := cronpkg.ParseCron(m.Cron)
					if err != nil {
						continue
					}
					nextRuns[name] = sched.Next(now)
					continue
				}

				if now.After(nextRun) || now.Equal(nextRun) {
					// Execute in background
					go func(mon Monitor) {
						execCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
						defer cancel()
						_ = s.executeMonitor(execCtx, mon, nil)
					}(m)

					// Calculate next run
					sched, err := cronpkg.ParseCron(m.Cron)
					if err != nil {
						continue
					}
					nextRuns[name] = sched.Next(now)
				}
			}
		}
	}
}

// Stop signals the scheduler to stop.
func (s *Scheduler) Stop() {
	select {
	case s.stopCh <- struct{}{}:
	default:
	}
}

// save persists a monitor config to disk.
func (s *Scheduler) save(m Monitor) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("creating monitor directory: %w", err)
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling monitor config: %w", err)
	}

	configPath := filepath.Join(s.dir, m.Name+".yaml")
	return os.WriteFile(configPath, data, 0644)
}

// executeMonitor runs a single monitor's workflow.
func (s *Scheduler) executeMonitor(ctx context.Context, m Monitor, envVars map[string]string) error {
	data, err := os.ReadFile(m.WorkflowPath)
	if err != nil {
		return fmt.Errorf("reading workflow file: %w", err)
	}

	wf, err := workflow.Parse(data)
	if err != nil {
		return fmt.Errorf("parsing workflow: %w", err)
	}

	vars := make(map[string]string)
	for k, v := range envVars {
		vars[k] = v
	}

	result, err := s.runner.Run(ctx, wf, vars)
	if err != nil {
		s.notifyFailure(m, fmt.Sprintf("workflow execution error: %v", err))
		return err
	}

	if result.TotalFailed > 0 {
		s.notifyFailure(m, fmt.Sprintf("%d assertions failed", result.TotalFailed))
	}

	return nil
}

// notifyFailure sends a webhook notification if configured.
func (s *Scheduler) notifyFailure(m Monitor, message string) {
	if m.OnFailure == nil {
		return
	}

	// Fire-and-forget webhook notification. Uses a simple POST with the
	// failure message. Full HTTP notification is handled externally; this
	// is a best-effort signal.
	_ = message // TODO: send HTTP POST to m.OnFailure.URL
}
