// Webhook: verify SQRIL webhooks over the raw body, then record the final status of the transaction.
package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/giangtv270304/gosqril"
)

func main() {
	secret := os.Getenv("SQRIL_WEBHOOK_SECRET")

	http.HandleFunc("/webhooks/sqril", func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, secret)
	})

	server := &http.Server{Addr: ":8080", ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(server.ListenAndServe())
}

func handle(w http.ResponseWriter, r *http.Request, secret string) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)

		return
	}

	if err := gosqril.VerifyWebhookSignature(secret, raw, r.Header.Get(gosqril.WebhookSignatureHeader)); err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)

		return
	}

	event, err := gosqril.ParseWebhookEvent(raw)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)

		return
	}

	switch event.Status {
	case gosqril.TransactionStatusSuccess, gosqril.TransactionStatusFailed, gosqril.TransactionStatusRefunded:
		log.Printf("tx %s: %s", event.TxID, event.Status)
	default:
		log.Printf("tx %s: unexpected status %s", event.TxID, event.Status)
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"received": true, "message": "Webhook processed successfully"}`))
}
