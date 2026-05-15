# syntax=docker/dockerfile:1.6

FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/oidc-server ./cmd/server

FROM alpine:3.19 AS runner

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/oidc-server /app/oidc-server
COPY --from=builder /src/configs /app/configs
COPY --from=builder /src/db/migrations /app/db/migrations

USER app

EXPOSE 8080

ENTRYPOINT ["/app/oidc-server"]
