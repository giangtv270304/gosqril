package gosqril

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type request struct {
	Method         string
	Path           string
	Query          url.Values
	Body           any
	IdempotencyKey string
}

func requireField(name, value string) error {
	if value == "" {
		return fmt.Errorf("%w: %s", ErrMissingField, name)
	}

	return nil
}

func requireCustomerID(value string) error {
	if err := requireField("CustomerID", value); err != nil {
		return err
	}

	if strings.Contains(value, "/") {
		return fmt.Errorf("%w: CustomerID must not contain '/'", ErrInvalidField)
	}

	return nil
}

func validateOptionalCustomerID(value string) error {
	if value == "" {
		return nil
	}

	return requireCustomerID(value)
}

func queryWithDP(dp *int) url.Values {
	query := url.Values{}
	if dp != nil {
		query.Set("dp", strconv.Itoa(*dp))
	}

	return query
}
