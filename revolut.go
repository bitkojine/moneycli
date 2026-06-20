package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func getRevolutToken() (string, error) {
	token, err := keychainGet("revolut-token")
	if err == nil && token != "" {
		return token, nil
	}

	fmt.Println("Opening Revolut login in your browser...")
	fmt.Println("Please log into your Revolut account in the opened window.")

	token, err = authRevolut()
	if err != nil {
		return "", err
	}

	keychainSet("revolut-token", token)
	return token, nil
}

func authRevolut() (string, error) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-background-networking", false),
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	tokenCh := make(chan string, 1)

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			if strings.Contains(e.Request.URL, "/api/retail/") {
				if a, ok := e.Request.Headers["Authorization"]; ok {
					if s, ok := a.(string); ok && strings.HasPrefix(s, "Bearer ") {
						select {
						case tokenCh <- strings.TrimPrefix(s, "Bearer "):
						default:
						}
					}
				}
			}
		}
	})

	chromedp.Run(ctx, chromedp.Navigate("https://app.revolut.com"))

	select {
	case token := <-tokenCh:
		return token, nil
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("login timed out")
	}
}

func getRevolutBalance(token string) (float64, string, error) {
	req, _ := http.NewRequest("GET", "https://app.revolut.com/api/retail/user/current", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return 0, "", fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, "", err
	}

	if state, ok := data["state"].(map[string]interface{}); ok {
		if accounts, ok := state["accounts"].([]interface{}); ok && len(accounts) > 0 {
			if acct, ok := accounts[0].(map[string]interface{}); ok {
				bal, hasBalance := acct["balance"].(float64)
				currency, hasCurrency := acct["currency"].(string)
				if hasBalance {
					if !hasCurrency {
						currency = "EUR"
					}
					return bal, currency, nil
				}
			}
		}
	}

	return 0, "", fmt.Errorf("balance not found in API response")
}
