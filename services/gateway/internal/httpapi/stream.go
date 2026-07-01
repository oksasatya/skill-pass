package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"google.golang.org/grpc"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// sseHeartbeatInterval keeps idle SSE connections alive through proxies/browsers.
// ponytail: hard-coded; expose as config if a deployment needs tuning.
const sseHeartbeatInterval = 30 * time.Second

// StreamCertificateEvents handles GET /certificates/stream?owner= — bridges the indexer's
// gRPC server-streaming RPC to a browser-consumable SSE feed. O(1) per forwarded event.
func StreamCertificateEvents(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSONError(w, http.StatusInternalServerError, "streaming unsupported")
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		stream, err := d.Cert.StreamCertificateEvents(ctx, &certv1.StreamCertificateEventsRequest{
			OwnerAddress: r.URL.Query().Get("owner"),
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		events := make(chan *certv1.CertificateEvent)
		errs := make(chan error, 1)
		go recvLoop(ctx, stream, events, errs)

		heartbeat := time.NewTicker(sseHeartbeatInterval)
		defer heartbeat.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errs:
				if err != nil && !errors.Is(err, io.EOF) {
					d.Log.Warn("sse stream ended", "err", err)
				}
				return
			case ev := <-events:
				writeSSEEvent(w, ev)
				flusher.Flush()
			case <-heartbeat.C:
				_, _ = w.Write([]byte(": ping\n\n"))
				flusher.Flush()
			}
		}
	}
}

// recvLoop pumps events off the gRPC stream into a channel, respecting ctx cancellation on
// both send paths so it can never leak a goroutine blocked on an abandoned channel.
func recvLoop(ctx context.Context, stream grpc.ServerStreamingClient[certv1.CertificateEvent], events chan<- *certv1.CertificateEvent, errs chan<- error) {
	for {
		ev, err := stream.Recv()
		if err != nil {
			select {
			case errs <- err:
			case <-ctx.Done():
			}
			return
		}
		select {
		case events <- ev:
		case <-ctx.Done():
			return
		}
	}
}

// writeSSEEvent writes one gRPC CertificateEvent as an SSE "data:" frame.
func writeSSEEvent(w http.ResponseWriter, ev *certv1.CertificateEvent) {
	b, err := json.Marshal(map[string]any{
		"eventType":   ev.GetEventType(),
		"certificate": toDTO(ev.GetCertificate()),
	})
	if err != nil {
		return
	}
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(b)
	_, _ = w.Write([]byte("\n\n"))
}
