// Package reserved holds the subdomains tunnel clients may not claim.
//
// It lives on its own because two unrelated packages need the same answer: the
// tunnel control plane, when a client asks for a subdomain, and the REST API,
// when a user reserves one. They were checking different things — a name the
// control plane refused could still be reserved through the API, which let a
// user sit on an infrastructure name and keep it from everyone else.
package reserved

import (
	"regexp"
	"strings"
)

// subdomains are refused for tunnels and for reservations alike.
//
// Three kinds of name are in here: ones that already exist in our DNS zones
// (taking them would shadow real infrastructure), ones a visitor would read as
// official (a phishing page on admin.fxtun.ru needs no further explanation),
// and the operational names any future service is likely to want.
var subdomains = map[string]bool{
	// Infrastructure and control surfaces
	"www": true, "admin": true, "administrator": true, "cp": true,
	"cpanel": true, "panel": true, "control": true, "dashboard": true,
	"api": true, "app": true, "tunnel": true, "tunnels": true,
	"console": true, "manage": true, "manager": true, "root": true,

	// Naming and mail, where a wrong answer breaks delivery or resolution
	"ns": true, "ns1": true, "ns2": true, "ns3": true, "ns4": true,
	"dns": true, "mx": true, "mail": true, "email": true, "webmail": true,
	"smtp": true, "imap": true, "pop": true, "pop3": true,
	"ftp": true, "sftp": true, "autoconfig": true, "autodiscover": true,
	"_dmarc": true, "_domainkey": true, "_acme-challenge": true,

	// Anything that speaks for the brand or handles credentials and money
	"login": true, "signin": true, "signup": true, "register": true,
	"auth": true, "oauth": true, "sso": true, "account": true, "accounts": true,
	"billing": true, "pay": true, "payment": true, "payments": true,
	"checkout": true, "invoice": true, "secure": true, "security": true,
	"support": true, "help": true, "helpdesk": true, "abuse": true,
	"postmaster": true, "hostmaster": true, "webmaster": true,
	"noreply": true, "no-reply": true,

	// Content and tooling we host or are likely to
	"blog": true, "docs": true, "doc": true, "wiki": true, "kb": true,
	"status": true, "health": true, "metrics": true, "monitor": true,
	"mon": true, "grafana": true, "prometheus": true, "kibana": true,
	"cdn": true, "static": true, "assets": true, "files": true,
	"download": true, "downloads": true, "git": true, "ci": true,
	"cd": true, "jenkins": true, "registry": true, "vpn": true, "proxy": true,

	// Environment names, so a tunnel cannot pose as our own staging
	"test": true, "testing": true, "stage": true, "staging": true,
	"dev": true, "prod": true, "production": true, "demo": true,
	"internal": true, "private": true, "local": true, "localhost": true,

	// The brand itself
	"fxtun": true, "fxtunnel": true,
}

// numberedInfraNames matches the families that are conventionally numbered:
// name servers and mail exchangers. Listing ns1 through ns4 by hand leaves ns5
// open, and the next person to add a name server would not think to update
// this file.
var numberedInfraNames = regexp.MustCompile(`^(ns|dns|mx|smtp|mail|pop|imap|www)[0-9]+$`)

// IsReserved reports whether a subdomain is off limits. Comparison is
// case-insensitive: hostnames are, and a check that is not would be trivially
// sidestepped with Admin instead of admin.
func IsReserved(subdomain string) bool {
	name := strings.ToLower(strings.TrimSpace(subdomain))
	return subdomains[name] || numberedInfraNames.MatchString(name)
}
