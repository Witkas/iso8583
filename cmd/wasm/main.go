//go:build js && wasm

// Command wasm exposes the iso8583 decoder to the browser. It registers a
// global JavaScript function, parseISO8583(hex), that decodes a hex-encoded
// message and returns a JSON string with the annotated fields and any
// validation findings. It powers the GitHub Pages demo (index.html) and runs
// entirely client-side — no message ever leaves the browser.
package main

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"syscall/js"

	"github.com/Witkas/iso8583"
	"github.com/Witkas/iso8583/annotate"
	"github.com/Witkas/iso8583/packager"
)

type result struct {
	OK         bool                      `json:"ok"`
	Error      string                    `json:"error,omitempty"`
	MTI        string                    `json:"mti,omitempty"`
	MTIMeaning string                    `json:"mtiMeaning,omitempty"`
	Fields     []annotate.AnnotatedField `json:"fields,omitempty"`
	Findings   []iso8583.Finding         `json:"findings,omitempty"`
}

func main() {
	pkg, err := packager.Default()
	if err != nil {
		panic(err)
	}
	ann, err := annotate.New()
	if err != nil {
		panic(err)
	}

	js.Global().Set("parseISO8583", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return toJSON(result{Error: "no input"})
		}
		return toJSON(decode(pkg, ann, args[0].String()))
	}))

	// Keep the Go runtime alive so the exported function stays callable.
	select {}
}

func decode(pkg *packager.Packager, ann *annotate.Annotator, input string) result {
	raw, err := decodeHex(input)
	if err != nil {
		return result{Error: err.Error()}
	}
	msg, err := iso8583.Parse(raw, pkg)
	if err != nil {
		return result{Error: err.Error()}
	}
	res := ann.Annotate(msg)
	return result{
		OK:         true,
		MTI:        res.MTI,
		MTIMeaning: res.MTIMeaning,
		Fields:     res.Fields,
		Findings:   iso8583.Validate(msg),
	}
}

func toJSON(r result) string {
	b, err := json.Marshal(r)
	if err != nil {
		return `{"ok":false,"error":"internal: failed to encode result"}`
	}
	return string(b)
}

// decodeHex tolerates whitespace and an optional 0x prefix, matching the CLI.
func decodeHex(s string) ([]byte, error) {
	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	return hex.DecodeString(s)
}
