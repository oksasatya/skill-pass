package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
	grpcadapter "github.com/oksasatya/skillpass/services/indexer/internal/adapter/grpc"
	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

const bufSize = 1 << 20 // 1 MiB

// --- fakes ---

type fakeRepo struct {
	cert     domain.Certificate
	certErr  error
	listPage usecase.CertificatePage
	listErr  error
	state    domain.IndexerState
	stateErr error
	count    int64
	countErr error
}

func (f *fakeRepo) GetByTokenID(_ context.Context, _ string) (domain.Certificate, error) {
	return f.cert, f.certErr
}
func (f *fakeRepo) List(_ context.Context, _ usecase.ListParams) (usecase.CertificatePage, error) {
	return f.listPage, f.listErr
}
func (f *fakeRepo) Count(_ context.Context) (int64, error) { return f.count, f.countErr }
func (f *fakeRepo) GetState(_ context.Context) (domain.IndexerState, error) {
	return f.state, f.stateErr
}
func (f *fakeRepo) Upsert(_ context.Context, _ domain.Certificate) error       { return nil }
func (f *fakeRepo) SaveState(_ context.Context, _ domain.IndexerState) error   { return nil }
func (f *fakeRepo) DeleteFromBlock(_ context.Context, _ int64, _ uint64) error { return nil }
func (f *fakeRepo) GetIssuanceTrend(_ context.Context, _ usecase.TrendBucket, _ time.Time) ([]usecase.TrendPoint, error) {
	return nil, nil
}

type fakeEventSource struct {
	head    uint64
	headErr error
}

func (f *fakeEventSource) HeadBlock(_ context.Context) (uint64, error) {
	return f.head, f.headErr
}
func (f *fakeEventSource) BlockHash(_ context.Context, _ uint64) (string, error) {
	return "0xfakehash", nil
}
func (f *fakeEventSource) IssuedLogs(_ context.Context, _, _ uint64) ([]domain.IssuedLog, error) {
	return nil, nil
}
func (f *fakeEventSource) GetCertificate(_ context.Context, _ string) (domain.OnchainCertificate, error) {
	return domain.OnchainCertificate{}, nil
}

// fakeSubscriber implements usecase.EventSubscriber for tests — a single-slot Broadcaster
// stand-in so tests can push events directly without going through the Worker.
type fakeSubscriber struct {
	ch chan domain.Certificate
}

func newFakeSubscriber() *fakeSubscriber {
	return &fakeSubscriber{ch: make(chan domain.Certificate, 8)}
}

func (f *fakeSubscriber) Subscribe() (<-chan domain.Certificate, func()) {
	return f.ch, func() {}
}

// --- helpers ---

func dialBufconn(t *testing.T, repo usecase.CertificateRepo, src usecase.EventSource, sub usecase.EventSubscriber, trend *usecase.TrendService) certv1.CertificateQueryClient {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { _ = lis.Close() })

	srv := grpc.NewServer()
	certv1.RegisterCertificateQueryServer(srv, grpcadapter.NewServer(repo, src, sub, trend, nil))
	t.Cleanup(srv.GracefulStop)

	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return certv1.NewCertificateQueryClient(conn)
}

func mustAddr(t *testing.T, s string) domain.Address {
	t.Helper()
	a, err := domain.NewAddress(s)
	if err != nil {
		t.Fatalf("NewAddress(%q): %v", s, err)
	}
	return a
}

// --- GetCertificate ---

