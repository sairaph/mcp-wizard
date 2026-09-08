package rpc_test

import (
	"encoding/json"
	"testing"

	"github.com/sairaph/mcp-wizard/daemon/rpc"
)

func TestNewRequest(t *testing.T) {
	req, err := rpc.NewRequest(1, "ping", map[string]string{"msg": "hello"})
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if req.ID != 1 {
		t.Fatalf("expected ID 1, got %d", req.ID)
	}
	if req.Method != "ping" {
		t.Fatalf("expected method ping, got %s", req.Method)
	}
	var params map[string]string
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params["msg"] != "hello" {
		t.Fatalf("expected msg=hello, got %s", params["msg"])
	}
}

func TestNewRequestNilParams(t *testing.T) {
	req, err := rpc.NewRequest(2, "status", nil)
	if err != nil {
		t.Fatalf("NewRequest with nil params failed: %v", err)
	}
	if req.ID != 2 {
		t.Fatalf("expected ID 2, got %d", req.ID)
	}
}

func TestNewResponse(t *testing.T) {
	resp, err := rpc.NewResponse(1, "ok")
	if err != nil {
		t.Fatalf("NewResponse failed: %v", err)
	}
	if resp.ID != 1 {
		t.Fatalf("expected ID 1, got %d", resp.ID)
	}
	var result string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected result ok, got %s", result)
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := rpc.NewErrorResponse(1, rpc.CodeMethod, "unknown method")
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != rpc.CodeMethod {
		t.Fatalf("expected code %d, got %d", rpc.CodeMethod, resp.Error.Code)
	}
	if resp.Error.Message != "unknown method" {
		t.Fatalf("expected message 'unknown method', got %s", resp.Error.Message)
	}
}

func TestResponseJSONSerialization(t *testing.T) {
	resp := rpc.NewErrorResponse(42, rpc.CodeParse, "parse error")
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded rpc.Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ID != 42 {
		t.Fatalf("expected ID 42, got %d", decoded.ID)
	}
	if decoded.Error == nil || decoded.Error.Code != rpc.CodeParse {
		t.Fatal("error not preserved through round-trip")
	}
}

func TestErrorCodesAreCorrect(t *testing.T) {
	tests := []struct {
		name string
		code int
		want int
	}{
		{"CodeParse", rpc.CodeParse, -32700},
		{"CodeInvalid", rpc.CodeInvalid, -32600},
		{"CodeMethod", rpc.CodeMethod, -32601},
		{"CodeParams", rpc.CodeParams, -32602},
		{"CodeInternal", rpc.CodeInternal, -32603},
		{"CodeCustom", rpc.CodeCustom, -32000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, tt.code)
			}
		})
	}
}
