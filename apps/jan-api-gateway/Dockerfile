FROM golang:1.24.6-alpine AS builder
WORKDIR /app

COPY application/go.mod ./
RUN go mod download

ARG VERSION_TAG=dev
COPY application/. .
RUN go mod tidy
RUN go build -o /jan-api-gateway \
    -ldflags="-s -w -X 'menlo.ai/jan-api-gateway/config.Version=${VERSION_TAG}'" \
    ./cmd/server

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /jan-api-gateway .

EXPOSE 8080
CMD ["./jan-api-gateway"]