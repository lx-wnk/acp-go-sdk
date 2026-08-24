package acp

import (
	"errors"
	"fmt"
	"net/url"
)

// SafeURL parses the elicitation target and rejects any scheme other than https
// and http. The agent chooses this URL, so treat it as untrusted input.
//
// A host carrying non-ASCII labels is returned in its ASCII (Punycode) form, so
// the caller displays a name that cannot be confused with a different one:
// https://аpple.com, whose first letter is Cyrillic, comes back as
// xn--pple-43d.com rather than a string that renders as apple.com.
//
// A successful parse is not permission to open the URL. Show the host to the
// user before navigating, and do not prefetch it for a preview.
func (u CreateElicitationUrl) SafeURL() (*url.URL, error) {
	parsed, err := url.Parse(u.Url)
	if err != nil {
		return nil, fmt.Errorf("elicitation url is not parseable: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, fmt.Errorf("elicitation url scheme %q is not allowed, want https or http", parsed.Scheme)
	}
	// https:///auth parses with an empty host and would otherwise pass.
	if parsed.Host == "" {
		return nil, errors.New("elicitation url has no host")
	}
	if !isASCII(parsed.Host) {
		ascii := asciiHost(parsed.Hostname())
		if port := parsed.Port(); port != "" {
			ascii += ":" + port
		}
		parsed.Host = ascii
	}
	return parsed, nil
}
