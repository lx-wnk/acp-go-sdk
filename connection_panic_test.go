package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// readLines drains the connection's output into a channel so a test can await
// individual responses without blocking the pipe.
func readLines(t *testing.T, r io.Reader) chan []byte {
	t.Helper()
	lines := make(chan []byte, 16)
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lines <- append([]byte(nil), scanner.Bytes()...)
		}
		close(lines)
	}()
	return lines
}

func awaitLine(t *testing.T, lines chan []byte) []byte {
	t.Helper()
	select {
	case b, ok := <-lines:
		if !ok {
			t.Fatal("connection closed before a response arrived")
		}
		return b
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for a response")
		return nil
	}
}

// A panic in a handler must not escape the connection. Go terminates the whole
// process on an unrecovered panic in any goroutine, so without containment one
// faulty handler ends every session multiplexed on this connection.
func TestHandlerPanic_RequestGetsInternalError(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer func() { _ = inW.Close(); _ = outW.Close(); _ = inR.Close(); _ = outR.Close() }()

	NewConnection(func(_ context.Context, method string, _ json.RawMessage) (any, *RequestError) {
		if method == "boom" {
			panic("simulated handler defect")
		}
		return map[string]any{"ok": true}, nil
	}, outW, inR)

	lines := readLines(t, outR)

	if _, err := inW.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"boom","params":{}}` + "\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	var res anyMessage
	if err := json.Unmarshal(awaitLine(t, lines), &res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Error == nil {
		t.Fatalf("panicking handler produced no error response: %+v", res)
	}
	if res.Error.Code != -32603 {
		t.Fatalf("code = %d, want -32603", res.Error.Code)
	}
	// The peer is not necessarily trusted, so the panic value stays in the log.
	if strings.Contains(res.Error.Error(), "simulated handler defect") {
		t.Fatalf("panic detail leaked to the peer: %s", res.Error.Error())
	}

	// The connection must still serve the next request.
	if _, err := inW.Write([]byte(`{"jsonrpc":"2.0","id":2,"method":"fine","params":{}}` + "\n")); err != nil {
		t.Fatalf("write second request: %v", err)
	}
	var res2 anyMessage
	if err := json.Unmarshal(awaitLine(t, lines), &res2); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if res2.Error != nil {
		t.Fatalf("connection unusable after a handler panic: %+v", res2.Error)
	}
}

// A notification has no response to carry the failure, so the panic must be
// contained and logged without stopping the serial notification worker.
func TestHandlerPanic_NotificationKeepsConnectionAlive(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer func() { _ = inW.Close(); _ = outW.Close(); _ = inR.Close(); _ = outR.Close() }()

	NewConnection(func(_ context.Context, method string, _ json.RawMessage) (any, *RequestError) {
		if method == "boom" {
			panic("simulated notification defect")
		}
		return map[string]any{"ok": true}, nil
	}, outW, inR)

	lines := readLines(t, outR)

	if _, err := inW.Write([]byte(`{"jsonrpc":"2.0","method":"boom","params":{}}` + "\n")); err != nil {
		t.Fatalf("write notification: %v", err)
	}
	if _, err := inW.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"fine","params":{}}` + "\n")); err != nil {
		t.Fatalf("write request: %v", err)
	}

	var res anyMessage
	if err := json.Unmarshal(awaitLine(t, lines), &res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("connection unusable after a notification handler panic: %+v", res.Error)
	}
}
