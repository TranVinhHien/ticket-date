# Build context: ./be (docker build -f be/user-service/Dockerfile ./be)
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY common/go.mod common/go.sum ./common/
COPY user-service/go.mod user-service/go.sum ./user-service/

WORKDIR /src/user-service
RUN go mod download

WORKDIR /src
COPY common/ ./common/
COPY user-service/ ./user-service/

WORKDIR /src/user-service
RUN CGO_ENABLED=0 GOOS=linux go build -tags=nomsgpack -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/server ./server

USER app

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["./server"]
