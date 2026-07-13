# --- build ---
FROM node:25-alpine AS builder
WORKDIR /src

COPY package.json package-lock.json* ./
RUN npm ci

COPY . .
ARG VITE_API_BASE_URL=/api/v1
ARG VITE_WS_BASE_URL=/ws
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_WS_BASE_URL=$VITE_WS_BASE_URL
RUN npm run build

# --- runtime ---
# Unprivileged variant: the master process runs as the non-root "nginx" user
# and listens on 8080 (container hardening — no root in the runtime image).
FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY --from=builder /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

# The base image already runs as "nginx" (uid 101); restated explicitly so
# static scanners (and readers) see the non-root guarantee in THIS file.
USER nginx

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
    CMD wget -qO- http://127.0.0.1:8080/ || exit 1
