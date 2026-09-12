# Architecture

The universal translator is an offline machine translation application built
with GTK4 and the [Bergamot](https://github.com/browsermt/bergamot-translator)
engine (a translation front-end over
[Marian](https://github.com/marian-nmt/marian-dev)). It is a greenfield Go
project that links the C++ inference engine into a single Go binary via CGo.

The guiding constraint is that the built application is self-contained: the
whole inference engine is compiled in, and the only dynamic libraries are GTK4
and common system libraries (OpenBLAS, libc, libm).

## Package layout

```
cmd/universal-translator/   application entry point and wiring
app/                        GTK4 user interface (MainWindow)
bindings/gtk/               hand-written CGo bindings for GTK4
adapters/bergamot/          CGo adapter over the native C ABI
services/
  translator/               background worker that owns the engine
  models/                   language-pair resolution and model preparation
  downloading/              HTTP downloader with SHA-256 verification
repositories/               on-disk model store (cache + system-wide)
models/                     embedded model catalogs and resolution logic
types/                      shared value types and constants
interfaces/                 ports (Downloader, ModelStore)
caches/                     XDG cache path resolution
native/
  bridge/                   thin extern "C" ABI over Bergamot
  CMakeLists.txt            builds libbergamot_bridge.so
  third_party/              vendored Bergamot/Marian/SentencePiece/intgemm/ssplit
scripts/                    catalog generation tooling
dist/                       desktop entry
packages/archlinux/         Arch Linux PKGBUILDs
```

## Layering

The code follows a ports-and-adapters style, with the dependency arrows
pointing inward:

```
UI (app, cmd) ──> interfaces ──> services ──> repositories/adapters
                    │  │              │              │
                    │  └── models ────┘              │
                    └────────────────────────────────┘
```

- `interfaces` defines the ports (`Downloader`, `ModelStore`). The concrete
  implementations live in `services/downloading` and `repositories`.
- `models` holds the embedded catalogs and the pure resolution logic
  (`Resolve`, `findModel`); it has no dependencies on the rest of the app.
- `adapters/bergamot` is the only place that touches C types. Everything above
  it deals in opaque handles and Go strings.
- `services/translator` serialises access to the engine (which is not
  thread-safe) behind a single worker goroutine.

## Data flow

1. **Language selection.** The UI lists source and target languages derived
   from the union of all catalogs. Choosing a pair triggers
   `ModelManager.IsAvailable` / `IsInstalled` to decide whether a model exists
   and whether it is already on disk.
2. **Resolution.** `models.Resolve` returns either a single direct model or two
   models to pivot through English.
3. **Preparation.** `ModelManager.Prepare` downloads any missing model into a
   staging directory and commits it to the cache.
4. **Loading.** `services/translator` loads the model(s) into the native
   engine through the C ABI.
5. **Translation.** Text flows from the UI into the worker, through the engine,
   and back out to the UI.

## Threading model

GTK4 is strictly single-threaded: all widget access must happen on the main
loop. The Bergamot engine is not safe to share across goroutines. These two
facts are reconciled with a single worker goroutine:

- `services/translator.Service` owns the engine. It consumes commands from a
  channel (`Load`, `Translate`, `Cancel`) and serialises all engine access.
- Results, errors and busy-state changes are reported through callbacks
  (`onResult`, `onError`, `onBusy`, `onLoaded`) which are invoked from the
  worker goroutine and marshalled onto the GTK main thread with
  `gtk.RunOnMain`.
- Downloads happen on ad-hoc goroutines in `cmd/universal-translator`; a
  monotonic `selectionGeneration` counter discards stale results when the user
  changes languages faster than downloads finish.
- The native bridge exposes a *blocking* `translate`/`pivot` (it wraps
  Bergamot's asynchronous API with a condition variable), so the worker can
  wait for a result and still be interrupted by `Cancel`, which calls
  `service->clear()` and unblocks the waiter.

## Key constants

- `models.EnglishCode = "en"` — the pivot language.
- `types.ModelConfigFileName = "config.intgemm8bitalpha.yml"` — the Marian
  config every model directory must contain.
- `caches.SystemModelsPath = "/usr/share/universal-translator/models"` — the
  read-only, system-wide model directory.
