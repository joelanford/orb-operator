FROM gcr.io/distroless/static:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/orb-operator /orb-operator
USER 65532:65532
ENTRYPOINT ["/orb-operator"]
