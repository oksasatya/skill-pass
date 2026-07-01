package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// fakeEventStream implements grpc.ServerStreamingClient[certv1.CertificateEvent] for tests.
// It replays a fixed sequence of events, then either returns a terminal error or blocks
// until ctx is cancelled (mirroring a real long-lived stream with nothing left to send).
type fakeEventStream struct {
	events []*certv1.CertificateEvent
	idx    int
	endErr error
	ctx    context.Context
}

func (f *fakeEventStream) Recv() (*certv1.CertificateEvent, error) {
	if f.idx < len(f.events) {
		ev := f.events[f.idx]
		f.idx++
		return ev, nil
	}
	if f.endErr != nil {
		return nil, f.endErr
	}
	<-f.ctx.Done()
	return nil, f.ctx.Err()
}

func (f *fakeEventStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeEventStream) Trailer() metadata.MD         { return nil }
func (f *fakeEventStream) CloseSend() error             { return nil }
func (f *fakeEventStream) Context() context.Context     { return f.ctx }
func (f *fakeEventStream) SendMsg(_ any) error          { return nil }
func (f *fakeEventStream) RecvMsg(_ any) error          { return nil }

func sampleEvent(tokenID string) *certv1.CertificateEvent {
	return &certv1.CertificateEvent{
		EventType: "issued",
		Certificate: &certv1.Certificate{
			TokenId:  tokenID,
			Title:    "Go Expert",
			IssuedAt: timestamppb.New(time.Unix(1700000000, 0).UTC()),
		},
	}
}

func TestRecvLoop_ForwardsEventsThenErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &fakeEventStream{events: []*certv1.CertificateEvent{sampleEvent("1"), sampleEvent("2")}, endErr: io.EOF, ctx: ctx}
	events := make(chan *certv1.CertificateEvent)
	errs := make(chan error, 1)

	go recvLoop(ctx, stream, events, errs)

	for _, want := range []string{"1", "2"} {
		select {
		case ev := <-events:
			if ev.GetCertificate().GetTokenId() != want {
				t.Fatalf("token_id = %q, want %q", ev.GetCertificate().GetTokenId(), want)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	}

	select {
	case err := <-errs:
		if err != io.EOF {
			t.Fatalf("err = %v, want io.EOF", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for terminal error")
	}
}

func TestRecvLoop_StopsOnCtxCancelWithoutLeaking(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stream := &fakeEventStream{events: []*certv1.CertificateEvent{sampleEvent("1")}, ctx: ctx}
	// unbuffered, never read — recvLoop must not block forever once ctx is cancelled
	events := make(chan *certv1.CertificateEvent)
	errs := make(chan error, 1)

	done := make(chan struct{})
	go func() {
		recvLoop(ctx, stream, events, errs)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("recvLoop leaked: did not return after ctx cancel")
	}
}

func TestWriteSSEEvent_Format(t *testing.T) {
	w := httptest.NewRecorder()
	writeSSEEvent(w, sampleEvent("7"))

	body := w.Body.String()
	if !strings.HasPrefix(body, "data: ") || !strings.HasSuffix(body, "\n\n") {
		t.Fatalf("unexpected SSE frame shape: %q", body)
	}

	raw := strings.TrimSuffix(strings.TrimPrefix(body, "data: "), "\n\n")
	var payload struct {
		EventType   string         `json:"eventType"`
		Certificate CertificateDTO `json:"certificate"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.EventType != "issued" || payload.Certificate.TokenID != "7" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
