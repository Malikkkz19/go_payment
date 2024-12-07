# Build stage
FROM golang:1.22.5-alpine AS builder

WORKDIR /app

# Install required build tools
RUN apk add --no-cache gcc musl-dev protoc

# Copy go mod and sum files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/payment-api ./cmd/api
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/payment-grpc ./cmd/grpc

# Final stage
FROM alpine:latest

WORKDIR /app

# Install necessary runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Copy the binaries from builder
COPY --from=builder /app/payment-api .
COPY --from=builder /app/payment-grpc .
COPY --from=builder /app/configs/config.yaml ./configs/

# Create non-root user
RUN adduser -D -g '' appuser
RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8080 50051

# Use shell form to allow environment variable substitution
CMD if [ "$SERVICE_TYPE" = "grpc" ]; then \
        ./payment-grpc; \
    else \
        ./payment-api; \
    fi
