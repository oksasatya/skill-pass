package webhook_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hibiken/asynq"

	"github.com/oksasatya/skillpass/services/notify/internal/adapter/webhook"
	"github.com/oksasatya/skillpass/services/notify/internal/usecase"
)

func TestHandler_ProcessTask_Success(t *testing.T) {
	var gotBody []byte
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotSig = r.Header.Get("X-SkillPass-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	h := webhook.NewHandler(srv.URL, "test-secret")
	payload := []byte(`{"event":"certificate.issued","data":{"tokenId":"1"}}`)
	task := asynq.NewTask(webhook.DeliverTaskType, payload)

	if err := h.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("ProcessTask: %v", err)
	}
	if string(gotBody) != string(payload) {
		t.Errorf("body = %s, want %s", gotBody, payload)
	}
	wantSig := "sha256=" + usecase.SignPayload("test-secret", payload)
	if gotSig != wantSig {
		t.Errorf("signature header = %q, want %q", gotSig, wantSig)
	}
}

func TestHandler_ProcessTask_NonSuccessStatus_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	h := webhook.NewHandler(srv.URL, "test-secret")
	task := asynq.NewTask(webhook.DeliverTaskType, []byte(`{}`))

	if err := h.ProcessTask(context.Background(), task); err == nil {
		t.Fatal("want an error on a 500 response (asynq must retry)")
	}
}

func TestHandler_ProcessTask_UnreachableURL_ReturnsError(t *testing.T) {
	h := webhook.NewHandler("http://127.0.0.1:1", "test-secret") // nothing listens on port 1
	task := asynq.NewTask(webhook.DeliverTaskType, []byte(`{}`))

	if err := h.ProcessTask(context.Background(), task); err == nil {
		t.Fatal("want an error when the endpoint is unreachable")
	}
}

var _ asynq.Handler = (*webhook.Handler)(nil)
