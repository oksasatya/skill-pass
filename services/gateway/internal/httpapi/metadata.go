package httpapi

import (
	"context"
	"net/http"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// MetadataAttributeDTO is one ERC-721-style attribute entry.
type MetadataAttributeDTO struct {
	TraitType string `json:"trait_type"`
	Value     string `json:"value"`
}

// MetadataDTO is the ERC-721 metadata JSON shape served at GET /certificates/{tokenId}/metadata.
// image is deliberately omitted -- no image asset exists in this project.
type MetadataDTO struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Attributes  []MetadataAttributeDTO `json:"attributes"`
}

// GetCertificateMetadata handles GET /certificates/{tokenId}/metadata -- reshapes the
// existing GetCertificate gRPC response into ERC-721-style JSON so wallets/marketplaces
// can render a certificate. No new gRPC call; zero business logic beyond the reshape.
func GetCertificateMetadata(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenID := r.PathValue("tokenId")
		if !isValidTokenID(tokenID) {
			writeJSONError(w, http.StatusBadRequest, "token_id must be a non-empty decimal digit string")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), d.RequestTimeout)
		defer cancel()

		resp, err := d.Cert.GetCertificate(ctx, &certv1.GetCertificateRequest{TokenId: tokenID})
		if err != nil {
			writeGRPCError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, toMetadataDTO(resp.GetCertificate()))
	}
}

// toMetadataDTO maps a proto Certificate to its ERC-721 metadata JSON representation.
// Pure function -- O(1).
func toMetadataDTO(c *certv1.Certificate) MetadataDTO {
	return MetadataDTO{
		Name:        c.GetTitle(),
		Description: c.GetDescription(),
		Attributes: []MetadataAttributeDTO{
			{TraitType: "Recipient", Value: c.GetRecipientName()},
			{TraitType: "Issuer", Value: c.GetIssuerName()},
			{TraitType: "Issued At", Value: c.GetIssuedAt().AsTime().UTC().Format("2006-01-02T15:04:05Z07:00")},
		},
	}
}
