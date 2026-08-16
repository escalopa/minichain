FROM golang:1.26-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /wallet ./cmd/wallet

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /wallet /wallet
ENV NODE_URL=http://node:3000

ENTRYPOINT ["/wallet"]
