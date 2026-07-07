# --- build ---
FROM golang:1.26-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- runtime ---
FROM alpine:3.24
RUN apk add --no-cache ca-certificates wget && \
    addgroup -S kisy && adduser -S kisy -G kisy
WORKDIR /app

COPY --from=builder /out/server ./server
COPY --from=builder /src/migrations ./migrations

USER kisy
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["./server"]
