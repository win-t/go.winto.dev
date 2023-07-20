FROM golang:alpine AS builder
WORKDIR /builder
COPY . .
RUN go build -o envreplace .

FROM alpine
COPY --from=builder /builder/envreplace /envreplace
