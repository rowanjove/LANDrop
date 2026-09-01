VERSION := 2.0.1
BINARY := landrop
LDFLAGS := -s -w

.PHONY: all clean build build-web build-all

all: build-all

build-web:
	cd web && npm ci && npm run build

build: build-web
	go build -ldflags "$(LDFLAGS)" -o $(BINARY).exe .

build-all: build-web build-windows build-darwin build-linux

dist:
	mkdir -p dist

build-windows: dist
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-windows-amd64.exe .

build-darwin: dist
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-darwin-arm64 .

build-linux: dist
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-v$(VERSION)-linux-arm64 .

clean:
	rm -rf dist/ $(BINARY).exe
