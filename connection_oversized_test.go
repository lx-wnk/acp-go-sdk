package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

// readMessage replaces bufio.Scanner in receive, which stops for good at its first
// over-long line. It must keep ScanLines' framing — a trailing "\r\n" or "\n" is
// dropped and a final line without a newline is still returned — and skip an
// oversized line instead of ending the stream.
func TestReadMessage(t *testing.T) {
	const max = 16
	r := bufio.NewReaderSize(strings.NewReader(
		"short\n"+
			"crlf\r\n"+
			strings.Repeat("x", 40)+"\n"+
			"after\n"+
			"last"), 16)

	var buf []byte
	want := []struct {
		line string
		err  error
	}{
		{"short", nil},
		{"crlf", nil},
		{"", errMessageTooLarge},
		{"after", nil},
		{"last", nil},
		{"", io.EOF},
	}
	for i, w := range want {
		line, err := readMessage(r, max, buf[:0])
		if !errors.Is(err, w.err) || string(line) != w.line {
			t.Fatalf("read %d = (%q, %v), want (%q, %v)", i, line, err, w.line, w.err)
		}
		buf = line
	}
}

// One message above the 10 MB cap used to end the connection with every request in
// flight. It must be dropped like an unparseable line, and the next message served.
func TestConnection_OversizedMessageIsSkipped(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer func() { _ = inW.Close(); _ = outW.Close(); _ = inR.Close(); _ = outR.Close() }()

	NewConnection(func(context.Context, string, json.RawMessage) (any, *RequestError) {
		return map[string]any{"ok": true}, nil
	}, outW, inR)
	lines := readLines(t, outR)

	go func() {
		huge := `{"jsonrpc":"2.0","id":1,"method":"x","params":{"blob":"` + strings.Repeat("A", 12<<20) + `"}}` + "\n"
		if _, err := inW.Write([]byte(huge)); err != nil {
			return
		}
		_, _ = inW.Write([]byte(`{"jsonrpc":"2.0","id":2,"method":"x","params":{}}` + "\n"))
	}()

	var res anyMessage
	if err := json.Unmarshal(awaitLine(t, lines), &res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Error != nil || res.ID == nil || string(*res.ID) != "2" {
		t.Fatalf("response = %+v, want a result for id 2", res)
	}
}

// A message well above the old 1 MB initial buffer but under the cap must still be
// delivered whole, since the reader now grows on demand instead.
func TestConnection_LargeMessageUnderCapIsDelivered(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer func() { _ = inW.Close(); _ = outW.Close(); _ = inR.Close(); _ = outR.Close() }()

	got := make(chan int, 1)
	NewConnection(func(_ context.Context, _ string, params json.RawMessage) (any, *RequestError) {
		got <- len(params)
		return map[string]any{"ok": true}, nil
	}, outW, inR)
	lines := readLines(t, outR)

	params := `{"blob":"` + strings.Repeat("B", 3<<20) + `"}`
	go func() {
		_, _ = inW.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"x","params":` + params + `}` + "\n"))
	}()

	awaitLine(t, lines)
	if n := <-got; n != len(params) {
		t.Fatalf("handler saw %d bytes of params, want %d", n, len(params))
	}
}
