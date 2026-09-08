// Package iso8583 decodes and encodes ISO 8583 card-payment messages.
//
// ISO 8583 is the message format that card networks use to move authorization
// and clearing traffic between terminals, acquirers, switches, and issuers.
// The format is dense and delimiter-free: a message is a message type
// indicator (MTI), one or two bitmaps declaring which data elements are
// present, and the data elements themselves packed back to back. This package
// does the error-prone parts for you — bitmap bit math, MTI handling, and
// variable-length (LLVAR/LLLVAR) length prefixes.
//
// # Field layout lives in data, not code
//
// How each message part is encoded on the wire — the MTI width, the bitmap
// encoding, and every data element's length discipline and encoding — is
// described by a [packager.Packager], loaded from YAML. A new scheme dialect is
// a new YAML file, not new Go. The shipped default implements the classic
// ISO 8583:1987 128-element ASCII layout; [packager.Default] returns it.
//
// # Decoding
//
//	pkg, _ := packager.Default()
//	msg, err := iso8583.Parse(raw, pkg)
//
// [Parse] turns raw bytes into a [Message]: the MTI, a [Bitmap] of which data
// elements are present, and the decoded [Field] values.
//
// # Encoding
//
// [Message.Pack] is the inverse of [Parse]: it serializes a Message back to
// wire bytes, computing the bitmap and length prefixes for you. Parse and Pack
// round-trip — packing a parsed message reproduces the original bytes.
//
// # Human-readable meaning
//
// This package intentionally stops at field values. To attach plain-language
// meaning (response codes, MCC descriptions, processing-code breakdowns,
// amounts), see the annotate package.
package iso8583
