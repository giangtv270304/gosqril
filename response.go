package gosqril

// Party is a sender or recipient object. SQRIL documents it as free-form, so keys vary by corridor
// (for example name_first, name_last, ic_number, account_no, bank_code). It holds personal data.
type Party map[string]any

// MissingField describes one piece of data SQRIL still needs before a payout.
type MissingField struct {
	// Field is a path such as "amount", "recipient.name" or "sender.occupation".
	Field string `json:"field"`
	// Type is the input kind: numeric, text, email, phone or select.
	Type  string `json:"type,omitempty"`
	Label string `json:"label,omitempty"`
	// Required is nil when the API did not send it.
	Required *bool `json:"required,omitempty"`
}
