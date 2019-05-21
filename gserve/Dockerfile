FROM scratch
FROM golang:alpine
RUN apk update
RUN apk --no-cache add make
RUN apk --no-cache add git
RUN go get github.com/samuel/go-zookeeper/zk
WORKDIR /
ADD /src/gserve /src/gserve
COPY Makefile Makefile

RUN make

ENTRYPOINT ["/gserve/gserve"]
