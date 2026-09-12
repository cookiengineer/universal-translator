NATIVE_BUILD_DIR := native/build
NATIVE_LIB      := $(NATIVE_BUILD_DIR)/libbergamot_bridge.so
BINARY          := bin/universal-translator
PREFIX          ?= /usr/local

.PHONY: all native build test test-e2e install clean

all: build

# Build the vendored Bergamot/Marian/SentencePiece C++ library.
native: $(NATIVE_LIB)

$(NATIVE_LIB):
	cd native && cmake -S . -B build -G Ninja \
		-DCMAKE_BUILD_TYPE=Release \
		-DCMAKE_POLICY_VERSION_MINIMUM=3.5
	cd native && cmake --build build --target bergamot_bridge

# Build the Go application and bundle the native library next to it.
build: native
	mkdir -p bin
	go build -o $(BINARY) ./cmd/universal-translator
	cp $(NATIVE_LIB) bin/

test:
	go test ./...

test-e2e:
	go test -tags e2e -run 'TestEndToEndTranslation|TestPivotTranslation' ./adapters/bergamot/

install: build
	install -Dm755 $(BINARY) $(PREFIX)/bin/universal-translator
	install -Dm755 bin/libbergamot_bridge.so $(PREFIX)/lib/libbergamot_bridge.so

clean:
	rm -rf bin native/build
