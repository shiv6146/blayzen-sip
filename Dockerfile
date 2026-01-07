# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g cmd/blayzen-sip/main.go -o docs

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o blayzen-sip ./cmd/blayzen-sip

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/blayzen-sip .

# Copy migrations for init
COPY --from=builder /app/migrations ./migrations

# Expose ports
# SIP UDP/TCP
EXPOSE 5060/udp
EXPOSE 5060/tcp
# REST API
EXPOSE 8080/tcp
# RTP range
EXPOSE 10000-10100/udp

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run
ENTRYPOINT ["./blayzen-sip"]
