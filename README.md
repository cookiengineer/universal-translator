# Universal Translator

An offline, privacy-focused machine translation application for Linux built
with GTK4 and the [Bergamot](https://github.com/browsermt/bergamot-translator)
translation engine.

The application is self-contained: the entire inference engine (Bergamot,
Marian, SentencePiece, intgemm and ssplit) is compiled into a native library
and linked into the Go binary via CGo. No translation is ever sent to a remote
service. Models are stored in one of two places:

- `~/.cache/universal-translator/models/` are downloaded on demand by the app.
- `/usr/share/universal-translator/models/` are system-wide, installed by the `universal-translator-languages` package.

## Building

Requirements:

- Go 1.27+
- CMake 3.12+, Ninja
- GCC or Clang (C++17)
- GTK4 development headers (`pkg-config gtk4`)
- OpenBLAS (`libopenblas`) and PCRE2 development headers

Build:

```sh
make build;
```

This builds the native library under `native/build/` and produces
`bin/universal-translator` with `bin/libbergamot_bridge.so` alongside it.

Install system-wide (optional):

```sh
sudo make install;
```

Arch Linux users can build the two provided packages instead
(`packages/archlinux/`):

```sh
# cd packages/archlinux/universal-translator
makepkg -si;

# in packages/archlinux/universal-translator-languages (bundles all models)
makepkg -si;
```

## Usage

Run `bin/universal-translator`. Choose the source and target languages with the
two dropdowns. If the required model is not installed yet it is downloaded
automatically (with checksum verification), then loaded.

Type text in the left panel and press **Translate**; the result appears in the
right panel. When no direct model exists for a pair, the app automatically pivots
through English (e.g. Czech to English to German).

## Layout

```
cmd/universal-translator/ application entry point and wiring
app/                     GTK4 user interface (MainWindow)
bindings/gtk/            hand-written CGo bindings for GTK4
adapters/bergamot/       CGo bridge to the native Bergamot C ABI
services/
  translator/            background worker owning the engine
  models/                language-pair resolution and model preparation
  downloading/           HTTP download with SHA-256 verification
repositories/            on-disk model store (cache + system-wide)
models/                  embedded catalogs (Bergamot + Firefox Translations)
types/                   shared value types
interfaces/              ports (ModelStore, Downloader)
caches/                  XDG cache path resolution
native/
  bridge/                thin extern "C" ABI over Bergamot
  third_party/           vendored Bergamot/Marian/SentencePiece/intgemm/ssplit
scripts/                 catalog generation tooling
dist/                    desktop entry
packages/archlinux/      Arch Linux PKGBUILDs
docs/                    architecture and workflow documentation
```

## Documentation

See [docs/](docs/) for the architecture and workflows: the package layout and
threading model, the native C++ bridge, the model system, and the Arch Linux
packaging.


## Native library

`native/CMakeLists.txt` builds a single `libbergamot_bridge.so` from the
vendored C++ sources with a small `extern "C"` shim (`native/bridge/`).

The Go side only ever talks to this opaque-handle C ABI, never to C++ types
directly.

## Tests

```sh
make test        # unit tests (no network, no model download)
make test-e2e    # end-to-end translation (downloads models)
```

## License

[GPL 3.0 or later](./GPL-3.0-or-later.txt)

