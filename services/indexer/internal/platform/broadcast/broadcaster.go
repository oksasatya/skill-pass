// Package broadcast is an in-process pub/sub fan-out for live indexed-certificate events.
// Single-instance only — no Redis (see BE architecture decision: gRPC streaming covers
// realtime at pilot scale; Redis pub/sub joins when the indexer scales past one replica).
package broadcast

import (
	"sync"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
)

// bufferSize bounds each subscriber's channel. Publish is O(k) over k subscribers and never
// blocks: a slow/stalled subscriber drops events rather than stalling the ingest worker.
const bufferSize = 32

// Broadcaster fans out indexed certificates to live subscribers (gRPC stream handlers).
// Safe for concurrent use.
type Broadcaster struct {
	mu   sync.Mutex
	subs map[chan domain.Certificate]struct{}
}

// NewBroadcaster constructs an empty Broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{subs: make(map[chan domain.Certificate]struct{})}
}

// Subscribe registers a new subscriber and returns its channel plus an unsubscribe func.
// Callers MUST call unsubscribe (typically via defer) or the channel and its map entry leak.
func (b *Broadcaster) Subscribe() (<-chan domain.Certificate, func()) {
	ch := make(chan domain.Certificate, bufferSize)

	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs, ch)
			b.mu.Unlock()
			close(ch)
		})
	}
	return ch, unsubscribe
}

// Publish fans c out to every current subscriber, non-blocking. A subscriber whose buffer
// is full drops the event — this is an at-most-once live feed; ListCertificates/GetCertificate
// remain authoritative, so a dropped live update is never data loss, only a missed nudge.
func (b *Broadcaster) Publish(c domain.Certificate) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs {
		select {
		case ch <- c:
		default:
		}
	}
}
