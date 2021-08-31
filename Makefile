build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build cmd/main.go
	mv main build/

build-docker: build
	docker build -t isoftstone/takeover build/

.PHONY: all build build-docker
