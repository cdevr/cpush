all:
	go generate
	go build

test:
	go test ./...

install:
	go install
