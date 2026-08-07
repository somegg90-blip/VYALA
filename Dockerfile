FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY go.sum ./ 
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /vyala ./cmd/scanner

FROM python:3.12-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && pip install --no-cache-dir --break-system-packages "semgrep==1.75.0"

# Create non-root user for security
RUN adduser --disabled-password --gecos "" --uid 1000 vyala

COPY --from=build /vyala /usr/local/bin/vyala

# FIX: Copy the Semgrep rules into the container so the engine can find them!
COPY --from=build /src/internal/rules /etc/vyala/rules

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /usr/local/bin/vyala

USER vyala
WORKDIR /github/workspace

ENTRYPOINT ["/entrypoint.sh"]