FROM rust:1-bookworm AS builder

WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo build --release --locked

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --create-home --uid 10001 minichain

COPY --from=builder /app/target/release/minichain-node /usr/local/bin/minichain-node

USER minichain
EXPOSE 3000 4001
ENV PORT=3000 P2P_PORT=4001 DIFFICULTY=4
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=6 \
    CMD curl --fail --silent http://127.0.0.1:3000/blocks > /dev/null || exit 1

ENTRYPOINT ["minichain-node"]
