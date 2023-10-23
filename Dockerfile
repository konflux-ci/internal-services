# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset:1.19 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY loader/ loader/
COPY metadata/ metadata/
COPY metrics/ metrics/
COPY tekton/ tekton/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

# Use ubi-micro as minimal base image to package the manager binary
# See https://catalog.redhat.com/software/containers/ubi9/ubi-micro/615bdf943f6014fa45ae1b58
FROM registry.access.redhat.com/ubi9/ubi-micro:9.2-15.1696515526
WORKDIR /
COPY --from=builder /workspace/manager .

# It is mandatory to set these labels
LABEL description="RHTAP Internal Services"
LABEL io.k8s.description="RHTAP Internal Services"
LABEL io.k8s.display-name="internal-services"
LABEL summary="RHTAP Internal Services"

USER 65532:65532

ENTRYPOINT ["/manager"]
