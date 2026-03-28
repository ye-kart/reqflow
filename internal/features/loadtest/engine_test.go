package loadtest

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
)

// mockHTTPClient is a test double for driven.HTTPClient.
type mockHTTPClient struct {
	doFunc    func(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error)
	callCount atomic.Int64
}

func (m *mockHTTPClient) Do(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
	m.callCount.Add(1)
	if m.doFunc != nil {
		return m.doFunc(ctx, req)
	}
	return domain.HTTPResponse{StatusCode: 200, Status: "200 OK"}, nil
}

func TestEngine_runs_correct_number_of_VUs(t *testing.T) {
	var peakVUs atomic.Int64
	var activeVUs atomic.Int64

	client := &mockHTTPClient{
		doFunc: func(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			current := activeVUs.Add(1)
			// Track peak concurrency.
			for {
				peak := peakVUs.Load()
				if current <= peak || peakVUs.CompareAndSwap(peak, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			activeVUs.Add(-1)
			return domain.HTTPResponse{StatusCode: 200, Status: "200 OK"}, nil
		},
	}

	engine := NewEngine(client)
	cfg := Config{
		URL:      "http://localhost/test",
		Method:   "GET",
		VUs:      5,
		Duration: 200 * time.Millisecond,
	}

	result, err := engine.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	peak := peakVUs.Load()
	if peak < 3 {
		t.Fatalf("peak VUs = %d, want at least 3 (configured 5)", peak)
	}
	if peak > 5 {
		t.Fatalf("peak VUs = %d, want at most 5", peak)
	}

	if result.TotalRequests == 0 {
		t.Fatal("TotalRequests = 0, want > 0")
	}
}

func TestEngine_respects_duration(t *testing.T) {
	client := &mockHTTPClient{}
	engine := NewEngine(client)

	duration := 200 * time.Millisecond
	cfg := Config{
		URL:      "http://localhost/test",
		Method:   "GET",
		VUs:      2,
		Duration: duration,
	}

	start := time.Now()
	_, err := engine.Run(context.Background(), cfg, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Should run for approximately the configured duration.
	if elapsed < duration-50*time.Millisecond {
		t.Fatalf("elapsed = %v, want >= %v", elapsed, duration-50*time.Millisecond)
	}
	// Should not significantly overshoot.
	if elapsed > duration+200*time.Millisecond {
		t.Fatalf("elapsed = %v, want <= %v", elapsed, duration+200*time.Millisecond)
	}
}

func TestEngine_rampup_gradually_increases_VUs(t *testing.T) {
	var snapshots []Snapshot
	client := &mockHTTPClient{
		doFunc: func(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			time.Sleep(5 * time.Millisecond)
			return domain.HTTPResponse{StatusCode: 200, Status: "200 OK"}, nil
		},
	}

	engine := NewEngine(client)
	cfg := Config{
		URL:      "http://localhost/test",
		Method:   "GET",
		VUs:      10,
		Duration: 500 * time.Millisecond,
		RampUp:   300 * time.Millisecond,
	}

	progress := make(chan Snapshot, 100)
	done := make(chan struct{})
	go func() {
		for s := range progress {
			snapshots = append(snapshots, s)
		}
		close(done)
	}()

	_, err := engine.Run(context.Background(), cfg, progress)
	close(progress)
	<-done

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(snapshots) < 2 {
		t.Fatalf("got %d snapshots, want at least 2", len(snapshots))
	}

	// First snapshot should have fewer VUs than the last.
	first := snapshots[0]
	last := snapshots[len(snapshots)-1]
	if first.VUs >= last.VUs {
		t.Fatalf("ramp-up not detected: first VUs=%d, last VUs=%d", first.VUs, last.VUs)
	}
}

func TestEngine_results_are_accurate(t *testing.T) {
	successCount := 0
	failCount := 0
	client := &mockHTTPClient{
		doFunc: func(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			successCount++
			if successCount%3 == 0 {
				failCount++
				return domain.HTTPResponse{}, errStub
			}
			return domain.HTTPResponse{StatusCode: 200, Status: "200 OK"}, nil
		},
	}

	engine := NewEngine(client)
	cfg := Config{
		URL:      "http://localhost/test",
		Method:   "GET",
		VUs:      1, // Single VU so counting is deterministic.
		Duration: 200 * time.Millisecond,
	}

	result, err := engine.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.TotalRequests == 0 {
		t.Fatal("TotalRequests = 0")
	}
	if result.Successes+result.Failures != result.TotalRequests {
		t.Fatalf("Successes(%d) + Failures(%d) != TotalRequests(%d)",
			result.Successes, result.Failures, result.TotalRequests)
	}
	if result.Failures == 0 {
		t.Fatal("expected some failures")
	}
	if result.Duration <= 0 {
		t.Fatalf("Duration = %v, want > 0", result.Duration)
	}
	if result.RequestsPerSec <= 0 {
		t.Fatalf("RequestsPerSec = %v, want > 0", result.RequestsPerSec)
	}

	expectedErrorRate := float64(result.Failures) / float64(result.TotalRequests)
	if diff := result.ErrorRate - expectedErrorRate; diff > 0.001 || diff < -0.001 {
		t.Fatalf("ErrorRate = %v, want ~%v", result.ErrorRate, expectedErrorRate)
	}
}

func TestEngine_context_cancellation_stops_run(t *testing.T) {
	client := &mockHTTPClient{
		doFunc: func(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			time.Sleep(5 * time.Millisecond)
			return domain.HTTPResponse{StatusCode: 200, Status: "200 OK"}, nil
		},
	}

	engine := NewEngine(client)
	cfg := Config{
		URL:      "http://localhost/test",
		Method:   "GET",
		VUs:      5,
		Duration: 10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := engine.Run(ctx, cfg, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Should stop well before the 10s duration.
	if elapsed > 1*time.Second {
		t.Fatalf("elapsed = %v, should have stopped after ctx cancel", elapsed)
	}
}

func TestEngine_sends_progress_snapshots(t *testing.T) {
	client := &mockHTTPClient{
		doFunc: func(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			time.Sleep(5 * time.Millisecond)
			return domain.HTTPResponse{StatusCode: 200, Status: "200 OK"}, nil
		},
	}

	engine := NewEngine(client)
	cfg := Config{
		URL:      "http://localhost/test",
		Method:   "GET",
		VUs:      3,
		Duration: 300 * time.Millisecond,
	}

	progress := make(chan Snapshot, 100)
	var snapshots []Snapshot
	done := make(chan struct{})
	go func() {
		for s := range progress {
			snapshots = append(snapshots, s)
		}
		close(done)
	}()

	_, err := engine.Run(context.Background(), cfg, progress)
	close(progress)
	<-done

	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(snapshots) == 0 {
		t.Fatal("expected at least one progress snapshot")
	}

	// Snapshots should have increasing elapsed times.
	for i := 1; i < len(snapshots); i++ {
		if snapshots[i].Elapsed < snapshots[i-1].Elapsed {
			t.Fatalf("snapshot[%d].Elapsed=%v < snapshot[%d].Elapsed=%v",
				i, snapshots[i].Elapsed, i-1, snapshots[i-1].Elapsed)
		}
	}
}

func TestEngine_headers_and_body_are_sent(t *testing.T) {
	var capturedReq domain.HTTPRequest
	client := &mockHTTPClient{
		doFunc: func(ctx context.Context, req domain.HTTPRequest) (domain.HTTPResponse, error) {
			capturedReq = req
			return domain.HTTPResponse{StatusCode: 200, Status: "200 OK"}, nil
		},
	}

	engine := NewEngine(client)
	cfg := Config{
		URL:     "http://localhost/test",
		Method:  "POST",
		Headers: map[string]string{"X-Test": "hello"},
		Body:    []byte(`{"key":"value"}`),
		VUs:     1,
		Duration: 50 * time.Millisecond,
	}

	_, err := engine.Run(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if string(capturedReq.Method) != "POST" {
		t.Fatalf("Method = %s, want POST", capturedReq.Method)
	}
	if string(capturedReq.Body) != `{"key":"value"}` {
		t.Fatalf("Body = %s, want {\"key\":\"value\"}", capturedReq.Body)
	}

	found := false
	for _, h := range capturedReq.Headers {
		if h.Key == "X-Test" && h.Value == "hello" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected header X-Test: hello not found")
	}
}
