# Multi-stage multi-arch build for PatchPulse (Go)
FROM --platform=$BUILDPLATFORM golang:alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and embedded public assets
COPY . .

# Target architecture passed by Docker buildx
ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-s -w" -o /app/patchpulse ./cmd/patchpulse

# Final lean runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/patchpulse /app/patchpulse

# Data directory for SQLite WAL database
RUN mkdir -p /app/data
ENV DB_DIR=/app/data
ENV PORT=3000
ENV HOST=0.0.0.0

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:3000/api/health || exit 1

ENTRYPOINT ["/app/patchpulse"]
CMD ["serve"]
