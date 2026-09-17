#!/bin/sh
set -eu
# Render runtime config so the API base can be set at deploy time
# (Vite bakes import.meta.env into the bundle at build time).
# API_PUBLIC_URL: full base, e.g. https://rhythm.example.com/api/v1
#   or empty = same origin (/api/v1 on the same host via Caddy).
cat > /usr/share/nginx/html/config.js <<EOF
window.__ENV__ = { API_URL: '${API_PUBLIC_URL:-}' };
EOF
exec nginx -g 'daemon off;'
