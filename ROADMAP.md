# Roadmap

The vision, architecture decisions, and phased plan for turning this project from
a CLI decoder into a reusable Go **library** for working with ISO 8583 messages.

## Vision

A Go library that lets developers work with ISO 8583 without hand-rolling the
error-prone parts — **bitmaps, MTI, length prefixes (LLVAR/LLLVAR), and field
encoding are computed for you**. A CLI is one consumer of the library; an
optional, provider-neutral LLM layer lets a developer describe a message in
plain English and get valid bytes back.

The wedge (versus the mature [`moov-io/iso8583`](https://github.com/moov-io/iso8583)):
**approachability, a learning-oriented lens, and the LLM-assisted generator** —
not feature parity.

Inspiration for the eventual web presence: <https://iso8583sim.com/>.

## Core architecture decisions

### 1. Public API must leave `internal/`

Go forbids importing `internal/` packages from outside the module. Today the
entire useful surface (`parser.Parse`, `packager.Packager`, the annotator) lives
under `internal/`, so the project is not importable as a library — it is a CLI
in a library's clothes. Fixing this is Phase 0 and everything builds on it.

Target layout:

```
iso8583/          # public library: Message, Field, Bitmap, Parse, Pack
  packager/       # public: Packager, FieldDef, dialects
  annotate/       # public: human-readable meaning
  llm/            # public: provider-agnostic generator interface
internal/         # only genuinely-private helpers remain
cmd/iso8583lens/  # the CLI, a thin consumer of the public library
```

The primary package is `iso8583`, so consumer code reads `iso8583.Parse(...)` /
`iso8583.Pack(...)`.

### 2. LLM support is provider-neutral and optional

The core library takes **no** LLM SDK dependency. It defines a small interface:

```go
package llm

// Describe turns natural language into structured field values; the
// deterministic packer then turns those values into valid bytes.
type Generator interface {
    Describe(ctx context.Context, prompt string, schema Schema) (FieldValues, error)
}
```

Adapters live in separate subpackages (`llm/claude`, `llm/openai`) so a provider
SDK is only pulled into the dependency tree if the developer imports that
adapter. The developer brings their own API key and picks the provider:

```go
gen := claude.New(os.Getenv("ANTHROPIC_API_KEY")) // or openai.New(...)
msg, _ := iso8583.Generate(ctx, gen, "a $45 grocery auth on card 4556...")
```

The LLM only ever maps English → structured field values. The deterministic
packer guarantees the resulting bytes are valid, which keeps a cheap model
(e.g. Claude Haiku, GPT-mini) sufficient. Constrain the model with structured
outputs / tool schemas **derived from the packager definition**, so it can only
emit fields the active dialect knows about.

### 3. No RAG

The generator's knowledge requirement is tiny and already structured (the
packager + lookup tables). Feed it directly into the prompt/schema. Revisit only
if free-form Q&A over full scheme manuals is ever added.

### 4. License: MIT

Permissive, professional, the conventional default for a library meant for broad
adoption.

### 5. Naming (open decision)

`iso8583lens` reads as "viewer/decoder" and undersells a library that also
*builds* messages and does the math. Under consideration: keep a memorable module
name but make the primary package `iso8583` (clean consumer code); `lens` can
survive as the CLI name. Final call pending.

## Phased plan

Each phase ships something useful on its own.

### Phase 0 — Make it a real library (foundation)
- Move the public API out of `internal/` into the `iso8583` package.
- Add `Pack` (struct → bytes): the missing half of `Parse`, and the foundation
  for both round-trip mode and the generator. Does the bitmap/length math.
- Package docs (`doc.go`) and runnable examples (`Example_parse`, `Example_pack`)
  so pkg.go.dev renders well.
- **Deliverable: v0.1.0** — `go get`-able, parse + pack with the math done for you.

### Phase 1 — Round-trip & core polish
- Parse → modify a field → re-pack → byte-level diff (great correctness test and
  teaching tool).
- Validation / lint: Luhn on PAN, currency/amount sanity, MCC lookup.
- Harden error messages.
- **Deliverable: v0.2.0** — useful to a Go dev with zero LLM involvement.

### Phase 2 — BCD / binary encodings
- Implement the packed (BCD) and binary field encodings the schema already
  anticipates — moves the tool from "synthetic ASCII only" to "parses real
  captures."
- Add a second dialect YAML to prove the schema generalizes.
- **Deliverable: v0.3.0**

### Phase 3 — LLM generator
- `llm.Generator` interface + `llm/claude` and `llm/openai` adapters, driven by
  structured outputs derived from the packager schema.
- CLI subcommand: `iso8583lens generate "..."`.
- **Deliverable: v0.4.0** — the headline feature.

### Phase 4 — Professional presentation
- `LICENSE` (MIT), README with badges (pkg.go.dev, Go version, license, CI),
  `CHANGELOG`, semantic-version tags.
- GitHub Actions CI: test + lint on push.
- GitHub Pages site (no custom domain), inspired by iso8583sim.com — likely a
  **WASM build** of the parser so messages decode in the browser with no server.

### Phase 5 — Ongoing
- More scheme dialects, an `explain` command, contribution docs.

## Guardrails

- Don't let the Pages/WASM shininess jump ahead of Phases 0–1. A beautiful site
  over a library nobody can import is backwards.
- The wedge is approachability + the LLM angle + the learning lens, not feature
  parity with `moov-io/iso8583`.
