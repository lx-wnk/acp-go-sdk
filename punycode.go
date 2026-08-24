package acp

import (
	"strings"
	"unicode/utf8"
)

// Punycode (RFC 3492) parameters.
const (
	punyBase        = 36
	punyTmin        = 1
	punyTmax        = 26
	punySkew        = 38
	punyDamp        = 700
	punyInitialBias = 72
	punyInitialN    = 0x80
)

// asciiHost rewrites a host so every label is ASCII, encoding the ones that are not with
// Punycode. A label that is already ASCII is returned unchanged, so this is a no-op for
// the vast majority of hosts.
//
// The SDK carries no dependencies, and Go's standard library exposes no IDNA API, so the
// encoder lives here. This is deliberately not full IDNA2008: there is no mapping,
// normalisation or validity checking. It exists so a host can be shown to a user in a
// form that cannot be confused with a different host, which the label-level ASCII
// encoding is sufficient for.
func asciiHost(host string) string {
	if isASCII(host) {
		return host
	}
	labels := strings.Split(host, ".")
	for i, label := range labels {
		if isASCII(label) {
			continue
		}
		encoded, ok := punyEncode(label)
		if !ok {
			// Leave a label the encoder cannot express alone rather than emit something
			// that looks authoritative and is not.
			continue
		}
		labels[i] = "xn--" + encoded
	}
	return strings.Join(labels, ".")
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// punyEncode implements the RFC 3492 encoding procedure for a single label.
func punyEncode(label string) (string, bool) {
	runes := []rune(label)
	var out strings.Builder
	basic := 0
	for _, r := range runes {
		if r < punyInitialN {
			out.WriteRune(r)
			basic++
		}
	}
	handled := basic
	if basic > 0 {
		out.WriteByte('-')
	}

	n := rune(punyInitialN)
	delta := 0
	bias := punyInitialBias
	for handled < len(runes) {
		m := rune(0x7FFFFFFF)
		for _, r := range runes {
			if r >= n && r < m {
				m = r
			}
		}
		if int(m-n) > (0x7FFFFFFF-delta)/(handled+1) {
			return "", false
		}
		delta += int(m-n) * (handled + 1)
		n = m
		for _, r := range runes {
			if r < n {
				delta++
				if delta < 0 {
					return "", false
				}
			}
			if r != n {
				continue
			}
			q := delta
			for k := punyBase; ; k += punyBase {
				t := k - bias
				switch {
				case t < punyTmin:
					t = punyTmin
				case t > punyTmax:
					t = punyTmax
				}
				if q < t {
					break
				}
				out.WriteByte(punyDigit(t + (q-t)%(punyBase-t)))
				q = (q - t) / (punyBase - t)
			}
			out.WriteByte(punyDigit(q))
			bias = punyAdapt(delta, handled+1, handled == basic)
			delta = 0
			handled++
		}
		delta++
		n++
	}
	return out.String(), true
}

func punyAdapt(delta, numPoints int, firstTime bool) int {
	if firstTime {
		delta /= punyDamp
	} else {
		delta /= 2
	}
	delta += delta / numPoints
	k := 0
	for delta > ((punyBase-punyTmin)*punyTmax)/2 {
		delta /= punyBase - punyTmin
		k += punyBase
	}
	return k + (punyBase-punyTmin+1)*delta/(delta+punySkew)
}

func punyDigit(d int) byte {
	if d < 26 {
		return byte('a' + d)
	}
	return byte('0' + d - 26)
}
