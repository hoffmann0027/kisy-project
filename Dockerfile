# All-in-one image: the Go backend serves the built SPA on the same origin.
# Suited to single-service free hosting (Render, Fly, Koyeb). The database
# and Redis come from managed add-ons via DATABASE_URL / REDIS_URL.
#
# Build context is the repository root.

# --- frontend build ---
FROM node:22-alpine AS frontend
WORKDIR /fe
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
# Same-origin defaults: the SPA calls /api/v1 and /ws on its own host.
ENV VITE_API_BASE_URL=/api/v1
ENV VITE_WS_BASE_URL=/ws
RUN npm run build

# --- backend build ---
FROM golang:1.26-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget && \
    addgroup -S kisy && adduser -S kisy -G kisy
WORKDIR /app

COPY --from=backend /out/server ./server
COPY --from=backend /src/migrations ./migrations
COPY --from=frontend /fe/dist ./web

ENV WEB_DIR=/app/web \
    APP_ENV=production \
    RUN_MIGRATIONS=true \
    BACKEND_HTTP_PORT=8080

USER kisy
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=5 \
    CMD wget -qO- http://127.0.0.1:${PORT:-8080}/health || exit 1

ENTRYPOINT ["./server"]
