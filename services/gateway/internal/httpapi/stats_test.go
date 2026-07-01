package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"

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
