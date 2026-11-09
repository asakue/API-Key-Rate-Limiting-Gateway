FROM golang:1.23-alpine AS builder

ENV GOWORK=off
ENV CGO_ENABLED=0

WORKDIR /app
COPY control-plane/go.mod control-plane/go.sum* ./control-plane/
WORKDIR /app/control-plane
RUN go mod download

WORKDIR /app
COPY control-plane/ ./control-plane/
WORKDIR /app/control-plane
RUN GOOS=linux go build -o /control-plane ./cmd/server

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /control-plane .
EXPOSE 8081
CMD ["./control-plane"]
