# See docs/release-artifact-policy.md for the release toolchain contract.
FROM golang:1.27.0 AS build

WORKDIR /app

COPY Makefile go.mod go.sum ./
COPY common ./common
COPY internal ./internal
COPY mongoarchive ./mongoarchive
COPY mongounarchive ./mongounarchive
COPY notification ./notification
COPY storage ./storage
COPY utils ./utils
RUN make build

# See https://hub.docker.com/_/alpine/tags
FROM alpine:3.23.4

RUN adduser -D -u 1000 nonroot

RUN apk --no-cache add tzdata

COPY --from=build /app/dist/mongo-archive /usr/local/bin/mongo-archive
COPY --from=build /app/dist/mongo-unarchive /usr/local/bin/mongo-unarchive

USER nonroot

WORKDIR /home/nonroot

ENV TZ=America/Vancouver
