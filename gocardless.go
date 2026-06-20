package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://bankaccountdata.gocardless.com/api/v2"

type Client struct {
	secretID   string
	secretKey  string
	httpClient *http.Client
}

type tokenResponse struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

type institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type requisitionResponse struct {
	ID       string   `json:"id"`
	Link     string   `json:"link"`
	Status   string   `json:"status"`
	Accounts []string `json:"accounts"`
}

type accountBalance struct {
	Balances []struct {
		BalanceAmount struct {
			Amount   string `json:"amount"`
			Currency string `json:"currency"`
		} `json:"balanceAmount"`
		BalanceType string `json:"balanceType"`
	} `json:"balances"`
}

func NewClient(secretID, secretKey string) *Client {
	return &Client{
		secretID:   secretID,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) tokenRequest(path string, body map[string]string) (*tokenResponse, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", baseURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token API error %d: %s", resp.StatusCode, string(raw))
	}

	var tr tokenResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func (c *Client) getAPIToken() error {
	tr, err := c.tokenRequest("/token/new/", map[string]string{
		"secret_id":  c.secretID,
		"secret_key": c.secretKey,
	})
	if err != nil {
		return err
	}
	keychainSet("gocardless-access", tr.Access)
	keychainSet("gocardless-refresh", tr.Refresh)
	return nil
}

func (c *Client) refreshAPIToken() error {
	refresh, err := keychainGet("gocardless-refresh")
	if err != nil {
		return c.getAPIToken()
	}
	tr, err := c.tokenRequest("/token/refresh/", map[string]string{"refresh": refresh})
	if err != nil {
		return c.getAPIToken()
	}
	keychainSet("gocardless-access", tr.Access)
	keychainSet("gocardless-refresh", tr.Refresh)
	return nil
}

func (c *Client) validToken() (string, error) {
	token, err := keychainGet("gocardless-access")
	if err != nil {
		if err := c.getAPIToken(); err != nil {
			return "", err
		}
		return keychainGet("gocardless-access")
	}
	return token, nil
}

func (c *Client) apiGet(path string, result interface{}) error {
	token, err := c.validToken()
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	req, _ := http.NewRequest("GET", baseURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		if err := c.refreshAPIToken(); err != nil {
			return fmt.Errorf("needs-auth")
		}
		token, _ = keychainGet("gocardless-access")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(raw))
	}

	if result != nil {
		return json.Unmarshal(raw, result)
	}
	return nil
}

func (c *Client) FindRevolut() (*institution, error) {
	countries := []string{"GB", "IE", "ES", "FR", "DE", "LT", "NL", "BE", "PT", "IT"}
	for _, country := range countries {
		var institutions []institution
		err := c.apiGet("/institutions/?country="+country, &institutions)
		if err != nil {
			continue
		}
		for _, inst := range institutions {
			if strings.Contains(strings.ToLower(inst.Name), "revolut") {
				return &inst, nil
			}
		}
	}
	return nil, fmt.Errorf("Revolut institution not found")
}

func (c *Client) CreateRequisition(redirectURI, institutionID string) (*requisitionResponse, error) {
	ref := fmt.Sprintf("moneycli-%d", time.Now().UnixMilli())
	body := map[string]string{
		"redirect":       redirectURI,
		"institution_id": institutionID,
		"reference":      ref,
	}
	data, _ := json.Marshal(body)

	token, err := c.validToken()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("POST", baseURL+"/requisitions/", bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("create requisition error %d: %s", resp.StatusCode, string(raw))
	}

	var rr requisitionResponse
	if err := json.Unmarshal(raw, &rr); err != nil {
		return nil, err
	}
	return &rr, nil
}

func (c *Client) GetRequisitionAccounts(requisitionID string) ([]string, error) {
	var rr requisitionResponse
	err := c.apiGet("/requisitions/"+requisitionID+"/", &rr)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 5 && len(rr.Accounts) == 0; i++ {
		time.Sleep(time.Second)
		var rr2 requisitionResponse
		if err := c.apiGet("/requisitions/"+requisitionID+"/", &rr2); err == nil {
			rr = rr2
		}
	}
	if len(rr.Accounts) == 0 {
		return nil, fmt.Errorf("no accounts linked yet (status: %s)", rr.Status)
	}
	return rr.Accounts, nil
}

func (c *Client) GetBalance(accountID string) (amount float64, currency string, err error) {
	var ab accountBalance
	err = c.apiGet("/accounts/"+accountID+"/balances/", &ab)
	if err != nil {
		return 0, "", err
	}

	var best struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}
	for _, b := range ab.Balances {
		amt := b.BalanceAmount
		if b.BalanceType == "interimAvailable" || b.BalanceType == "closingBooked" {
			best = amt
			break
		}
		if best.Amount == "" {
			best = amt
		}
	}

	if best.Amount == "" {
		return 0, "", fmt.Errorf("no balance found")
	}

	fmt.Sscanf(best.Amount, "%f", &amount)
	return amount, best.Currency, nil
}
