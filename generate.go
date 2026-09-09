package iso8583

import (
	"context"
	"fmt"
	"sort"

	"github.com/Witkas/iso8583/llm"
	"github.com/Witkas/iso8583/packager"
)

// Generate turns a natural-language request into a valid ISO 8583 message using
// the given Generator and packager. The Generator maps the prompt to field
// values; Generate then validates and packs them, so the resulting bytes are
// guaranteed well-formed (bitmap and length prefixes computed here, not by the
// model). It returns the assembled Message and its wire bytes.
//
// The Generator is provider-neutral (see the llm package); pass an
// implementation from llm/claude or llm/openai, or your own. Generate itself
// makes no network calls and is deterministic given the Generator's output.
func Generate(ctx context.Context, gen llm.Generator, p *packager.Packager, prompt string) (*Message, []byte, error) {
	if gen == nil {
		return nil, nil, fmt.Errorf("generate: nil generator")
	}
	if p == nil {
		return nil, nil, fmt.Errorf("generate: nil packager")
	}

	draft, err := gen.Describe(ctx, prompt, catalogFrom(p))
	if err != nil {
		return nil, nil, fmt.Errorf("generate: %w", err)
	}

	msg := NewMessage(draft.MTI, draft.Fields)

	// Pack validates every field against the packager (lengths, encoding) and
	// computes the bitmap. A model that returned an over-long PAN or a
	// missized fixed field is caught here rather than emitting bad bytes.
	raw, err := msg.Pack(p)
	if err != nil {
		return nil, nil, fmt.Errorf("generate: the model produced an invalid message: %w", err)
	}
	return msg, raw, nil
}

// catalogFrom builds the field vocabulary handed to the model from a packager,
// in ascending field order.
func catalogFrom(p *packager.Packager) llm.Catalog {
	nums := make([]int, 0, len(p.Fields))
	for n := range p.Fields {
		nums = append(nums, n)
	}
	sort.Ints(nums)

	specs := make([]llm.FieldSpec, 0, len(nums))
	for _, n := range nums {
		def := p.Fields[n]
		specs = append(specs, llm.FieldSpec{
			Number:    n,
			Name:      def.Name,
			Fixed:     def.Type == packager.Fixed,
			Length:    def.Length,
			MaxLength: def.MaxLength,
		})
	}
	return llm.Catalog{
		MTIExample: "0100",
		Fields:     specs,
	}
}
