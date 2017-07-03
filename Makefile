BINPATH ?= build

build:
	go build -tags 'production' -o $(BINPATH)/dp-frontend-upload-prototype

debug:
	go build -tags 'debug' -o $(BINPATH)/dp-frontend-upload-prototype
	HUMAN_LOG=1 DEBUG=1 $(BINPATH)/dp-frontend-upload-prototype

.PHONY: build debug
