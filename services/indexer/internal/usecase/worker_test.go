package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// fakeEventSource implements usecase.EventSource for tests.
type fakeEventSource struct {
	head        uint64
	logs        map[uint64][]domain.IssuedLog // keyed by fromBlock for simplicity
	certs       map[string]domain.OnchainCertificate
	certErr     error             // if set, GetCertificate returns this error
	blockHashes map[uint64]string // canonical hash override per block; deterministic default if unset
}

func (f *fakeEventSource) HeadBlock(_ context.Context) (uint64, error) {
	return f.head, nil
}

func (f *fakeEventSource) BlockHash(_ context.Context, blockNumber uint64) (string, error) {
	if h, ok := f.blockHashes[blockNumber]; ok {
		return h, nil
	}
	return fmt.Sprintf("0xcanonical%d", blockNumber), nil
}

func (f *fakeEventSource) IssuedLogs(_ context.Context, from, to uint64) ([]domain.IssuedLog, error) {
	var out []domain.IssuedLog
	for b := from; b <= to; b++ {
		out = append(out, f.logs[b]...)
	}
	return out, nil
}

func (f *fakeEventSource) GetCertificate(_ context.Context, tokenID string) (domain.OnchainCertificate, error) {
	if f.certErr != nil {
		return domain.OnchainCertificate{}, f.certErr
	}
	c, ok := f.certs[tokenID]
	if !ok {
		return domain.OnchainCertificate{}, errors.New("fake: cert not found")
	}
	return c, nil
}

// fakeRepo implements usecase.CertificateRepo for tests.
type fakeRepo struct {
	certs     map[string]domain.Certificate
	state     domain.IndexerState
	outbox    map[string]int64 // "txHash:tokenID" -> assigned id
	outboxSeq int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{certs: make(map[string]domain.Certificate)}
}

func (r *fakeRepo) Upsert(_ context.Context, c domain.Certificate) error {
	r.certs[c.TokenID] = c
	return nil
}

func (r *fakeRepo) GetByTokenID(_ context.Context, id string) (domain.Certificate, error) {
	c, ok := r.certs[id]
	if !ok {
		return domain.Certificate{}, domain.ErrNotFound
	}
	return c, nil
}

func (r *fakeRepo) List(_ context.Context, _ usecase.ListParams) (usecase.CertificatePage, error) {
	return usecase.CertificatePage{}, nil
}

func (r *fakeRepo) Count(_ context.Context) (int64, error) {
	return int64(len(r.certs)), nil
}

func (r *fakeRepo) GetState(_ context.Context) (domain.IndexerState, error) {
	return r.state, nil
}

func (r *fakeRepo) SaveState(_ context.Context, s domain.IndexerState) error {
	r.state = s
	return nil
}

func (r *fakeRepo) DeleteFromBlock(_ context.Context, _ int64, blockNumber uint64) error {
	// delete all certs with block_number >= blockNumber
	for id, c := range r.certs {
		if uint64(c.BlockNumber) >= blockNumber {
			delete(r.certs, id)
		}
	}
	return nil
}

// GetIssuanceTrend is unused by worker tests; stubbed only to satisfy usecase.CertificateRepo.
func (r *fakeRepo) GetIssuanceTrend(_ context.Context, _ usecase.TrendBucket, _ time.Time) ([]usecase.TrendPoint, error) {
	return nil, nil
}

// InsertWebhookOutbox fakes the (chain_id, tx_hash, token_id) dedup for tests.
func (r *fakeRepo) InsertWebhookOutbox(_ context.Context, _ int64, txHash, tokenID string, _ []byte) (int64, bool, error) {
	if r.outbox == nil {
		r.outbox = make(map[string]int64)
	}
	key := txHash + ":" + tokenID
	if _, exists := r.outbox[key]; exists {
		return 0, false, nil
	}
	r.outboxSeq++
	r.outbox[key] = r.outboxSeq
	return r.outboxSeq, true, nil
}

// ListUnenqueuedWebhookOutbox is unused by worker tests; stubbed only to satisfy the port.
func (r *fakeRepo) ListUnenqueuedWebhookOutbox(_ context.Context, _ int) ([]usecase.WebhookOutboxEntry, error) {
	return nil, nil
}

// MarkWebhookOutboxEnqueued is unused by worker tests; stubbed only to satisfy the port.
func (r *fakeRepo) MarkWebhookOutboxEnqueued(_ context.Context, _ int64) error {
	return nil
}

// helpers

func mustAddr(s string) domain.Address {
	a, err := domain.NewAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}

func sampleLog(tokenID string, block uint64) domain.IssuedLog {
	return domain.IssuedLog{
		TokenID:     tokenID,
		BlockNumber: block,
		BlockHash:   "0xaabbcc",
		TxHash:      "0xdeadbeef",
		LogIndex:    0,
	}
}

