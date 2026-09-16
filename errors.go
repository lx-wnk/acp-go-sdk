package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// RequestError represents a JSON-RPC error response.
type RequestError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error renders the error as compact JSON, so a log line carries the code, message and
// data rather than a summary: {"code":-32603,"message":"Internal error","data":{}}.
// A nil receiver renders as "<nil>".
func (e *RequestError) Error() string {
	if e == nil {
		return "<nil>"
	}
	// Try to pretty-print compact JSON for stability in logs.
	type view struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data,omitempty"`
	}
	v := view{Code: e.Code, Message: e.Message, Data: e.Data}
	b, err := json.Marshal(v)
	if err == nil {
		return string(b)
	}
	// Fallback if marshal fails.
	if e.Data != nil {
		return fmt.Sprintf("code %d: %s (data: %v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("code %d: %s", e.Code, e.Message)
}

// NewParseError builds a JSON-RPC -32700 "Parse error", for a payload that is not valid
// JSON at all. data is carried through unchanged.
func NewParseError(data any) *RequestError {
	return &RequestError{Code: -32700, Message: "Parse error", Data: data}
}

// NewInvalidRequest builds a JSON-RPC -32600 "Invalid request", for valid JSON that is
// not a well-formed request object. data is carried through unchanged.
func NewInvalidRequest(data any) *RequestError {
	return &RequestError{Code: -32600, Message: "Invalid request", Data: data}
}

// NewMethodNotFound builds a JSON-RPC -32601 "Method not found" naming the method that
// was asked for. Return it from a handler for a capability this peer does not implement,
// rather than an answer that asserts a decision nobody made.
func NewMethodNotFound(method string) *RequestError {
	return &RequestError{Code: -32601, Message: "Method not found", Data: map[string]any{"method": method}}
}

// NewInvalidParams builds a JSON-RPC -32602 "Invalid params", for a request whose
// parameters are missing or malformed. data is carried through unchanged.
func NewInvalidParams(data any) *RequestError {
	return &RequestError{Code: -32602, Message: "Invalid params", Data: data}
}

// NewInternalError builds a JSON-RPC -32603 "Internal error", for a failure on this side
// that the peer cannot act on. data is carried through unchanged.
func NewInternalError(data any) *RequestError {
	return &RequestError{Code: -32603, Message: "Internal error", Data: data}
}

// NewRequestCancelled builds a -32800 "Request cancelled". The code is reserved by ACP,
// not by JSON-RPC 2.0. toReqErr returns it for a context.Canceled.
func NewRequestCancelled(data any) *RequestError {
	return &RequestError{Code: -32800, Message: "Request cancelled", Data: data}
}

// NewAuthRequired builds a -32000 "Authentication required", from the
// implementation-defined range JSON-RPC reserves. data is carried through unchanged.
func NewAuthRequired(data any) *RequestError {
	return &RequestError{Code: -32000, Message: "Authentication required", Data: data}
}

// toReqErr coerces arbitrary errors into JSON-RPC RequestError.
func toReqErr(err error) *RequestError {
	if err == nil {
		return nil
	}
	if re, ok := err.(*RequestError); ok {
		return re
	}
	if errors.Is(err, context.Canceled) {
		return NewRequestCancelled(map[string]any{"error": err.Error()})
	}
	return NewInternalError(map[string]any{"error": err.Error()})
}
