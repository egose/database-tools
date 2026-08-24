# See docs/release-artifact-policy.md for the release toolchain contract.
FROM golang:1.26.6@sha256:0d1d3a794be25f809dd2cb3160d8c73276c4056a9f8242a138e908ddeee7b6b6 AS build

WORKDIR /app

ARG VERSION=localdev
ARG REVISION=unknown

COPY Makefile go.mod go.sum ./
COPY internal ./internal
COPY mongoarchive ./mongoarchive
COPY mongounarchive ./mongounarchive
COPY postgresarchive ./postgresarchive
COPY postgresunarchive ./postgresunarchive
COPY notification ./notification
COPY storage ./storage
COPY utils ./utils
RUN make build VERSION="${VERSION}"

# See https://hub.docker.com/_/alpine/tags
FROM alpine:3.23.5@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40

ARG VERSION=localdev
ARG REVISION=unknown
ARG SOURCE=https://github.com/egose/database-tools
ARG POSTGRESQL_CLIENT_PACKAGE=postgresql18-client=18.6-r0

LABEL org.opencontainers.image.title="database-tools" \
      org.opencontainers.image.description="MongoDB and PostgreSQL archive and restore tools" \
      org.opencontainers.image.source="${SOURCE}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.licenses="Apache-2.0"

RUN adduser -D -u 1000 nonroot

RUN apk --no-cache add tzdata "${POSTGRESQL_CLIENT_PACKAGE}" \
    && install -d -m 0700 -o nonroot -g nonroot /home/nonroot/tmp /home/nonroot/work

COPY --from=build /app/dist/mongo-archive /usr/local/bin/mongo-archive
COPY --from=build /app/dist/mongo-unarchive /usr/local/bin/mongo-unarchive
COPY --from=build /app/dist/postgres-archive /usr/local/bin/postgres-archive
COPY --from=build /app/dist/postgres-unarchive /usr/local/bin/postgres-unarchive

USER nonroot

WORKDIR /home/nonroot/work

ENV TZ=America/Vancouver
ENV TMPDIR=/home/nonroot/tmp
