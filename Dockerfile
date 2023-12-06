# See https://hub.docker.com/_/golang/tags
FROM golang:1.21.5-bullseye AS builder

WORKDIR /app

COPY . .
RUN make build

# See https://hub.docker.com/_/alpine/tags
FROM alpine:3.18.5

RUN adduser -D -u 1000 nonroot

RUN apk --no-cache add tzdata

COPY --from=builder /app/dist /usr/local/bin

USER nonroot

WORKDIR /home/nonroot

# ENV TZ America/Vancouver
