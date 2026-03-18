BINARY := claude-relay
ifeq ($(OS),Windows_NT)
	BINARY := claude-relay.exe
endif

.PHONY: build run clean

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY) --token dev --dir .

clean:
	rm -f claude-relay claude-relay.exe
