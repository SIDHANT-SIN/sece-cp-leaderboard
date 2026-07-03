FROM golang:1.26-trixie AS builder

WORKDIR /opt/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /opt/app/main src/main.go

FROM gcr.io/distroless/static-debian13

WORKDIR /opt/app

COPY --from=builder /opt/app/main .
COPY --from=builder /opt/app/templates ./templates

EXPOSE 8080

CMD ["/opt/app/main"]