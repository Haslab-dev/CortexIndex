# T13 — Makefile, native build, install, PATH

**Status:** Done
**Depends on:** T01, T08

## Goal

Provide repeatable native macOS/Linux build and user-local install commands without requiring a package manager or mutating shell startup files.

## Acceptance criteria

- [x] `make build` writes executable `dist/cortex` with host-native cgo enabled (macOS verified).
- [x] `make test`, `make vet`, and `make check` work.
- [x] `make install` defaults to `$HOME/.local/bin`, supports `PREFIX`, `BINDIR`, and `DESTDIR`, and preserves executable mode (staged install verified mode 0755).
- [x] `make uninstall` removes only the selected install target.
- [x] `make env` prints a sourceable PATH export.
- [x] Unsupported host platforms fail with a clear Windows-future message.
- [x] README documents cgo/C compiler requirements, native macOS/Linux scope, PATH setup, and direct Go fallback.
