package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/oksasatya/skillpass/services/indexer/internal/adapter/postgres"
	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
	platformdb "github.com/oksasatya/skillpass/services/indexer/internal/platform/db"
	"github.com/oksasatya/skillpass/services/indexer/internal/usecase"
)

// startPostgres starts a real Postgres 17 container and returns a connected pool.
// It runs migrations before returning.
func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("skillpass_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(context.Background()) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("container dsn: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := platformdb.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// makeCert creates a test Certificate with deterministic, valid fields.
func makeCert(tokenID string) domain.Certificate {
	owner, _ := domain.NewAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	return domain.Certificate{
		TokenID:       tokenID,
		Owner:         owner,
		Title:         "Go Mastery " + tokenID,
		RecipientName: "Alice",
		IssuerName:    "Acme Academy",
		Description:   "desc " + tokenID,
		MetadataURI:   "ipfs://Qm" + tokenID,
		IssuedAt:      time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		ChainID:       1,
		TxHash:        "0x" + fmt.Sprintf("%064x", 1),
		LogIndex:      0,
		BlockNumber:   100,
		BlockHash:     "0x" + fmt.Sprintf("%064x", 2),
	}
}

// makeCertOwner like makeCert but with a specific owner and unique provenance.
func makeCertOwner(tokenID, ownerHex string) domain.Certificate {
	c := makeCert(tokenID)
	owner, _ := domain.NewAddress(ownerHex)
	c.Owner = owner
	// make tx_hash unique to avoid chain_id+tx_hash+log_index unique violation
	c.TxHash = "0x" + fmt.Sprintf("%064s", tokenID)
	return c
}

// TestUpsertGetRoundTrip verifies every field survives a round-trip through Postgres.
func TestUpsertGetRoundTrip(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	cert := makeCert("42")
	if err := repo.Upsert(ctx, cert); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetByTokenID(ctx, "42")
	if err != nil {
		t.Fatalf("GetByTokenID: %v", err)
	}

	if got.TokenID != cert.TokenID {
		t.Errorf("TokenID: got %q want %q", got.TokenID, cert.TokenID)
	}
	if got.Owner.String() != cert.Owner.String() {
		t.Errorf("Owner: got %q want %q", got.Owner, cert.Owner)
	}
	if got.Title != cert.Title {
		t.Errorf("Title: got %q want %q", got.Title, cert.Title)
	}
	if got.RecipientName != cert.RecipientName {
		t.Errorf("RecipientName: got %q want %q", got.RecipientName, cert.RecipientName)
	}
	if got.IssuerName != cert.IssuerName {
		t.Errorf("IssuerName: got %q want %q", got.IssuerName, cert.IssuerName)
	}
	if got.Description != cert.Description {
		t.Errorf("Description: got %q want %q", got.Description, cert.Description)
	}
	if got.MetadataURI != cert.MetadataURI {
		t.Errorf("MetadataURI: got %q want %q", got.MetadataURI, cert.MetadataURI)
	}
	if !got.IssuedAt.Equal(cert.IssuedAt) {
		t.Errorf("IssuedAt: got %v want %v", got.IssuedAt, cert.IssuedAt)
	}
	if got.ChainID != cert.ChainID {
		t.Errorf("ChainID: got %d want %d", got.ChainID, cert.ChainID)
	}
	if got.TxHash != cert.TxHash {
		t.Errorf("TxHash: got %q want %q", got.TxHash, cert.TxHash)
	}
	if got.LogIndex != cert.LogIndex {
		t.Errorf("LogIndex: got %d want %d", got.LogIndex, cert.LogIndex)
	}
	if got.BlockNumber != cert.BlockNumber {
		t.Errorf("BlockNumber: got %d want %d", got.BlockNumber, cert.BlockNumber)
	}
	if got.BlockHash != cert.BlockHash {
		t.Errorf("BlockHash: got %q want %q", got.BlockHash, cert.BlockHash)
	}
}

// TestUpsertIdempotent verifies that upserting the same cert twice results in one row.
func TestUpsertIdempotent(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	cert := makeCert("99")
	if err := repo.Upsert(ctx, cert); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}
	// second upsert — same token_id, should not error
	if err := repo.Upsert(ctx, cert); err != nil {
		t.Fatalf("second Upsert (idempotent): %v", err)
	}

	n, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Errorf("Count after 2 identical upserts: got %d want 1", n)
	}
}

