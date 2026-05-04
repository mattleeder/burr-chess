FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/web

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/internal/models/schema.sql ./internal/models/schema.sql
COPY cmd/web/localhost.crt cmd/web/localhost.key ./cmd/web/
EXPOSE 8080
CMD ["./server"]
