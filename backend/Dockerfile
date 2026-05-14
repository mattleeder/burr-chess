FROM dhi.io/golang:1-alpine3.23-dev AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/web

FROM scratch
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/internal/models/schema.sql ./internal/models/schema.sql
EXPOSE 8080
CMD ["./server"]
