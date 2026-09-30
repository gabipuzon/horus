FROM golang:1.27-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/horus ./cmd/server \
    && CGO_ENABLED=0 go build -o /out/horus-migrate ./cmd/migrate

FROM alpine:3.21
RUN apk add --no-cache ca-certificates \
    && addgroup -S horus \
    && adduser -S -G horus horus
COPY --from=builder /out/horus /usr/local/bin/horus
COPY --from=builder /out/horus-migrate /usr/local/bin/horus-migrate
USER horus
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/horus"]
