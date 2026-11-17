package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

const customerBody = `{"customer_id":"cus_abc123","name_first":"John","name_last":"Doe","gender":"M","ic_number":"E1234567","ic_type":"PP","ic_country":"SG","occupation":"OCC2","country_of_residence":"SG","phone":"+6591234567","email":"john.doe@example.com","nationality":"SG","dob":"1990-01-31","address":"1 Raffles Place, Singapore","ic_expiry_date":"2099-01-01","created_at":"2026-04-22T10:00:00.000Z","updated_at":"2026-04-22T10:00:00.000Z"}`

func assertCustomer(t *testing.T, customer *gosqril.Customer) {
	t.Helper()

	if customer.CustomerID != "cus_abc123" || customer.NameFirst != "John" || customer.NameLast != "Doe" {
		t.Errorf("customer = %s %s %s", customer.CustomerID, customer.NameFirst, customer.NameLast)
	}

	if customer.ICType != gosqril.ICTypePassport || customer.ICNumber != "E1234567" || customer.ICCountry != "SG" {
		t.Errorf("identity = %s %s %s", customer.ICType, customer.ICNumber, customer.ICCountry)
	}

	if customer.DOB != "1990-01-31" || customer.ICExpiryDate != "2099-01-01" || customer.CreatedAt != "2026-04-22T10:00:00.000Z" {
		t.Errorf("dates = %s %s %s", customer.DOB, customer.ICExpiryDate, customer.CreatedAt)
	}

	if customer.Occupation != gosqril.OccupationEmployee || customer.Nationality != "SG" || customer.Phone != "+6591234567" {
		t.Errorf("profile = %s %s %s", customer.Occupation, customer.Nationality, customer.Phone)
	}
}

func TestRegisterCustomer_Success(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusCreated, http.StatusOK} {
		client, recorded := newFakeServer(t, status, customerBody)

		resp, err := client.RegisterCustomer(context.Background(), &gosqril.RegisterCustomerRequest{
			NameFirst:          "John",
			NameLast:           "Doe",
			ICNumber:           "E1234567",
			ICType:             gosqril.ICTypePassport,
			ICCountry:          "SG",
			Occupation:         gosqril.OccupationEmployee,
			CountryOfResidence: "SG",
			DOB:                "1990-01-31",
			Address:            "1 Raffles Place, Singapore",
			ICExpiryDate:       "2099-01-01",
			Gender:             "",
			Phone:              "+6591234567",
			Email:              "",
			Nationality:        "SG",
		})
		if err != nil {
			t.Fatalf("status %d: RegisterCustomer: %v", status, err)
		}

		assertRequest(t, recorded, http.MethodPost, "/registerCustomer", "",
			`{"name_first":"John","name_last":"Doe","ic_number":"E1234567","ic_type":"PP","ic_country":"SG","occupation":"OCC2","country_of_residence":"SG","dob":"1990-01-31","address":"1 Raffles Place, Singapore","ic_expiry_date":"2099-01-01","phone":"+6591234567","nationality":"SG"}`)
		assertCustomer(t, resp)

		if resp.Allowed != nil {
			t.Errorf("allowed = %v, want nil", resp.Allowed)
		}
	}
}

func TestRegisterCustomer_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusBadRequest,
		`{"error_code":"MISSING_REQUIRED_FIELD","error_message":"name_first is required"}`)

	_, err := client.RegisterCustomer(context.Background(), &gosqril.RegisterCustomerRequest{NameLast: "Doe"})

	apiErr := requireAPIError(t, err, http.StatusBadRequest, gosqril.ErrorCodeMissingRequiredField)
	if apiErr.Message != "name_first is required" {
		t.Errorf("message = %q", apiErr.Message)
	}
}
