package httpapi

import (
	"time"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// CertificateDTO is the certificate JSON shape returned to the frontend.
type CertificateDTO struct {
	TokenID       string `json:"tokenId"`
	OwnerAddress  string `json:"ownerAddress"`
	Title         string `json:"title"`
	RecipientName string `json:"recipientName"`
	IssuerName    string `json:"issuerName"`
	Description   string `json:"description"`
	MetadataURI   string `json:"metadataUri"`
	IssuedAt      string `json:"issuedAt"` // RFC3339
	TxHash        string `json:"txHash"`
	BlockNumber   uint64 `json:"blockNumber"`
}

// ListCertificatesDTO is the JSON response for GET /certificates.
type ListCertificatesDTO struct {
	Certificates []CertificateDTO `json:"certificates"`
	NextCursor   string           `json:"nextCursor"`
	HasMore      bool             `json:"hasMore"`
}

// toDTO maps a proto Certificate to its JSON representation. Pure function — O(1).
func toDTO(c *certv1.Certificate) CertificateDTO {
	return CertificateDTO{
		TokenID:       c.GetTokenId(),
		OwnerAddress:  c.GetOwnerAddress(),
		Title:         c.GetTitle(),
		RecipientName: c.GetRecipientName(),
		IssuerName:    c.GetIssuerName(),
		Description:   c.GetDescription(),
		MetadataURI:   c.GetMetadataUri(),
		IssuedAt:      c.GetIssuedAt().AsTime().UTC().Format(time.RFC3339),
		TxHash:        c.GetTxHash(),
		BlockNumber:   c.GetBlockNumber(),
	}
}

// toDTOs maps a slice of proto Certificates. O(n) time, O(n) space — unavoidable, the
// response must carry every returned certificate.
func toDTOs(certs []*certv1.Certificate) []CertificateDTO {
	out := make([]CertificateDTO, 0, len(certs))
	for _, c := range certs {
		out = append(out, toDTO(c))
	}
	return out
}
