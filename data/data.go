// Package data embeds the shipped packager definitions and annotation
// lookup tables so the compiled binary is fully self-contained.
package data

import "embed"

//go:embed packagers/default.yaml
//go:embed packagers/bcd-demo.yaml
//go:embed tables/response-codes.yaml
//go:embed tables/mcc.yaml
//go:embed tables/processing-code.yaml
//go:embed tables/mti.yaml
var FS embed.FS
