build:
	go build -o bin/po_translator cmd/po-translator/main.go

run: build
	OPENAI_API_KEY=$(OPENAI_API_KEY) bin/po_translator

clean:
	rm -rf bin

.PHONY: build run clean