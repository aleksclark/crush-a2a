package a2a

import "encoding/json"

// Standard JSON-RPC error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// A2A-specific error codes.
const (
	CodeTaskNotFound      = -32001
	CodeTaskNotCancelable = -32002
	CodeUnsupported       = -32003
)

// ErrParseError returns a parse error response.
func ErrParseError(id json.RawMessage, detail string) *JSONRPCError {
	return NewJSONRPCError(id, CodeParseError, "Parse error", detail)
}

// ErrInvalidRequest returns an invalid request error.
func ErrInvalidRequest(id json.RawMessage, detail string) *JSONRPCError {
	return NewJSONRPCError(id, CodeInvalidRequest, "Invalid Request", detail)
}

// ErrMethodNotFound returns a method not found error.
func ErrMethodNotFound(id json.RawMessage, method string) *JSONRPCError {
	return NewJSONRPCError(id, CodeMethodNotFound, "Method not found", method)
}

// ErrInvalidParams returns an invalid params error.
func ErrInvalidParams(id json.RawMessage, detail string) *JSONRPCError {
	return NewJSONRPCError(id, CodeInvalidParams, "Invalid params", detail)
}

// ErrInternalError returns an internal error.
func ErrInternalError(id json.RawMessage, detail string) *JSONRPCError {
	return NewJSONRPCError(id, CodeInternalError, "Internal error", detail)
}

// ErrTaskNotFound returns a task not found error.
func ErrTaskNotFound(id json.RawMessage, taskID string) *JSONRPCError {
	return NewJSONRPCError(id, CodeTaskNotFound, "Task not found", taskID)
}

// ErrTaskNotCancelable returns a task not cancelable error.
func ErrTaskNotCancelable(id json.RawMessage, taskID string) *JSONRPCError {
	return NewJSONRPCError(id, CodeTaskNotCancelable, "Task not cancelable", taskID)
}

// ErrUnsupportedOperation returns an unsupported operation error.
func ErrUnsupportedOperation(id json.RawMessage, detail string) *JSONRPCError {
	return NewJSONRPCError(id, CodeUnsupported, "Unsupported operation", detail)
}
