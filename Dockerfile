# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.25 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=0.0.0

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.mod
COPY go.sum go.sum

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Copy LICENSE for inclusion in final image
COPY LICENSE LICENSE

# Build for target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -ldflags "-X github.com/openshift-assessment/cluster-assessment-operator/pkg/version.Version=${VERSION}" -a -o manager main.go

# Use Red Hat UBI minimal as base image for Red Hat certification
FROM registry.access.redhat.com/ubi9/ubi-micro:latest

ARG VERSION=0.0.0

# Required labels for Red Hat certification
LABEL name="cluster-assessment-operator" \
    vendor="Community" \
    version="${VERSION}" \
    release="1" \
    summary="OpenShift Cluster Assessment Operator" \
    description="Read-only operator that performs assessments of OpenShift cluster configuration and generates reports with findings and recommendations." \
    maintainer="Diego <dperez@crossvale.com>" \
    io.k8s.display-name="Cluster Assessment Operator" \
    io.k8s.description="Read-only operator that performs assessments of OpenShift cluster configuration." \
    io.openshift.tags="openshift,assessment,operator,compliance"

WORKDIR /

# Copy binary
COPY --from=builder /workspace/manager .

# Create licenses directory and copy license
RUN mkdir -p /licenses
COPY --from=builder /workspace/LICENSE /licenses/LICENSE

USER 65532:65532

ENTRYPOINT ["/manager"]
