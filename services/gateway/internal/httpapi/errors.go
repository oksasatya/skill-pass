package httpapi

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// writeJSON writes v as a JSON body with the given HTTP status.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeJSONError writes a {"error": msg} JSON body with the given HTTP status.
func writeJSONError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// writeGRPCError maps a gRPC error to an HTTP status + JSON body at the REST boundary.
// This is the ONLY place gRPC codes are translated to HTTP status — keep every call site
// routing through here so the mapping never drifts.
func writeGRPCError(w http.ResponseWriter, err error) {
	st := status.Convert(err)
	writeJSONError(w, grpcCodeToHTTPStatus(st.Code()), st.Message())
}

// grpcCodeToHTTPStatus maps gRPC status codes to HTTP status codes.
func grpcCodeToHTTPStatus(c codes.Code) int {
	switch c {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unavailable, codes.DeadlineExceeded:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
