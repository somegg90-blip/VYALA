FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /vyala ./cmd/scanner

FROM python:3.12-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && pip install --no-cache-dir --break-system-packages "semgrep>=1.70,<2"
COPY --from=build /vyala /usr/local/bin/vyala
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /usr/local/bin/vyala
ENTRYPOINT ["/entrypoint.sh"]# Force rebuild 2026-07-18
