# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.26-alpine AS build

WORKDIR /src

# Download dependencies first so they cache independently of source changes.
# go-tgbot is fetched from GitHub via the pseudo-version in go.mod (no local
# replace). If that repo is private, pass build credentials (see README).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary (mongo-driver and go-tgbot are pure Go; CGO not needed).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bot ./cmd/bot

# ---- runtime stage ----
FROM alpine:3.20

# TLS roots for the Telegram API (HTTPS) and Mongo.
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app

WORKDIR /app
COPY --from=build /bot /app/bot
COPY data/ /app/data/

# Drop root: the bot needs no privileges.
USER app

# Defaults; BOT_TOKEN must be provided at runtime.
ENV MONGO_URI=mongodb://mongo:27017 \
    MONGO_DB=irregular_verbs \
    VERBS_PATH=/app/data/verbs.json

ENTRYPOINT ["/app/bot"]
