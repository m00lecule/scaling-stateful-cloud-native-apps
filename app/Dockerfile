FROM golang:1.18-alpine AS builder

WORKDIR /go/src/github.com/m00lecule/stateful-scaling
COPY . /go/src/github.com/m00lecule/stateful-scaling
RUN go build -o app

FROM alpine:3.16
WORKDIR /stateful-app
COPY --from=builder /go/src/github.com/m00lecule/stateful-scaling/app .
EXPOSE 8080
ENTRYPOINT ["./app"]
