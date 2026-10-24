.PHONY: all build clean cross-compile run serve

BINARY=patchpulse
CMD=./cmd/patchpulse

all: build

build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BINARY) $(CMD)

run:
	go run $(CMD)

serve:
	go run $(CMD) serve

cross-compile:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags "-s -w" -o dist/patchpulse-windows-arm64.exe $(CMD)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o dist/patchpulse-windows-amd64.exe $(CMD)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o dist/patchpulse-linux-arm64 $(CMD)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dist/patchpulse-linux-amd64 $(CMD)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o dist/patchpulse-darwin-arm64 $(CMD)

clean:
	rm -rf $(BINARY) $(BINARY).exe dist/

