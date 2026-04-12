VERSION := 1.0.0
BINARY := landrop
LDFLAGS := -s -w

.PHONY: all clean build-all

all: build-all

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY).exe .

build-all: build-windows build-darwin build-linux

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-windows-amd64.exe .

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-darwin-arm64 .

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-linux-arm64 .

clean:
	rm -rf dist/ $(BINARY).exe
