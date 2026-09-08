SHELL := /bin/sh

BINARY ?= cortex
CMD ?= ./cmd/cortex
DIST ?= dist
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=
GO ?= go
CGO_ENABLED ?= 1
GOFLAGS ?=

.PHONY: all build test vet check install uninstall env run clean help

all: build

# Cortex uses Tree-sitter's cgo bindings. Build natively on macOS/Linux.
build:
	@case "$$(uname -s)" in Darwin|Linux) ;; *) echo "Cortex currently supports native macOS and Linux builds only (Windows support is planned)." >&2; exit 1 ;; esac
	@mkdir -p "$(DIST)"
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath -o "$(DIST)/$(BINARY)" $(CMD)

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: test vet build

install: build
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 0755 "$(DIST)/$(BINARY)" "$(DESTDIR)$(BINDIR)/$(BINARY)"
	@echo "Installed $(BINARY) to $(DESTDIR)$(BINDIR)/$(BINARY)"

uninstall:
	rm -f "$(DESTDIR)$(BINDIR)/$(BINARY)"

# Print a command that updates PATH in the calling shell:
# eval "$(make env)"
env:
	@printf 'export PATH="%s:$$PATH"\n' "$(BINDIR)"

run: build
	"$(DIST)/$(BINARY)" $(ARGS)

clean:
	rm -rf "$(DIST)"

help:
	@printf '%s\n' \
		'make build                         Build dist/cortex' \
		'make test                         Run Go tests' \
		'make vet                          Run go vet' \
		'make check                        Run tests, vet, and build' \
		'make install                      Install to $$(HOME)/.local/bin' \
		'make install PREFIX=/usr/local   Override install prefix' \
		'make env                          Print a sourceable PATH export' \
		'make run ARGS="version"          Build and run Cortex' \
		'make clean                        Remove build output'

bench: build
	$(DIST)/$(BINARY) benchmark --iterations $${ITERATIONS:-3}
