FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o payflow ./cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/payflow .
COPY migrations/ ./migrations/
EXPOSE 8080
CMD ["./payflow"]