package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/quiet-terminal-interactive/qtidocs/internal/registry"
)

func main() {
	sitesDir := flag.String("sites-dir", "sites", "registry repo directory holding one <subdomain>.yaml per site")
	platformURL := flag.String("platform-url", "", "URL of the platform's sweep endpoint (required)")
	secret := flag.String("secret", os.Getenv("QTIDOCS_PLATFORM_SECRET"), "shared secret authenticating against the platform (defaults to $QTIDOCS_PLATFORM_SECRET)")
	flag.Parse()

	if *platformURL == "" {
		fmt.Fprintln(os.Stderr, "registrysweep: -platform-url is required")
		flag.Usage()
		os.Exit(2)
	}
	if *secret == "" {
		fmt.Fprintln(os.Stderr, "registrysweep: -secret (or $QTIDOCS_PLATFORM_SECRET) is required")
		flag.Usage()
		os.Exit(2)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	if err := run(*sitesDir, *platformURL, *secret, client); err != nil {
		fmt.Fprintf(os.Stderr, "registrysweep: FAIL\n%v\n", err)
		os.Exit(1)
	}
	fmt.Println("registrysweep: PASS")
}

func run(sitesDir, platformURL, secret string, client *http.Client) error {
	current, err := registry.LoadAll(sitesDir)
	if err != nil {
		return err
	}

	registered := make([]string, 0, len(current))
	for subdomain := range current {
		registered = append(registered, subdomain)
	}

	body, err := json.Marshal(struct {
		Registered []string `json:"registered"`
	}{registered})
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, platformURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secret)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("calling platform: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("platform returned %s", resp.Status)
	}

	var result struct {
		TornDown int `json:"torn_down"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding platform response: %w", err)
	}

	fmt.Printf("registrysweep: reported %d registered site(s); platform tore down %d unregistered entry(s)\n", len(registered), result.TornDown)
	return nil
}
