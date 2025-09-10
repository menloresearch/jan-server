# Simple Docker image to run k6-based load tests for Jan Server
# Supports two modes:
# 1) Mount your local tests directory to /tests to use local files and persist results
# 2) Use the baked-in copy of the tests (copied at build time)

FROM grafana/k6:latest

USER root
RUN apt-get update \
    && apt-get install -y --no-install-recommends bash jq ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Baked-in tests (fallback when no volume is mounted)
WORKDIR /app
COPY run-loadtest.sh /app/run-loadtest.sh
COPY src /app/src
RUN chmod +x /app/run-loadtest.sh

# Lightweight entrypoint that prefers mounted /tests when available
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD [""]
