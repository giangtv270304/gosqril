package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestCheckCustomer_Success(t *testing.T) {
	t.Parallel()

	body := customerBody[:len(customerBody)-1] + `,"allowed":{"VN":true,"PE":false},"future_field":1}`
	client, recorded := newFakeServer(t, http.StatusOK, body)

	resp, err := client.CheckCustomer(context.Background(), &gosqril.CheckCustomerRequest{CustomerID: "cus abc+123"})
	if err != nil {
		t.Fatalf("CheckCustomer: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/checkCustomer", "customer_id=cus+abc%2B123", "")
	assertCustomer(t, resp)

	if len(resp.Allowed) != 2 || !resp.Allowed["VN"] || resp.Allowed["PE"] {
		t.Errorf("allowed = %v, want VN true and PE false", resp.Allowed)
	}
}

func TestCheckCustomer_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusNotFound,
		`{"error_code":"RESOURCE_NOT_FOUND","error_message":"Customer not found"}`)

	_, err := client.CheckCustomer(context.Background(), &gosqril.CheckCustomerRequest{CustomerID: "cus_missing"})

	assertAPIError(t, err, http.StatusNotFound, gosqril.ErrorCodeResourceNotFound)

	if !gosqril.IsNotFound(err) {
		t.Error("IsNotFound = false, want true")
	}
}
