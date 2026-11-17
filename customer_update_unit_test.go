package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestUpdateCustomer_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, customerBody)

	resp, err := client.UpdateCustomer(context.Background(), &gosqril.UpdateCustomerRequest{
		CustomerID:  "cus_abc123",
		Email:       ptr("john.doe@example.com"),
		Phone:       nil,
		Occupation:  nil,
		Address:     nil,
		Gender:      ptr("MALE"),
		Nationality: ptr("SG"),
	})
	if err != nil {
		t.Fatalf("UpdateCustomer: %v", err)
	}

	assertRequest(t, recorded, http.MethodPut, "/updateCustomer", "customer_id=cus_abc123",
		`{"email":"john.doe@example.com","gender":"MALE","nationality":"SG"}`)
	assertCustomer(t, resp)
}

func TestUpdateCustomer_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusBadRequest,
		`{"error_code":"INVALID_REQUEST","error_message":"Provide one or more updatable sender fields."}`)

	_, err := client.UpdateCustomer(context.Background(), &gosqril.UpdateCustomerRequest{CustomerID: "cus_abc123"})

	assertAPIError(t, err, http.StatusBadRequest, gosqril.ErrorCodeInvalidRequest)
}
