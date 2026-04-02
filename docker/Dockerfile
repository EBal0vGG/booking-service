FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /booking-service ./cmd/app

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /booking-service /booking-service
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["/booking-service"]