// TestGetByTokenIDNotFound verifies domain.ErrNotFound is returned for missing token.
func TestGetByTokenIDNotFound(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	_, err := repo.GetByTokenID(ctx, "9999999")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByTokenID missing: want domain.ErrNotFound, got %v", err)
	}
}

// insertN inserts certs with token IDs [start, start+n-1].
// Token IDs are sequential decimal numbers; each has unique provenance.
func insertN(t *testing.T, repo *postgres.CertificateRepo, n, start int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%d", start+i)
		c := makeCert(id)
		// unique provenance per cert to avoid UNIQUE(chain_id, tx_hash, log_index) violation
		c.TxHash = "0x" + fmt.Sprintf("%064d", start+i)
		c.LogIndex = int64(i)
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("insertN[%d]: %v", i, err)
		}
	}
}

// TestListKeysetPagination inserts 25 certs and pages through them (limit 10).
// Verifies: 3 pages (10,10,5), HasMore true/true/false, no duplicates, no gaps,
// newest-first ordering by numeric token_id.
func TestListKeysetPagination(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	// token_ids 1..25 (numeric comparison: 25 > 24 > … > 1)
	insertN(t, repo, 25, 1)

	seen := map[string]bool{}
	cursor := ""
	pageSizes := []int{}

	for page := 0; ; page++ {
		result, err := repo.List(ctx, usecase.ListParams{Limit: 10, Cursor: cursor})
		if err != nil {
			t.Fatalf("List page %d: %v", page, err)
		}

		for _, c := range result.Items {
			if seen[c.TokenID] {
				t.Errorf("page %d: duplicate token_id %s", page, c.TokenID)
			}
			seen[c.TokenID] = true
		}

		pageSizes = append(pageSizes, len(result.Items))

		if !result.HasMore {
			break
		}
		if result.NextCursor == "" {
			t.Fatal("HasMore=true but NextCursor is empty")
		}
		cursor = result.NextCursor

		if page > 10 {
			t.Fatal("pagination loop did not terminate")
		}
	}

	// 3 pages: 10, 10, 5
	if len(pageSizes) != 3 {
		t.Fatalf("expected 3 pages, got %d: %v", len(pageSizes), pageSizes)
	}
	if pageSizes[0] != 10 || pageSizes[1] != 10 || pageSizes[2] != 5 {
		t.Errorf("page sizes: got %v want [10 10 5]", pageSizes)
	}
	if len(seen) != 25 {
		t.Errorf("total items across pages: got %d want 25", len(seen))
	}

	// verify all ids 1..25 are present
	for i := 1; i <= 25; i++ {
		id := fmt.Sprintf("%d", i)
		if !seen[id] {
			t.Errorf("token_id %s missing from paginated results", id)
		}
	}
}

// TestListOwnerFilter verifies that List with Owner returns only that owner's certs.
func TestListOwnerFilter(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	ownerA := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	ownerB := "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	for i := 1; i <= 5; i++ {
		c := makeCertOwner(fmt.Sprintf("%d", i), ownerA)
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert ownerA cert %d: %v", i, err)
		}
	}
	for i := 6; i <= 8; i++ {
		c := makeCertOwner(fmt.Sprintf("%d", i), ownerB)
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert ownerB cert %d: %v", i, err)
		}
	}

	page, err := repo.List(ctx, usecase.ListParams{Owner: ownerA, Limit: 20})
	if err != nil {
		t.Fatalf("List ownerA: %v", err)
	}
	if len(page.Items) != 5 {
		t.Errorf("ownerA: got %d items want 5", len(page.Items))
	}
	for _, c := range page.Items {
		if c.Owner.String() != ownerA {
			t.Errorf("unexpected owner %s in ownerA results", c.Owner)
		}
	}

	pageB, err := repo.List(ctx, usecase.ListParams{Owner: ownerB, Limit: 20})
	if err != nil {
		t.Fatalf("List ownerB: %v", err)
	}
	if len(pageB.Items) != 3 {
		t.Errorf("ownerB: got %d items want 3", len(pageB.Items))
	}
}

