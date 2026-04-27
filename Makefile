.PHONY: build clean run tidy

BINARY := crush-a2a

build: tidy
	go build -o $(BINARY) ./cmd/crush-a2a

tidy:
	go mod tidy

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
