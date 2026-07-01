package broadcast_test

import (
	"testing"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
	"github.com/oksasatya/skillpass/services/indexer/internal/platform/broadcast"
)

func mustAddr(t *testing.T, s string) domain.Address {
	t.Helper()
	a, err := domain.NewAddress(s)
	if err != nil {
		t.Fatalf("NewAddress(%q): %v", s, err)
	}
	return a
}

func sampleCert(t *testing.T, tokenID string) domain.Certificate {
	t.Helper()
	return domain.Certificate{
		TokenID:     tokenID,
		Owner:       mustAddr(t, "0xabcdef0123456789abcdef0123456789abcdef01"),
		Title:       "T",
		IssuerName:  "I",
		TxHash:      "0x1",
		BlockHash:   "0xb",
		BlockNumber: 1,
		IssuedAt:    time.Now(),
	}
}

func TestBroadcaster_DeliversToSubscriber(t *testing.T) {
	b := broadcast.NewBroadcaster()
	ch, unsubscribe := b.Subscribe()
	defer unsubscribe()

	want := sampleCert(t, "1")
	b.Publish(want)

	select {
	case got := <-ch:
		if got.TokenID != want.TokenID {
			t.Errorf("TokenID = %q, want %q", got.TokenID, want.TokenID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestBroadcaster_FanOutToMultipleSubscribers(t *testing.T) {
	b := broadcast.NewBroadcaster()
	ch1, unsub1 := b.Subscribe()
	defer unsub1()
	ch2, unsub2 := b.Subscribe()
	defer unsub2()

	b.Publish(sampleCert(t, "1"))

	for _, ch := range []<-chan domain.Certificate{ch1, ch2} {
		select {
		case got := <-ch:
			if got.TokenID != "1" {
				t.Errorf("TokenID = %q, want 1", got.TokenID)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for fan-out delivery")
		}
	}
}

func TestBroadcaster_UnsubscribeStopsDelivery(t *testing.T) {
	b := broadcast.NewBroadcaster()
	ch, unsubscribe := b.Subscribe()
	unsubscribe()

	b.Publish(sampleCert(t, "1")) // must not panic (send on closed channel) or block

	if _, ok := <-ch; ok {
		t.Fatal("channel should be closed after unsubscribe")
	}
}

func TestBroadcaster_UnsubscribeIsIdempotent(t *testing.T) {
	b := broadcast.NewBroadcaster()
	_, unsubscribe := b.Subscribe()
	unsubscribe()
	unsubscribe() // must not panic (double close)
}

func TestBroadcaster_FullBufferDropsWithoutBlocking(t *testing.T) {
	b := broadcast.NewBroadcaster()
	_, unsubscribe := b.Subscribe() // never drained
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		for range 1000 {
			b.Publish(sampleCert(t, "1"))
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a full subscriber buffer")
	}
}
