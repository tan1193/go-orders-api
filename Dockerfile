FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/go-order-service ./cmd/server

FROM gcr.io/distroless/static-debian12

COPY --from=builder /bin/go-order-service /go-order-service

EXPOSE 8080

ENTRYPOINT ["/go-order-service"]