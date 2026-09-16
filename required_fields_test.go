package acp

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// sessionId is required by the schema but typed allOf:[{$ref: SessionId}], and the
// generator used to emit no check for that shape at all.
func TestValidate_RefTypedRequiredField(t *testing.T) {
	err := (&PromptRequest{Prompt: []ContentBlock{TextBlock("hi")}}).Validate()
	if err == nil || !strings.Contains(err.Error(), "sessionId") {
		t.Fatalf("PromptRequest without sessionId: Validate() = %v, want an error naming sessionId", err)
	}

	err = (&CreateTerminalResponse{}).Validate()
	if err == nil || !strings.Contains(err.Error(), "terminalId") {
		t.Fatalf("CreateTerminalResponse without terminalId: Validate() = %v, want an error naming terminalId", err)
	}

	if err := (&PromptRequest{SessionId: "s", Prompt: []ContentBlock{TextBlock("hi")}}).Validate(); err != nil {
		t.Fatalf("complete PromptRequest rejected: %v", err)
	}
}

// Dispatch validates params before a handler runs. A peer omitting sessionId must get
// -32602 and never reach Prompt with an empty identifier.
func TestDispatch_MissingSessionIdIsInvalidParams(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	defer func() { _ = inW.Close(); _ = outW.Close(); _ = inR.Close(); _ = outR.Close() }()

	reached := make(chan struct{}, 1)
	NewAgentSideConnection(agentFuncs{
		PromptFunc: func(context.Context, PromptRequest) (PromptResponse, error) {
			reached <- struct{}{}
			return PromptResponse{StopReason: StopReasonEndTurn}, nil
		},
	}, outW, inR)
	lines := readLines(t, outR)

	req := `{"jsonrpc":"2.0","id":1,"method":"session/prompt","params":{"prompt":[{"type":"text","text":"hi"}]}}`
	if _, err := inW.Write([]byte(req + "\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	var res anyMessage
	if err := json.Unmarshal(awaitLine(t, lines), &res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Error == nil || res.Error.Code != -32602 {
		t.Fatalf("response = %+v, want error -32602", res)
	}
	select {
	case <-reached:
		t.Fatal("handler ran for a request missing its required sessionId")
	default:
	}
}
