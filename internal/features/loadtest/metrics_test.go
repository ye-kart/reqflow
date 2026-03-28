package loadtest

import (
	"testing"
	"time"
)

func TestMetrics_Record_counts_successes_and_failures(t *testing.T) {
	m := &Metrics{}

	m.Record(10*time.Millisecond, nil)
	m.Record(20*time.Millisecond, nil)
	m.Record(0, errStub)

	if m.TotalRequests != 3 {
		t.Fatalf("TotalRequests = %d, want 3", m.TotalRequests)
	}
	if m.Successes != 2 {
		t.Fatalf("Successes = %d, want 2", m.Successes)
	}
	if m.Failures != 1 {
		t.Fatalf("Failures = %d, want 1", m.Failures)
	}
}

func TestMetrics_Record_stores_durations_only_on_success(t *testing.T) {
	m := &Metrics{}

	m.Record(10*time.Millisecond, nil)
	m.Record(0, errStub)

	if len(m.Durations) != 1 {
		t.Fatalf("len(Durations) = %d, want 1", len(m.Durations))
	}
}

func TestMetrics_Average(t *testing.T) {
	m := &Metrics{}
	m.Record(10*time.Millisecond, nil)
	m.Record(20*time.Millisecond, nil)
	m.Record(30*time.Millisecond, nil)

	avg := m.Average()
	if avg != 20*time.Millisecond {
		t.Fatalf("Average = %v, want 20ms", avg)
	}
}

func TestMetrics_Average_empty(t *testing.T) {
	m := &Metrics{}
	if avg := m.Average(); avg != 0 {
		t.Fatalf("Average = %v, want 0", avg)
	}
}

func TestMetrics_Min(t *testing.T) {
	m := &Metrics{}
	m.Record(30*time.Millisecond, nil)
	m.Record(10*time.Millisecond, nil)
	m.Record(20*time.Millisecond, nil)

	if got := m.Min(); got != 10*time.Millisecond {
		t.Fatalf("Min = %v, want 10ms", got)
	}
}

func TestMetrics_Min_empty(t *testing.T) {
	m := &Metrics{}
	if got := m.Min(); got != 0 {
		t.Fatalf("Min = %v, want 0", got)
	}
}

func TestMetrics_Max(t *testing.T) {
	m := &Metrics{}
	m.Record(10*time.Millisecond, nil)
	m.Record(30*time.Millisecond, nil)
	m.Record(20*time.Millisecond, nil)

	if got := m.Max(); got != 30*time.Millisecond {
		t.Fatalf("Max = %v, want 30ms", got)
	}
}

func TestMetrics_Max_empty(t *testing.T) {
	m := &Metrics{}
	if got := m.Max(); got != 0 {
		t.Fatalf("Max = %v, want 0", got)
	}
}

func TestMetrics_Percentile_P50(t *testing.T) {
	m := &Metrics{}
	// Record 100 values: 1ms, 2ms, ..., 100ms
	for i := 1; i <= 100; i++ {
		m.Record(time.Duration(i)*time.Millisecond, nil)
	}

	p50 := m.Percentile(50)
	if p50 != 50*time.Millisecond {
		t.Fatalf("P50 = %v, want 50ms", p50)
	}
}

func TestMetrics_Percentile_P90(t *testing.T) {
	m := &Metrics{}
	for i := 1; i <= 100; i++ {
		m.Record(time.Duration(i)*time.Millisecond, nil)
	}

	p90 := m.Percentile(90)
	if p90 != 90*time.Millisecond {
		t.Fatalf("P90 = %v, want 90ms", p90)
	}
}

func TestMetrics_Percentile_P95(t *testing.T) {
	m := &Metrics{}
	for i := 1; i <= 100; i++ {
		m.Record(time.Duration(i)*time.Millisecond, nil)
	}

	p95 := m.Percentile(95)
	if p95 != 95*time.Millisecond {
		t.Fatalf("P95 = %v, want 95ms", p95)
	}
}

func TestMetrics_Percentile_P99(t *testing.T) {
	m := &Metrics{}
	for i := 1; i <= 100; i++ {
		m.Record(time.Duration(i)*time.Millisecond, nil)
	}

	p99 := m.Percentile(99)
	if p99 != 99*time.Millisecond {
		t.Fatalf("P99 = %v, want 99ms", p99)
	}
}

func TestMetrics_Percentile_empty(t *testing.T) {
	m := &Metrics{}
	if got := m.Percentile(50); got != 0 {
		t.Fatalf("Percentile(50) = %v, want 0", got)
	}
}

func TestMetrics_Percentile_single_value(t *testing.T) {
	m := &Metrics{}
	m.Record(42*time.Millisecond, nil)

	if got := m.Percentile(50); got != 42*time.Millisecond {
		t.Fatalf("Percentile(50) = %v, want 42ms", got)
	}
	if got := m.Percentile(99); got != 42*time.Millisecond {
		t.Fatalf("Percentile(99) = %v, want 42ms", got)
	}
}

func TestMetrics_Percentile_unsorted_input(t *testing.T) {
	m := &Metrics{}
	// Record in random order
	for _, d := range []int{50, 10, 90, 30, 70, 20, 80, 40, 60, 100} {
		m.Record(time.Duration(d)*time.Millisecond, nil)
	}

	p50 := m.Percentile(50)
	if p50 != 50*time.Millisecond {
		t.Fatalf("P50 = %v, want 50ms", p50)
	}
}

func TestMetrics_RequestsPerSecond(t *testing.T) {
	m := &Metrics{}
	m.Record(10*time.Millisecond, nil)
	m.Record(10*time.Millisecond, nil)
	m.Record(10*time.Millisecond, nil)
	m.Record(10*time.Millisecond, nil)
	m.Record(10*time.Millisecond, nil)

	rps := m.RequestsPerSecond(1 * time.Second)
	if rps != 5.0 {
		t.Fatalf("RPS = %v, want 5.0", rps)
	}
}

func TestMetrics_RequestsPerSecond_half_second(t *testing.T) {
	m := &Metrics{}
	m.Record(10*time.Millisecond, nil)
	m.Record(10*time.Millisecond, nil)

	rps := m.RequestsPerSecond(500 * time.Millisecond)
	if rps != 4.0 {
		t.Fatalf("RPS = %v, want 4.0", rps)
	}
}

func TestMetrics_RequestsPerSecond_zero_elapsed(t *testing.T) {
	m := &Metrics{}
	m.Record(10*time.Millisecond, nil)

	if rps := m.RequestsPerSecond(0); rps != 0 {
		t.Fatalf("RPS = %v, want 0", rps)
	}
}

func TestMetrics_Record_is_concurrent_safe(t *testing.T) {
	m := &Metrics{}
	done := make(chan struct{})

	for i := 0; i < 100; i++ {
		go func() {
			m.Record(time.Millisecond, nil)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}

	if m.TotalRequests != 100 {
		t.Fatalf("TotalRequests = %d, want 100", m.TotalRequests)
	}
}