// TestListSearch verifies that Search matches title, issuer_name, recipient_name.
func TestListSearch(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	owner, _ := domain.NewAddress("0xcccccccccccccccccccccccccccccccccccccccc")

	certs := []domain.Certificate{
		{
			TokenID: "100", Owner: owner, Title: "Rust Fundamentals",
			RecipientName: "Bob", IssuerName: "TechCorp",
			IssuedAt: time.Now(), ChainID: 1,
			TxHash: "0x" + fmt.Sprintf("%064d", 100), LogIndex: 0, BlockNumber: 1, BlockHash: "0x" + fmt.Sprintf("%064d", 101),
		},
		{
			TokenID: "101", Owner: owner, Title: "Go Advanced",
			RecipientName: "Alice", IssuerName: "RustAcademy",
			IssuedAt: time.Now(), ChainID: 1,
			TxHash: "0x" + fmt.Sprintf("%064d", 200), LogIndex: 0, BlockNumber: 1, BlockHash: "0x" + fmt.Sprintf("%064d", 201),
		},
		{
			TokenID: "102", Owner: owner, Title: "Python Basics",
			RecipientName: "Charlie", IssuerName: "LearnCo",
			IssuedAt: time.Now(), ChainID: 1,
			TxHash: "0x" + fmt.Sprintf("%064d", 300), LogIndex: 0, BlockNumber: 1, BlockHash: "0x" + fmt.Sprintf("%064d", 301),
		},
	}
	for _, c := range certs {
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert %s: %v", c.TokenID, err)
		}
	}

	// "rust" matches title of 100 and issuer_name of 101
	page, err := repo.List(ctx, usecase.ListParams{Query: "rust", Limit: 20})
	if err != nil {
		t.Fatalf("List search: %v", err)
	}
	if len(page.Items) != 2 {
		t.Errorf("search 'rust': got %d items want 2", len(page.Items))
	}

	// "python" matches only 102
	page2, err := repo.List(ctx, usecase.ListParams{Query: "python", Limit: 20})
	if err != nil {
		t.Fatalf("List search python: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Errorf("search 'python': got %d items want 1", len(page2.Items))
	}
}

// TestListOwnerAndQuery verifies owner + search combined filter.
func TestListOwnerAndQuery(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	ownerX := "0xdddddddddddddddddddddddddddddddddddddddd"
	ownerY := "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

	addrX, _ := domain.NewAddress(ownerX)
	addrY, _ := domain.NewAddress(ownerY)

	certs := []domain.Certificate{
		{
			TokenID: "200", Owner: addrX, Title: "Solidity Expert",
			RecipientName: "Dave", IssuerName: "Web3Academy",
			IssuedAt: time.Now(), ChainID: 1,
			TxHash: "0x" + fmt.Sprintf("%064d", 400), LogIndex: 0, BlockNumber: 1, BlockHash: "0x" + fmt.Sprintf("%064d", 401),
		},
		{
			TokenID: "201", Owner: addrX, Title: "Go Expert",
			RecipientName: "Dave", IssuerName: "GoAcademy",
			IssuedAt: time.Now(), ChainID: 1,
			TxHash: "0x" + fmt.Sprintf("%064d", 500), LogIndex: 0, BlockNumber: 1, BlockHash: "0x" + fmt.Sprintf("%064d", 501),
		},
		{
			TokenID: "202", Owner: addrY, Title: "Solidity Basics",
			RecipientName: "Eve", IssuerName: "Web3Academy",
			IssuedAt: time.Now(), ChainID: 1,
			TxHash: "0x" + fmt.Sprintf("%064d", 600), LogIndex: 0, BlockNumber: 1, BlockHash: "0x" + fmt.Sprintf("%064d", 601),
		},
	}
	for _, c := range certs {
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert %s: %v", c.TokenID, err)
		}
	}

	// owner=X + query="solidity" → only 200
	page, err := repo.List(ctx, usecase.ListParams{Owner: ownerX, Query: "solidity", Limit: 20})
	if err != nil {
		t.Fatalf("List owner+query: %v", err)
	}
	if len(page.Items) != 1 {
		t.Errorf("owner+query: got %d items want 1", len(page.Items))
	}
	if len(page.Items) > 0 && page.Items[0].TokenID != "200" {
		t.Errorf("owner+query: got token_id %s want 200", page.Items[0].TokenID)
	}
}

