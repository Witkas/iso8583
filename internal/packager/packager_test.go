package packager

import (
	"strings"
	"testing"
)

func TestDefaultLoads(t *testing.T) {
	p, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if p.Name == "" {
		t.Error("packager name is empty")
	}
	// Spot-check a few well-known field definitions.
	if f, ok := p.Field(2); !ok || f.Type != LLVAR || f.MaxLength != 19 {
		t.Errorf("field 2 = %+v, ok=%v", f, ok)
	}
	if f, ok := p.Field(3); !ok || f.Type != Fixed || f.Length != 6 {
		t.Errorf("field 3 = %+v, ok=%v", f, ok)
	}
	if f, ok := p.Field(39); !ok || f.Name != "Response Code" {
		t.Errorf("field 39 = %+v, ok=%v", f, ok)
	}
	// The full 1987 layout should define all of fields 2..128.
	for n := 2; n <= 128; n++ {
		if _, ok := p.Field(n); !ok {
			t.Errorf("field %d not defined", n)
		}
	}
}

func TestLoadRejectsBadDefinitions(t *testing.T) {
	tests := map[string]string{
		"missing fixed length": `
name: bad
mti: {length: 4, encoding: ascii}
bitmap: {encoding: binary}
fields:
  3: {name: X, type: FIXED, encoding: ascii}
`,
		"missing var maxLength": `
name: bad
mti: {length: 4, encoding: ascii}
bitmap: {encoding: binary}
fields:
  2: {name: X, type: LLVAR, encoding: ascii}
`,
		"unknown field type": `
name: bad
mti: {length: 4, encoding: ascii}
bitmap: {encoding: binary}
fields:
  3: {name: X, type: WOBBLE, length: 6, encoding: ascii}
`,
		"unsupported bitmap encoding": `
name: bad
mti: {length: 4, encoding: ascii}
bitmap: {encoding: ascii}
fields:
  3: {name: X, type: FIXED, length: 6, encoding: ascii}
`,
		"field out of range": `
name: bad
mti: {length: 4, encoding: ascii}
bitmap: {encoding: binary}
fields:
  200: {name: X, type: FIXED, length: 6, encoding: ascii}
`,
	}
	for name, yaml := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Load([]byte(yaml)); err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestLengthPrefixDigits(t *testing.T) {
	cases := map[FieldType]int{Fixed: 0, LLVAR: 2, LLLVAR: 3}
	for typ, want := range cases {
		if got := typ.LengthPrefixDigits(); got != want {
			t.Errorf("%s.LengthPrefixDigits() = %d, want %d", typ, got, want)
		}
	}
}

func TestValidateErrorMentionsField(t *testing.T) {
	_, err := Load([]byte(`
name: bad
mti: {length: 4, encoding: ascii}
bitmap: {encoding: binary}
fields:
  3: {name: Processing Code, type: FIXED, encoding: ascii}
`))
	if err == nil || !strings.Contains(err.Error(), "field 3") {
		t.Errorf("error = %v, want it to mention field 3", err)
	}
}
