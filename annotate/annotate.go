// Package annotate implements layer 2 of iso8583lens: taking a parsed
// Message and attaching human-readable meaning to it using static lookup
// tables (shipped as YAML, no network or model calls).
package annotate

import (
	"fmt"
	"strconv"
	"strings"

	iso8583 "github.com/Witkas/iso8583lens"
	"github.com/Witkas/iso8583lens/data"
	"gopkg.in/yaml.v3"
)

// AnnotatedField is one decoded field plus its plain-language meaning.
// Meaning is empty when no annotation applies to the field.
type AnnotatedField struct {
	Number  int    `json:"number"`
	Name    string `json:"name"`
	Value   string `json:"value"`
	Meaning string `json:"meaning,omitempty"`
}

// Result is the annotated view of a whole message.
type Result struct {
	MTI        string           `json:"mti"`
	MTIMeaning string           `json:"mtiMeaning,omitempty"`
	Fields     []AnnotatedField `json:"fields"`
}

// Annotator holds the loaded lookup tables. Build one with New and reuse it.
type Annotator struct {
	responseCodes map[string]string
	mcc           map[string]string
	mti           mtiTables
	proc          procTables
}

type mtiTables struct {
	Names    map[string]string `yaml:"names"`
	Version  map[string]string `yaml:"version"`
	Class    map[string]string `yaml:"class"`
	Function map[string]string `yaml:"function"`
	Origin   map[string]string `yaml:"origin"`
}

type procTables struct {
	TransactionType map[string]string `yaml:"transactionType"`
	AccountType     map[string]string `yaml:"accountType"`
}

// New builds an Annotator from the embedded lookup tables.
func New() (*Annotator, error) {
	a := &Annotator{}

	var rc struct {
		Codes map[string]string `yaml:"codes"`
	}
	if err := loadYAML("tables/response-codes.yaml", &rc); err != nil {
		return nil, err
	}
	a.responseCodes = rc.Codes

	var mcc struct {
		Codes map[string]string `yaml:"codes"`
	}
	if err := loadYAML("tables/mcc.yaml", &mcc); err != nil {
		return nil, err
	}
	a.mcc = mcc.Codes

	if err := loadYAML("tables/mti.yaml", &a.mti); err != nil {
		return nil, err
	}
	if err := loadYAML("tables/processing-code.yaml", &a.proc); err != nil {
		return nil, err
	}
	return a, nil
}

func loadYAML(name string, into any) error {
	b, err := data.FS.ReadFile(name)
	if err != nil {
		return fmt.Errorf("annotate: reading embedded %s: %w", name, err)
	}
	if err := yaml.Unmarshal(b, into); err != nil {
		return fmt.Errorf("annotate: parsing %s: %w", name, err)
	}
	return nil
}

// Annotate produces the annotated view of a parsed message.
func (a *Annotator) Annotate(m *iso8583.Message) Result {
	res := Result{
		MTI:        m.MTI,
		MTIMeaning: a.mtiMeaning(m.MTI),
	}
	for _, n := range m.PresentFields() {
		f := m.Fields[n]
		res.Fields = append(res.Fields, AnnotatedField{
			Number:  n,
			Name:    f.Name,
			Value:   f.Value,
			Meaning: a.fieldMeaning(n, f.Value),
		})
	}
	return res
}

// mtiMeaning returns a label for an MTI, preferring the exact-match table and
// otherwise composing one from the four digit positions.
func (a *Annotator) mtiMeaning(mti string) string {
	if name, ok := a.mti.Names[mti]; ok {
		return name
	}
	if len(mti) != 4 {
		return ""
	}
	parts := []string{
		lookupDigit(a.mti.Version, mti[0:1]),
		lookupDigit(a.mti.Class, mti[1:2]),
		lookupDigit(a.mti.Function, mti[2:3]),
		lookupDigit(a.mti.Origin, mti[3:4]),
	}
	nonEmpty := parts[:0]
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, " / ")
}

func lookupDigit(table map[string]string, digit string) string {
	if v, ok := table[digit]; ok {
		return v
	}
	return ""
}

// fieldMeaning returns a plain-language meaning for a field value, or "" if
// no annotation is defined for that field number.
func (a *Annotator) fieldMeaning(n int, value string) string {
	switch n {
	case 3:
		return a.processingCode(value)
	case 4, 5, 6:
		return amountMeaning(value)
	case 18:
		if desc, ok := a.mcc[value]; ok {
			return desc
		}
		return "unknown merchant category code"
	case 39:
		if desc, ok := a.responseCodes[value]; ok {
			return desc
		}
		return "unknown response code"
	default:
		return ""
	}
}

// processingCode decodes field 3's three 2-digit sub-parts.
func (a *Annotator) processingCode(v string) string {
	if len(v) != 6 {
		return ""
	}
	txn := v[0:2]
	from := v[2:4]
	to := v[4:6]

	txnDesc := a.proc.TransactionType[txn]
	if txnDesc == "" {
		txnDesc = "unknown transaction type " + txn
	}
	fromDesc := a.proc.AccountType[from]
	if fromDesc == "" {
		fromDesc = "account type " + from
	}
	toDesc := a.proc.AccountType[to]
	if toDesc == "" {
		toDesc = "account type " + to
	}
	return fmt.Sprintf("%s; from %s; to %s", txnDesc, fromDesc, toDesc)
}

// amountMeaning renders an ISO amount (minor units, no decimal point) as a
// decimal figure. It assumes a 2-decimal currency, the common case; the true
// exponent depends on the transaction currency (field 49). This simplifying
// assumption is documented in the README.
func amountMeaning(v string) string {
	if v == "" {
		return ""
	}
	digits := v
	sign := ""
	// Some amount fields carry a leading C/D sign; tolerate it.
	switch v[0] {
	case 'C', 'c':
		digits = v[1:]
	case 'D', 'd':
		digits, sign = v[1:], "-"
	}
	n, err := strconv.ParseUint(digits, 10, 64)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s%d.%02d", sign, n/100, n%100)
}
