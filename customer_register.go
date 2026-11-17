package gosqril

import (
	"context"
	"net/http"
)

// RegisterCustomerRequest is a sender profile. All fields are required except Gender, Phone,
// Email and Nationality; the API validates them.
type RegisterCustomerRequest struct {
	// NameFirst and NameLast cannot be changed after registration.
	NameFirst string `json:"name_first"`
	NameLast  string `json:"name_last"`
	ICNumber  string `json:"ic_number"`
	ICType    ICType `json:"ic_type"`
	// ICCountry and CountryOfResidence are ISO 3166 alpha-2 codes.
	ICCountry string `json:"ic_country"`
	// Occupation is a code from ListOccupations, such as OccupationEmployee.
	Occupation         string `json:"occupation"`
	CountryOfResidence string `json:"country_of_residence"`
	// DOB is YYYY-MM-DD.
	DOB     string `json:"dob"`
	Address string `json:"address"`
	// ICExpiryDate is YYYY-MM-DD; use 2099-01-01 for a document that does not expire.
	ICExpiryDate string `json:"ic_expiry_date"`
	Gender       string `json:"gender,omitempty"`
	// Phone is in E.164 format. Needed for PE, KE, TZ and UG.
	Phone string `json:"phone,omitempty"`
	// Email is needed for TH and PE.
	Email string `json:"email,omitempty"`
	// Nationality is an ISO 3166 alpha-2 code. Needed for VN.
	Nationality string `json:"nationality,omitempty"`
}

// RegisterCustomerResponse is the stored customer. Allowed is not returned by this call.
type RegisterCustomerResponse = Customer

// RegisterCustomer creates a sender profile, or updates the existing one. Call it once per
// customer and keep CustomerID: decode, lookup, quotation and payout calls need it.
func (c *Client) RegisterCustomer(ctx context.Context, req *RegisterCustomerRequest) (*RegisterCustomerResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	return call[RegisterCustomerResponse](ctx, c, request{
		Method:         http.MethodPost,
		Path:           "/registerCustomer",
		Query:          nil,
		Body:           req,
		IdempotencyKey: "",
	})
}
