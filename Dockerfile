FROM golang:bookworm AS builder

WORKDIR /opt/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the app into a single, highly-optimized static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /opt/app/main src/main.go

FROM debian:bookworm-slim

WORKDIR /opt/app

# Install SSL/TLS root certificates so Go can trust Turso over HTTPS
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy the compiled binary from the builder stage
COPY --from=builder /opt/app/main .

# Copy your HTML templates so Gin can serve the views
COPY --from=builder /opt/app/templates ./templates

EXPOSE 8080

CMD ["./main"]