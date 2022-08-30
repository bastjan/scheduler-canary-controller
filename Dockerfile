FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY scheduler-canary-controller manager
USER 65532:65532

ENTRYPOINT ["/manager"]
