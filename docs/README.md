# Documentation

This directory describes the architecture and workflows of the universal
translator.

- [Architecture](architecture.md) — overall system design, package layout,
  data flow and threading model.
- [Native library](native-library.md) — the vendored C++ engine and the C ABI
  bridge that links it into the Go binary.
- [Models](models.md) — how translation models are catalogued, downloaded,
  stored and resolved.
- [Packaging](packaging.md) — the Arch Linux packages.
