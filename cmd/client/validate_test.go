package main

import "testing"

// SUSPECT: the "username must not contain ':'" check after
// strings.SplitN(auth, ":", 2) was dead code — SplitN with limit 2 can never
// put a ':' into the first part. A colon in the password half must still
// pass through untouched.
func TestSplitAuthFlag(t *testing.T) {
	cases := []struct {
		name     string
		auth     string
		wantUser string
		wantPass string
		wantErr  bool
	}{
		{name: "valid", auth: "alice:longpassword", wantUser: "alice", wantPass: "longpassword"},
		{name: "colon in password", auth: "alice:pass:word12", wantUser: "alice", wantPass: "pass:word12"},
		{name: "no colon", auth: "aliceonly", wantErr: true},
		{name: "empty username", auth: ":longpassword", wantErr: true},
		{name: "short password", auth: "alice:short", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			user, pass, err := splitAuthFlag(c.auth)
			if (err != nil) != c.wantErr {
				t.Fatalf("splitAuthFlag(%q): err=%v, want error=%v", c.auth, err, c.wantErr)
			}
			if err != nil {
				return
			}
			if user != c.wantUser || pass != c.wantPass {
				t.Fatalf("splitAuthFlag(%q) = (%q, %q), want (%q, %q)", c.auth, user, pass, c.wantUser, c.wantPass)
			}
		})
	}
}

// SUSPECT: `tcp --remote-port -5` / `--remote-port 99999` were accepted and
// only failed (or silently misbehaved) once the client tried to connect.
func TestValidateRemotePort(t *testing.T) {
	cases := []struct {
		port int
		ok   bool
	}{
		{0, true}, // auto-assign
		{1, true}, // lower bound
		{2222, true},
		{65535, true}, // upper bound
		{-5, false},
		{65536, false},
		{99999, false},
	}
	for _, c := range cases {
		err := validateRemotePort(c.port)
		if (err == nil) != c.ok {
			t.Errorf("validateRemotePort(%d): err=%v, want ok=%v", c.port, err, c.ok)
		}
	}
}
