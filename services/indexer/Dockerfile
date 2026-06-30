# syntax=docker/dockerfile:1
# CGO_ENABLED=0 — go-ethereum's C bindings (c-kzg, blst) are excluded at build time
# via their own build tags when CGO is disabled; static binary links cleanly.
# Runtime: distroless/static-debian12:nonroot (no shell, no libc, minimal attack surface).

FROM golang:1.26 AS builder

WORKDIR /build

# Download deps first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Static binary — CGO off, strip debug info for smaller image
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /indexer \
    ./services/indexer/cmd/indexer

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /indexer /indexer

ENTRYPOINT ["/indexer"]
