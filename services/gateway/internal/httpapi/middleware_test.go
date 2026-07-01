package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithObservability_RecoversFromPanic(t *testing.T) {
	panicking := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})
	h := WithObservability(panicking, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := httptest.NewRequest(http.MethodGet, "/certificates", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req) // must not panic out of the test

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestWithObservability_PassesThroughNormalResponse(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := WithObservability(ok, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// Regression: the observability wrapper must not hide http.Flusher from a streaming handler
// (e.g. StreamCertificateEvents) — a naive ResponseWriter wrapper breaks SSE silently.
func TestWithObservability_PreservesFlusher(t *testing.T) {
	streaming := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Error("http.Flusher not available through the observability wrapper")
		}
	})
	h := WithObservability(streaming, slog.New(slog.NewTextHandler(io.Discard, nil)))

	req := httptest.NewRequest(http.MethodGet, "/certificates/stream", nil)
	w := httptest.NewRecorder() // *httptest.ResponseRecorder implements http.Flusher

	h.ServeHTTP(w, req)
}
