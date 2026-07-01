package httpapi

import (
	"context"
	"net/http"
	"strconv"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// defaultPageSize is applied when page_size is absent/invalid/non-positive.
const defaultPageSize = 20

// GetCertificate handles GET /certificates/{tokenId} — one certificate by token_id.
func GetCertificate(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenID := r.PathValue("tokenId")
		if tokenID == "" {
			writeJSONError(w, http.StatusBadRequest, "token_id is required")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), d.RequestTimeout)
		defer cancel()

		resp, err := d.Cert.GetCertificate(ctx, &certv1.GetCertificateRequest{TokenId: tokenID})
		if err != nil {
			writeGRPCError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]CertificateDTO{"certificate": toDTO(resp.GetCertificate())})
	}
}

// ListCertificates handles GET /certificates?owner=&q=&cursor=&page_size= — a keyset-paginated,
// optionally owner-filtered and text-searched list.
func ListCertificates(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		ctx, cancel := context.WithTimeout(r.Context(), d.RequestTimeout)
		defer cancel()

		resp, err := d.Cert.ListCertificates(ctx, &certv1.ListCertificatesRequest{
			OwnerAddress: q.Get("owner"),
			Query:        q.Get("q"),
			Cursor:       q.Get("cursor"),
			PageSize:     parsePageSize(q.Get("page_size")),
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, ListCertificatesDTO{
			Certificates: toDTOs(resp.GetCertificates()),
			NextCursor:   resp.GetNextCursor(),
			HasMore:      resp.GetHasMore(),
		})
	}
}

// parsePageSize parses the page_size query param, defaulting to defaultPageSize on
// empty/invalid/non-positive input. Pure function — O(1).
func parsePageSize(raw string) int32 {
	if raw == "" {
		return defaultPageSize
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultPageSize
	}
	return int32(n)
}
