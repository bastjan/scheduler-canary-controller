FROM docker.io/library/alpine:3.21 as runtime

RUN \
  apk add --update --no-cache \
    bash \
    curl \
    ca-certificates \
    tzdata

ENTRYPOINT ["/manager"]
COPY scheduler-canary-controller manager

USER 65536:0
