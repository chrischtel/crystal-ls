# Build stage
FROM golang:1.21-alpine AS builder

# Set build arguments
ARG VERSION=dev

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o crystal-ls main.go

# Final stage
FROM scratch

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /build/crystal-ls /crystal-ls

# Copy documentation
COPY README.md LICENSE ./

# Expose port (if needed for future features)
EXPOSE 8080

# Set the binary as entrypoint
ENTRYPOINT ["/crystal-ls"]
