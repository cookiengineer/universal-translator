# Models

Translation models come from two sources and are downloaded on demand into a
local cache. The catalog is embedded in the binary, so discovering which models
exist requires no network access.

## Catalog sources

Both catalogs are embedded with `go:embed` and merged at init in
`models/catalog.go`.

1. **`models/catalog.json`** — the official Bergamot models hosted at
   `data.statmt.org`. Each model is a single `.tar.gz` archive that already
   contains its config. These are preferred when a pair is served by both
   sources.

2. **`models/firefox_catalog.json`** — the Firefox Translations models
   (111 language pairs) hosted on Mozilla's Remote Settings CDN. Each model is
   split into individual files (`model.<pair>.intgemm.alphas.bin`,
   `vocab.<pair>.spm`, `lex.50.50.<pair>.s2t.bin`) and its
   `config.intgemm8bitalpha.yml` is generated rather than shipped.

### The Model struct

`models.Model` supports both forms:

```go
type Model struct {
    ShortName, Name              string
    SourceName, SourceCode       string
    TargetName, TargetCode       string
    ModelType                    string   // "base" or "tiny"

    URL, Checksum                string   // archive form
    Files     []ModelFile        // multi-file form
    ConfigYAML string            // generated config (multi-file form)
}
```

`IsArchive()` reports whether the model is the archive form. A `ModelFile`
carries `Filename`, `URL` and `SHA256`.

## Storage

Models are looked up in two locations, in priority order (see
`repositories.ModelStore`):

1. `~/.cache/universal-translator/models/` — writable; populated by in-app
   downloads. Takes precedence over the system-wide directory.
2. `/usr/share/universal-translator/models/` — read-only; installed by the
   `universal-translator-languages` package.

Both share the same on-disk layout: one directory per model (named by its
`ShortName`) containing `config.intgemm8bitalpha.yml` plus the model and vocab
files. A model counts as installed only if that config file is present.

## Download / install flow

`ModelManager.ensureInstalled` (in `services/models`):

1. Returns early if `ModelStore.IsInstalled(shortName)`.
2. Creates a staging directory (`<cache>/<shortName>.extracting`).
3. Calls `Downloader.Download(model, stagingDir, onProgress)`:
   - archive form — downloads the `.tar.gz`, verifies its SHA-256, and
     extracts it (stripping the leading directory);
   - multi-file form — downloads each file, verifies its SHA-256, and writes
     the generated config.
4. Calls `ModelStore.Commit`, which validates that the config exists and
   atomically renames the staging directory into place.

Downloads are checksum-verified throughout; a failed download leaves no
partial state behind.

## Resolution: direct vs pivot

`models.Resolve(pair)`:

1. Looks for a direct model whose source/target codes match the pair,
   preferring a `tiny` variant when both `tiny` and `base` exist.
2. Otherwise falls back to pivoting through English: `source → en` and
   `en → target`. This is what lets, for example, Russian → Dutch work even
   though no direct model exists.

The translator service loads one model for a direct pair, or two for a pivot,
and uses `bergamot_bridge_pivot` in the latter case.

## Regenerating the Firefox catalog

The Firefox catalog is generated from Mozilla's Remote Settings registry:

```sh
python3 scripts/generate_firefox_catalog.py
```

The script picks the best version per pair (prefers tiny, falls back to base),
handles shared vs split vocabularies (`srcvocab`/`trgvocab` for Chinese), and
writes the per-model config. Re-run it and commit `models/firefox_catalog.json`
when the upstream registry changes.
