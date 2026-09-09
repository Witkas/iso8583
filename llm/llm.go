// Package llm defines the provider-neutral seam for turning a natural-language
// request into an ISO 8583 message draft. It has no dependency on any LLM SDK
// or on the iso8583 package — callers pass a Catalog describing the available
// fields, and an implementation returns a Draft of field values. The iso8583
// package's Generate function then validates and packs that draft into bytes,
// so the model never has to compute a bitmap or a length prefix.
//
// Concrete implementations live in subpackages (llm/claude, llm/openai) and are
// selected by the caller, keeping this package and the core library free of
// provider SDKs.
package llm

import "context"

// FieldSpec describes one data element a Generator may populate.
type FieldSpec struct {
	Number    int
	Name      string
	Fixed     bool // true for FIXED, false for LLVAR/LLLVAR
	Length    int  // exact length in digits/chars (FIXED only)
	MaxLength int  // maximum length (variable fields only)
}

// Catalog is the field vocabulary handed to a Generator: the fields it may set
// and their length constraints. It carries no wire-format detail — the model's
// job is only to choose an MTI and field values.
type Catalog struct {
	MTIExample string
	Fields     []FieldSpec
}

// Draft is what a Generator returns: an MTI and a value for each field it chose
// to set, keyed by field number. Values are logical (digit/character) strings;
// the iso8583 packer handles wire encoding.
type Draft struct {
	MTI    string
	Fields map[int]string
}

// Generator maps a natural-language request into a Draft, guided by the Catalog.
// Implementations should return values that satisfy the catalog's constraints,
// but need not be perfect — the caller validates and packs the draft, surfacing
// any violation as an error.
type Generator interface {
	Describe(ctx context.Context, prompt string, cat Catalog) (Draft, error)
}
