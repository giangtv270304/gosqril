package gosqril

import (
	"context"
	"net/http"
	"net/url"
)

// ICType is the kind of identity document. Unknown values from the API are kept as sent.
type ICType string

// Identity document types accepted by RegisterCustomer.
const (
	ICTypeNationalID     ICType = "NIC"
	ICTypePassport       ICType = "PP"
	ICTypeWorkPermit     ICType = "WEP"
	ICTypeDriversLicense ICType = "DL"
)

// Customer is the sender profile returned by RegisterCustomer, CheckCustomer and UpdateCustomer.
// Every field is optional in the API; an absent field is the zero value. It holds personal data.
type Customer struct {
	CustomerID string `json:"customer_id"`
	NameFirst  string `json:"name_first"`
	NameLast   string `json:"name_last"`
	// Gender is echoed as stored: M or F, or MALE or FEMALE.
	Gender string `json:"gender,omitempty"`
	// ICNumber is the number of the identity document of kind ICType.
	ICNumber string `json:"ic_number"`
	ICType   ICType `json:"ic_type"`
	// ICCountry, CountryOfResidence and Nationality are ISO 3166 alpha-2 codes.
	ICCountry string `json:"ic_country"`
	// Occupation is a legacy code (OCC1 to OCC12) or an ISCO-08 code.
	Occupation         string `json:"occupation"`
	CountryOfResidence string `json:"country_of_residence"`
	// Phone is in E.164 format.
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	Nationality string `json:"nationality,omitempty"`
	// DOB and ICExpiryDate are YYYY-MM-DD.
	DOB          string `json:"dob"`
	Address      string `json:"address"`
	ICExpiryDate string `json:"ic_expiry_date"`
	// CreatedAt and UpdatedAt are ISO 8601, empty when the API does not send them.
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	// Allowed maps a corridor country code to true when the profile satisfies that corridor.
	// False means UpdateCustomer is needed first. Nil when the API does not send it.
	Allowed map[string]bool `json:"allowed,omitempty"`
}

// CheckCustomerRequest selects the customer to read.
type CheckCustomerRequest struct {
	// CustomerID is the id returned by RegisterCustomer. Required; must not contain "/".
	CustomerID string
}

// CheckCustomerResponse is the customer, including Allowed.
type CheckCustomerResponse = Customer

// CheckCustomer returns one customer. An unknown id is an *APIError; check it with IsNotFound.
func (c *Client) CheckCustomer(ctx context.Context, req *CheckCustomerRequest) (*CheckCustomerResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	if err := requireCustomerID(req.CustomerID); err != nil {
		return nil, err
	}

	return call[CheckCustomerResponse](ctx, c, request{
		Method:         http.MethodGet,
		Path:           "/checkCustomer",
		Query:          url.Values{"customer_id": {req.CustomerID}},
		Body:           nil,
		IdempotencyKey: "",
	})
}
