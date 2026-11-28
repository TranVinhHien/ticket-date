# Build context: ./be (docker build -f be/order-service/Dockerfile ./be)
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY common/go.mod common/go.sum ./common/
COPY order-service/go.mod order-service/go.sum ./order-service/

WORKDIR /src/order-service
RUN go mod download

WORKDIR /src
COPY common/ ./common/
COPY order-service/ ./order-service/

WORKDIR /src/order-service
RUN CGO_ENABLED=0 GOOS=linux go build -tags=nomsgpack -trimpath -ldflags="-s -w" -o /out/server ./server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/server ./server

USER app

EXPOSE 8083 50053

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8083/health || exit 1

ENTRYPOINT ["./server"]
