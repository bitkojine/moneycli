package main

import (
	"fmt"
	"os"
	"time"
)

func mockMain() {
	amount := 4281.17
	currency := "EUR"
	symbol := currencySymbol(currency)

	fmt.Printf("Revolut balance\n\n%s %s%.2f\n\nLast updated: %s\n",
		currency, symbol, amount, time.Now().Format("2006-01-02 15:04"))

	os.Exit(0)
}
