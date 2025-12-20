FROM golang:1.25 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -o husna-bot ./cmd/bot/main.go

#RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/husna-bot .

#COPY --from=builder /go/bin/goose /usr/local/bin/goose

COPY config ./config
COPY assets ./assets
#COPY migrations ./migrations
#
#COPY entrypoint.sh ./entrypoint.sh
#RUN chmod +x ./entrypoint.sh
#
#CMD ["./entrypoint.sh"]
CMD ["./husna-bot"]