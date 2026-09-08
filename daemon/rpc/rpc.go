// Package rpc defines the JSON-RPC wire types used by daemon/socket.
package rpc

import "encoding/json"

// Request is a JSON-RPC request.
type Request struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC response carrying either Result or Error.
type Response struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Error is a JSON-RPC error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Standard JSON-RPC error codes; CodeCustom starts the server-defined range.
const (
	CodeParse    = -32700
	CodeInvalid  = -32600
	CodeMethod   = -32601
	CodeParams   = -32602
	CodeInternal = -32603
	CodeCustom   = -32000
)

// NewRequest builds a Request with params JSON-encoded.
func NewRequest(id int64, method string, params any) (Request, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return Request{}, err
	}
	return Request{ID: id, Method: method, Params: raw}, nil
}

// NewResponse builds a success Response with result JSON-encoded.
func NewResponse(id int64, result any) (Response, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return Response{}, err
	}
	return Response{ID: id, Result: raw}, nil
}

// NewErrorResponse builds an error Response.
func NewErrorResponse(id int64, code int, message string) Response {
	return Response{ID: id, Error: &Error{Code: code, Message: message}}
}
