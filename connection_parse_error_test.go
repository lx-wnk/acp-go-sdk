package acp

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// JSON-RPC 2.0 §5: when the id of an unreadable message cannot be determined, the
// error response carries an id member that is present and null. Decoding into
// anyMessage cannot tell null from absent, so the raw object is inspected.
func parseErrorReply(t *testing.T, line []byte) (bool, *RequestError) {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		t.Fatalf("reply is not a JSON object: %v", err)
	}
	id, hasID := raw["id"]
	if !hasID || string(id) != "null" {
		return false, nil
	}
	var e RequestError
	if err := json.Unmarshal(raw["error"], &e); err != nil {
		t.Fatalf("decode error member: %v", err)
	}
	return true, &e
}

func startEchoConnection(t *testing.T) (io.Writer, chan []byte) {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	t.Cleanup(func() { _ = inW.Close(); _ = outW.Close(); _ = inR.Close(); _ = outR.Close() })
	NewConnection(func(context.Context, string, json.RawMessage) (any, *RequestError) {
		return map[string]any{"ok": true}, nil
	}, outW, inR)
	return inW, readLines(t, outR)
}

// The reply is sent asynchronously so receive never blocks on a write, which means
// it may arrive before or after the answer to the next request.
func awaitParseErrorAndResult(t *testing.T, lines chan []byte, resultID string) *RequestError {
	t.Helper()
	var parseErr *RequestError
	gotResult := false
	for i := 0; i < 2; i++ {
		line := awaitLine(t, lines)
		if ok, e := parseErrorReply(t, line); ok {
			parseErr = e
			continue
		}
		var res anyMessage
		if err := json.Unmarshal(line, &res); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if res.ID != nil && string(*res.ID) == resultID && res.Error == nil {
			gotResult = true
		}
	}
	if parseErr == nil {
		t.Fatal("no -32700 reply with a null id for the unreadable message")
	}
	if !gotResult {
		t.Fatalf("no result for request %s after the unreadable message", resultID)
	}
	return parseErr
}

func TestConnection_MalformedJSONGetsParseError(t *testing.T) {
	w, lines := startEchoConnection(t)
	go func() {
		_, _ = w.Write([]byte(`{"method": "x", "id": 1, INVALID}` + "\n"))
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"method":"x","params":{}}` + "\n"))
	}()
	if e := awaitParseErrorAndResult(t, lines, "2"); e.Code != -32700 {
		t.Fatalf("code = %d, want -32700", e.Code)
	}
}

func TestConnection_OversizedMessageGetsParseError(t *testing.T) {
	w, lines := startEchoConnection(t)
	go func() {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"x","params":"` + strings.Repeat("A", 12<<20) + `"}` + "\n"))
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"method":"x","params":{}}` + "\n"))
	}()
	e := awaitParseErrorAndResult(t, lines, "2")
	if e.Code != -32700 {
		t.Fatalf("code = %d, want -32700", e.Code)
	}
	if !strings.Contains(e.Error(), "size limit") {
		t.Fatalf("reply does not say why the message was dropped: %s", e.Error())
	}
}
