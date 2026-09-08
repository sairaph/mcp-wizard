package proxy

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

type recordingRT struct{ req *http.Request }

func (r *recordingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	r.req = req
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
}

func post(t *testing.T, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "http://remote/mcp", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func TestInjectorStandardHeadersOnlyForNewProtocols(t *testing.T) {
	rec := &recordingRT{}
	inj := &injector{headers: map[string]string{"Authorization": "Bearer x"}, base: rec}
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"greet","arguments":{}}}`

	inj.setProtocolVersion("2025-11-25")
	if _, err := inj.RoundTrip(post(t, body)); err != nil {
		t.Fatal(err)
	}
	if rec.req.Header.Get("Mcp-Method") != "" {
		t.Fatal("Mcp-Method must not be sent for protocol versions before 2026-06-30")
	}
	if rec.req.Header.Get("MCP-Protocol-Version") != "2025-11-25" || rec.req.Header.Get("Authorization") != "Bearer x" {
		t.Fatalf("headers = %v", rec.req.Header)
	}

	inj.setProtocolVersion("2026-06-30")
	if _, err := inj.RoundTrip(post(t, body)); err != nil {
		t.Fatal(err)
	}
	if rec.req.Header.Get("Mcp-Method") != "tools/call" || rec.req.Header.Get("Mcp-Name") != "greet" {
		t.Fatalf("standard headers missing: %v", rec.req.Header)
	}
	// The body must still be readable by the base transport.
	got, _ := io.ReadAll(rec.req.Body)
	if string(got) != body {
		t.Fatalf("body consumed: %q", got)
	}

	cases := map[string][2]string{
		`{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"file:///x"}}`: {"resources/read", "file:///x"},
		`{"jsonrpc":"2.0","id":3,"method":"prompts/get","params":{"name":"p"}}`:           {"prompts/get", "p"},
		`{"jsonrpc":"2.0","id":4,"method":"tools/list"}`:                                  {"tools/list", ""},
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`:                          {"notifications/initialized", ""},
	}
	for b, want := range cases {
		inj.RoundTrip(post(t, b))
		if rec.req.Header.Get("Mcp-Method") != want[0] || rec.req.Header.Get("Mcp-Name") != want[1] {
			t.Fatalf("%s: got method=%q name=%q", b, rec.req.Header.Get("Mcp-Method"), rec.req.Header.Get("Mcp-Name"))
		}
	}
}

func TestInjectorDoesNotOverrideExplicitMethodHeader(t *testing.T) {
	rec := &recordingRT{}
	inj := &injector{base: rec}
	inj.setProtocolVersion("2026-06-30")
	req := post(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	req.Header.Set("Mcp-Method", "tools/list")
	req.Header.Set("Mcp-Name", "already")
	inj.RoundTrip(req)
	if rec.req.Header.Get("Mcp-Name") != "already" {
		t.Fatal("explicit headers set by the SDK must be preserved")
	}
}
