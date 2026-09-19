# syntax=docker/dockerfile:1

# CodeAtlas with the real Tree-sitter parser. CGO is enabled here so the
# high-fidelity parser (internal/parser/*.go, //go:build cgo) and the bundled
# grammar C sources are compiled in — the pure-Go fallback parser is only used
# when CGO is off. The frontend is already built into internal/webui/dist and
# embedded via go:embed, so no Node stage is needed.
FROM golang:1.26-bookworm AS build
WORKDIR /src
ENV CGO_ENABLED=1

# Cache modules first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/codeatlas ./cmd/codeatlas

# CGO links against glibc, so the runtime is Debian slim (not scratch/alpine).
# git is included so remote repositories can be cloned for analysis.
FROM debian:bookworm-slim
RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates git \
	&& rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/codeatlas /usr/local/bin/codeatlas

# PORT defaults to 7331 for a plain `docker run`; platforms like Railway inject
# their own PORT and the server binds 0.0.0.0:$PORT (see defaultListen). The
# database lives under CODEATLAS_DATA_DIR (/data); mount a persistent volume
# there so it survives restarts and redeploys. There is deliberately no VOLUME
# instruction: Railway rejects it and manages its own volumes, while a plain
# `docker run` uses `-v <name>:/data`.
ENV PORT=7331 \
	CODEATLAS_DATA_DIR=/data
EXPOSE 7331

ENTRYPOINT ["codeatlas", "serve"]
