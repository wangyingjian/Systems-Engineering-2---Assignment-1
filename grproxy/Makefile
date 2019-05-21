GOPATH := ${PWD}:${GOPATH}
export GOPATH

default: build

build:
	go build -tags netgo -v -o ./grproxy/grproxy ./src/grproxy
