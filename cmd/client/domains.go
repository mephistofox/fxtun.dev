package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type domainDTO struct {
	ID        int64     `json:"id"`
	Subdomain string    `json:"subdomain"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type domainsListResponse struct {
	Domains    []domainDTO `json:"domains"`
	Total      int         `json:"total"`
	MaxDomains int         `json:"max_domains"`
}

type domainCheckResponse struct {
	Subdomain string `json:"subdomain"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

func newDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Manage reserved subdomains",
		RunE:  runDomainsList,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List reserved subdomains",
		RunE:  runDomainsList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "add <subdomain>",
		Short: "Reserve a subdomain",
		Args:  cobra.ExactArgs(1),
		RunE:  runDomainsAdd,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "remove <subdomain>",
		Short: "Release a reserved subdomain",
		Args:  cobra.ExactArgs(1),
		RunE:  runDomainsRemove,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "check <subdomain>",
		Short: "Check if a subdomain is available",
		Args:  cobra.ExactArgs(1),
		RunE:  runDomainsCheck,
	})

	return cmd
}

func runDomainsList(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	resp, err := client.get("/domains")
	if err != nil {
		return fmt.Errorf("failed to fetch domains: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}

	data, err := decodeJSON[domainsListResponse](resp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(data.Domains) == 0 {
		fmt.Printf("No reserved domains. (limit: %d)\n", data.MaxDomains)
		return nil
	}

	fmt.Printf("Reserved domains (%d/%d):\n\n", data.Total, data.MaxDomains)
	for _, d := range data.Domains {
		fmt.Printf("  %-20s  %s  (created %s)\n",
			d.Subdomain,
			d.URL,
			d.CreatedAt.Format("2006-01-02"))
	}
	fmt.Println()
	return nil
}

func runDomainsAdd(cmd *cobra.Command, args []string) error {
	subdomain := strings.ToLower(args[0])

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	resp, err := client.post("/domains", map[string]string{"subdomain": subdomain})
	if err != nil {
		return fmt.Errorf("failed to reserve domain: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}

	data, err := decodeJSON[domainDTO](resp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Reserved: %s → %s\n", data.Subdomain, data.URL)
	return nil
}

func runDomainsRemove(cmd *cobra.Command, args []string) error {
	subdomain := strings.ToLower(args[0])

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	resp, err := client.get("/domains")
	if err != nil {
		return fmt.Errorf("failed to fetch domains: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}

	data, err := decodeJSON[domainsListResponse](resp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	var domainID int64
	for _, d := range data.Domains {
		if d.Subdomain == subdomain {
			domainID = d.ID
			break
		}
	}
	if domainID == 0 {
		return fmt.Errorf("domain '%s' not found in your reserved domains", subdomain)
	}

	delResp, err := client.delete(fmt.Sprintf("/domains/%d", domainID))
	if err != nil {
		return fmt.Errorf("failed to release domain: %w", err)
	}
	if delResp.StatusCode != http.StatusOK && delResp.StatusCode != http.StatusNoContent {
		return apiError(delResp)
	}
	delResp.Body.Close()

	fmt.Printf("Released: %s\n", subdomain)
	return nil
}

func runDomainsCheck(cmd *cobra.Command, args []string) error {
	subdomain := strings.ToLower(args[0])

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	resp, err := client.get("/domains/check/" + subdomain)
	if err != nil {
		return fmt.Errorf("failed to check domain: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}

	data, err := decodeJSON[domainCheckResponse](resp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if data.Available {
		fmt.Printf("%s — available\n", data.Subdomain)
	} else {
		fmt.Printf("%s — unavailable (%s)\n", data.Subdomain, data.Reason)
	}
	return nil
}
