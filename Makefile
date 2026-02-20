BINARY := smartir-tuya-converter
BINDIR := bin

.PHONY: build test install clean

build:
	go build -o $(BINDIR)/$(BINARY) ./cmd/smartir-tuya-converter

test:
	go test ./...

install:
	go install ./cmd/smartir-tuya-converter

clean:
	rm -rf $(BINDIR)
