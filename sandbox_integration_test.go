package gosqril_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/giangtv270304/gosqril"
)

const (
	sandboxTimeout       = 60 * time.Second
	sandboxMarker        = "sdk-sandbox-test"
	sandboxQuotationCOP  = gosqril.Decimal("5000")
	sandboxMissingTxID   = "tx_sdk_sandbox_test_missing"
	sandboxBREBStaticKey = "sqril-breb-qa-manual-amount"
)

func sandboxContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), sandboxTimeout)
	t.Cleanup(cancel)

	return ctx
}

func registerSandboxCustomer(ctx context.Context, t *testing.T, client *gosqril.Client) string {
	t.Helper()

	customer, err := client.RegisterCustomer(ctx, &gosqril.RegisterCustomerRequest{
		NameFirst:          sandboxMarker,
		NameLast:           "Tester",
		ICNumber:           "E0000001X",
		ICType:             gosqril.ICTypePassport,
		ICCountry:          "SG",
		Occupation:         gosqril.OccupationEmployee,
		CountryOfResidence: "SG",
		DOB:                "1990-01-01",
		Address:            "1 Sandbox Test Street, Singapore 000001",
		ICExpiryDate:       "2099-01-01",
		Gender:             "M",
		Phone:              "+6580000001",
		Email:              "sdk-sandbox-test@example.com",
		Nationality:        "SG",
	})
	if err != nil {
		t.Fatalf("RegisterCustomer: %v", err)
	}

	if customer.CustomerID == "" {
		t.Fatal("RegisterCustomer returned an empty customer_id")
	}

	return customer.CustomerID
}

func registerAndUpdateSandboxCustomer(ctx context.Context, t *testing.T, client *gosqril.Client) string {
	t.Helper()

	customerID := registerSandboxCustomer(ctx, t, client)

	updated, err := client.UpdateCustomer(ctx, &gosqril.UpdateCustomerRequest{
		CustomerID:  customerID,
		Nationality: ptr("SG"),
	})
	if err != nil {
		t.Errorf("UpdateCustomer with nationality: %v", err)

		return customerID
	}

	if updated.NameFirst != sandboxMarker {
		t.Errorf("name_first = %q, want %q", updated.NameFirst, sandboxMarker)
	}

	return customerID
}

func sandboxCustomerID(ctx context.Context, t *testing.T, client *gosqril.Client) string {
	t.Helper()

	if customerID := os.Getenv(envCustomerID); customerID != "" {
		return customerID
	}

	return registerSandboxCustomer(ctx, t, client)
}

func TestSandboxHelper_StopsWhenBaseURLIsProduction(t *testing.T) {
	for _, baseURL := range []string{gosqril.ProductionBaseURL, "https://api.sqril.io/", "https://API.SQRIL.IO"} {
		t.Setenv(envBaseURL, baseURL)

		if got, err := sandboxBaseURL(); !errors.Is(err, errSandboxPointsAtProduction) || got != "" {
			t.Errorf("%s: sandboxBaseURL = %q, %v, want the production error", baseURL, got, err)
		}

		client, err := sandboxClientWith("any-client-id", "any-client-secret")
		if !errors.Is(err, errSandboxPointsAtProduction) || client != nil {
			t.Errorf("%s: sandboxClientWith = %v, %v, want no client and the production error", baseURL, client, err)
		}
	}

	t.Setenv(envBaseURL, "")

	if got, err := sandboxBaseURL(); err != nil || got != gosqril.SandboxBaseURL {
		t.Errorf("unset: sandboxBaseURL = %q, %v, want the sandbox URL", got, err)
	}
}

func TestSandboxHelper_RefusesProduction(t *testing.T) {
	t.Parallel()

	for _, baseURL := range []string{gosqril.ProductionBaseURL, "https://api.sqril.io/", "https://API.SQRIL.IO", " https://api.sqril.io/v1", "::bad url"} {
		if !isProductionURL(baseURL) {
			t.Errorf("isProductionURL(%q) = false, want true", baseURL)
		}
	}

	if isProductionURL(gosqril.SandboxBaseURL) {
		t.Error("isProductionURL(sandbox) = true, want false")
	}
}

