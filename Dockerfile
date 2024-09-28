# See https://hub.docker.com/_/golang/tags
FROM golang:1.23.1-bullseye AS build

WORKDIR /app

COPY . .
RUN make build

# See https://hub.docker.com/_/alpine/tags
FROM alpine:3.20.1

RUN adduser -D -u 1000 nonroot

RUN apk --no-cache add tzdata

COPY --from=build /app/dist /usr/local/bin

USER nonroot

WORKDIR /home/nonroot

ENV TZ=America/Vancouver
