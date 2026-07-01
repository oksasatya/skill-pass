package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// fakeCertClient implements certv1.CertificateQueryClient for handler tests.
type fakeCertClient struct {
	getResp  *certv1.GetCertificateResponse
	getErr   error
	listResp *certv1.ListCertificatesResponse
	listErr  error
}

func (f *fakeCertClient) GetCertificate(_ context.Context, _ *certv1.GetCertificateRequest, _ ...grpc.CallOption) (*certv1.GetCertificateResponse, error) {
	return f.getResp, f.getErr
}

func (f *fakeCertClient) ListCertificates(_ context.Context, _ *certv1.ListCertificatesRequest, _ ...grpc.CallOption) (*certv1.ListCertificatesResponse, error) {
	return f.listResp, f.listErr
}

func (f *fakeCertClient) StreamCertificateEvents(_ context.Context, _ *certv1.StreamCertificateEventsRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[certv1.CertificateEvent], error) {
	return nil, status.Error(codes.Unimplemented, "not used in this test")
}

func (f *fakeCertClient) GetIndexerStatus(_ context.Context, _ *certv1.GetIndexerStatusRequest, _ ...grpc.CallOption) (*certv1.GetIndexerStatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not used in this test")
}

func sampleProtoCert(tokenID string) *certv1.Certificate {
	return &certv1.Certificate{
		TokenId:       tokenID,
		OwnerAddress:  "0x1234567890123456789012345678901234567890",
		Title:         "Go Expert",
		RecipientName: "Alice",
		IssuerName:    "Skillpass",
		Description:   "Backend cert",
		MetadataUri:   "ipfs://Qm...",
		IssuedAt:      timestamppb.New(time.Unix(1700000000, 0).UTC()),
		TxHash:        "0xdeadbeef",
		BlockNumber:   42,
	}
}

func newDeps(cert certv1.CertificateQueryClient) Deps {
	return Deps{Cert: cert, RequestTimeout: time.Second}
}

// --- parsePageSize ---

func TestParsePageSize(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int32
	}{
		{"empty defaults", "", defaultPageSize},
		{"valid", "50", 50},
		{"zero defaults", "0", defaultPageSize},
		{"negative defaults", "-5", defaultPageSize},
		{"non-numeric defaults", "abc", defaultPageSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePageSize(tt.raw); got != tt.want {
				t.Errorf("parsePageSize(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

// --- grpcCodeToHTTPStatus ---

func TestGRPCCodeToHTTPStatus(t *testing.T) {
	tests := []struct {
		code codes.Code
		want int
	}{
		{codes.NotFound, http.StatusNotFound},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.DeadlineExceeded, http.StatusServiceUnavailable},
		{codes.Internal, http.StatusInternalServerError},
		{codes.Unknown, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			if got := grpcCodeToHTTPStatus(tt.code); got != tt.want {
				t.Errorf("grpcCodeToHTTPStatus(%v) = %d, want %d", tt.code, got, tt.want)
			}
		})
	}
}

// --- toDTO ---

func TestToDTO(t *testing.T) {
	dto := toDTO(sampleProtoCert("1"))
	if dto.TokenID != "1" || dto.Title != "Go Expert" || dto.BlockNumber != 42 {
		t.Fatalf("unexpected dto: %+v", dto)
	}
	if dto.IssuedAt != "2023-11-14T22:13:20Z" {
		t.Fatalf("unexpected issuedAt: %q", dto.IssuedAt)
	}
}

// --- GetCertificate handler ---

func TestGetCertificate_Found(t *testing.T) {
	client := &fakeCertClient{getResp: &certv1.GetCertificateResponse{Certificate: sampleProtoCert("1")}}
	req := httptest.NewRequest(http.MethodGet, "/certificates/1", nil)
	req.SetPathValue("tokenId", "1")
	w := httptest.NewRecorder()

	GetCertificate(newDeps(client))(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]CertificateDTO
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["certificate"].TokenID != "1" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestGetCertificate_NotFound(t *testing.T) {
	client := &fakeCertClient{getErr: status.Error(codes.NotFound, "certificate not found")}
	req := httptest.NewRequest(http.MethodGet, "/certificates/999", nil)
	req.SetPathValue("tokenId", "999")
	w := httptest.NewRecorder()

	GetCertificate(newDeps(client))(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestGetCertificate_EmptyTokenID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/certificates/", nil)
	w := httptest.NewRecorder()

	GetCertificate(newDeps(&fakeCertClient{}))(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// --- ListCertificates handler ---

func TestListCertificates_MapsQueryParams(t *testing.T) {
	var captured *certv1.ListCertificatesRequest
	client := &fakeCertClient{
		listResp: &certv1.ListCertificatesResponse{
			Certificates: []*certv1.Certificate{sampleProtoCert("1"), sampleProtoCert("2")},
			NextCursor:   "2",
			HasMore:      true,
		},
	}
	// wrap to capture the request the handler builds
	capturing := &capturingClient{fakeCertClient: client, onList: func(r *certv1.ListCertificatesRequest) { captured = r }}

	req := httptest.NewRequest(http.MethodGet, "/certificates?owner=0xabc&q=go&cursor=5&page_size=10", nil)
	w := httptest.NewRecorder()

	ListCertificates(newDeps(capturing))(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if captured.GetOwnerAddress() != "0xabc" || captured.GetQuery() != "go" || captured.GetCursor() != "5" || captured.GetPageSize() != 10 {
		t.Fatalf("unexpected captured request: %+v", captured)
	}

	var body ListCertificatesDTO
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Certificates) != 2 || !body.HasMore || body.NextCursor != "2" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

// capturingClient wraps fakeCertClient to record the ListCertificates request it received.
type capturingClient struct {
	*fakeCertClient
	onList func(*certv1.ListCertificatesRequest)
}

func (c *capturingClient) ListCertificates(ctx context.Context, req *certv1.ListCertificatesRequest, opts ...grpc.CallOption) (*certv1.ListCertificatesResponse, error) {
	c.onList(req)
	return c.fakeCertClient.ListCertificates(ctx, req, opts...)
}
