# Native library

The inference engine is Bergamot + Marian + SentencePiece + intgemm + ssplit,
all C++ projects that are not available as system packages. They are vendored
under `native/third_party/` and compiled by `native/CMakeLists.txt` into a
single shared library, `libbergamot_bridge.so`, which is linked into the Go
binary via CGo.

## What is vendored

```
native/third_party/bergamot-translator/
  CMakeLists.txt            upstream build (patched)
  src/translator/           the bergamot-translator library
  3rd_party/
    marian-dev/             the Marian inference library
      src/3rd_party/
        sentencepiece/      browsermt fork (vocabulary handling)
        intgemm/            int8 GEMM kernels
    ssplit-cpp/             sentence splitting
```

The sources are copied into the tree (not git submodules) at pinned commits.
Several small patches were needed to make the vendored code build cleanly with
modern toolchains:

- `marian-dev`: removed `-Werror` (newer GCC emits false-positive
  array-bounds/stringop warnings); emitted a static `git_revision.h`; removed
  `microsoft/quicksand.cpp` and `microsoft/cosmos.cpp` (non-standalone
  Microsoft-internal sources).
- `bergamot-translator/.gitignore`: `models` changed to `/models` so the
  vendored `marian-dev/src/models/` *source* directory is not accidentally
  ignored (this previously caused those files to be missing from the git
  history).

## Build configuration

`native/CMakeLists.txt` pins a CPU-only, int8 configuration:

- `COMPILE_CUDA=OFF`, `USE_FBGEMM=OFF`, `USE_RUY=OFF`, `USE_ONNX_SGEMM=OFF`
- `USE_SENTENCEPIECE=ON`, `COMPILE_LIBRARY_ONLY=ON`
- `USE_STATIC_LIBS=OFF` — this lets CMake discover the system OpenBLAS shared
  library for Marian's float GEMMs; intgemm only covers the quantised int8
  GEMMs, so OpenBLAS is still required.
- `BLA_VENDOR=OpenBLAS` — avoids accidentally linking an installed MKL.
- `BUILD_ARCH=x86-64-v3` — a portable AVX2 baseline (avoids an AVX-512 codegen
  ICE in some GCC versions); intgemm still performs its own runtime CPU
  dispatch for the hot kernels.
- tcmalloc is disabled so its allocator does not interpose `malloc` inside the
  Go process.

## The C ABI

`native/bridge/bridge.{h,cpp}` exposes a tiny `extern "C"` ABI with opaque
handles, so the Go side never sees C++ types:

```c
BergamotBridge *bergamot_bridge_create(size_t num_workers, size_t cache_size);
void            bergamot_bridge_destroy(BergamotBridge *);
BergamotTranslationModel *bergamot_bridge_load_model(BergamotBridge *, const char *config_path);
void            bergamot_bridge_unload_model(BergamotTranslationModel *);
int             bergamot_bridge_translate(BergamotBridge *, BergamotTranslationModel *, const char *text, char **out);
int             bergamot_bridge_pivot(BergamotBridge *, BergamotTranslationModel *, BergamotTranslationModel *, const char *text, char **out);
void            bergamot_bridge_cancel(BergamotBridge *);
void            bergamot_bridge_free_string(char *);
const char     *bergamot_bridge_last_error(BergamotBridge *);
```

Internally the bridge owns an `AsyncService` and makes its asynchronous
interface appear blocking: `translate`/`pivot` enqueue a request and wait on a
condition variable for the worker-thread callback, while `cancel` calls
`service->clear()` and signals the waiter. A `PendingTranslation` state object
is heap-owned and shared through a `shared_ptr` so it survives a cancel that
returns before the callback fires.

## The CGo adapter

`adapters/bergamot/engine.go` is the only Go package that imports C. It:

- declares the bridge header and links the library with
  `#cgo LDFLAGS: -L…/native/build -lbergamot_bridge` plus rpaths (`$ORIGIN`
  and the source build directory);
- wraps the opaque handles in Go types (`Engine`, `Model`) with finalizers;
- returns Go errors from `bergamot_bridge_last_error`.

The linker needs the library built first, which is handled by the `native`
target in the top-level `Makefile`.
