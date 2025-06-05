# Build stage
FROM --platform=$BUILDPLATFORM golang:1.23.8-alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN apk update && apk add --no-cache bash git gcc musl-dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN export CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} && \
  go build -ldflags="-w -s" -o build/blobcast .

# Runtime stage
FROM alpine:latest

ARG UID=10001
ARG USER_NAME=blobcast

ENV BLOBCAST_HOME=/app/${USER_NAME}

RUN apk update && apk add --no-cache bash ca-certificates

RUN adduser ${USER_NAME} \
  -D \
  -g ${USER_NAME} \
  -h ${BLOBCAST_HOME} \
  -s /sbin/nologin \
  -u ${UID}

# Copy the binary from builder
COPY --from=builder /src/build/blobcast /bin/blobcast

USER ${USER_NAME}

WORKDIR ${BLOBCAST_HOME}

# Expose ports
# gRPC port
EXPOSE 50051
# HTTP gateway port
EXPOSE 8080
# REST API port
EXPOSE 8081

ENTRYPOINT ["/bin/blobcast"]
