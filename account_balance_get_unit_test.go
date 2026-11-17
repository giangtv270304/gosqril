package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestGetAccountBalances_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"balances":{"USD":{"available":100,"locked":0,"total":100,"currency":"USD"},"VND":{"available":5000000,"locked":0,"total":5000000,"currency":"VND"},"PHP":{"available":1234567890123456789012345.67,"locked":0.10,"total":1234567890123456789012345.77,"currency":"PHP"}}}`)

	resp, err := client.GetAccountBalances(context.Background(), &gosqril.GetAccountBalancesRequest{})
	if err != nil {
		t.Fatalf("GetAccountBalances: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/getAccountBalances", "", "")

	want := map[string]gosqril.Balance{
		"USD": {Available: "100", Locked: "0", Total: "100", Currency: "USD"},
		"VND": {Available: "5000000", Locked: "0", Total: "5000000", Currency: "VND"},
		"PHP": {Available: "1234567890123456789012345.67", Locked: "0.10", Total: "1234567890123456789012345.77", Currency: "PHP"},
	}

	if len(resp.Balances) != len(want) {
		t.Fatalf("balances = %d, want %d", len(resp.Balances), len(want))
	}

	for currency, balance := range want {
		if resp.Balances[currency] != balance {
			t.Errorf("%s = %+v, want %+v", currency, resp.Balances[currency], balance)
		}
	}
}

func TestGetAccountBalances_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusUnauthorized,
		`{"error_code":"INVALID_CREDENTIALS","error_message":"Credentials invalid"}`)

	_, err := client.GetAccountBalances(context.Background(), nil)

	assertAPIError(t, err, http.StatusUnauthorized, gosqril.ErrorCodeInvalidCredentials)

	if !gosqril.IsUnauthorized(err) {
		t.Error("IsUnauthorized = false, want true")
	}
}
