GO=go

build: # Build binaries
	./build.sh dione

test: # Run tests
	$(GO) test

run: # Run app
	$(GO) run .

run-watch: # Run app & watch for changes and restart
	watcher

.PHONY: build test

clean: # clean workspace
	$(GO) clean
	rm -rf ./build/*
