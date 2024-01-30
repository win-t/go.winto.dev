FROM golang:alpine AS builder
WORKDIR /builder
COPY . .
RUN CGO_ENABLED=0 go build -o envreplace .

FROM alpine
COPY --from=builder /builder/envreplace /envreplace
ENTRYPOINT ["/envreplace", "-"]
