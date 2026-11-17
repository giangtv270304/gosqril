package gosqril_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/giangtv270304/gosqril"
)

type moneyHolder struct {
	Amount   gosqril.Decimal  `json:"amount"`
	Optional *gosqril.Decimal `json:"optional,omitempty"`
}

func TestDecimal_RoundTripKeepsDigits(t *testing.T) {
	t.Parallel()

	values := []string{
		"1000.50",
		"0.0000000001",
		"1234567890123456789012345",
		"1234567890123456789012345.123456789012345678",
		"0.0000639967",
		"9007199254740993",
		"0.10",
		"-12.30",
		"0",
		"100000",
	}

	for _, value := range values {
		wire := `{"amount":` + value + `,"optional":` + value + `}`

		var holder moneyHolder
		if err := json.Unmarshal([]byte(wire), &holder); err != nil {
			t.Errorf("%s: unmarshal: %v", value, err)

			continue
		}

		if string(holder.Amount) != value || holder.Optional == nil || string(*holder.Optional) != value {
			t.Errorf("%s: decoded amount = %s, optional = %v", value, holder.Amount, holder.Optional)
		}

		encoded, err := json.Marshal(holder)
		if err != nil {
			t.Errorf("%s: marshal: %v", value, err)

			continue
		}

		if string(encoded) != wire {
			t.Errorf("%s: encoded = %s, want %s", value, encoded, wire)
		}
	}
}

func TestDecimal_UnmarshalForms(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wire string
		want string
	}{
		{`"1000.50"`, "1000.50"},
		{`"1234567890123456789012345.10"`, "1234567890123456789012345.10"},
		{`6.4e-8`, "0.000000064"},
		{`6.39967E-5`, "0.0000639967"},
		{`1e3`, "1000"},
		{`1E+3`, "1000"},
		{`1.5e1`, "15"},
		{`1.50e1`, "15.0"},
		{`-1.25e2`, "-125"},
		{`1.25e-1`, "0.125"},
		{`0.5e1`, "5"},
		{`12345.678e-2`, "123.45678"},
		{`1e0`, "1"},
		{`"1e3"`, "1000"},
	}

	for _, testCase := range testCases {
		var got gosqril.Decimal
		if err := json.Unmarshal([]byte(testCase.wire), &got); err != nil {
			t.Errorf("%s: %v", testCase.wire, err)

			continue
		}

		if string(got) != testCase.want {
			t.Errorf("%s: got %s, want %s", testCase.wire, got, testCase.want)
		}
	}
}

func TestDecimal_NullAndAbsent(t *testing.T) {
	t.Parallel()

	var holder moneyHolder
	if err := json.Unmarshal([]byte(`{"amount":null,"optional":null}`), &holder); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if holder.Amount != "" || holder.Optional != nil {
		t.Errorf("holder = %+v, want empty amount and nil optional", holder)
	}

	encoded, err := json.Marshal(moneyHolder{Amount: "5", Optional: nil})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(encoded) != `{"amount":5}` {
		t.Errorf("encoded = %s, want {\"amount\":5}", encoded)
	}
}

func TestDecimal_EmptyStringDecodesToEmpty(t *testing.T) {
	t.Parallel()

	var holder moneyHolder
	if err := json.Unmarshal([]byte(`{"amount":"","optional":""}`), &holder); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if holder.Amount != "" || holder.Optional == nil || *holder.Optional != "" {
		t.Errorf("holder = %q, %v, want empty decimals", holder.Amount, holder.Optional)
	}

	if _, err := json.Marshal(holder); !errors.Is(err, gosqril.ErrInvalidDecimal) {
		t.Errorf("marshal of an empty Decimal: error = %v, want ErrInvalidDecimal", err)
	}
}

func TestDecimal_MarshalAcceptsNegative(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(moneyHolder{Amount: "-12.30"})
	if err != nil || string(encoded) != `{"amount":-12.30}` {
		t.Errorf("encoded = %s, error = %v, want {\"amount\":-12.30}", encoded, err)
	}
}

func TestDecimal_UnmarshalRejectsNonNumeric(t *testing.T) {
	t.Parallel()

	for _, wire := range []string{`"abc"`, `"12abc"`, `"1,000"`, `"1e99999"`, `true`, `{}`, `[1]`, `" 1"`, `"+1"`, `"1."`} {
		var got gosqril.Decimal
		if err := json.Unmarshal([]byte(wire), &got); err == nil {
			t.Errorf("%s: decoded to %q, want an error", wire, got)
		}
	}
}

func TestDecimal_MarshalRejectsNonNumeric(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "abc", "1e5", "1.", ".5", "1,000", " 1", "1 ", "+1", "--1", "1.2.3", "0x10", "NaN", "007", "１２"} {
		_, err := json.Marshal(moneyHolder{Amount: gosqril.Decimal(value)})
		if !errors.Is(err, gosqril.ErrInvalidDecimal) {
			t.Errorf("%q: error = %v, want ErrInvalidDecimal", value, err)
		}
	}
}
