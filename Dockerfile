# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.27-bookworm AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=secret,id=build_ca,required=false \
            if [ -f /run/secrets/build_ca ]; then \
                  cat /etc/ssl/certs/ca-certificates.crt /run/secrets/build_ca > /tmp/ca-certificates.crt; \
                  SSL_CERT_FILE=/tmp/ca-certificates.crt go mod download; \
                  rm /tmp/ca-certificates.crt; \
            else \
                  go mod download; \
            fi
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/d2mcp ./cmd

FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="d2mcp" \
      org.opencontainers.image.description="D2 diagram generation and editing MCP server" \
      org.opencontainers.image.source="https://github.com/recursivefunctions/d2mcp" \
      org.opencontainers.image.licenses="MIT" \
      io.modelcontextprotocol.server.name="io.github.RecursiveFunctions/d2mcp"

WORKDIR /workspace
COPY --from=build /out/d2mcp /usr/local/bin/d2mcp

ENTRYPOINT ["/usr/local/bin/d2mcp"]
CMD ["-transport=stdio", "-workspace-root=default=/workspace"]