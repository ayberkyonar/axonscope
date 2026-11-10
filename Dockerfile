# Builds a small image that runs axonscope:
#   docker build -t axonscope .
#   docker run --rm axonscope events --server host.docker.internal:8124

FROM golang:1.27 AS build
WORKDIR /src
# Download dependencies first, so they are cached when only the code changes.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# A static binary without C dependencies runs on the minimal image below.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /axonscope ./cmd/axonscope

# Contains only the binary and CA certificates (for TLS), and runs as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /axonscope /axonscope
ENTRYPOINT ["/axonscope"]