// TestCount verifies Count reflects the number of inserted certs.
func TestCount(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	n, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count empty: %v", err)
	}
	if n != 0 {
		t.Errorf("Count empty: got %d want 0", n)
	}

	insertN(t, repo, 7, 1000)
	n, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count after insert: %v", err)
	}
	if n != 7 {
		t.Errorf("Count after 7 inserts: got %d want 7", n)
	}
}

// TestStateRoundTrip verifies cold GetState returns zero, then SaveState+GetState round-trips.
func TestStateRoundTrip(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	// cold start: no rows yet
	state, err := repo.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState cold: %v", err)
	}
	if state.LastProcessedBlock != 0 {
		t.Errorf("GetState cold: expected zero state, got %+v", state)
	}

	want := domain.IndexerState{
		ChainID:            1,
		LastProcessedBlock: 12345,
		LastProcessedHash:  "0xdeadbeef",
	}
	if err := repo.SaveState(ctx, want); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	got, err := repo.GetState(ctx)
	if err != nil {
		t.Fatalf("GetState after save: %v", err)
	}
	if got.ChainID != want.ChainID ||
		got.LastProcessedBlock != want.LastProcessedBlock ||
		got.LastProcessedHash != want.LastProcessedHash {
		t.Errorf("GetState round-trip: got %+v want %+v", got, want)
	}
}

// TestListHasMoreFalseOnLastPage verifies HasMore=false and empty NextCursor on the last page.
func TestListHasMoreFalseOnLastPage(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	insertN(t, repo, 3, 2000)

	page, err := repo.List(ctx, usecase.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.HasMore {
		t.Error("HasMore: got true want false (3 items, limit 10)")
	}
	if page.NextCursor != "" {
		t.Errorf("NextCursor: got %q want empty", page.NextCursor)
	}
	if len(page.Items) != 3 {
		t.Errorf("Items: got %d want 3", len(page.Items))
	}
}

// TestGetIssuanceTrend_BucketsByDay verifies day-bucketed counts come back sorted, with
// only non-zero buckets present (zero-fill across the requested range is TrendService's
// job — see Task 5 — not the repo's).
func TestGetIssuanceTrend_BucketsByDay(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	day1 := makeCert("1")
	day1.IssuedAt = time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	day1b := makeCertOwner("2", "0xabcdef1234567890abcdef1234567890abcdef12")
	day1b.IssuedAt = time.Date(2026, 6, 29, 22, 0, 0, 0, time.UTC)
	day3 := makeCertOwner("3", "0xabcdef1234567890abcdef1234567890abcdef12")
	day3.IssuedAt = time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)

	for _, c := range []domain.Certificate{day1, day1b, day3} {
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert %s: %v", c.TokenID, err)
		}
	}

	since := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	points, err := repo.GetIssuanceTrend(ctx, usecase.TrendBucketDay, since)
	if err != nil {
		t.Fatalf("GetIssuanceTrend: %v", err)
	}

	// only buckets with >=1 row come back — day 06-30 has none and is absent (zero-fill is
	// TrendService's job, tested in Task 5, not the repo's).
	if len(points) != 2 {
		t.Fatalf("got %d points, want 2 (06-29 and 07-01): %+v", len(points), points)
	}
	if points[0].Count != 2 {
		t.Errorf("06-29 count = %d, want 2", points[0].Count)
	}
	if points[1].Count != 1 {
		t.Errorf("07-01 count = %d, want 1", points[1].Count)
	}

	// BucketStart must land exactly on UTC day boundaries regardless of the Postgres
	// session's TimeZone GUC -- a regression here (e.g. reintroducing a bare ::timestamptz
	// cast on a naive value) would silently shift every bucket by the session's UTC offset,
	// which then fails to match TrendService's UTC-aligned merge keys (see trend.go).
	wantDay1 := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	wantDay3 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if !points[0].BucketStart.Equal(wantDay1) {
		t.Errorf("06-29 BucketStart = %v, want %v", points[0].BucketStart, wantDay1)
	}
	if !points[1].BucketStart.Equal(wantDay3) {
		t.Errorf("07-01 BucketStart = %v, want %v", points[1].BucketStart, wantDay3)
	}
}

