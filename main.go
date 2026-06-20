package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if os.Getenv("MOCK") == "true" {
		mockMain()
		return
	}

	secretID, err := keychainGet("secret-id")
	if err != nil {
		secretID = os.Getenv("GOCARDLESS_SECRET_ID")
	}
	secretKey, err := keychainGet("secret-key")
	if err != nil {
		secretKey = os.Getenv("GOCARDLESS_SECRET_KEY")
	}

	if secretID == "" || secretKey == "" {
		fmt.Fprintln(os.Stderr, "GoCardless API credentials not found.")
		fmt.Fprintln(os.Stderr, "Set GOCARDLESS_SECRET_ID and GOCARDLESS_SECRET_KEY environment variables.")
		os.Exit(1)
	}

	// Store credentials in keychain for future runs
	keychainSet("secret-id", secretID)
	keychainSet("secret-key", secretKey)

	client := NewClient(secretID, secretKey)

	accountID, err := keychainGet("account-id")
	if err != nil || accountID == "" {
		if err := authenticate(client, secretID, secretKey); err != nil {
			fmt.Fprintln(os.Stderr, "Authentication failed:", err)
			os.Exit(1)
		}
		accountID, _ = keychainGet("account-id")
	}

	amount, currency, err := client.GetBalance(accountID)
	if err != nil {
		if err.Error() == "needs-auth" || err.Error() == "auth: needs-auth" {
			keychainDeleteAll()
			if err := authenticate(client, secretID, secretKey); err != nil {
				fmt.Fprintln(os.Stderr, "Authentication failed:", err)
				os.Exit(1)
			}
			accountID, _ = keychainGet("account-id")
			amount, currency, err = client.GetBalance(accountID)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Unable to fetch balance. Try again.")
				os.Exit(1)
			}
		} else {
			fmt.Fprintln(os.Stderr, "Unable to fetch balance. Try again.")
			os.Exit(1)
		}
	}

	symbol := currencySymbol(currency)
	fmt.Printf("Revolut balance\n\n%s %s%.2f\n\nLast updated: %s\n",
		currency, symbol, amount, time.Now().Format("2006-01-02 15:04"))
}

func currencySymbol(code string) string {
	switch code {
	case "EUR":
		return "€"
	case "USD":
		return "$"
	case "GBP":
		return "£"
	case "JPY":
		return "¥"
	case "CHF":
		return "CHF "
	case "SEK":
		return "kr "
	case "NOK":
		return "kr "
	case "DKK":
		return "kr "
	case "PLN":
		return "zł"
	case "CZK":
		return "Kč "
	case "HUF":
		return "Ft "
	}
	return code + " "
}
