# iso8583lens

A command-line tool that decodes a raw **ISO 8583** message and prints a
fully annotated, human-readable breakdown of every field. Pure, deterministic
parsing and static table lookups — no network calls, no external services, no
AI.

## What is ISO 8583?

ISO 8583 is the international standard message format that card payment
networks use to move authorization and clearing traffic between terminals,
acquirers, switches, and card issuers. Every time you tap or swipe a card, a
compact binary message in roughly this shape is what actually crosses the
wire. It is dense and delimiter-free, which makes it fast to transmit and
genuinely hard to read by eye — which is what this tool is for.

## What it does

Given a raw message (as a hex string or a file), `iso8583lens`:

- reads the **MTI** (message type indicator) and labels it,
- parses the **bitmap(s)** to determine which data elements are present,
- decodes each **data element** (fixed-length and `LLVAR`/`LLLVAR`
  variable-length) into its value, and
- **annotates** the important fields with plain-language meaning: response
  codes, the processing-code breakdown, merchant category codes, and amounts.

## Quick start

Requires Go 1.22+.

```sh
# Build the binary
go build -o iso8583lens ./cmd/iso8583lens

# Decode the shipped sample (a 0100 authorization request)
./iso8583lens parse testdata/auth-0100.hex
```

Expected output:

```
MTI: 0100 (Authorization Request)

FIELD  NAME                              VALUE             MEANING
002    Primary Account Number (PAN)      455673xxxxxx9855
003    Processing Code                   000000            Purchase (goods and services); from Default (unspecified) account; to Default (unspecified) account
004    Amount, Transaction               000000004500      45.00
007    Transmission Date and Time        0906120000
011    System Trace Audit Number (STAN)  000123
012    Time, Local Transaction           120000
013    Date, Local Transaction           0906
018    Merchant Type (MCC)               5411              Grocery stores and supermarkets
041    Card Acceptor Terminal ID         TERM0001
042    Card Acceptor ID Code             MERCHANT0000123
049    Currency Code, Transaction        840

Amounts are shown assuming a 2-decimal currency.
```

## Usage

```
iso8583lens parse <file>           # decode a file, print a formatted table
iso8583lens parse <file> --json    # decode a file, print JSON
iso8583lens parse --hex <string>   # decode from a hex string argument
```

A `.bin` file is read as raw bytes; any other file (e.g. `.hex`) is read as a
hex string. Hex input tolerates whitespace and an optional `0x` prefix.

## Authorization vs. clearing

A card transaction generally happens in two stages. **Authorization**
(message types in the `01xx`/`02xx` range) is the real-time check that asks
the issuer "is this card valid and are the funds available?" and gets back an
approve/decline — it places a hold but usually moves no money. **Clearing and
settlement** (typically `02xx` presentments feeding batch settlement) is the
later step where the actual funds are exchanged between the acquirer and
issuer. The two stages carry overlapping but different fields, which is
exactly why a tool that decodes any message by its bitmap — rather than
assuming one fixed layout — is useful.

## How it is built

Two layers, each with its data kept out of the Go code so the tool is easy to
extend:

- **Layer 1 — parser** (`internal/parser`): decodes the wire format into a
  structured `Message`. Field layouts (length discipline, encoding, names)
  live in a **packager** definition — `data/packagers/default.yaml` — not in
  Go, so a new dialect can be added by writing YAML.
- **Layer 2 — annotator** (`internal/annotate`): attaches human-readable
  meaning using static lookup tables in `data/tables/` (response codes, MCC,
  MTI, and the processing-code sub-fields).

### The default dialect

The shipped packager implements the classic 128-element **ISO 8583:1987**
layout (the same field set commonly referenced via jPOS's generic packager),
in an **ASCII** flavour:

- **MTI**: 4 ASCII digits.
- **Bitmap**: raw binary, 8 bytes per block; bit 1 set means a secondary
  bitmap follows, extending coverage to fields 65–128.
- **Fields**: ASCII. Fixed-length fields have a known width; `LLVAR`/`LLLVAR`
  fields carry a 2- or 3-digit ASCII length prefix.

Encoding is declared per message part in the packager schema, so a future
binary/BCD dialect can reuse the same structure.

## Notes and simplifications

- **PAN masking**: the table output masks the card number (first 6 + last 4).
  The `--json` output preserves the decoded value as parsed.
- **Amounts** are formatted assuming a 2-decimal currency, which covers most
  currencies; the true minor-unit exponent depends on the transaction
  currency (field 49).
- **MCC table** (`data/tables/mcc.yaml`) is a common-category subset, not the
  full ISO 18245 registry. Unknown codes are reported as unknown rather than
  guessed.

## Tests

```sh
go test ./...
```

Coverage includes a full realistic 0100 message fixture, secondary-bitmap
handling, the annotation lookups, and every malformed-input failure mode
(short MTI, short/missing bitmap, truncated fixed and variable fields,
non-numeric length prefixes, and undefined fields flagged in the bitmap).

## Not yet built (future work)

This is phase 1 of a larger idea. Explicitly out of scope here, planned for
later specs:

- A **generator / reverse mode** that builds a valid message from a
  natural-language or structured description.
- Any **LLM / AI-assisted** explanation layer, and retrieval (RAG) of scheme
  documentation.
- **Binary/BCD** field encodings (the schema is structured for it; the parser
  does not implement it yet).
- Additional **scheme dialects** beyond the one default packager.
