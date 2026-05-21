# Step 1: Use the Go image to build the app
FROM golang:bookworm AS builder

WORKDIR /opt/app

# Copy module files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of your code
COPY . .

# Build the app into a single, highly-optimized static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /opt/app/main src/main.go

# Step 2: Use a tiny base image to run the binary
FROM debian:bookworm-slim

WORKDIR /opt/app

# Copy the compiled binary from the builder stage
COPY --from=builder /opt/app/main .

# Copy your HTML templates so Gin can serve the views
COPY --from=builder /opt/app/templates ./templates

EXPOSE 8080

# Run the pre-compiled binary instantly
CMD ["./main"]