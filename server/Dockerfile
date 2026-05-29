# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=$(cat VERSION 2>/dev/null || echo dev)" \
    -o /out/stonepad-server ./cmd/stonepad-server

# Runtime stage
FROM scratch
COPY --from=builder /out/stonepad-server /stonepad-server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
ENTRYPOINT ["/stonepad-server"]
