FROM golang:1.23-alpine AS builder

ENV GOWORK=off
ENV GOTOOLCHAIN=auto

WORKDIR /app
COPY mock-backend/go.mod ./mock-backend/
COPY mock-backend/main.go ./mock-backend/

WORKDIR /app/mock-backend
RUN CGO_ENABLED=0 GOOS=linux go build -o /mock-backend main.go

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /mock-backend .
EXPOSE 8080
CMD ["./mock-backend"]