func TestSandbox_GetAccountBalances(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.GetAccountBalances(sandboxContext(t), nil)
	if err != nil {
		t.Fatalf("GetAccountBalances: %v", err)
	}

	if _, ok := resp.Balances["USD"]; !ok {
		t.Errorf("balances = %v, want a USD entry", resp.Balances)
	}
}

func TestSandbox_ListBanks(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.ListBanks(sandboxContext(t), &gosqril.ListBanksRequest{CountryCode: "VN"})
	if err != nil {
		t.Fatalf("ListBanks: %v", err)
	}

	if len(resp.Banks) == 0 || resp.Banks[0].Code == "" {
		t.Errorf("banks = %d, want at least one with a code", len(resp.Banks))
	}
}

func TestSandbox_ListOccupations(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.ListOccupations(sandboxContext(t), &gosqril.ListOccupationsRequest{
		Format: gosqril.OccupationFormatLegacy,
		Level:  0,
	})
	if err != nil {
		t.Fatalf("ListOccupations: %v", err)
	}

	if total, ok := resp.CountTotal(); !ok || total != 12 || len(resp.Occupations) != 12 {
		t.Errorf("count = %d (%v), occupations = %d, want 12", total, ok, len(resp.Occupations))
	}
}

func TestSandbox_ListThenGetTransaction(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)
	ctx := sandboxContext(t)

	page, err := client.ListTransactions(ctx, &gosqril.ListTransactionsRequest{Limit: 1})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}

	if len(page.Transactions) == 0 {
		t.Skip("the sandbox account has no transactions to read back")
	}

	listed := page.Transactions[0]

	fetched, err := client.GetTransaction(ctx, &gosqril.GetTransactionRequest{TransactionID: listed.ID, DP: nil})
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}

	if fetched.Transaction.ID != listed.ID || fetched.Transaction.Amount != listed.Amount {
		t.Errorf("fetched = %s %s, want %s %s", fetched.Transaction.ID, fetched.Transaction.Amount, listed.ID, listed.Amount)
	}
}

func TestSandbox_CheckCustomer(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	customerID := os.Getenv(envCustomerID)
	if customerID == "" {
		t.Skipf("sandbox test skipped: set %s", envCustomerID)
	}

	resp, err := client.CheckCustomer(sandboxContext(t), &gosqril.CheckCustomerRequest{CustomerID: customerID})
	if err != nil {
		t.Fatalf("CheckCustomer: %v", err)
	}

	if resp.CustomerID != customerID {
		t.Errorf("customer_id = %q, want %q", resp.CustomerID, customerID)
	}
}