func sampleCert(tokenID string) domain.OnchainCertificate {
	return domain.OnchainCertificate{
		TokenID:       tokenID,
		Owner:         mustAddr("0x1234567890123456789012345678901234567890"),
		Title:         "Go Expert",
		RecipientName: "Alice",
		IssuerName:    "Skillpass",
		Description:   "Backend cert",
		MetadataURI:   "ipfs://Qm...",
		IssuedAt:      time.Unix(1700000000, 0).UTC(),
	}
}

func newWorker(src usecase.EventSource, repo usecase.CertificateRepo) *usecase.Worker {
	return usecase.NewWorker(src, repo, usecase.WorkerConfig{
		ChainID:      31337,
		StartBlock:   0,
		BatchSize:    100,
		PollInterval: time.Hour, // large so ticker never fires in tests
	}, nil)
}

// --- tests ---

func TestWorker_ColdStart(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 5,
		logs: map[uint64][]domain.IssuedLog{
			1: {sampleLog("1", 1)},
			3: {sampleLog("2", 3)},
		},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
			"2": sampleCert("2"),
		},
	}
	w := newWorker(src, repo)
	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(repo.certs) != 2 {
		t.Fatalf("want 2 certs, got %d", len(repo.certs))
	}
	if repo.state.LastProcessedBlock != 5 {
		t.Fatalf("want state.LastProcessedBlock=5, got %d", repo.state.LastProcessedBlock)
	}
}

func TestWorker_Resume(t *testing.T) {
	repo := newFakeRepo()
	repo.state = domain.IndexerState{ChainID: 31337, LastProcessedBlock: 3, LastProcessedHash: "0xcanonical3"}

	src := &fakeEventSource{
		head: 7,
		logs: map[uint64][]domain.IssuedLog{
			// blocks 0-3 should NOT be reprocessed
			1: {sampleLog("99", 1)},
			4: {sampleLog("4", 4)},
		},
		certs: map[string]domain.OnchainCertificate{
			"99": sampleCert("99"),
			"4":  sampleCert("4"),
		},
	}
	w := newWorker(src, repo)
	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	// tokenID "99" at block 1 must NOT be present (it's before resume point)
	if _, ok := repo.certs["99"]; ok {
		t.Fatal("should not have processed block 1 after resume at block 3")
	}
	if _, ok := repo.certs["4"]; !ok {
		t.Fatal("should have processed block 4")
	}
	if repo.state.LastProcessedBlock != 7 {
		t.Fatalf("want state=7, got %d", repo.state.LastProcessedBlock)
	}
}

func TestWorker_IdempotentReplay(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{
			1: {sampleLog("1", 1)},
		},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	w := newWorker(src, repo)
	// poll twice with the same head (state resets to allow replay)
	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("first poll: %v", err)
	}
	// reset state so second poll re-processes the same range
	repo.state = domain.IndexerState{}
	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("second poll: %v", err)
	}
	// upsert is idempotent — still exactly 1 cert
	if len(repo.certs) != 1 {
		t.Fatalf("want 1 cert after replay, got %d", len(repo.certs))
	}
}

func TestWorker_EmptyRange(t *testing.T) {
	repo := newFakeRepo()
	repo.state = domain.IndexerState{ChainID: 31337, LastProcessedBlock: 10}
	src := &fakeEventSource{head: 10} // head == last → nothing to do
	w := newWorker(src, repo)
	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(repo.certs) != 0 {
		t.Fatal("no certs should be processed when head == last")
	}
	// state should remain at 10, not advance
	if repo.state.LastProcessedBlock != 10 {
		t.Fatalf("state should stay at 10, got %d", repo.state.LastProcessedBlock)
	}
}

func TestWorker_GetCertificateError_StateNotAdvanced(t *testing.T) {
	repo := newFakeRepo()
	certErr := errors.New("rpc timeout")
	src := &fakeEventSource{
		head: 5,
		logs: map[uint64][]domain.IssuedLog{
			1: {sampleLog("1", 1)},
		},
		certErr: certErr,
	}
	w := newWorker(src, repo)
	err := w.Poll(t.Context())
	if err == nil {
		t.Fatal("want error when GetCertificate fails")
	}
	// state must NOT have advanced
	if repo.state.LastProcessedBlock != 0 {
		t.Fatalf("state must not advance on error, got %d", repo.state.LastProcessedBlock)
	}
}

func TestWorker_CtxCancel(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{head: 0}
	w := usecase.NewWorker(src, repo, usecase.WorkerConfig{
		ChainID:      31337,
		StartBlock:   0,
		BatchSize:    100,
		PollInterval: time.Millisecond, // tiny so at least one tick fires
	}, nil)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after ctx cancel")
	}
}

