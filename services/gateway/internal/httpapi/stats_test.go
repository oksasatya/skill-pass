package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// fakeTrendCertClient extends fakeCertClient (certificates_test.go) with GetIssuanceTrend.
// Embedding promotes GetCertificate/ListCertificates/StreamCertificateEvents/
// GetIndexerStatus from *fakeCertClient, so fakeTrendCertClient as a whole still satisfies
// certv1.CertificateQueryClient once this method is added.
type fakeTrendCertClient struct {
	*fakeCertClient
	trendResp *certv1.GetIssuanceTrendResponse
	trendErr  error
}

func (f *fakeTrendCertClient) GetIssuanceTrend(_ context.Context, _ *certv1.GetIssuanceTrendRequest, _ ...grpc.CallOption) (*certv1.GetIssuanceTrendResponse, error) {
	return f.trendResp, f.trendErr
}

func TestGetIssuanceTrendHandler_MissingBucket(t *testing.T) {
	client := &fakeTrendCertClient{fakeCertClient: &fakeCertClient{}}
	req := httptest.NewRequest(http.MethodGet, "/stats/trend?range=30d", nil)
	w := httptest.NewRecorder()

	GetIssuanceTrend(newDeps(client))(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetIssuanceTrendHandler_UnknownBucket(t *testing.T) {
	client := &fakeTrendCertClient{fakeCertClient: &fakeCertClient{}}
	req := httptest.NewRequest(http.MethodGet, "/stats/trend?bucket=year&range=30d", nil)
	w := httptest.NewRecorder()

	GetIssuanceTrend(newDeps(client))(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetIssuanceTrendHandler_Success(t *testing.T) {
	bucketStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	client := &fakeTrendCertClient{
		fakeCertClient: &fakeCertClient{},
		trendResp: &certv1.GetIssuanceTrendResponse{
			Points: []*certv1.TrendPoint{
				{BucketStart: timestamppb.New(bucketStart), Count: 42},
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/stats/trend?bucket=day&range=30d", nil)
	w := httptest.NewRecorder()

	GetIssuanceTrend(newDeps(client))(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body map[string][]TrendPointDTO
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	points := body["points"]
	if len(points) != 1 {
		t.Fatalf("got %d points, want 1", len(points))
	}
	if points[0].Count != 42 {
		t.Errorf("count = %d, want 42", points[0].Count)
	}
	if points[0].BucketStart != "2026-07-01T00:00:00Z" {
		t.Errorf("bucketStart = %q, want RFC3339 UTC", points[0].BucketStart)
	}
}
