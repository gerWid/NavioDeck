# ── Stage 1: Build Frontend ───────────────────────────────────────────────────
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# ── Stage 2: Build Backend ────────────────────────────────────────────────────
FROM golang:1.23-alpine AS backend-builder
WORKDIR /app/backend

COPY backend/go.mod ./
RUN go mod download

COPY backend/ ./
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o naviodeck .

# ── Stage 3: Final Image ──────────────────────────────────────────────────────
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata su-exec

WORKDIR /app

COPY --from=backend-builder /app/backend/naviodeck ./naviodeck
COPY --from=frontend-builder /app/frontend/dist ./static
COPY entrypoint.sh /entrypoint.sh

RUN adduser -D -u 1000 naviodeck && \
    mkdir -p /data && \
    chown -R naviodeck:naviodeck /app /data && \
    sed -i 's/\r//' /entrypoint.sh && \
    chmod +x /entrypoint.sh

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
CMD ["--addr", ":8080", "--data", "/data", "--static", "/app/static"]
