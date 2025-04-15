# Build the manager binary
FROM docker.io/golang:1.23 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace


COPY go.mod go.mod
COPY go.sum go.sum


RUN go env -w GO111MODULE=on && \
        go env -w GOPROXY=https://goproxy.cn,direct && \
        go mod download


COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/


RUN go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.cn,direct && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go



FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
