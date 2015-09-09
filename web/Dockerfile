FROM golang:1.5

COPY . /go/src/github.com/tombee/docker-demo
WORKDIR /go/src/github.com/tombee/docker-demo

RUN go get
RUN go build

CMD ["docker-demo"]

