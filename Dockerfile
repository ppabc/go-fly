
FROM golang:alpine as builder
RUN mkdir /app
COPY . /app
ENV GOPROXY https://mirrors.aliyun.com/goproxy
VOLUME ["/app/config"]
WORKDIR /app/
RUN ["go","build","go-fly.go"]


FROM alpine:latest

COPY --from=builder /app/go-fly  /app/go-fly

EXPOSE 8081
##RUN ["/app/go-fly","install"]
CMD ["/app/go-fly","server"]