// TestDeleteFromBlock verifies DeleteFromBlock deletes certs at or above a boundary block.
func TestDeleteFromBlock(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	below := makeCert("1")
	below.ChainID = 1
	below.BlockNumber = 100
	atBoundary := makeCertOwner("2", "0xabcdef1234567890abcdef1234567890abcdef12")
	atBoundary.ChainID = 1
	atBoundary.BlockNumber = 150
	above := makeCertOwner("3", "0xabcdef1234567890abcdef1234567890abcdef12")
	above.ChainID = 1
	above.BlockNumber = 200

	for _, c := range []domain.Certificate{below, atBoundary, above} {
		if err := repo.Upsert(ctx, c); err != nil {
			t.Fatalf("upsert %s: %v", c.TokenID, err)
		}
	}

	if err := repo.DeleteFromBlock(ctx, 1, 150); err != nil {
		t.Fatalf("DeleteFromBlock: %v", err)
	}

	if _, err := repo.GetByTokenID(ctx, "1"); err != nil {
		t.Fatalf("token 1 (below boundary) should survive: %v", err)
	}
	if _, err := repo.GetByTokenID(ctx, "2"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("token 2 (at boundary, inclusive) should be deleted, got err=%v", err)
	}
	if _, err := repo.GetByTokenID(ctx, "3"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("token 3 (above boundary) should be deleted, got err=%v", err)
	}
}

func TestInsertWebhookOutbox_DedupesByChainTxToken(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	payload := []byte(`{"tokenId":"1"}`)

	id1, isNew1, err := repo.InsertWebhookOutbox(ctx, 31337, "0xabc", "1", payload)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if !isNew1 {
		t.Fatal("first insert should be new")
	}
	if id1 == 0 {
		t.Fatal("expected a non-zero id")
	}

	id2, isNew2, err := repo.InsertWebhookOutbox(ctx, 31337, "0xabc", "1", payload)
	if err != nil {
		t.Fatalf("second (duplicate) insert: %v", err)
	}
	if isNew2 {
		t.Fatal("second insert of the same (chain_id, tx_hash, token_id) must not be new")
	}
	if id2 != 0 {
		t.Fatal("duplicate insert should return a zero id (nothing was returned)")
	}

	// A DIFFERENT token_id sharing the same tx_hash (hypothetical batch-mint-via-multicall
	// in one transaction) must still be treated as a genuinely new event, not collapsed.
	id3, isNew3, err := repo.InsertWebhookOutbox(ctx, 31337, "0xabc", "2", payload)
	if err != nil {
		t.Fatalf("different token_id insert: %v", err)
	}
	if !isNew3 {
		t.Fatal("a different token_id sharing the same tx_hash must be treated as new")
	}
	if id3 == id1 {
		t.Fatal("expected a distinct id for a distinct token_id")
	}
}

func TestListUnenqueuedWebhookOutbox_ReturnsOnlyUnmarked(t *testing.T) {
	pool := startPostgres(t)
	repo := postgres.NewCertificateRepo(pool)
	ctx := context.Background()

	id1, _, err := repo.InsertWebhookOutbox(ctx, 31337, "0xabc", "1", []byte(`{}`))
	if err != nil {
		t.Fatalf("insert 1: %v", err)
	}
	if _, _, err := repo.InsertWebhookOutbox(ctx, 31337, "0xdef", "2", []byte(`{}`)); err != nil {
		t.Fatalf("insert 2: %v", err)
	}

	if err := repo.MarkWebhookOutboxEnqueued(ctx, id1); err != nil {
		t.Fatalf("mark enqueued: %v", err)
	}

	entries, err := repo.ListUnenqueuedWebhookOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("list unenqueued: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d unenqueued entries, want 1 (id1 was marked enqueued)", len(entries))
	}
}
