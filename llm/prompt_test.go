package llm

import "testing"

func TestParseDraftJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"plain", `{"mti":"0100","fields":{"2":"4111111111111111","4":"000000004500"}}`},
		{"fenced", "```json\n{\"mti\":\"0100\",\"fields\":{\"4\":\"000000004500\"}}\n```"},
		{"prose", `Sure! Here you go:  {"mti":"0200","fields":{"3":"000000"}}  Hope that helps.`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := ParseDraftJSON(tc.in)
			if err != nil {
				t.Fatalf("ParseDraftJSON: %v", err)
			}
			if d.MTI == "" {
				t.Error("empty MTI")
			}
			if len(d.Fields) == 0 {
				t.Error("no fields parsed")
			}
		})
	}
}

func TestParseDraftJSONErrors(t *testing.T) {
	bad := []string{
		"",
		"no json here",
		`{"fields":{"4":"x"}}`,              // missing mti
		`{"mti":"0100","fields":{"x":"y"}}`, // non-numeric field key
	}
	for _, in := range bad {
		if _, err := ParseDraftJSON(in); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestSystemPromptListsFields(t *testing.T) {
	cat := Catalog{
		MTIExample: "0100",
		Fields: []FieldSpec{
			{Number: 2, Name: "PAN", Fixed: false, MaxLength: 19},
			{Number: 4, Name: "Amount", Fixed: true, Length: 12},
		},
	}
	got := SystemPrompt(cat)
	for _, want := range []string{"0100", "2: PAN", "max 19", "4: Amount", "exactly 12", "JSON"} {
		if !contains(got, want) {
			t.Errorf("system prompt missing %q", want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
