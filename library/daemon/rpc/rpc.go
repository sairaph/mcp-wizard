package rpc

import "encoding/json"

type Request struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const (
	CodeParse    = -32700
	CodeInvalid  = -32600
	CodeMethod   = -32601
	CodeParams   = -32602
	CodeInternal = -32603
	CodeCustom   = -32000
)

func NewRequest(id int64, method string, params any) (Request, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return Request{}, err
	}
	return Request{ID: id, Method: method, Params: raw}, nil
}

func NewResponse(id int64, result any) (Response, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return Response{}, err
	}
	return Response{ID: id, Result: raw}, nil
}

func NewErrorResponse(id int64, code int, message string) Response {
	return Response{ID: id, Error: &Error{Code: code, Message: message}}
}