func TestGetCertificate_Found(t *testing.T) {
	issuedAt := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	cert := domain.Certificate{
		TokenID:       "42",
		Owner:         mustAddr(t, "0xabcdef0123456789abcdef0123456789abcdef01"),
		Title:         "Go Expert",
		RecipientName: "Alice",
		IssuerName:    "SkillPass",
		TxHash:        "0xdeadbeef",
		BlockNumber:   100,
		IssuedAt:      issuedAt,
		BlockHash:     "0xblockhash",
	}
	repo := &fakeRepo{cert: cert}
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), usecase.NewTrendService(&fakeRepo{}, 1))

	resp, err := client.GetCertificate(context.Background(), &certv1.GetCertificateRequest{TokenId: "42"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := resp.GetCertificate()
	if c.GetTokenId() != "42" {
		t.Errorf("token_id: got %q, want %q", c.GetTokenId(), "42")
	}
	if c.GetOwnerAddress() != "0xabcdef0123456789abcdef0123456789abcdef01" {
		t.Errorf("owner_address: got %q", c.GetOwnerAddress())
	}
	if c.GetTitle() != "Go Expert" {
		t.Errorf("title: got %q", c.GetTitle())
	}
	if c.GetBlockNumber() != 100 {
		t.Errorf("block_number: got %d", c.GetBlockNumber())
	}
	if c.GetIssuedAt().AsTime().UTC() != issuedAt {
		t.Errorf("issued_at: got %v, want %v", c.GetIssuedAt().AsTime().UTC(), issuedAt)
	}
}

func TestGetCertificate_NotFound(t *testing.T) {
	repo := &fakeRepo{certErr: fmt.Errorf("%w: token_id 99", domain.ErrNotFound)}
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), usecase.NewTrendService(&fakeRepo{}, 1))

	_, err := client.GetCertificate(context.Background(), &certv1.GetCertificateRequest{TokenId: "99"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("want codes.NotFound, got %v", status.Code(err))
	}
}

func TestGetCertificate_EmptyTokenID(t *testing.T) {
	client := dialBufconn(t, &fakeRepo{}, &fakeEventSource{}, newFakeSubscriber(), usecase.NewTrendService(&fakeRepo{}, 1))

	_, err := client.GetCertificate(context.Background(), &certv1.GetCertificateRequest{TokenId: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want codes.InvalidArgument, got %v", status.Code(err))
	}
}

func TestGetCertificate_InternalError(t *testing.T) {
	repo := &fakeRepo{certErr: errors.New("db exploded")}
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), usecase.NewTrendService(&fakeRepo{}, 1))

	_, err := client.GetCertificate(context.Background(), &certv1.GetCertificateRequest{TokenId: "1"})
	if status.Code(err) != codes.Internal {
		t.Errorf("want codes.Internal, got %v", status.Code(err))
	}
	// must NOT leak internal error text
	if status.Convert(err).Message() == "db exploded" {
		t.Errorf("internal error text leaked to client")
	}
}

// --- ListCertificates ---

func TestListCertificates_MapsRequest(t *testing.T) {
	addr := "0xabcdef0123456789abcdef0123456789abcdef01"
	cert := domain.Certificate{
		TokenID:     "1",
		Owner:       mustAddr(t, addr),
		Title:       "T",
		IssuerName:  "I",
		TxHash:      "0x1",
		BlockHash:   "0xb",
		BlockNumber: 1,
		IssuedAt:    time.Now(),
	}
	page := usecase.CertificatePage{
		Items:      []domain.Certificate{cert},
		NextCursor: "cursor-abc",
		HasMore:    true,
	}
	repo := &fakeRepo{listPage: page}
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), usecase.NewTrendService(&fakeRepo{}, 1))

	resp, err := client.ListCertificates(context.Background(), &certv1.ListCertificatesRequest{
		OwnerAddress: addr,
		Cursor:       "start",
		PageSize:     10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetCertificates()) != 1 {
		t.Errorf("certificates: got %d, want 1", len(resp.GetCertificates()))
	}
	if resp.GetNextCursor() != "cursor-abc" {
		t.Errorf("next_cursor: got %q", resp.GetNextCursor())
	}
	if !resp.GetHasMore() {
		t.Error("has_more: want true")
	}
}

// --- GetIndexerStatus ---

