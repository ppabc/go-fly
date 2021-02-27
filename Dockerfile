
FROM golang:1.11.3-alpine3.8 as builder
RUN mkdir /app
COPY . /app
ENV GOPROXY https://mirrors.aliyun.com/goproxy
VOLUME ["/app/config"]
WORKDIR /app/
RUN ["go","build","go-fly.go"]
RUN ["go-fly","install"]

FROM alpine:latest

COPY --from=builder /app/go-fly  /app/go-fly

EXPOSE 8081
CMD ["/app/go-fly","server"]
