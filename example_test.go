package iso8583_test

import (
	"encoding/hex"
	"fmt"

	"github.com/Witkas/iso8583"
	"github.com/Witkas/iso8583/packager"
)

// ExampleParse decodes a raw 0100 authorization request into its fields.
func ExampleParse() {
	p, _ := packager.Default()

	// A 0100 authorization request (also shipped in testdata/auth-0100.hex).
	raw, _ := hex.DecodeString(
		"303130307238400000c08000" + // MTI 0100 + bitmap (fields 2,3,4,7,11,12,13,18,41,42,49)
			"3136" + "34353536373337353836383939383535" + // DE2 LLVAR 16-digit PAN
			"303030303030" + // DE3 "000000"
			"303030303030303034353030" + // DE4 "000000004500"
			"30393036313230303030" + // DE7 "0906120000"
			"303030313233" + // DE11 "000123"
			"313230303030" + // DE12 "120000"
			"30393036" + // DE13 "0906"
			"35343131" + // DE18 "5411"
			"5445524d30303031" + // DE41 "TERM0001"
			"4d45524348414e5430303030313233" + // DE42 "MERCHANT0000123"
			"383430", // DE49 "840"
	)

	msg, err := iso8583.Parse(raw, p)
	if err != nil {
		panic(err)
	}
	fmt.Println("MTI:", msg.MTI)
	fmt.Println("PAN:", msg.Fields[2].Value)
	fmt.Println("Amount:", msg.Fields[4].Value)
	// Output:
	// MTI: 0100
	// PAN: 4556737586899855
	// Amount: 000000004500
}

// ExampleMessage_Pack builds a message from field values and serializes it —
// the bitmap and the PAN's variable-length prefix are computed for you.
func ExampleMessage_Pack() {
	p, _ := packager.Default()

	msg := iso8583.NewMessage("0100", map[int]string{
		2:  "4556737586899855", // PAN (LLVAR — length prefix added automatically)
		3:  "000000",           // processing code
		4:  "000000004500",     // amount: 45.00
		49: "840",              // currency: USD
	})

	raw, err := msg.Pack(p)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%x\n", raw)
	// Output:
	// 303130307000000000008000313634353536373337353836383939383535303030303030303030303030303034353030383430
}
