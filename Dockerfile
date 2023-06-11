# See https://hub.docker.com/_/golang/tags
FROM golang:1.20.5-bullseye as builder

WORKDIR /app

COPY . .
RUN make build

# See https://hub.docker.com/_/alpine/tags
FROM alpine:3.18

RUN adduser -D -u 1000 nonroot

COPY --from=builder /app/dist /usr/local/bin

USER nonroot

WORKDIR /home/nonroot

ENV TZ America/Vancouver
