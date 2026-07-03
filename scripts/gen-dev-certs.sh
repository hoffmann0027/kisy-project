#!/usr/bin/env bash
# Generate a self-signed TLS certificate for local HTTPS development.
# Production must use certificates from a real CA (e.g. Let's Encrypt).
set -euo pipefail

CERT_DIR="$(cd "$(dirname "$0")/.." && pwd)/deploy/nginx/certs"
mkdir -p "$CERT_DIR"

if [[ -f "$CERT_DIR/privkey.pem" && "${FORCE:-0}" != "1" ]]; then
  echo "Certificates already exist in $CERT_DIR (set FORCE=1 to overwrite)."
  exit 0
fi

openssl req -x509 -newkey rsa:2048 -sha256 -days 365 -nodes \
  -keyout "$CERT_DIR/privkey.pem" \
  -out "$CERT_DIR/fullchain.pem" \
  -subj "/C=US/O=KISY/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

chmod 600 "$CERT_DIR/privkey.pem"
echo "Self-signed certificate written to $CERT_DIR"
echo "Enable HTTPS with: docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d"
