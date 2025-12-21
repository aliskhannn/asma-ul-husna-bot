# Stage 1: Goose builder
FROM golang:1.25-alpine AS goose-builder

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/pressly/goose/v3/cmd/goose@latest

# Stage 2: Application builder
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -ldflags="-s -w" -o husna-bot ./cmd/bot/main.go

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Stage 3: Runtime
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=goose-builder /go/bin/goose /usr/local/bin/goose

COPY --from=builder /app/husna-bot .

COPY --from=builder /app/migrations ./migrations

COPY --from=builder /app/config ./config
COPY --from=builder /app/assets ./assets

COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

CMD ["./entrypoint.sh"]