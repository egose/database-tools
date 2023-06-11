# See https://hub.docker.com/_/golang/tags
FROM golang:1.20.5-bullseye as builder

WORKDIR /app

COPY . .
RUN make build

# See https://hub.docker.com/_/alpine/tags
FROM alpine:3.18

RUN adduser -D -u 1000 mongotool
RUN chown mongotool:mongotool /tmp

COPY --from=builder /app/dist /usr/local/bin

USER mongotool

WORKDIR /home/mongotool

ENV TZ America/Vancouver
