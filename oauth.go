package main

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"
)

func authenticate(c *Client, secretID, secretKey string) error {
	fmt.Println("Opening authentication...")

	inst, err := c.FindRevolut()
	if err != nil {
		return fmt.Errorf("finding Revolut: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	refChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			ref := r.URL.Query().Get("ref")
			if ref != "" {
				refChan <- ref
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Authentication successful! You can close this window."))
		})
		errChan <- http.Serve(listener, mux)
	}()

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
	rr, err := c.CreateRequisition(redirectURI, inst.ID)
	if err != nil {
		listener.Close()
		return fmt.Errorf("creating requisition: %w", err)
	}

	exec.Command("open", rr.Link).Start()

	select {
	case <-refChan:
	case err := <-errChan:
		listener.Close()
		return fmt.Errorf("server error: %w", err)
	case <-time.After(5 * time.Minute):
		listener.Close()
		return fmt.Errorf("authentication timed out")
	}
	listener.Close()

	accounts, err := c.GetRequisitionAccounts(rr.ID)
	if err != nil {
		return fmt.Errorf("getting accounts: %w", err)
	}

	keychainSet("account-id", accounts[0])
	keychainSet("requisition-id", rr.ID)

	_ = secretID
	_ = secretKey
	return nil
}
