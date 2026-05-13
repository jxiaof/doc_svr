FROM golang:1.26-alpine AS builder

ARG VERSION=dev
WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY main.go ./main.go
COPY internal ./internal
COPY web ./web
COPY content ./content

RUN CGO_ENABLED=0 GOFLAGS=-mod=vendor go build \
	-trimpath \
	-ldflags="-s -w -X main.version=${VERSION}" \
	-o /out/doc-svr .

FROM alpine:3.21

RUN addgroup -S app && adduser -S -G app app && apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /out/doc-svr /usr/local/bin/doc-svr

ENV PORT=3000
EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD wget -q -O - "http://127.0.0.1:${PORT}/healthz" >/dev/null || exit 1

USER app
ENTRYPOINT ["/usr/local/bin/doc-svr"]
