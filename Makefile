OUT=lambdabot
GO=$(shell which go)
GOFMT=$(shell which gofmt)

all: build

install: install-rc
	$(GO) install .
	@mv $(OUT) /usr/local/bin/$(OUT)
	echo "The binary has been installed to /usr/local/bin/$(OUT)"

install-rc:
	@echo "Installing rc.d script..."
	@cp ./rc /usr/local/etc/rc.d/lambdabot
	@chmod +x /usr/local/etc/rc.d/lambdabot
	@echo "The rc.d script has been installed successfully."

build: fmt
	GOOS=freebsd GOARCH=amd64 $(GO) build -o $(OUT) .

run: build
	./$(BINARY_NAME)

fmt:
	$(GOFMT) -s -w .

.PHONY: all build run fmt install install-rc