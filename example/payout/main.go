// Payout: decode a QR, quote it, and pay it. ExecutePayout runs only with -execute, because it moves money.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/giangtv270304/gosqril"
)

func main() {
	execute := flag.Bool("execute", false, "execute the payout after the quotation")
	amount := flag.String("amount", "", "local-currency amount, for a QR that carries none")
	flag.Parse()

	client, err := gosqril.NewClient(gosqril.SandboxBaseURL, os.Getenv("SQRIL_CLIENT_ID"), os.Getenv("SQRIL_CLIENT_SECRET"))
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	customerID := os.Getenv("SQRIL_TEST_CUSTOMER_ID")
	partnerTxID := fmt.Sprintf("sdk-sandbox-test-%d", time.Now().UnixMilli())

	decoded, err := client.DecodeQR(ctx, &gosqril.DecodeQRRequest{
		QRString: os.Getenv("SQRIL_TEST_QR_STRING"), CustomerID: customerID, PartnerTransactionID: partnerTxID,
	})
	if err != nil {
		var apiErr *gosqril.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("decode qr: %s (status %d, code %s)", apiErr.Message, apiErr.StatusCode, apiErr.Code)
		}

		log.Fatalf("decode qr: %v", err)
	}

	for _, missing := range decoded.MissingFields {
		fmt.Printf("still needed before payout: %s\n", missing.Field)
	}

	quoteReq := &gosqril.GetQuotationRequest{TxID: decoded.TxID, CustomerID: customerID}
	if *amount != "" {
		value := gosqril.Decimal(*amount)
		quoteReq.Amount = &value
	}

	quotation, err := client.GetQuotation(ctx, quoteReq)
	if err != nil {
		log.Fatalf("get quotation %s: %v", decoded.TxID, err)
	}

	fmt.Printf("tx %s: %s %s = %s USD, fee %s USD, valid until %s\n",
		quotation.TxID, quotation.Amount, quotation.Currency, quotation.AmountUSD, quotation.Fee, quotation.ExpiresAt)

	if !*execute {
		fmt.Println("not executed; run with -execute to pay")

		return
	}

	payout, err := client.ExecutePayout(ctx, &gosqril.ExecutePayoutRequest{
		IdempotencyKey:  partnerTxID,
		TxID:            decoded.TxID,
		AmountConfirmed: quotation.Amount,
		Currency:        quotation.Currency,
		CustomerID:      customerID,
	})
	if err != nil {
		var apiErr *gosqril.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			log.Fatalf("payout refused: %s (status %d, code %s)", apiErr.Message, apiErr.StatusCode, apiErr.Code)
		}

		log.Fatalf("payout outcome unknown: call GetTransaction %s or resend with idempotency key %s: %v", decoded.TxID, partnerTxID, err)
	}

	fmt.Printf("tx %s: %s\n", payout.TxID, payout.Status)
}
