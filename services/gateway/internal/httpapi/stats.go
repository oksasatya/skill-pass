package httpapi

import (
	"context"
	"net/http"

	certv1 "github.com/oksasatya/skillpass/proto/gen/go/skillpass/cert/v1"
)

// bucketParams maps the REST ?bucket= value to its proto enum.
var bucketParams = map[string]certv1.TrendBucket{
	"day":   certv1.TrendBucket_TREND_BUCKET_DAY,
	"week":  certv1.TrendBucket_TREND_BUCKET_WEEK,
	"month": certv1.TrendBucket_TREND_BUCKET_MONTH,
}

// TrendPointDTO is one bucketed count in the JSON response.
type TrendPointDTO struct {
	BucketStart string `json:"bucketStart"` // RFC3339
	Count       uint64 `json:"count"`
}

// GetIssuanceTrend handles GET /stats/trend?bucket=day|week|month&range=<preset> — a
// certificate-issuance time series. Thin REST wrapper: validate, call the indexer's gRPC
// method, map to JSON, no business logic here (that's TrendService, indexer-side).
func GetIssuanceTrend(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		bucket, ok := bucketParams[q.Get("bucket")]
		if !ok {
			writeJSONError(w, http.StatusBadRequest, "bucket must be one of: day, week, month")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), d.RequestTimeout)
		defer cancel()

		resp, err := d.Cert.GetIssuanceTrend(ctx, &certv1.GetIssuanceTrendRequest{
			Bucket:      bucket,
			RangePreset: q.Get("range"),
		})
		if err != nil {
			writeGRPCError(w, err)
			return
		}

		points := make([]TrendPointDTO, 0, len(resp.GetPoints()))
		for _, p := range resp.GetPoints() {
			points = append(points, TrendPointDTO{
				BucketStart: p.GetBucketStart().AsTime().UTC().Format("2006-01-02T15:04:05Z07:00"),
				Count:       p.GetCount(),
			})
		}
		writeJSON(w, http.StatusOK, map[string][]TrendPointDTO{"points": points})
	}
}
