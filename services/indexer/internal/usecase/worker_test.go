package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// fakeEventSource implements usecase.EventSource for tests.
type fakeEventSource struct {
	head    uint64
	logs    map[uint64][]domain.IssuedLog // keyed by fromBlock for simplicity
	certs   map[string]domain.OnchainCertificate
	certErr error // if set, GetCertificate returns this error
}

func (f *fakeEventSource) HeadBlock(_ context.Context) (uint64, error) {
	return f.head, nil
}

func (f *fakeEventSource) BlockHash(_ context.Context, _ uint64) (string, error) {
	return "0xfakehash", nil
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
	certs map[string]domain.Certificate
	state domain.IndexerState
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
	repo.state = domain.IndexerState{ChainID: 31337, LastProcessedBlock: 3, LastProcessedHash: "0xaabbcc"}

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
