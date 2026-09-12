# Arch Linux packaging

Two packages are provided:

- `universal-translator` builds the GTK4 application and the bundled
  `libbergamot_bridge.so` (the vendored Bergamot/Marian engine).
- `universal-translator-languages` bundles every language model and
  installs them into the system-wide models directory, so the app can translate
  offline without downloading models first.

## System-wide models directory

The application looks for models in two places:

1. `~/.cache/universal-translator/models/` is writable and populated by in-app
   downloads (takes precedence).
2. `/usr/share/universal-translator/models/` is read-only and populated by the
   `universal-translator-languages` package.

Both share the same on-disk layout: a directory per model containing
`config.intgemm8bitalpha.yml`, `model.intgemm.alphas.bin`, the sentencepiece
vocabularies and the shortlist.

## Building

Build the binary package:

```sh
cd packages/archlinux/universal-translator;
makepkg -si;
```

Build the model bundle (downloads ~600 MB of model archives and produces a
large package):

```sh
cd packages/archlinux/universal-translator-languages;
makepkg -si;
```

Installing both gives a fully offline translator with all supported language
pairs.

## Notes

- `_git_repo` at the top of `universal-translator/PKGBUILD` must point at the
  source repository.
- `pkgver` is a fixed placeholder; bump it (or switch to a `pkgver()`
  function using `git describe`) once tags exist.
- `license=('MIT')` is a placeholder until a LICENSE file is added to the
  repository.
- The `-languages` package model list is generated from `models/catalog.json`;
  regenerate the `source=`/`sha256sums=` arrays if the catalog changes.
