all:
	go generate ./...
	go build ./cmd/cpush
	go build ./cmd/clitable
	go build ./cmd/rcheck

test:
	go test ./...

install: all test
	go install ./cmd/cpush
