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
