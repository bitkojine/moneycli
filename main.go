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

	token, err := getRevolutToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Authentication failed:", err)
		os.Exit(1)
	}

	amount, currency, err := getRevolutBalance(token)
	if err != nil {
		// Token may have expired — clear and retry
		keychainDelete("revolut-token")
		token, err = getRevolutToken()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Authentication failed:", err)
			os.Exit(1)
		}
		amount, currency, err = getRevolutBalance(token)
		if err != nil {
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
