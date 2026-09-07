package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/Witkas/iso8583lens/internal/annotate"
)

// renderJSON writes the annotated result as indented JSON.
func renderJSON(w io.Writer, r annotate.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// renderTable writes the annotated result as an aligned text table.
func renderTable(w io.Writer, r annotate.Result) error {
	if r.MTIMeaning != "" {
		fmt.Fprintf(w, "MTI: %s (%s)\n\n", r.MTI, r.MTIMeaning)
	} else {
		fmt.Fprintf(w, "MTI: %s\n\n", r.MTI)
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tNAME\tVALUE\tMEANING")
	for _, f := range r.Fields {
		fmt.Fprintf(tw, "%03d\t%s\t%s\t%s\n",
			f.Number, f.Name, displayValue(f.Number, f.Value), f.Meaning)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if hasAmount(r) {
		fmt.Fprintln(w, "\nAmounts are shown assuming a 2-decimal currency.")
	}
	return nil
}

// displayValue formats a field value for the human table, masking the PAN
// (field 2) so a full card number is never printed to the terminal.
func displayValue(number int, value string) string {
	if number == 2 {
		return maskPAN(value)
	}
	return value
}

// maskPAN keeps the leading 6 (issuer/BIN) and trailing 4 digits and masks
// the middle, matching common PCI display rules. Short values are left as-is.
func maskPAN(pan string) string {
	if len(pan) <= 10 {
		return pan
	}
	middle := strings.Repeat("x", len(pan)-10)
	return pan[:6] + middle + pan[len(pan)-4:]
}

func hasAmount(r annotate.Result) bool {
	for _, f := range r.Fields {
		switch f.Number {
		case 4, 5, 6:
			return true
		}
	}
	return false
}
