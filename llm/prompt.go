package llm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// SystemPrompt builds the provider-agnostic instruction handed to the model: it
// explains the task, lists the fields the model may set (from the catalog), and
// demands a strict JSON reply. Both adapters use it so their behaviour matches.
func SystemPrompt(cat Catalog) string {
	var b strings.Builder
	b.WriteString("You translate a natural-language description of a card-payment ")
	b.WriteString("transaction into ISO 8583 field values. Do not compute bitmaps or ")
	b.WriteString("length prefixes — only choose the MTI and the value of each field.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Reply with ONLY a JSON object, no prose and no markdown code fences.\n")
	b.WriteString("- Shape: {\"mti\":\"0100\",\"fields\":{\"<number>\":\"<value>\"}}.\n")
	b.WriteString("- Field values are plain digit/character strings. Amounts are in minor ")
	b.WriteString("units with no decimal point, left-padded to the field length ")
	b.WriteString("(e.g. $45.00 in a 12-digit amount field is \"000000004500\").\n")
	b.WriteString("- Fixed fields must be exactly their length; variable fields must not ")
	b.WriteString("exceed their maximum. Omit fields you have no value for.\n")
	b.WriteString(fmt.Sprintf("- Use MTI \"%s\" unless the description clearly implies another.\n\n", cat.MTIExample))
	b.WriteString("Available fields (number: name [constraint]):\n")

	for _, f := range cat.Fields {
		constraint := fmt.Sprintf("max %d", f.MaxLength)
		if f.Fixed {
			constraint = fmt.Sprintf("exactly %d", f.Length)
		}
		b.WriteString(fmt.Sprintf("  %d: %s [%s]\n", f.Number, f.Name, constraint))
	}
	return b.String()
}

// ParseDraftJSON parses a model reply into a Draft. It tolerates surrounding
// prose or markdown fences by extracting the outermost JSON object, and accepts
// field keys as JSON strings (the natural shape for a numeric-keyed object).
func ParseDraftJSON(s string) (Draft, error) {
	obj := extractJSONObject(s)
	if obj == "" {
		return Draft{}, fmt.Errorf("no JSON object found in model reply")
	}

	var raw struct {
		MTI    string            `json:"mti"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal([]byte(obj), &raw); err != nil {
		return Draft{}, fmt.Errorf("model reply is not valid JSON: %w", err)
	}
	if raw.MTI == "" {
		return Draft{}, fmt.Errorf("model reply has no \"mti\"")
	}

	draft := Draft{MTI: raw.MTI, Fields: make(map[int]string, len(raw.Fields))}
	// Sort keys so an error mentions the first offending one deterministically.
	keys := make([]string, 0, len(raw.Fields))
	for k := range raw.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		n, err := strconv.Atoi(k)
		if err != nil {
			return Draft{}, fmt.Errorf("field key %q is not a number", k)
		}
		draft.Fields[n] = raw.Fields[k]
	}
	return draft, nil
}

// extractJSONObject returns the substring from the first '{' to the last '}',
// inclusive, or "" if there is no plausible object. This strips markdown fences
// and any leading/trailing prose without a full tokenizer.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end < 0 || end < start {
		return ""
	}
	return s[start : end+1]
}