func TestGetIndexerStatus_LagAndHealthy(t *testing.T) {
	repo := &fakeRepo{
		state: domain.IndexerState{LastProcessedBlock: 90},
		count: 5,
	}
	src := &fakeEventSource{head: 100}
	client := dialBufconn(t, repo, src, newFakeSubscriber(), usecase.NewTrendService(&fakeRepo{}, 1))

	resp, err := client.GetIndexerStatus(context.Background(), &certv1.GetIndexerStatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetLastProcessedBlock() != 90 {
		t.Errorf("last_processed_block: got %d", resp.GetLastProcessedBlock())
	}
	if resp.GetChainHeadBlock() != 100 {
		t.Errorf("chain_head_block: got %d", resp.GetChainHeadBlock())
	}
	if resp.GetIndexLag() != 10 {
		t.Errorf("index_lag: got %d, want 10", resp.GetIndexLag())
	}
	if !resp.GetHealthy() {
		t.Error("healthy: want true (lag=10 < 500)")
	}
	if resp.GetTotalCertificates() != 5 {
		t.Errorf("total_certificates: got %d", resp.GetTotalCertificates())
	}
}

func TestGetIndexerStatus_ChainUnreachable_Degrades(t *testing.T) {
	repo := &fakeRepo{
		state: domain.IndexerState{LastProcessedBlock: 90},
		count: 3,
	}
	src := &fakeEventSource{headErr: errors.New("RPC timeout")}
	client := dialBufconn(t, repo, src, newFakeSubscriber(), usecase.NewTrendService(&fakeRepo{}, 1))

	resp, err := client.GetIndexerStatus(context.Background(), &certv1.GetIndexerStatusRequest{})
	if err != nil {
		// RPC must succeed (degraded, not failed)
		t.Fatalf("unexpected RPC error: %v", err)
	}
	if resp.GetChainHeadBlock() != 0 {
		t.Errorf("chain_head_block: got %d, want 0 (degraded)", resp.GetChainHeadBlock())
	}
	if resp.GetHealthy() {
		t.Error("healthy: want false when chain is unreachable")
	}
}

func TestGetIndexerStatus_LagUnderflowGuard(t *testing.T) {
	// last_processed > head (e.g. reorg/reset)
	repo := &fakeRepo{state: domain.IndexerState{LastProcessedBlock: 200}}
	src := &fakeEventSource{head: 100}
	client := dialBufconn(t, repo, src, newFakeSubscriber(), usecase.NewTrendService(&fakeRepo{}, 1))

	resp, err := client.GetIndexerStatus(context.Background(), &certv1.GetIndexerStatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetIndexLag() != 0 {
		t.Errorf("index_lag underflow: got %d, want 0", resp.GetIndexLag())
	}
}

// --- StreamCertificateEvents ---

func TestStreamCertificateEvents_ForwardsPublishedEvent(t *testing.T) {
	sub := newFakeSubscriber()
	client := dialBufconn(t, &fakeRepo{}, &fakeEventSource{}, sub, usecase.NewTrendService(&fakeRepo{}, 1))

	stream, err := client.StreamCertificateEvents(context.Background(), &certv1.StreamCertificateEventsRequest{})
	if err != nil {
		t.Fatalf("unexpected dial error: %v", err)
	}

	sub.ch <- domain.Certificate{
		TokenID:     "1",
		Owner:       mustAddr(t, "0xabcdef0123456789abcdef0123456789abcdef01"),
		Title:       "T",
		IssuerName:  "I",
		TxHash:      "0x1",
		BlockHash:   "0xb",
		BlockNumber: 1,
		IssuedAt:    time.Now(),
	}

	ev, err := stream.Recv()
	if err != nil {
		t.Fatalf("unexpected recv error: %v", err)
	}
	if ev.GetEventType() != "issued" {
		t.Errorf("event_type: got %q, want issued", ev.GetEventType())
	}
	if ev.GetCertificate().GetTokenId() != "1" {
		t.Errorf("token_id: got %q, want 1", ev.GetCertificate().GetTokenId())
	}
}

func TestStreamCertificateEvents_FiltersByOwner(t *testing.T) {
	const wantOwner = "0xabcdef0123456789abcdef0123456789abcdef01"
	sub := newFakeSubscriber()
	client := dialBufconn(t, &fakeRepo{}, &fakeEventSource{}, sub, usecase.NewTrendService(&fakeRepo{}, 1))

	stream, err := client.StreamCertificateEvents(context.Background(), &certv1.StreamCertificateEventsRequest{OwnerAddress: wantOwner})
	if err != nil {
		t.Fatalf("unexpected dial error: %v", err)
	}

	// other owner — must be filtered out
	sub.ch <- domain.Certificate{
		TokenID: "1", Owner: mustAddr(t, "0x1111111111111111111111111111111111111111"),
		Title: "T", IssuerName: "I", TxHash: "0x1", BlockHash: "0xb", IssuedAt: time.Now(),
	}
	// matching owner — must be delivered
	sub.ch <- domain.Certificate{
		TokenID: "2", Owner: mustAddr(t, wantOwner),
		Title: "T", IssuerName: "I", TxHash: "0x2", BlockHash: "0xb", IssuedAt: time.Now(),
	}

	ev, err := stream.Recv()
	if err != nil {
		t.Fatalf("unexpected recv error: %v", err)
	}
	if ev.GetCertificate().GetTokenId() != "2" {
		t.Errorf("token_id: got %q, want 2 (owner-filtered)", ev.GetCertificate().GetTokenId())
	}
}

func TestStreamCertificateEvents_StopsOnClientCancel(t *testing.T) {
	sub := newFakeSubscriber()
	client := dialBufconn(t, &fakeRepo{}, &fakeEventSource{}, sub, usecase.NewTrendService(&fakeRepo{}, 1))

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := client.StreamCertificateEvents(ctx, &certv1.StreamCertificateEventsRequest{})
	if err != nil {
		t.Fatalf("unexpected dial error: %v", err)
	}
	cancel()

	_, recvErr := stream.Recv()
	if status.Code(recvErr) != codes.Canceled {
		t.Errorf("want codes.Canceled, got %v", status.Code(recvErr))
	}
}

// --- GetIssuanceTrend ---

func TestGetIssuanceTrend_Valid(t *testing.T) {
	repo := &fakeRepo{} // GetIssuanceTrend on fakeRepo returns (nil, nil) by default — added above
	trend := usecase.NewTrendService(repo, 31337)
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), trend)

	resp, err := client.GetIssuanceTrend(context.Background(), &certv1.GetIssuanceTrendRequest{
		Bucket:      certv1.TrendBucket_TREND_BUCKET_DAY,
		RangePreset: "30d",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetPoints()) == 0 {
		t.Fatal("expected zero-filled points for a 30d range, got none")
	}
}

func TestGetIssuanceTrend_InvalidPresetForBucket(t *testing.T) {
	repo := &fakeRepo{}
	trend := usecase.NewTrendService(repo, 31337)
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), trend)

	_, err := client.GetIssuanceTrend(context.Background(), &certv1.GetIssuanceTrendRequest{
		Bucket:      certv1.TrendBucket_TREND_BUCKET_WEEK,
		RangePreset: "30d", // not a valid preset for WEEK
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want codes.InvalidArgument, got %v", status.Code(err))
	}
}

func TestGetIssuanceTrend_UnspecifiedBucket(t *testing.T) {
	repo := &fakeRepo{}
	trend := usecase.NewTrendService(repo, 31337)
	client := dialBufconn(t, repo, &fakeEventSource{}, newFakeSubscriber(), trend)

	_, err := client.GetIssuanceTrend(context.Background(), &certv1.GetIssuanceTrendRequest{
		Bucket: certv1.TrendBucket_TREND_BUCKET_UNSPECIFIED,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want codes.InvalidArgument, got %v", status.Code(err))
	}
}
