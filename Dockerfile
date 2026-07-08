# --- build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src

# Download dependencies first so they cache independently of source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 produces a fully static binary (no libc) suitable for the
# distroless "static" base. -ldflags "-s -w" strips debug info to shrink it.
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/signpost .

# --- runtime stage ---
# distroless static: just the binary + CA certs, no shell or package manager.
# Runs as the nonroot user (UID 65532); the mounted config.toml must be
# world-readable (the default 0644 from `cp` is fine).
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /usr/src/app
COPY --from=build /out/signpost ./signpost
EXPOSE 3035
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD ["/usr/src/app/signpost", "-healthcheck"]
ENTRYPOINT ["/usr/src/app/signpost"]
