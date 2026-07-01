FROM golang:bookworm AS builder

WORKDIR /opt/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /opt/app/main src/main.go

FROM debian:bookworm-slim

WORKDIR /opt/app

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /opt/app/main .

COPY --from=builder /opt/app/templates ./templates

EXPOSE 8080

CMD ["./main"]