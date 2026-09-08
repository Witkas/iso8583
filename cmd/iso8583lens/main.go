// Command iso8583lens decodes a raw ISO 8583 message into a human-readable
// breakdown of every field. It performs pure, deterministic parsing and
// static table lookups — no network calls, no external services.
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	iso8583 "github.com/Witkas/iso8583lens"
	"github.com/Witkas/iso8583lens/annotate"
	"github.com/Witkas/iso8583lens/packager"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		// Cobra already prints the error; exit non-zero.
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "iso8583lens",
		Short:         "Decode and annotate ISO 8583 card-payment messages",
		Long:          "iso8583lens decodes a raw ISO 8583 message and prints a fully\nannotated, human-readable breakdown of every data element.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(newParseCmd())
	return root
}

func newParseCmd() *cobra.Command {
	var (
		hexStr string
		asJSON bool
	)

	cmd := &cobra.Command{
		Use:   "parse [file]",
		Short: "Decode a message from a file or a hex string",
		Long: "Decode an ISO 8583 message and print it as a formatted table (default)\n" +
			"or as JSON (--json).\n\n" +
			"Provide the message either as a positional file argument or via --hex.\n" +
			"A .bin file is read as raw bytes; any other file is read as a hex string.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := readInput(hexStr, args)
			if err != nil {
				return err
			}

			pkg, err := packager.Default()
			if err != nil {
				return err
			}
			msg, err := iso8583.Parse(raw, pkg)
			if err != nil {
				return err
			}
			annotator, err := annotate.New()
			if err != nil {
				return err
			}
			result := annotator.Annotate(msg)

			out := cmd.OutOrStdout()
			if asJSON {
				return renderJSON(out, result)
			}
			return renderTable(out, result)
		},
	}

	cmd.Flags().StringVar(&hexStr, "hex", "", "decode from this hex string instead of a file")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON instead of a formatted table")
	return cmd
}

// readInput resolves the message bytes from either the --hex flag or a file
// argument, returning a clear error if the inputs are missing or conflicting.
func readInput(hexStr string, args []string) ([]byte, error) {
	switch {
	case hexStr != "" && len(args) > 0:
		return nil, fmt.Errorf("provide either --hex or a file, not both")
	case hexStr != "":
		return decodeHex(hexStr)
	case len(args) == 1:
		return readFile(args[0])
	default:
		return nil, fmt.Errorf("no input: pass a file argument or --hex <string>")
	}
}

func readFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(strings.ToLower(path), ".bin") {
		return b, nil
	}
	return decodeHex(string(b))
}

// decodeHex parses a hex string, tolerating whitespace and an optional 0x
// prefix so pasted messages "just work".
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
	if s == "" {
		return nil, fmt.Errorf("empty hex input")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex input: %w", err)
	}
	return b, nil
}
