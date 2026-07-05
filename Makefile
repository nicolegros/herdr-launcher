.PHONY: build clean

build:
	go build -o bin/herdr-launcher .

clean:
	rm -rf bin/
