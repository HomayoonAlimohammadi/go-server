.PHONY: test
test:
	go test ./app/ -v --coverprofile=coverage.out
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
