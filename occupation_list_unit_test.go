package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestListOccupations_SuccessISCO(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"occupations":[{"code":"1","level":1,"title":"Managers","legacy_occ":"OCC1"},{"code":"2","level":1,"title":"Professionals","legacy_occ":"OCC9"}],"format":"isco","count":2,"level":1}`)

	resp, err := client.ListOccupations(context.Background(), &gosqril.ListOccupationsRequest{
		Format: gosqril.OccupationFormatISCO,
		Level:  1,
	})
	if err != nil {
		t.Fatalf("ListOccupations: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/listOccupations", "format=isco&level=1", "")

	if len(resp.Occupations) != 2 || resp.Format != gosqril.OccupationFormatISCO || resp.Level == nil || *resp.Level != 1 {
		t.Fatalf("response = %+v", resp)
	}

	second := resp.Occupations[1]
	if second.Code != "2" || second.Title != "Professionals" || second.Level == nil || *second.Level != 1 ||
		second.LegacyOCC == nil || *second.LegacyOCC != gosqril.OccupationProfessional || second.ISCOCode != nil {
		t.Errorf("occupation = %+v", second)
	}

	if total, ok := resp.CountTotal(); !ok || total != 2 {
		t.Errorf("CountTotal = %d, %v, want 2, true", total, ok)
	}
}

func TestListOccupations_SuccessLegacy(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"occupations":[{"code":"OCC1","title":"C-Suite executive / Board member","isco_code":"1120"},{"code":"OCC12","title":"Others","isco_code":null}],"format":"legacy","count":12}`)

	resp, err := client.ListOccupations(context.Background(), &gosqril.ListOccupationsRequest{Format: gosqril.OccupationFormatLegacy})
	if err != nil {
		t.Fatalf("ListOccupations: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/listOccupations", "format=legacy", "")

	if len(resp.Occupations) != 2 || resp.Level != nil {
		t.Fatalf("response = %+v", resp)
	}

	first, last := resp.Occupations[0], resp.Occupations[1]
	if first.Code != gosqril.OccupationCSuiteExecutive || first.ISCOCode == nil || *first.ISCOCode != "1120" || first.Level != nil {
		t.Errorf("first = %+v", first)
	}

	if last.Code != gosqril.OccupationOthers || last.ISCOCode != nil {
		t.Errorf("last = %+v", last)
	}

	if total, ok := resp.CountTotal(); !ok || total != 12 {
		t.Errorf("CountTotal = %d, %v, want 12, true", total, ok)
	}
}

func TestListOccupations_SuccessAllWithNilRequest(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"isco":[{"code":"1","level":1,"title":"Managers","legacy_occ":null}],"legacy":[{"code":"OCC12","title":"Others","isco_code":null}],"count":{"isco":1,"legacy":1}}`)

	resp, err := client.ListOccupations(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListOccupations: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/listOccupations", "", "")

	if len(resp.ISCO) != 1 || len(resp.Legacy) != 1 || resp.Occupations != nil || resp.ISCO[0].LegacyOCC != nil {
		t.Errorf("response = %+v", resp)
	}

	if total, ok := resp.CountTotal(); ok {
		t.Errorf("CountTotal = %d, true, want false for a count object", total)
	}

	if string(resp.Count) != `{"isco":1,"legacy":1}` {
		t.Errorf("count = %s, want the raw object", resp.Count)
	}
}

func TestListOccupations_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusUnauthorized,
		`{"error_code":"AUTHENTICATION_REQUIRED","error_message":"Authentication is missing"}`)

	_, err := client.ListOccupations(context.Background(), nil)

	assertAPIError(t, err, http.StatusUnauthorized, gosqril.ErrorCodeAuthenticationRequired)

	if !gosqril.IsUnauthorized(err) {
		t.Error("IsUnauthorized = false, want true")
	}
}
