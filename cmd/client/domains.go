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

type customDomainDTO struct {
	ID              int64      `json:"id"`
	Domain          string     `json:"domain"`
	TargetSubdomain string     `json:"target_subdomain"`
	Verified        bool       `json:"verified"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type customDomainsListResponse struct {
	Domains    []customDomainDTO `json:"domains"`
	Total      int               `json:"total"`
	MaxDomains int               `json:"max_domains"`
	BaseDomain string            `json:"base_domain"`
	ServerIP   string            `json:"server_ip"`
}

type verifyResponse struct {
	Verified bool   `json:"verified"`
	Error    string `json:"error,omitempty"`
	Expected string `json:"expected,omitempty"`
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

	customCmd := &cobra.Command{
		Use:   "custom",
		Short: "Manage custom domains (2nd level)",
		RunE:  runCustomDomainsList,
	}

	customCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List custom domains",
		RunE:  runCustomDomainsList,
	})

	addCustomCmd := &cobra.Command{
		Use:   "add <domain>",
		Short: "Add a custom domain",
		Args:  cobra.ExactArgs(1),
		RunE:  runCustomDomainsAdd,
	}
	addCustomCmd.Flags().StringP("target", "t", "", "Target subdomain (required)")
	_ = addCustomCmd.MarkFlagRequired("target")
	customCmd.AddCommand(addCustomCmd)

	customCmd.AddCommand(&cobra.Command{
		Use:   "remove <domain>",
		Short: "Remove a custom domain",
		Args:  cobra.ExactArgs(1),
		RunE:  runCustomDomainsRemove,
	})

	customCmd.AddCommand(&cobra.Command{
		Use:   "verify <domain>",
		Short: "Verify DNS for a custom domain",
		Args:  cobra.ExactArgs(1),
		RunE:  runCustomDomainsVerify,
	})

	cmd.AddCommand(customCmd)

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

func runCustomDomainsList(_ *cobra.Command, _ []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	resp, err := client.get("/custom-domains")
	if err != nil {
		return fmt.Errorf("failed to fetch custom domains: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}

	data, err := decodeJSON[customDomainsListResponse](resp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(data.Domains) == 0 {
		fmt.Printf("No custom domains. (limit: %d)\n", data.MaxDomains)
		fmt.Printf("Base domain: %s | Server IP: %s\n", data.BaseDomain, data.ServerIP)
		return nil
	}

	fmt.Printf("Custom domains (%d/%d):\n\n", data.Total, data.MaxDomains)
	for _, d := range data.Domains {
		status := "pending"
		if d.Verified {
			status = "verified"
		}
		fmt.Printf("  %-30s → %s.%s  [%s]\n",
			d.Domain, d.TargetSubdomain, data.BaseDomain, status)
	}
	fmt.Printf("\nDNS setup: CNAME → <subdomain>.%s  or  A → %s\n", data.BaseDomain, data.ServerIP)
	fmt.Println()
	return nil
}

func runCustomDomainsAdd(cmd *cobra.Command, args []string) error {
	domain := strings.ToLower(args[0])
	target, _ := cmd.Flags().GetString("target")

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	resp, err := client.post("/custom-domains", map[string]string{
		"domain":           domain,
		"target_subdomain": target,
	})
	if err != nil {
		return fmt.Errorf("failed to add custom domain: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}

	data, err := decodeJSON[customDomainDTO](resp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	status := "pending DNS verification"
	if data.Verified {
		status = "verified"
	}
	fmt.Printf("Added: %s → %s [%s]\n", data.Domain, data.TargetSubdomain, status)
	if !data.Verified {
		fmt.Println("Set up DNS records and run: fxtunnel domains custom verify " + domain)
	}
	return nil
}

func runCustomDomainsRemove(_ *cobra.Command, args []string) error {
	domain := strings.ToLower(args[0])

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	resp, err := client.get("/custom-domains")
	if err != nil {
		return fmt.Errorf("failed to fetch custom domains: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}

	data, err := decodeJSON[customDomainsListResponse](resp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	var domainID int64
	for _, d := range data.Domains {
		if d.Domain == domain {
			domainID = d.ID
			break
		}
	}
	if domainID == 0 {
		return fmt.Errorf("custom domain '%s' not found", domain)
	}

	delResp, err := client.delete(fmt.Sprintf("/custom-domains/%d", domainID))
	if err != nil {
		return fmt.Errorf("failed to remove custom domain: %w", err)
	}
	if delResp.StatusCode != http.StatusOK && delResp.StatusCode != http.StatusNoContent {
		return apiError(delResp)
	}
	delResp.Body.Close()

	fmt.Printf("Removed: %s\n", domain)
	return nil
}

func runCustomDomainsVerify(_ *cobra.Command, args []string) error {
	domain := strings.ToLower(args[0])

	client, err := newAPIClient()
	if err != nil {
		return err
	}

	resp, err := client.get("/custom-domains")
	if err != nil {
		return fmt.Errorf("failed to fetch custom domains: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}

	listData, err := decodeJSON[customDomainsListResponse](resp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	var domainID int64
	for _, d := range listData.Domains {
		if d.Domain == domain {
			domainID = d.ID
			break
		}
	}
	if domainID == 0 {
		return fmt.Errorf("custom domain '%s' not found", domain)
	}

	verifyResp, err := client.post(fmt.Sprintf("/custom-domains/%d/verify", domainID), nil)
	if err != nil {
		return fmt.Errorf("failed to verify domain: %w", err)
	}
	if verifyResp.StatusCode != http.StatusOK {
		return apiError(verifyResp)
	}

	result, err := decodeJSON[verifyResponse](verifyResp)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Verified {
		fmt.Printf("%s — verified! TLS certificate will be obtained automatically.\n", domain)
	} else {
		fmt.Printf("%s — verification failed: %s\n", domain, result.Error)
		if result.Expected != "" {
			fmt.Printf("Expected DNS target: %s\n", result.Expected)
		}
	}
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
