all:
	go generate
	go build
	go install

test:
	go test ./...