// fakePublisher implements usecase.EventPublisher for tests.
type fakePublisher struct {
	published []domain.Certificate
}

func (f *fakePublisher) Publish(c domain.Certificate) {
	f.published = append(f.published, c)
}

func TestWorker_PublishesOnSuccessfulUpsert(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	pub := &fakePublisher{}
	w := newWorker(src, repo)
	w.SetPublisher(pub)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(pub.published) != 1 || pub.published[0].TokenID != "1" {
		t.Fatalf("want 1 published cert with TokenID=1, got %+v", pub.published)
	}
}

func TestWorker_NilPublisher_NoPanic(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	w := newWorker(src, repo) // SetPublisher never called
	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
}

func TestWorker_Reconcile_NoReorg_IsNoop(t *testing.T) {
	repo := newFakeRepo()
	repo.state = domain.IndexerState{ChainID: 31337, LastProcessedBlock: 10, LastProcessedHash: "0xcanonical10"}
	repo.certs["1"] = domain.Certificate{TokenID: "1", BlockNumber: 5}

	src := &fakeEventSource{head: 10} // BlockHash(10) defaults to "0xcanonical10" — matches stored
	w := newWorker(src, repo)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if _, ok := repo.certs["1"]; !ok {
		t.Fatal("no reorg should have occurred — cert must survive")
	}
}

func TestWorker_Reconcile_DetectsReorgAndRewinds(t *testing.T) {
	repo := newFakeRepo()
	repo.state = domain.IndexerState{ChainID: 31337, LastProcessedBlock: 20, LastProcessedHash: "0xstale-hash"}
	repo.certs["1"] = domain.Certificate{TokenID: "1", BlockNumber: 5}  // below the rewind window — survives
	repo.certs["2"] = domain.Certificate{TokenID: "2", BlockNumber: 15} // within [20-12+1, 20] — deleted

	src := &fakeEventSource{
		head: 20,
		blockHashes: map[uint64]string{
			20: "0xcanonical20", // mismatches stored "0xstale-hash" -> reorg detected
			8:  "0xcanonical8",  // rewindTo = 20-12 = 8
		},
		// no logs configured for [9,20] — nothing re-appears there in this test
	}
	w := newWorker(src, repo)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}

	if _, ok := repo.certs["1"]; !ok {
		t.Fatal("token 1 (block 5, below rewind point) must survive")
	}
	if _, ok := repo.certs["2"]; ok {
		t.Fatal("token 2 (block 15, within rewound window) must be deleted")
	}
	// reconcile() rewinds the checkpoint to 8, but poll() does not return early — it falls
	// straight through to the normal fetch-and-advance logic using the now-rewound w.next,
	// re-scanning [9,20] in the SAME Poll() call (no logs there in this test, so nothing is
	// re-added) and advancing the checkpoint to head. This converges in one poll cycle
	// instead of requiring an extra tick — the final observable state is at head, not at
	// the intermediate rewound point.
	if repo.state.LastProcessedBlock != 20 {
		t.Fatalf("state.LastProcessedBlock = %d, want 20 (rewound to 8, then re-scanned forward to head in the same Poll() call)", repo.state.LastProcessedBlock)
	}
	if repo.state.LastProcessedHash != "0xcanonical20" {
		t.Fatalf("state.LastProcessedHash = %q, want the canonical hash of head (20)", repo.state.LastProcessedHash)
	}
}

func TestWorker_Reconcile_ColdStart_IsNoop(t *testing.T) {
	repo := newFakeRepo() // zero-value state: LastProcessedHash == ""
	src := &fakeEventSource{head: 0}
	w := newWorker(src, repo)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	// must not panic or attempt a delete on an uninitialized checkpoint
}

// fakeEnqueuer implements usecase.TaskEnqueuer for tests.
type fakeEnqueuer struct {
	enqueued []string // taskType per call
	payloads [][]byte // payload per call, parallel to enqueued
}

func (f *fakeEnqueuer) EnqueueUnique(_ context.Context, taskType, _ string, payload []byte) error {
	f.enqueued = append(f.enqueued, taskType)
	f.payloads = append(f.payloads, payload)
	return nil
}

func TestWorker_EnqueuesTrendRefresh_OnSuccessfulUpsert(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	enq := &fakeEnqueuer{}
	w := newWorker(src, repo)
	w.SetEnqueuer(enq)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if countTaskType(enq.enqueued, usecase.TrendRefreshTaskType) != 1 {
		t.Fatalf("want 1 enqueue of %q, got %v", usecase.TrendRefreshTaskType, enq.enqueued)
	}
}

