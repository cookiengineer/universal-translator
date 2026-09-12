# Packaging (Arch Linux)

Two packages live under `packages/archlinux/`:

- **`universal-translator`** — the GTK4 application and the bundled
  `libbergamot_bridge.so`.
- **`universal-translator-languages`** — bundles the official Bergamot models
  (`models/catalog.json`) into the system-wide model directory.

## universal-translator

`PKGBUILD` builds from the git repository (`_git_repo`), runs `make build`
with the standard Arch Go flags (`-buildmode=pie`, `-trimpath`, `-mod=readonly`)
and installs:

- `bin/universal-translator` → `/usr/bin/universal-translator`
- `bin/libbergamot_bridge.so` → `/usr/lib/libbergamot_bridge.so`
- `dist/universal-translator.desktop` → `/usr/share/applications/`
- the GPL-3.0-or-later license → `/usr/share/licenses/`

Dependencies: `gtk4`, `openblas`, `gcc-libs` (runtime); `go`, `cmake`, `ninja`,
`gcc`, `openblas`, `pcre2` (build).

Build and install:

```sh
cd packages/archlinux/universal-translator
makepkg -sf
sudo pacman -U universal-translator-*.pkg.tar.zst
```

## universal-translator-languages

This package installs the pre-built Bergamot models into
`/usr/share/universal-translator/models/`, which the application reads as the
read-only system-wide model root. Its `source=`/`sha256sums=` arrays are
generated from `models/catalog.json`; regenerate them if the catalog changes.

Only the data.statmt.org archive models are bundled. The Firefox Translations
models use UUID-based CDN URLs and multi-file downloads, so they are fetched
on demand by the application rather than packaged.

## Notes for maintainers

- `_git_repo` and `url` in `universal-translator/PKGBUILD` must point at the
  source repository; `pkgver` is a placeholder until releases are tagged.
- `makepkg` leaves `src/`, `pkg/` and a git cache directory behind; these are
  ignored by the repository `.gitignore`.
