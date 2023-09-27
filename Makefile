all:
	go generate
	go build ./...
	go build

test:
	go test ./...

install: all test
	go install
