package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

func TestGetCertificateMetadataHandler_InvalidTokenID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/certificates/1abc/metadata", nil)
	req.SetPathValue("tokenId", "1; DROP TABLE certificates;")
	w := httptest.NewRecorder()

	GetCertificateMetadata(newDeps(&fakeCertClient{}))(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetCertificateMetadataHandler_Success(t *testing.T) {
	client := &fakeCertClient{getResp: &certv1.GetCertificateResponse{Certificate: sampleProtoCert("1")}}
	req := httptest.NewRequest(http.MethodGet, "/certificates/1/metadata", nil)
	req.SetPathValue("tokenId", "1")
	w := httptest.NewRecorder()

	GetCertificateMetadata(newDeps(client))(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body MetadataDTO
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "Go Expert" {
		t.Errorf("name = %q, want %q", body.Name, "Go Expert")
	}
	if body.Description != "Backend cert" {
		t.Errorf("description = %q, want %q", body.Description, "Backend cert")
	}
	if len(body.Attributes) != 3 {
		t.Fatalf("got %d attributes, want 3", len(body.Attributes))
	}
	if body.Attributes[0].TraitType != "Recipient" || body.Attributes[0].Value != "Alice" {
		t.Errorf("attributes[0] = %+v, want Recipient=Alice", body.Attributes[0])
	}
	if body.Attributes[1].TraitType != "Issuer" || body.Attributes[1].Value != "Skillpass" {
		t.Errorf("attributes[1] = %+v, want Issuer=Skillpass", body.Attributes[1])
	}
}

func TestGetCertificateMetadataHandler_NotFound(t *testing.T) {
	client := &fakeCertClient{getErr: status.Error(codes.NotFound, "certificate not found")}
	req := httptest.NewRequest(http.MethodGet, "/certificates/999/metadata", nil)
	req.SetPathValue("tokenId", "999")
	w := httptest.NewRecorder()

	GetCertificateMetadata(newDeps(client))(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
