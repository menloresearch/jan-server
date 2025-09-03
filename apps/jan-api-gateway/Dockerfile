FROM golang:1.24.6-alpine AS builder
WORKDIR /app

RUN apk add --no-cache build-base

COPY application/go.mod ./
RUN go mod download

ARG VERSION_TAG=dev
COPY application/. .
RUN go mod tidy

RUN CGO_ENABLED=1 GOOS=linux \
    go build -o /jan-api-gateway \
      -gcflags="all=-N -l" \
      -ldflags="-linkmode=external -X menlo.ai/jan-api-gateway/config.Version=${VERSION_TAG}" \
      ./cmd/server

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /jan-api-gateway .

EXPOSE 8080
CMD ["./jan-api-gateway"]
