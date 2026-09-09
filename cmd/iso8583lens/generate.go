package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Witkas/iso8583"
	"github.com/Witkas/iso8583/annotate"
	"github.com/Witkas/iso8583/llm"
	"github.com/Witkas/iso8583/llm/claude"
	"github.com/Witkas/iso8583/llm/openai"
	"github.com/Witkas/iso8583/packager"
)

func newGenerateCmd() *cobra.Command {
	var provider string

	cmd := &cobra.Command{
		Use:   "generate [description...]",
		Short: "Build a valid message from a natural-language description (uses an LLM)",
		Long: "Describe a transaction in plain English and get back a valid ISO 8583\n" +
			"message. The model only chooses field values; the bitmap and length\n" +
			"prefixes are computed deterministically, so the output is always well-formed.\n\n" +
			"Requires an API key in the environment for the chosen provider:\n" +
			"  --provider claude  -> ANTHROPIC_API_KEY\n" +
			"  --provider openai  -> OPENAI_API_KEY\n\n" +
			"Example:\n" +
			"  iso8583lens generate --provider claude \"a $45 grocery auth on card 4556737586899855\"",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gen, err := newGenerator(provider)
			if err != nil {
				return err
			}
			pkg, err := packager.Default()
			if err != nil {
				return err
			}

			prompt := strings.Join(args, " ")
			msg, raw, err := iso8583.Generate(cmd.Context(), gen, pkg, prompt)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "hex: %x\n\n", raw)

			annotator, err := annotate.New()
			if err != nil {
				return err
			}
			return renderTable(out, annotator.Annotate(msg))
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "claude", "LLM provider: claude or openai")
	return cmd
}

// newGenerator constructs the provider adapter, reading the API key from the
// provider's conventional environment variable.
func newGenerator(provider string) (llm.Generator, error) {
	switch strings.ToLower(provider) {
	case "claude", "anthropic":
		return claude.New(""), nil
	case "openai", "gpt":
		return openai.New(""), nil
	default:
		return nil, fmt.Errorf("unknown provider %q (use claude or openai)", provider)
	}
}
