package gosqril

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// OccupationFormat selects the code set returned by ListOccupations.
type OccupationFormat string

// Formats accepted by ListOccupations.
const (
	OccupationFormatISCO   OccupationFormat = "isco"
	OccupationFormatLegacy OccupationFormat = "legacy"
	OccupationFormatAll    OccupationFormat = "all"
)

// Legacy occupation codes for RegisterCustomerRequest.Occupation.
const (
	OccupationCSuiteExecutive       = "OCC1"
	OccupationEmployee              = "OCC2"
	OccupationEntrepreneur          = "OCC3"
	OccupationForeignWorker         = "OCC4"
	OccupationForeignDomesticWorker = "OCC5"
	OccupationHomeMaker             = "OCC6"
	OccupationMiddleManagement      = "OCC7"
	OccupationPilot                 = "OCC8"
	OccupationProfessional          = "OCC9"
	OccupationSeniorManagement      = "OCC10"
	OccupationStudent               = "OCC11"
	OccupationOthers                = "OCC12"
)

// Occupation is one ISCO-08 or legacy entry. Level and LegacyOCC are set on ISCO entries,
// ISCOCode on legacy entries (nil for OCC12).
type Occupation struct {
	Code      string  `json:"code"`
	Title     string  `json:"title"`
	Level     *int    `json:"level,omitempty"`
	LegacyOCC *string `json:"legacy_occ,omitempty"`
	ISCOCode  *string `json:"isco_code,omitempty"`
}

// ListOccupationsRequest: zero values are left out and the API applies its defaults.
type ListOccupationsRequest struct {
	// Format defaults to isco.
	Format OccupationFormat
	// Level (1 to 4) filters ISCO entries by hierarchy level.
	Level int
}

// ListOccupationsResponse changes shape with the format: isco and legacy fill Occupations;
// all fills ISCO and Legacy.
type ListOccupationsResponse struct {
	Occupations []Occupation     `json:"occupations,omitempty"`
	ISCO        []Occupation     `json:"isco,omitempty"`
	Legacy      []Occupation     `json:"legacy,omitempty"`
	Format      OccupationFormat `json:"format,omitempty"`
	// Count is an integer for isco and legacy and an object for all; read the integer with CountTotal.
	Count json.RawMessage `json:"count,omitempty"`
	// Level echoes the requested level; nil when none was sent.
	Level *int `json:"level,omitempty"`
}

// CountTotal returns Count when the API sent an integer. ok is false otherwise (format all).
func (r *ListOccupationsResponse) CountTotal() (total int, ok bool) {
	if err := json.Unmarshal(r.Count, &total); err != nil {
		return 0, false
	}

	return total, true
}

// ListOccupations returns the occupation codes accepted by RegisterCustomer. Pass nil for defaults.
func (c *Client) ListOccupations(ctx context.Context, req *ListOccupationsRequest) (*ListOccupationsResponse, error) {
	query := url.Values{}

	if req != nil {
		if req.Format != "" {
			query.Set("format", string(req.Format))
		}

		if req.Level > 0 {
			query.Set("level", strconv.Itoa(req.Level))
		}
	}

	return call[ListOccupationsResponse](ctx, c, request{
		Method:         http.MethodGet,
		Path:           "/listOccupations",
		Query:          query,
		Body:           nil,
		IdempotencyKey: "",
	})
}
