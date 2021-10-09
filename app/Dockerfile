FROM golang:alpine AS builder

WORKDIR /go/src/github.com/m00lecule/stateful-scaling
COPY . /go/src/github.com/m00lecule/stateful-scaling
RUN go build -o app

FROM alpine:3.12
COPY --from=builder /go/src/github.com/m00lecule/stateful-scaling/app .
EXPOSE 8080
ENTRYPOINT ["./app"]