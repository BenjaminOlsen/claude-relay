.PHONY: build run clean

build:
	go build -o claude-relay.exe .

run: build
	./claude-relay.exe --token dev --dir .

clean:
	rm -f claude-relay.exe
