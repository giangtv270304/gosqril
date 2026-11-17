package gosqril

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const maxDecimalExponent = 1000

// ErrInvalidDecimal is returned when a Decimal does not hold a plain decimal number.
var ErrInvalidDecimal = errors.New("gosqril: invalid decimal")

// Decimal is an amount or a rate as a decimal string such as "1000.50", in standard currency
// units (not minor units). It is written to JSON as a bare number and read from a JSON number
// or string with the digits kept as sent. It never passes through float64; the SDK does no math.
// JSON "" decodes to the empty Decimal: an optional *Decimal field is then non-nil and empty.
type Decimal string

// String returns the decimal text.
func (d Decimal) String() string { return string(d) }

// MarshalJSON writes the value as a JSON number. It fails with ErrInvalidDecimal unless the
// value is digits with an optional leading "-" and an optional fraction ("12", "-0.5", "1000.50").
// Negative values are sent as they are; the API decides whether an amount is acceptable.
func (d Decimal) MarshalJSON() ([]byte, error) {
	text := string(d)
	if !isPlainDecimal(text) || hasLeadingZero(text) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidDecimal, text)
	}

	return []byte(text), nil
}

// UnmarshalJSON reads a JSON number or string. Exponent notation ("6.4e-8") is expanded to plain
// digits; JSON null and "" leave the value unchanged; other text fails with ErrInvalidDecimal.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "null" {
		return nil
	}

	if strings.HasPrefix(text, `"`) {
		if err := json.Unmarshal(data, &text); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidDecimal, data)
		}

		if text == "" {
			return nil
		}
	}

	parsed, err := parseDecimal(text)
	if err != nil {
		return err
	}

	*d = parsed

	return nil
}

func parseDecimal(text string) (Decimal, error) {
	index := strings.IndexAny(text, "eE")
	if index < 0 {
		if !isPlainDecimal(text) {
			return "", fmt.Errorf("%w: %q", ErrInvalidDecimal, text)
		}

		return Decimal(text), nil
	}

	mantissa := text[:index]

	exponent, err := strconv.Atoi(text[index+1:])
	if err != nil || !isPlainDecimal(mantissa) || exponent > maxDecimalExponent || exponent < -maxDecimalExponent {
		return "", fmt.Errorf("%w: %q", ErrInvalidDecimal, text)
	}

	return Decimal(shiftDecimalPoint(mantissa, exponent)), nil
}

func shiftDecimalPoint(mantissa string, exponent int) string {
	unsigned, negative := strings.CutPrefix(mantissa, "-")
	intPart, fracPart, _ := strings.Cut(unsigned, ".")
	digits := intPart + fracPart
	point := len(intPart) + exponent

	switch {
	case point <= 0:
		intPart, fracPart = "0", strings.Repeat("0", -point)+digits
	case point >= len(digits):
		intPart, fracPart = digits+strings.Repeat("0", point-len(digits)), ""
	default:
		intPart, fracPart = digits[:point], digits[point:]
	}

	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}

	out := intPart
	if fracPart != "" {
		out += "." + fracPart
	}

	if negative {
		out = "-" + out
	}

	return out
}

func isPlainDecimal(text string) bool {
	intPart, fracPart, hasFraction := strings.Cut(strings.TrimPrefix(text, "-"), ".")

	return isDigits(intPart) && (!hasFraction || isDigits(fracPart))
}

func hasLeadingZero(text string) bool {
	intPart, _, _ := strings.Cut(strings.TrimPrefix(text, "-"), ".")

	return len(intPart) > 1 && intPart[0] == '0'
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}

	for _, char := range text {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}