// countTaskType counts occurrences of taskType in enqueued -- Task 4 adds a second
// enqueue call (webhook:deliver) alongside trend:refresh, so exact-length assertions on
// the whole slice are no longer meaningful; count the specific type instead.
func countTaskType(enqueued []string, taskType string) int {
	n := 0
	for _, t := range enqueued {
		if t == taskType {
			n++
		}
	}
	return n
}

func TestWorker_EnqueuesWebhookDeliver_OnNewCertificate(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	enq := &fakeEnqueuer{}
	w := newWorker(src, repo)
	w.SetEnqueuer(enq)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if countTaskType(enq.enqueued, usecase.WebhookDeliverTaskType) != 1 {
		t.Fatalf("want 1 enqueue of %q, got %v", usecase.WebhookDeliverTaskType, enq.enqueued)
	}
}

func TestWorker_DoesNotReenqueueWebhook_OnIdempotentReplay(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	enq := &fakeEnqueuer{}
	w := newWorker(src, repo)
	w.SetEnqueuer(enq)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("first poll: %v", err)
	}
	if got := countTaskType(enq.enqueued, usecase.WebhookDeliverTaskType); got != 1 {
		t.Fatalf("want 1 webhook enqueue after first poll, got %d", got)
	}

	// Simulate a reorg replay of the same certificate: a fresh Worker instance re-derives
	// its resume point from repo.state (reset below), so it genuinely re-fetches and
	// re-processes block 1's log through processLog/enqueueWebhook/InsertWebhookOutbox --
	// exercising the outbox dedup path for real. Reusing the SAME Worker instance would not
	// do this: its in-memory w.next cursor already advanced past head after the first poll
	// and isn't reset by mutating the fake repo's state, so a second Poll() on it would
	// short-circuit before ever reaching processLog again -- a vacuous test that passes for
	// the wrong reason. Sharing the same repo (and its outbox map) and the same enqueuer
	// across w and w2 is what lets this test observe whether the dedup key actually
	// prevented a second enqueue.
	repo.state = domain.IndexerState{}
	w2 := newWorker(src, repo)
	w2.SetEnqueuer(enq)
	if err := w2.Poll(t.Context()); err != nil {
		t.Fatalf("second (replay) poll: %v", err)
	}
	if got := countTaskType(enq.enqueued, usecase.WebhookDeliverTaskType); got != 1 {
		t.Fatalf("want still only 1 webhook enqueue after replay (outbox dedup), got %d", got)
	}
}

// fakeFailingOutboxRepo wraps fakeRepo but makes InsertWebhookOutbox always fail --
// verifies processLog propagates the error (fatal) rather than swallowing it.
type fakeFailingOutboxRepo struct {
	*fakeRepo
}

func (r *fakeFailingOutboxRepo) InsertWebhookOutbox(_ context.Context, _ int64, _, _ string, _ []byte) (int64, bool, error) {
	return 0, false, errors.New("fake: outbox insert failed")
}

func TestWorker_InsertWebhookOutboxError_IsFatal_StateNotAdvanced(t *testing.T) {
	repo := &fakeFailingOutboxRepo{fakeRepo: newFakeRepo()}
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	w := newWorker(src, repo)
	w.SetEnqueuer(&fakeEnqueuer{})

	err := w.Poll(t.Context())
	if err == nil {
		t.Fatal("want an error when InsertWebhookOutbox fails")
	}
	if repo.state.LastProcessedBlock != 0 {
		t.Fatalf("state must not advance when outbox insert fails, got %d", repo.state.LastProcessedBlock)
	}
}

func TestWorker_NilEnqueuer_NoPanic(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 1,
		logs: map[uint64][]domain.IssuedLog{1: {sampleLog("1", 1)}},
		certs: map[string]domain.OnchainCertificate{
			"1": sampleCert("1"),
		},
	}
	w := newWorker(src, repo) // SetEnqueuer never called
	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
}

func TestWorker_ChecksInCanonicalHash_NotLastLogHash(t *testing.T) {
	repo := newFakeRepo()
	src := &fakeEventSource{
		head: 5,
		logs: map[uint64][]domain.IssuedLog{
			1: {sampleLog("1", 1)}, // this log's own BlockHash field is "0xaabbcc" (see sampleLog)
		},
		certs:       map[string]domain.OnchainCertificate{"1": sampleCert("1")},
		blockHashes: map[uint64]string{5: "0xcanonical-head-5"},
	}
	w := newWorker(src, repo)

	if err := w.Poll(t.Context()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if repo.state.LastProcessedHash != "0xcanonical-head-5" {
		t.Fatalf("state.LastProcessedHash = %q, want the canonical head hash, not the log's own block hash (0xaabbcc)", repo.state.LastProcessedHash)
	}
}
