FROM golang:1.26-bookworm AS builder

WORKDIR /app
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /explorer ./cmd/explorer

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /explorer /explorer
EXPOSE 8080
ENV PORT=8080 NODE_URL=http://node:3000

ENTRYPOINT ["/explorer"]
