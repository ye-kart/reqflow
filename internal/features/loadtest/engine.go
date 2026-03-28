package loadtest

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
	"github.com/ye-kart/reqflow/internal/ports/driven"
)

// Config holds the configuration for a load test run.
type Config struct {
	URL      string
	Method   string
	Headers  map[string]string
	Body     []byte
	VUs      int           // virtual users (concurrent goroutines)
	Duration time.Duration
	RampUp   time.Duration // ramp-up period
}

// Engine orchestrates a load test using virtual users.
type Engine struct {
	httpClient driven.HTTPClient
}

// NewEngine creates a new Engine with the given HTTP client.
func NewEngine(hc driven.HTTPClient) *Engine {
	return &Engine{httpClient: hc}
}

// Run executes the load test according to cfg, sending progress updates
// to the progress channel (if non-nil). Returns aggregated results.
func (e *Engine) Run(ctx context.Context, cfg Config, progress chan<- Snapshot) (Result, error) {
	metrics := &Metrics{}
	req := e.buildRequest(cfg)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	start := time.Now()
	deadline := start.Add(cfg.Duration)

	var activeVUs atomic.Int64
	var wg sync.WaitGroup

	// Start progress reporter before launching VUs so ramp-up is visible.
	var progressWg sync.WaitGroup
	if progress != nil {
		progressWg.Add(1)
		go func() {
			defer progressWg.Done()
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					elapsed := time.Since(start)
					if elapsed > cfg.Duration {
						return
					}
					metrics.mu.Lock()
					reqs := metrics.TotalRequests
					metrics.mu.Unlock()
					progress <- Snapshot{
						Elapsed:  elapsed,
						VUs:      int(activeVUs.Load()),
						Requests: reqs,
						RPS:      metrics.RequestsPerSecond(elapsed),
					}
				}
			}
		}()
	}

	// Launch VUs, respecting ramp-up schedule.
	for i := 0; i < cfg.VUs; i++ {
		if cfg.RampUp > 0 && i > 0 {
			// Target launch time for VU i: rampUp * i / VUs from start.
			target := start.Add(cfg.RampUp * time.Duration(i) / time.Duration(cfg.VUs))
			wait := time.Until(target)
			if wait > 0 {
				select {
				case <-ctx.Done():
				case <-time.After(wait):
				}
			}
		}

		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		activeVUs.Add(1)
		go e.vuWorker(ctx, req, deadline, metrics, &activeVUs, &wg)
	}

	wg.Wait()
	cancel() // Stop progress reporter.
	progressWg.Wait()

	elapsed := time.Since(start)

	return e.buildResult(metrics, elapsed), nil
}

// vuWorker is a single virtual user goroutine that sends requests in a loop
// until the deadline or context cancellation.
func (e *Engine) vuWorker(ctx context.Context, req domain.HTTPRequest, deadline time.Time, metrics *Metrics, activeVUs *atomic.Int64, wg *sync.WaitGroup) {
	defer wg.Done()
	defer activeVUs.Add(-1)

	for {
		if time.Now().After(deadline) {
			return
		}
		if ctx.Err() != nil {
			return
		}

		start := time.Now()
		_, err := e.httpClient.Do(ctx, req)
		duration := time.Since(start)

		// Don't record context cancellation as a failure.
		if ctx.Err() != nil {
			return
		}

		metrics.Record(duration, err)
	}
}

// buildRequest creates a domain.HTTPRequest from the load test Config.
func (e *Engine) buildRequest(cfg Config) domain.HTTPRequest {
	req := domain.HTTPRequest{
		Method: domain.HTTPMethod(cfg.Method),
		URL:    cfg.URL,
		Body:   cfg.Body,
	}
	for k, v := range cfg.Headers {
		req.Headers = append(req.Headers, domain.Header{Key: k, Value: v})
	}
	return req
}

// buildResult aggregates metrics into a final Result.
func (e *Engine) buildResult(m *Metrics, elapsed time.Duration) Result {
	m.mu.Lock()
	total := m.TotalRequests
	successes := m.Successes
	failures := m.Failures
	m.mu.Unlock()

	var errorRate float64
	if total > 0 {
		errorRate = float64(failures) / float64(total)
	}

	return Result{
		TotalRequests:  total,
		Successes:      successes,
		Failures:       failures,
		Average:        m.Average(),
		Min:            m.Min(),
		Max:            m.Max(),
		P50:            m.Percentile(50),
		P90:            m.Percentile(90),
		P95:            m.Percentile(95),
		P99:            m.Percentile(99),
		RequestsPerSec: m.RequestsPerSecond(elapsed),
		Duration:       elapsed,
		ErrorRate:      errorRate,
	}
}
