# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) once it
reaches v1.

## [0.1.0] - 2026-09-09

First release.

### Added
- Public, importable library API. The decoder moved out of `internal/` into the
  root `iso8583` package (`Parse`, `Message`, `Field`, `Bitmap`), with
  `packager` and `annotate` as top-level subpackages.
- `Message.Pack` — serialize a message to wire bytes, computing the bitmap
  (including a secondary bitmap) and `LLVAR`/`LLLVAR` length prefixes.
- `NewMessage` — assemble a message from field values, ready to `Pack`.
- `Validate` — advisory sanity checks over a parsed message (Luhn on the PAN,
  numeric amounts, well-formed currency codes), returning structured `Finding`s.
- CLI `validate` subcommand, exiting non-zero on error-severity findings.
- **BCD (packed-decimal) field encoding.** Fields may be declared `encoding:
  bcd`; `Parse` and `Pack` handle packed values and packed length prefixes
  (two digits per byte). A shipped `bcd-demo` dialect exercises it.
- **Browser demo (WebAssembly).** The decoder compiles to WASM (`cmd/wasm`) and
  is served via GitHub Pages — paste a hex message and decode it client-side.
- **LLM-assisted generation.** `Generate` turns a natural-language description
  into a valid message via a provider-neutral `llm.Generator` interface; the
  model only picks field values, and `Pack` guarantees well-formed bytes.
  Adapters `llm/claude` and `llm/openai` call their APIs over plain HTTP (no SDK
  dependency added to the module). New CLI `generate --provider claude|openai`.
  Keys are read from the environment (`ANTHROPIC_API_KEY` / `OPENAI_API_KEY`);
  see `.env.example`.
- Package documentation and runnable examples (`ExampleParse`,
  `ExampleMessage_Pack`).
- `ROADMAP.md`, `LICENSE` (MIT), and this changelog.

### Changed
- Module path is now `github.com/Witkas/iso8583` (package `iso8583`), removing
  the package-name/path mismatch; the GitHub repository was renamed to match.
  The CLI command remains `iso8583lens`. Consumers import
  `github.com/Witkas/iso8583` and call `iso8583.Parse` / `iso8583.Pack` with no
  import alias.

### Notes
- `Parse` and `Pack` round-trip: packing a parsed message reproduces the
  original bytes.
