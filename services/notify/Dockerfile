# syntax=docker/dockerfile:1
# Mirrors services/indexer/Dockerfile and services/gateway/Dockerfile: static binary,
# distroless/nonroot runtime.

FROM golang:1.26 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /notify \
    ./services/notify/cmd/notify

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /notify /notify

ENTRYPOINT ["/notify"]
