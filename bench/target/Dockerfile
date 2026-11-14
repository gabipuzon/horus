FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY bench/target/main.go ./main.go
RUN CGO_ENABLED=0 go build -o /out/target ./main.go

FROM alpine:3.21
COPY --from=builder /out/target /target
ENTRYPOINT ["/target"]
