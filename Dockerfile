# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.20.5 as builder

ARG GOARCH=''

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    go mod download

# Copy the go source
COPY cmd/ cmd
COPY pkg/ pkg/
COPY charts charts/

ARG TARGETOS
ARG TARGETARCH

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} go build -a -o gardener-extension-provider-onmetal ./cmd/gardener-extension-provider-onmetal/main.go && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} go build -a -o gardener-extension-admission-onmetal ./cmd/gardener-extension-admission-onmetal/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS gardener-extension-provider-onmetal
WORKDIR /
COPY charts /charts
COPY --from=builder /workspace/gardener-extension-provider-onmetal /gardener-extension-provider-onmetal
USER 65532:65532

ENTRYPOINT ["/gardener-extension-provider-onmetal"]

FROM gcr.io/distroless/static:nonroot AS gardener-extension-admission-onmetal
WORKDIR /
COPY charts /charts
COPY --from=builder /workspace/gardener-extension-admission-onmetal /gardener-extension-admission-onmetal
USER 65532:65532

ENTRYPOINT ["/gardener-extension-admission-onmetal"]