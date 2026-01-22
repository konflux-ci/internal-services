# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset:9.7-1768393489 as builder

ARG TARGETOS
ARG TARGETARCH

USER 1001

# Copy the Go Modules manifests
COPY --chown=1001:0 go.mod go.mod
COPY --chown=1001:0 go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY --chown=1001:0 . .

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

# Use ubi-minimal as minimal base image to package the manager binary
# See https://catalog.redhat.com/software/containers/ubi9-minimal/61832888c0d15aff4912fe0d
FROM registry.access.redhat.com/ubi9-minimal:9.7-1764794109
COPY --from=builder /opt/app-root/src/manager /

# It is mandatory to set these labels
LABEL name="Konflux Internal Services"
LABEL description="Konflux Internal Services"
LABEL io.k8s.description="Konflux Internal Services"
LABEL io.k8s.display-name="internal-services"
LABEL io.openshift.tags="internal-services"
LABEL summary="Konflux Internal Services"
LABEL com.redhat.component="internal-services"
LABEL vendor="Red Hat, Inc."
LABEL distribution-scope="public"
LABEL release="1"
LABEL url="github.com/konflux-ci/internal-services"
LABEL version="1"

USER 65532:65532

ENTRYPOINT ["/manager"]
