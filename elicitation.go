package acp

import (
	"errors"
	"fmt"
	"net/url"
)

// SafeURL parses the elicitation target and rejects any scheme other than https
// and http.
//
// The agent chooses this URL, so an unchecked value is a navigation primitive
// controlled by the peer: javascript: reaches an embedded webview, file:// and
// smb:// reach the local filesystem or a UNC path.
//
// A successful parse is not permission to open the URL. Show the host to the
// user before navigating, and do not prefetch it for a preview.
func (u CreateElicitationUrl) SafeURL() (*url.URL, error) {
	parsed, err := url.Parse(u.Url)
	if err != nil {
		return nil, fmt.Errorf("elicitation url is not parseable: %w", err)
	}
	switch parsed.Scheme {
	case "https", "http":
	default:
		return nil, fmt.Errorf("elicitation url scheme %q is not allowed, want https or http", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, errors.New("elicitation url has no host")
	}
	return parsed, nil
}
