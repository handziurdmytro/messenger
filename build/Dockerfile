# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder

ARG SERVICE

WORKDIR /build

COPY go.work go.work.sum ./
COPY pkg/ ./pkg/
COPY services/ ./services/

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build \
    -ldflags="-w -s" \
    -trimpath \
    -o /app \
    ./services/${SERVICE}/cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /app /app

ENTRYPOINT ["/app"]