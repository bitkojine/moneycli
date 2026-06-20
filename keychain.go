package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func keychainSet(account, value string) error {
	cmd := exec.Command("security", "add-generic-password",
		"-s", "moneycli",
		"-a", account,
		"-w", value,
		"-U",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("keychain set %s: %s: %s", account, err, string(out))
	}
	return nil
}

func keychainGet(account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", "moneycli",
		"-a", account,
		"-w",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("keychain get %s: %w", account, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func keychainDelete(account string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", "moneycli",
		"-a", account,
	)
	return cmd.Run()
}

func keychainDeleteAll() {
	for _, a := range []string{"gocardless-access", "gocardless-refresh", "account-id", "requisition-id", "secret-id", "secret-key"} {
		keychainDelete(a)
	}
}
