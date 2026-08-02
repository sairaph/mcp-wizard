package render

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TextResult wraps a pre-rendered string into an MCP CallToolResult.
func TextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// SuccessResult wraps a Document into an MCP CallToolResult.
func SuccessResult(front any, body string) *mcp.CallToolResult {
	doc := Document{Front: front, Body: body}
	text, err := doc.String()
	if err != nil {
		return errorResult(Error{Code: CodeInternal, Message: "could not render result: " + err.Error()})
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// ErrorResult wraps an Error into an MCP CallToolResult with IsError=true.
func ErrorResult(e Error) *mcp.CallToolResult {
	return errorResult(e)
}

func errorResult(e Error) *mcp.CallToolResult {
	front := struct {
		Error Error `yaml:"error"`
	}{Error: e}

	body := "## Error\n\n" + e.Message
	if e.Hint != "" {
		body += "\n\n" + e.Hint
	}

	doc := Document{Front: front, Body: body, IsError: true}
	text, err := doc.String()
	if err != nil {
		text = fmt.Sprintf("---\nerror:\n  code: %s\n  message: %s\n---\n## Error\n\n%s\n", e.Code, e.Message, e.Message)
	}

	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}
