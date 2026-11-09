FROM golang:1.23-alpine AS builder

ENV GOWORK=off
ENV CGO_ENABLED=0

WORKDIR /app
COPY gateway/go.mod gateway/go.sum* ./gateway/
WORKDIR /app/gateway
RUN go mod download

WORKDIR /app
COPY gateway/ ./gateway/
WORKDIR /app/gateway
RUN GOOS=linux go build -o /gateway ./cmd/gateway

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /gateway .
EXPOSE 8000
CMD ["./gateway"]