func TestSandbox_PreviewQuotations(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	resp, err := client.PreviewQuotations(sandboxContext(t), &gosqril.PreviewQuotationsRequest{
		DP:       nil,
		Amount:   "100",
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("PreviewQuotations: %v", err)
	}

	for _, quotation := range resp.Quotations {
		if quotation.Currency == "" || quotation.Amount == "" || quotation.ExchangeRate == "" {
			t.Errorf("quotation = %+v, want currency, amount and exchange rate", quotation)
		}
	}
}

func TestSandbox_DecodeQR(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	qrString := os.Getenv(envQRString)
	if qrString == "" {
		t.Skipf("sandbox test skipped: set %s", envQRString)
	}

	ctx := sandboxContext(t)

	resp, err := client.DecodeQR(ctx, &gosqril.DecodeQRRequest{
		QRString:   qrString,
		CustomerID: sandboxCustomerID(ctx, t, client),
	})
	if err != nil {
		t.Fatalf("DecodeQR: %v", err)
	}

	if resp.TxID == "" || resp.Currency == "" {
		t.Errorf("decode = %+v, want tx_id and currency", resp)
	}
}

func TestSandbox_RegisterLookupQuoteFlow(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)
	ctx := sandboxContext(t)

	customerID := os.Getenv(envCustomerID)
	if customerID == "" {
		customerID = registerAndUpdateSandboxCustomer(ctx, t, client)
	}

	checked, err := client.CheckCustomer(ctx, &gosqril.CheckCustomerRequest{CustomerID: customerID})
	if err != nil {
		t.Fatalf("CheckCustomer: %v", err)
	}

	if checked.CustomerID != customerID {
		t.Errorf("customer_id = %q, want %q", checked.CustomerID, customerID)
	}

	lookup, err := client.LookupPaymentCode(ctx, &gosqril.LookupPaymentCodeRequest{
		CustomerID:             customerID,
		CountryCode:            "CO",
		PaymentMethod:          gosqril.PaymentMethodPaymentKey,
		PaymentDestination:     sandboxBREBStaticKey,
		PaymentDestinationCode: "BREB",
	})
	if err != nil {
		t.Fatalf("LookupPaymentCode: %v", err)
	}

	quotationRequest := &gosqril.GetQuotationRequest{TxID: lookup.TxID, CustomerID: customerID}
	if lookup.Amount == "" || lookup.Amount == "0" {
		quotationRequest.Amount = ptr(sandboxQuotationCOP)
	}

	quotation, err := client.GetQuotation(ctx, quotationRequest)
	if err != nil {
		t.Fatalf("GetQuotation: %v", err)
	}

	if quotation.TxID != lookup.TxID || quotation.ExchangeRate == "" || quotation.ExpiresAt == "" {
		t.Errorf("quotation = %+v", quotation)
	}

	wantStatus := gosqril.TransactionStatusPending

	if os.Getenv(envExecutePayout) == "1" {
		payout, err := client.ExecutePayout(ctx, &gosqril.ExecutePayoutRequest{
			IdempotencyKey:       fmt.Sprintf("%s-%d", sandboxMarker, time.Now().UnixNano()),
			TxID:                 lookup.TxID,
			AmountConfirmed:      quotation.Amount,
			Currency:             quotation.Currency,
			CustomerID:           customerID,
			PartnerTransactionID: fmt.Sprintf("%s-%d", sandboxMarker, time.Now().Unix()),
		})
		if err != nil {
			t.Fatalf("ExecutePayout: %v", err)
		}

		if payout.Status != gosqril.TransactionStatusProcessing {
			t.Errorf("payout status = %s, want PROCESSING", payout.Status)
		}

		wantStatus = ""
	}

	fetched, err := client.GetTransaction(ctx, &gosqril.GetTransactionRequest{TransactionID: lookup.TxID})
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}

	if fetched.Transaction.ID != lookup.TxID || fetched.Transaction.CustomerID != customerID {
		t.Errorf("transaction = %s of %s, want %s of %s", fetched.Transaction.ID, fetched.Transaction.CustomerID, lookup.TxID, customerID)
	}

	if wantStatus != "" && fetched.Transaction.Status != wantStatus {
		t.Errorf("status = %s, want %s", fetched.Transaction.Status, wantStatus)
	}
}

func TestSandbox_GetTransaction_NotFound(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	_, err := client.GetTransaction(sandboxContext(t), &gosqril.GetTransactionRequest{TransactionID: sandboxMissingTxID})

	assertAPIError(t, err, http.StatusNotFound, gosqril.ErrorCodeTransactionNotFound)

	if !gosqril.IsNotFound(err) {
		t.Errorf("IsNotFound = false for %v", err)
	}
}

func TestSandbox_WrongSecret_Unauthorized(t *testing.T) {
	t.Parallel()

	newSandboxClient(t)

	client, err := sandboxClientWith(os.Getenv(envClientID), "sdk-sandbox-test-wrong-secret")
	if err != nil {
		t.Fatalf("sandbox client: %v", err)
	}

	_, err = client.GetAccountBalances(sandboxContext(t), nil)
	if !gosqril.IsUnauthorized(err) {
		t.Errorf("error = %v, want a 401 *gosqril.APIError", err)
	}
}

func TestSandbox_PreviewQuotations_InvalidAmount(t *testing.T) {
	t.Parallel()

	client := newSandboxClient(t)

	_, err := client.PreviewQuotations(sandboxContext(t), &gosqril.PreviewQuotationsRequest{Amount: "0", Currency: "USD"})

	assertAPIError(t, err, http.StatusBadRequest, gosqril.ErrorCodeInvalidAmount)
}
