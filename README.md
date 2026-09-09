# iso8583lens

[![Go Reference](https://pkg.go.dev/badge/github.com/Witkas/iso8583.svg)](https://pkg.go.dev/github.com/Witkas/iso8583)
[![CI](https://github.com/Witkas/iso8583/actions/workflows/ci.yml/badge.svg)](https://github.com/Witkas/iso8583/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Go library for **decoding and encoding ISO 8583 card-payment messages** — with
the error-prone parts (bitmaps, MTI, variable-length prefixes, field encoding)
done for you. A CLI ships alongside it for quick inspection from the terminal.

> **Status:** early and evolving. The library API may change before v1. See
> [ROADMAP.md](ROADMAP.md) for where this is headed (round-trip editing,
> BCD/binary encodings, an optional LLM-assisted generator, and a browser demo).

## What is ISO 8583?

ISO 8583 is the international message format card networks use to move
authorization and clearing traffic between terminals, acquirers, switches, and
issuers. Every time you tap or swipe a card, a compact binary message in roughly
this shape crosses the wire. It is dense and delimiter-free — fast to transmit,
genuinely hard to read by eye, and easy to get wrong when you hand-roll the
bitmap and length math. This library exists to take that math off your plate.

## Install

```sh
go get github.com/Witkas/iso8583
```

Requires Go 1.22+.

## Library usage

### Decode

```go
import (
    "encoding/hex"

    "github.com/Witkas/iso8583"
    "github.com/Witkas/iso8583/packager"
)

p, _ := packager.Default()
raw, _ := hex.DecodeString("30313030...") // your message bytes
msg, err := iso8583.Parse(raw, p)
// msg.MTI, msg.Bitmap, msg.Fields[2].Value, ...
```

`Parse` reads the MTI, walks the bitmap(s) to learn which data elements are
present, and decodes each one — fixed-length and `LLVAR`/`LLLVAR`
variable-length.

### Encode

```go
msg := iso8583.NewMessage("0100", map[int]string{
    2:  "4556737586899855", // PAN — LLVAR length prefix computed for you
    3:  "000000",           // processing code
    4:  "000000004500",     // amount: 45.00
    49: "840",              // currency: USD
})
raw, err := msg.Pack(p)
```

`Pack` is the inverse of `Parse`: it computes the bitmap (including a secondary
bitmap when any field above 64 is present) and writes each variable-length
field's length prefix. You set field values; the library handles the wire
format. `Parse` and `Pack` round-trip — packing a parsed message reproduces the
original bytes.

### Human-readable meaning

The `annotate` package attaches plain-language meaning to a parsed message —
response codes, MCC descriptions, processing-code breakdowns, and amounts —
using static lookup tables (no network, no model calls).

```go
import "github.com/Witkas/iso8583/annotate"

a, _ := annotate.New()
result := a.Annotate(msg) // result.Fields[i].Meaning
```

## Field layout lives in data, not code

How each part of a message is encoded — the MTI width, the bitmap encoding, and
every data element's length discipline and encoding — is described by a
**packager** definition loaded from YAML ([`data/packagers/default.yaml`](data/packagers/default.yaml)),
not hardcoded in Go. A new scheme dialect is a new YAML file. The shipped default
implements the classic **ISO 8583:1987** 128-element layout in an ASCII flavour:

- **MTI:** 4 ASCII digits.
- **Bitmap:** raw binary, 8 bytes per block; bit 1 set means a secondary bitmap
  follows, extending coverage to fields 65–128.
- **Fields:** ASCII. Fixed-length fields have a known width; `LLVAR`/`LLLVAR`
  fields carry a 2- or 3-digit ASCII length prefix.

Encoding is declared per message part, so a future binary/BCD dialect reuses the
same structure.

## CLI

The `iso8583lens` command is a thin consumer of the library.

```sh
go build -o iso8583lens ./cmd/iso8583lens
./iso8583lens parse testdata/auth-0100.hex        # formatted table
./iso8583lens parse testdata/auth-0100.hex --json # JSON
./iso8583lens parse --hex 30313030...             # from a hex string
./iso8583lens validate testdata/auth-0100.hex     # sanity checks (non-zero exit on errors)
```

A `.bin` file is read as raw bytes; any other file (e.g. `.hex`) is read as a
hex string, tolerating whitespace and an optional `0x` prefix. The table output
masks the PAN (first 6 + last 4); `--json` preserves the decoded value.

Example output:

```
MTI: 0100 (Authorization Request)

FIELD  NAME                              VALUE             MEANING
002    Primary Account Number (PAN)      455673xxxxxx9855
003    Processing Code                   000000            Purchase (goods and services); from Default account; to Default account
004    Amount, Transaction               000000004500      45.00
018    Merchant Type (MCC)               5411              Grocery stores and supermarkets
049    Currency Code, Transaction        840
```

## Authorization vs. clearing

A card transaction generally happens in two stages. **Authorization**
(`01xx`/`02xx`) is the real-time check that asks the issuer "is this card valid
and are the funds available?" — it places a hold but usually moves no money.
**Clearing and settlement** is the later step where funds actually change hands.
The two stages carry overlapping but different fields, which is exactly why a
library that decodes any message by its bitmap — rather than assuming one fixed
layout — is useful.

## Notes and simplifications

- **Amounts** are formatted assuming a 2-decimal currency (the common case); the
  true minor-unit exponent depends on the transaction currency (field 49).
- **MCC table** is a common-category subset, not the full ISO 18245 registry.
  Unknown codes are reported as unknown rather than guessed.
- **Pack is strict about fixed-field width** — it does not silently pad, because
  correct padding (zero-left for numeric, space-right for alphanumeric) depends
  on field semantics the schema does not yet carry.

## Development

```sh
go test ./...   # unit tests, round-trip tests, and runnable examples
go vet ./...
```

## License

[MIT](LICENSE)
