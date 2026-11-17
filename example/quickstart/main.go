// Quickstart: read the sandbox balances, then preview what 100 USD buys in each corridor.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/giangtv270304/gosqril"
)

func main() {
	client, err := gosqril.NewClient(
		gosqril.SandboxBaseURL,
		os.Getenv("SQRIL_CLIENT_ID"),
		os.Getenv("SQRIL_CLIENT_SECRET"),
	)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	balances, err := client.GetAccountBalances(ctx, nil)
	if err != nil {
		var apiErr *gosqril.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("get balances: %s (status %d, code %s)", apiErr.Message, apiErr.StatusCode, apiErr.Code)
		}

		log.Fatalf("get balances: %v", err)
	}

	for currency, balance := range balances.Balances {
		fmt.Printf("%s available %s locked %s\n", currency, balance.Available, balance.Locked)
	}

	preview, err := client.PreviewQuotations(ctx, &gosqril.PreviewQuotationsRequest{Amount: "100", Currency: "USD"})
	if err != nil {
		log.Fatalf("preview quotations: %v", err)
	}

	for _, quotation := range preview.Quotations {
		fmt.Printf("%s %s at %s USD each, fee %s USD\n", quotation.Amount, quotation.Currency, quotation.ExchangeRate, quotation.Fee)
	}
}
