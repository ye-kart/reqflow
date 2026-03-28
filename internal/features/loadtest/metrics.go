package loadtest

import (
	"sort"
	"sync"
	"time"
)

// Metrics collects per-request timing data for a load test run.
// All methods are safe for concurrent use.
type Metrics struct {
	mu            sync.Mutex
	TotalRequests int
	Successes     int
	Failures      int
	Durations     []time.Duration
}

// Record records the result of a single request.
func (m *Metrics) Record(duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	if err != nil {
		m.Failures++
		return
	}
	m.Successes++
	m.Durations = append(m.Durations, duration)
}

// Percentile returns the p-th percentile latency (e.g. 50, 90, 95, 99).
// It sorts a copy of the durations to avoid mutating the original slice.
func (m *Metrics) Percentile(p float64) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(m.Durations))
	copy(sorted, m.Durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := int(p/100*float64(len(sorted))+0.5) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// Average returns the mean request duration.
func (m *Metrics) Average() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range m.Durations {
		total += d
	}
	return total / time.Duration(len(m.Durations))
}

// Min returns the minimum request duration.
func (m *Metrics) Min() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Durations) == 0 {
		return 0
	}
	min := m.Durations[0]
	for _, d := range m.Durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

// Max returns the maximum request duration.
func (m *Metrics) Max() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Durations) == 0 {
		return 0
	}
	max := m.Durations[0]
	for _, d := range m.Durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

// RequestsPerSecond calculates the throughput over the given elapsed time.
func (m *Metrics) RequestsPerSecond(elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	m.mu.Lock()
	total := m.TotalRequests
	m.mu.Unlock()
	return float64(total) / elapsed.Seconds()
}

// Result holds the final aggregated metrics for a load test run.
type Result struct {
	TotalRequests  int
	Successes      int
	Failures       int
	Average        time.Duration
	Min            time.Duration
	Max            time.Duration
	P50            time.Duration
	P90            time.Duration
	P95            time.Duration
	P99            time.Duration
	RequestsPerSec float64
	Duration       time.Duration
	ErrorRate      float64
}

// Snapshot represents a point-in-time progress update during a load test.
type Snapshot struct {
	Elapsed  time.Duration
	VUs      int
	Requests int
	RPS      float64
}
