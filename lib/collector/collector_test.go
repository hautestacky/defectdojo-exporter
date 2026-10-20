package collector

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// newProductsServer serves an empty product list and counts how many times it is called.
func newProductsServer(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()

	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/products/" {
			calls.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"next":"","results":[]}`)); err != nil {
			t.Errorf("error writing response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	return server, &calls
}

func TestCollectMetricsWaitsForInterval(t *testing.T) {
	server, calls := newProductsServer(t)

	go CollectMetrics(server.URL, "token", 1, time.Hour, time.Second, false)

	time.Sleep(300 * time.Millisecond)

	if got := calls.Load(); got != 1 {
		t.Errorf("expected 1 collection pass before the interval elapsed, got %d", got)
	}
}

func TestCollectMetricsRunsAgainAfterInterval(t *testing.T) {
	server, calls := newProductsServer(t)

	go CollectMetrics(server.URL, "token", 1, 100*time.Millisecond, time.Second, false)

	time.Sleep(550 * time.Millisecond)

	// One pass at start plus one per tick: about 6, with room for scheduling jitter.
	if got := calls.Load(); got < 3 || got > 8 {
		t.Errorf("expected between 3 and 8 collection passes, got %d", got)
	}
}
