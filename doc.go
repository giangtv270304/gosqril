// Package gosqril is a Go client for the SQRIL SaaS payout API.
//
//	client, err := gosqril.NewClient(gosqril.SandboxBaseURL, clientID, clientSecret)
//	if err != nil {
//		return err
//	}
//
//	resp, err := client.GetAccountBalances(ctx, nil)
//
// Amounts and rates are Decimal values (decimal strings); they never pass through float64.
// The client secret is for servers only.
package gosqril
