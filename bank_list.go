package gosqril

import (
	"context"
	"net/http"
	"net/url"
)

// Bank is one entry of ListBanks. For Indonesia the same Code can appear twice with different
// SwiftCode values.
type Bank struct {
	// Code is the value for LookupPaymentCodeRequest.PaymentDestinationCode.
	Code string `json:"code"`
	Name string `json:"name"`
	// ShortName and SwiftCode are sent for Indonesia only; SwiftCode not for every bank.
	ShortName string `json:"short_name,omitempty"`
	SwiftCode string `json:"swift_code,omitempty"`
	// Type is not documented by SQRIL; it is kept as sent.
	Type string `json:"type,omitempty"`
	// CountryCode is an ISO 3166 alpha-2 code.
	CountryCode string `json:"country_code,omitempty"`
}

// ListBanksRequest selects the country.
type ListBanksRequest struct {
	// CountryCode is required: ID, PH, VN, BO, NG, GH or ZM.
	CountryCode string
}

// ListBanksResponse lists the banks of one country.
type ListBanksResponse struct {
	Banks       []Bank `json:"banks"`
	CountryCode string `json:"country_code"`
	// Total is the number of entries in Banks.
	Total int `json:"total"`
}

// ListBanks returns the bank codes for LookupPaymentCode. A country the account does not
// support is a 403 *APIError.
func (c *Client) ListBanks(ctx context.Context, req *ListBanksRequest) (*ListBanksResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	if err := requireField("CountryCode", req.CountryCode); err != nil {
		return nil, err
	}

	return call[ListBanksResponse](ctx, c, request{
		Method:         http.MethodGet,
		Path:           "/listBanks",
		Query:          url.Values{"country_code": {req.CountryCode}},
		Body:           nil,
		IdempotencyKey: "",
	})
}
