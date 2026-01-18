# Build stage
FROM golang:1.24.11 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
# Install swag for generating Swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest
COPY . .
# Generate Swagger docs
RUN swag init -g cmd/meerkat/main.go -o docs
# Build the application
RUN CGO_ENABLED=1 go build -o meerkat ./cmd/meerkat

# Runtime stage
FROM debian:bookworm-slim
# Install ca-certificates for HTTPS
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /build/meerkat /app/meerkat
# Create data directory for SQLite database with proper permissions
RUN mkdir -p /app/data && chown -R nobody:nogroup /app/data && chmod 755 /app/data
USER nobody
EXPOSE 8080
ENTRYPOINT ["/app/meerkat"]

