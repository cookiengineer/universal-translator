#!/usr/bin/env python3
"""Generate the Firefox Translations model catalog.

Fetches the Mozilla Remote Settings "translations-models" collection and emits
models/firefox_catalog.json, a static catalog used by the application at
runtime (no network access needed at startup).

Usage:
    python3 scripts/generate_firefox_catalog.py
"""

import json
import os
import re
import sys
import urllib.request

REMOTE_SETTINGS_URL = (
    "https://firefox.settings.services.mozilla.com/v1/buckets/main/"
    "collections/translations-models/records"
)
CDN_BASE = "https://firefox-settings-attachments.cdn.mozilla.net/"

# ISO code -> human-readable name, matching the style of the existing catalog.
LANGUAGE_NAMES = {
    "en": "English",
    "af": "Afrikaans",
    "ar": "Arabic",
    "az": "Azerbaijani",
    "be": "Belarusian",
    "bg": "Bulgarian",
    "bn": "Bengali",
    "bs": "Bosnian",
    "ca": "Catalan",
    "cs": "Czech",
    "da": "Danish",
    "de": "German",
    "el": "Greek",
    "es": "Spanish",
    "et": "Estonian",
    "eu": "Basque",
    "fa": "Persian",
    "fi": "Finnish",
    "fr": "French",
    "gl": "Galician",
    "gu": "Gujarati",
    "he": "Hebrew",
    "hi": "Hindi",
    "hr": "Croatian",
    "hu": "Hungarian",
    "id": "Indonesian",
    "is": "Icelandic",
    "it": "Italian",
    "ja": "Japanese",
    "kn": "Kannada",
    "ko": "Korean",
    "lt": "Lithuanian",
    "lv": "Latvian",
    "ml": "Malayalam",
    "mr": "Marathi",
    "ms": "Malay",
    "mt": "Maltese",
    "nb": "Norwegian (Bokmål)",
    "nl": "Dutch",
    "nn": "Norwegian (Nynorsk)",
    "pl": "Polish",
    "pt": "Portuguese",
    "ro": "Romanian",
    "ru": "Russian",
    "sk": "Slovak",
    "sl": "Slovenian",
    "sq": "Albanian",
    "sr": "Serbian",
    "sv": "Swedish",
    "ta": "Tamil",
    "te": "Telugu",
    "th": "Thai",
    "tr": "Turkish",
    "uk": "Ukrainian",
    "ur": "Urdu",
    "vi": "Vietnamese",
    "zh-Hans": "Chinese (Simplified)",
    "zh-Hant": "Chinese (Traditional)",
}

VERSION_RE = re.compile(r"^(\d+)\.(\d+)([a-z]*)(\d*)$")
TINY_MODEL_SIZE = 20_000_000  # bytes; below this a model is considered "tiny"


def version_key(version):
    match = VERSION_RE.match(version)
    if not match:
        return (0, 0, False, 0)
    major = int(match.group(1))
    minor = int(match.group(2))
    alpha_suffix = match.group(3)
    alpha_number = int(match.group(4) or 0)
    is_stable = alpha_suffix == ""
    return (major, minor, is_stable, alpha_number)


def fetch_records():
    with urllib.request.urlopen(REMOTE_SETTINGS_URL, timeout=60) as response:
        return json.load(response)["data"]


def group_records(records):
    """group -> version -> fileType -> record."""
    grouped = {}
    for record in records:
        source = record.get("fromLang")
        target = record.get("toLang")
        version = record.get("version")
        file_type = record.get("fileType")
        if not source or not target or not version or not file_type:
            continue
        grouped.setdefault((source, target), {}).setdefault(version, {})[file_type] = record
    return grouped


def complete_versions(pair_files):
    """Return (version, files) for versions that have model + lex + vocab."""
    results = []
    for version, files in pair_files.items():
        if "model" not in files or "lex" not in files:
            continue
        if "vocab" not in files and not ("srcvocab" in files and "trgvocab" in files):
            continue
        results.append((version, files))
    return results


def model_size(files):
    return files["model"].get("attachment", {}).get("size", 0)


def pick_version(complete):
    """Pick the best version: prefer tiny, then the highest version."""
    tiny = [(v, f) for v, f in complete if model_size(f) < TINY_MODEL_SIZE]
    candidates = tiny or complete
    candidates.sort(key=lambda item: version_key(item[0]), reverse=True)
    return candidates[0]


def file_entry(filename, record):
    attachment = record["attachment"]
    return {
        "filename": filename,
        "url": CDN_BASE + attachment["location"],
        "sha256": attachment["hash"],
    }


def build_config(source, target, files):
    vocab = _vocab_names(files)
    model_name = files["model"]["name"]
    lex_name = files["lex"]["name"]
    lines = [
        "relative-paths: true",
        "models:",
        f"  - {model_name}",
        "vocabs:",
        f"  - {vocab[0]}",
        f"  - {vocab[1]}",
        "shortlist:",
        f"    - {lex_name}",
        "    - false",
        "beam-size: 1",
        "normalize: 1.0",
        "word-penalty: 0",
        "mini-batch: 64",
        "maxi-batch: 1000",
        "maxi-batch-sort: src",
        "workspace: 2000",
        "max-length-factor: 2.5",
        "gemm-precision: int8shiftAlphaAll",
    ]
    return "\n".join(lines) + "\n"


def _vocab_names(files):
    if "srcvocab" in files and "trgvocab" in files:
        return (files["srcvocab"]["name"], files["trgvocab"]["name"])
    vocab_name = files["vocab"]["name"]
    return (vocab_name, vocab_name)


def build_model(source, target, version, files):
    model_files = []
    for file_type in ("model", "lex"):
        model_files.append(file_entry(files[file_type]["name"], files[file_type]))

    if "srcvocab" in files and "trgvocab" in files:
        model_files.append(file_entry(files["srcvocab"]["name"], files["srcvocab"]))
        model_files.append(file_entry(files["trgvocab"]["name"], files["trgvocab"]))
    else:
        model_files.append(file_entry(files["vocab"]["name"], files["vocab"]))

    model_type = "tiny" if model_size(files) < TINY_MODEL_SIZE else "base"
    source_name = LANGUAGE_NAMES.get(source, source)
    target_name = LANGUAGE_NAMES.get(target, target)

    return {
        "shortName": f"fx-{source}-{target}",
        "name": f"{source_name}-{target_name}",
        "sourceName": source_name,
        "sourceCode": source,
        "targetName": target_name,
        "targetCode": target,
        "type": model_type,
        "files": model_files,
        "config": build_config(source, target, files),
    }


def main():
    records = fetch_records()
    grouped = group_records(records)

    models = []
    for (source, target), pair_files in sorted(grouped.items()):
        complete = complete_versions(pair_files)
        if not complete:
            continue
        version, files = pick_version(complete)
        models.append(build_model(source, target, version, files))

    models.sort(key=lambda m: m["shortName"])

    output_path = os.path.join(
        os.path.dirname(__file__), "..", "models", "firefox_catalog.json"
    )
    with open(output_path, "w") as output:
        json.dump({"models": models}, output, indent=2, ensure_ascii=False)
        output.write("\n")

    print(f"wrote {output_path} with {len(models)} models")


if __name__ == "__main__":
    sys.exit(main())
