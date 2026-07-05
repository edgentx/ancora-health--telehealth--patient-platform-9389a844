FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
# Build both deployable processes: the HTTP API server and the Temporal worker.
RUN CGO_ENABLED=0 go build -o /server ./cmd/server
RUN CGO_ENABLED=0 go build -o /worker ./cmd/worker

FROM alpine:3.19
COPY --from=builder /server /server
COPY --from=builder /worker /worker
EXPOSE 8000
# Default to the API server; deploy the worker with `CMD ["/worker"]` (or an
# override) as a separate process on the same image.
CMD ["/server"]
