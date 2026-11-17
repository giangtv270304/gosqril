package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestListBanks_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"banks":[{"code":"013","name":"Bank Permata","short_name":"Permata","swift_code":"BBBAIDJA","type":"bank","country_code":"ID"},{"code":"013","name":"Bank Permata Syariah","swift_code":"BBBAIDJS"},{"code":"688","name":"BPR KS","channel_category":"x"}],"country_code":"ID","total":3}`)

	resp, err := client.ListBanks(context.Background(), &gosqril.ListBanksRequest{CountryCode: "ID"})
	if err != nil {
		t.Fatalf("ListBanks: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/listBanks", "country_code=ID", "")

	if len(resp.Banks) != 3 || resp.CountryCode != "ID" || resp.Total != 3 {
		t.Fatalf("response = %+v", resp)
	}

	want := gosqril.Bank{Code: "013", Name: "Bank Permata", ShortName: "Permata", SwiftCode: "BBBAIDJA", Type: "bank", CountryCode: "ID"}
	if resp.Banks[0] != want {
		t.Errorf("bank = %+v, want %+v", resp.Banks[0], want)
	}

	if resp.Banks[1].Code != "013" || resp.Banks[1].SwiftCode != "BBBAIDJS" || resp.Banks[2].SwiftCode != "" {
		t.Errorf("banks = %+v", resp.Banks)
	}
}

func TestListBanks_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusForbidden,
		`{"error_code":"COUNTRY_NOT_SUPPORTED_BY_ACCOUNT","error_message":"Account does not support this country"}`)

	_, err := client.ListBanks(context.Background(), &gosqril.ListBanksRequest{CountryCode: "ID"})

	assertAPIError(t, err, http.StatusForbidden, gosqril.ErrorCodeCountryNotSupportedByAccount)
}
