package gosqril

import (
	"context"
	"net/http"
	"net/url"
)

// UpdateCustomerRequest changes sender fields. Set at least one; nil fields are left unchanged.
type UpdateCustomerRequest struct {
	// CustomerID is required and must not contain "/". It is sent in the query, not in the body.
	CustomerID string  `json:"-"`
	Email      *string `json:"email,omitempty"`
	// Phone is in E.164 format.
	Phone      *string `json:"phone,omitempty"`
	Occupation *string `json:"occupation,omitempty"`
	Address    *string `json:"address,omitempty"`
	// Gender is MALE or FEMALE for Peru.
	Gender *string `json:"gender,omitempty"`
	// Nationality is an ISO 3166 alpha-2 code. SQRIL's guides ask for it but the endpoint
	// schema does not list it, so the API may ignore or reject it.
	Nationality *string `json:"nationality,omitempty"`
}

// UpdateCustomerResponse is the customer after the update.
type UpdateCustomerResponse = Customer

// UpdateCustomer changes the updatable fields of a customer. Names cannot be updated.
func (c *Client) UpdateCustomer(ctx context.Context, req *UpdateCustomerRequest) (*UpdateCustomerResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	if err := requireCustomerID(req.CustomerID); err != nil {
		return nil, err
	}

	return call[UpdateCustomerResponse](ctx, c, request{
		Method:         http.MethodPut,
		Path:           "/updateCustomer",
		Query:          url.Values{"customer_id": {req.CustomerID}},
		Body:           req,
		IdempotencyKey: "",
	})
}
