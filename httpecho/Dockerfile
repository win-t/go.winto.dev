FROM golang:alpine AS builder
WORKDIR /builder
COPY . .
RUN go build -o server .

FROM scratch
COPY --from=builder /builder/server /server
ENTRYPOINT ["/server"]
