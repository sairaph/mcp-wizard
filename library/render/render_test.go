package render_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sairaph/mcp-wizard/render"
)

// ---- helpers ----

type testError struct {
	msg  string
	code int
}

func (e *testError) Error() string { return e.msg }

type notFoundErr struct{ name string }

func (e *notFoundErr) Error() string { return fmt.Sprintf("%q not found", e.name) }

// ---- Document tests ----

func TestDocumentRendersYamlFrontmatterAndBody(t *testing.T) {
	front := struct {
		Name string `yaml:"name"`
	}{Name: "test"}
	doc := render.Document{Front: front, Body: "hello world"}
	got, err := doc.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "---\n") {
		t.Fatalf("expected yaml frontmatter, got %q", got)
	}
	if !strings.Contains(got, "name: test") {
		t.Fatalf("expected name in frontmatter, got %q", got)
	}
	if !strings.Contains(got, "hello world") {
		t.Fatalf("expected body, got %q", got)
	}
}

func TestDocumentNilFrontRendersOnlyFences(t *testing.T) {
	doc := render.Document{Front: nil, Body: ""}
	got, err := doc.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "---\n---\n" {
		t.Fatalf("expected ---\\n---\\n, got %q", got)
	}
}

func TestDocumentEmptyBodyRendersFrontmatterOnly(t *testing.T) {
	front := struct {
		Key string `yaml:"key"`
	}{Key: "val"}
	doc := render.Document{Front: front, Body: ""}
	got, err := doc.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "---\n") {
		t.Fatalf("expected frontmatter, got %q", got)
	}
	if strings.Count(got, "---") != 2 {
		t.Fatalf("expected exactly two --- fences, got %q", got)
	}
}

func TestDocumentRejectsMapStringAny(t *testing.T) {
	doc := render.Document{Front: map[string]any{"a": 1}, Body: ""}
	_, err := doc.String()
	if err == nil {
		t.Fatal("expected error for map[string]any front")
	}
	if !strings.Contains(err.Error(), "non-deterministic") {
		t.Fatalf("expected non-deterministic error, got %v", err)
	}
}

func TestDocumentExceedsMaxBytesReturnsError(t *testing.T) {
	big := strings.Repeat("x", render.MaxBytes)
	doc := render.Document{Front: nil, Body: big}
	_, err := doc.String()
	if err == nil {
		t.Fatal("expected error for oversized document")
	}
}

// ---- Result tests ----

func TestSuccessResultWrapsDocument(t *testing.T) {
	front := struct {
		ID int `yaml:"id"`
	}{ID: 42}
	res := render.SuccessResult(front, "content")
	if res.IsError {
		t.Fatal("expected IsError=false")
	}
	if len(res.Content) != 1 {
		t.Fatalf("expected 1 content, got %d", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", res.Content[0])
	}
	if !strings.Contains(tc.Text, "id: 42") {
		t.Fatalf("expected id in text, got %q", tc.Text)
	}
	if !strings.Contains(tc.Text, "content") {
		t.Fatalf("expected body in text, got %q", tc.Text)
	}
}

func TestErrorResultProducesIsErrorTrueWithErrorFrontmatter(t *testing.T) {
	e := render.Error{Code: render.CodeNotFound, Message: "item missing", Hint: "try another id"}
	res := render.ErrorResult(e)
	if !res.IsError {
		t.Fatal("expected IsError=true")
	}
	if len(res.Content) != 1 {
		t.Fatalf("expected 1 content, got %d", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", res.Content[0])
	}
	if !strings.Contains(tc.Text, "code: not_found") {
		t.Fatalf("expected error code in text, got %q", tc.Text)
	}
	if !strings.Contains(tc.Text, "item missing") {
		t.Fatalf("expected error message in text, got %q", tc.Text)
	}
	if !strings.Contains(tc.Text, "try another id") {
		t.Fatalf("expected hint in text, got %q", tc.Text)
	}
}

func TestTextResultWrapsRawString(t *testing.T) {
	res := render.TextResult("plain text")
	if res.IsError {
		t.Fatal("expected IsError=false")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", res.Content[0])
	}
	if tc.Text != "plain text" {
		t.Fatalf("expected 'plain text', got %q", tc.Text)
	}
}

// ---- Fence tests ----

func TestFenceGrowsPastContentTildes(t *testing.T) {
	content := "~~~\nhello"
	got := render.Fence(content, "")
	if !strings.Contains(got, "~~~~") {
		t.Fatalf("expected fence longer than content tildes, got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("expected content in fence, got %q", got)
	}
}

func TestFenceWithLanguageInfoString(t *testing.T) {
	got := render.Fence("code", "go")
	if !strings.HasPrefix(got, "~~~go\n") {
		t.Fatalf("expected ~~~go prefix, got %q", got)
	}
	if !strings.Contains(got, "code") {
		t.Fatalf("expected content, got %q", got)
	}
}

func TestScreenEmptyLinesRendersBlankMessage(t *testing.T) {
	got := render.Screen(nil)
	if !strings.Contains(got, "(the screen is blank)") {
		t.Fatalf("expected blank screen message, got %q", got)
	}
}

func TestScreenWithLinesRendersInFence(t *testing.T) {
	lines := []string{"line1", "line2"}
	got := render.Screen(lines)
	if !strings.Contains(got, "line1") || !strings.Contains(got, "line2") {
		t.Fatalf("expected lines in fence, got %q", got)
	}
	if !strings.HasPrefix(got, "~~~text\n") {
		t.Fatalf("expected ~~~text prefix, got %q", got)
	}
}

// ---- ProgressBar tests ----

func TestProgressBarAt0Percent(t *testing.T) {
	got := render.ProgressBar(0, 10, 12)
	if got != "[----------]" {
		t.Fatalf("expected all dashes, got %q", got)
	}
}

func TestProgressBarAt50Percent(t *testing.T) {
	got := render.ProgressBar(5, 10, 12)
	if got != "[#####-----]" {
		t.Fatalf("expected half filled, got %q", got)
	}
}

func TestProgressBarAt100Percent(t *testing.T) {
	got := render.ProgressBar(10, 10, 12)
	if got != "[##########]" {
		t.Fatalf("expected all filled, got %q", got)
	}
}

func TestProgressBarTotalZeroReturnsEmpty(t *testing.T) {
	got := render.ProgressBar(0, 0, 12)
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestProgressBarDoneClampsToTotal(t *testing.T) {
	got := render.ProgressBar(20, 10, 12)
	if got != "[##########]" {
		t.Fatalf("expected clamped to full, got %q", got)
	}
}

// ---- PageMeta tests ----

func TestPageMetaSerialization(t *testing.T) {
	doc := render.Document{
		Front: render.PageMeta{Page: 1, Total: 50, TotalPages: 3},
		Body:  "items",
	}
	got, err := doc.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "page: 1") {
		t.Fatalf("expected page in frontmatter, got %q", got)
	}
	if !strings.Contains(got, "total: 50") {
		t.Fatalf("expected total in frontmatter, got %q", got)
	}
	if !strings.Contains(got, "total_pages: 3") {
		t.Fatalf("expected total_pages in frontmatter, got %q", got)
	}
}

func TestNextPageHintAtFirstPage(t *testing.T) {
	meta := render.PageMeta{Page: 0, Total: 100, TotalPages: 3}
	hint := render.NextPageHint(meta)
	if !strings.Contains(hint, "page=1") {
		t.Fatalf("expected page=1 hint, got %q", hint)
	}
}

func TestNextPageHintAtLastPage(t *testing.T) {
	meta := render.PageMeta{Page: 2, Total: 100, TotalPages: 3}
	hint := render.NextPageHint(meta)
	if hint != "" {
		t.Fatalf("expected empty hint on last page, got %q", hint)
	}
}

func TestNextPageHintSinglePage(t *testing.T) {
	meta := render.PageMeta{Page: 0, Total: 5, TotalPages: 1}
	hint := render.NextPageHint(meta)
	if hint != "" {
		t.Fatalf("expected empty hint for single page, got %q", hint)
	}
}

// ---- Classify tests ----

func TestClassifyMatchesErrorInTable(t *testing.T) {
	table := []render.ErrorMapping{
		{Target: &notFoundErr{}, Code: render.CodeNotFound, Hint: "check the name"},
	}
	err := &notFoundErr{name: "my-resource"}
	e := render.Classify(err, table)
	if e.Code != render.CodeNotFound {
		t.Fatalf("expected code not_found, got %q", e.Code)
	}
	if e.Hint != "check the name" {
		t.Fatalf("expected hint, got %q", e.Hint)
	}
	if !strings.Contains(e.Message, "not found") {
		t.Fatalf("expected message about not found, got %q", e.Message)
	}
}

func TestClassifyUnmatchedErrorFallsBackToCodeInternal(t *testing.T) {
	table := []render.ErrorMapping{
		{Target: &notFoundErr{}, Code: render.CodeNotFound},
	}
	err := errors.New("something broke")
	e := render.Classify(err, table)
	if e.Code != render.CodeInternal {
		t.Fatalf("expected code internal_error, got %q", e.Code)
	}
	if !strings.Contains(e.Message, "something broke") {
		t.Fatalf("expected original message, got %q", e.Message)
	}
}

func TestClassifyWithExtractFunction(t *testing.T) {
	table := []render.ErrorMapping{
		{
			Target: &testError{},
			Code:   render.CodeInvalidInput,
			Hint:   "fix your input",
			Extract: func(err error) (string, map[string]any) {
				te, ok := err.(*testError)
				if !ok {
					return "", nil
				}
				return te.msg, map[string]any{"code": te.code}
			},
		},
	}
	err := &testError{msg: "bad value", code: 400}
	e := render.Classify(err, table)
	if e.Code != render.CodeInvalidInput {
		t.Fatalf("expected code invalid_input, got %q", e.Code)
	}
	if e.Message != "bad value" {
		t.Fatalf("expected extracted message, got %q", e.Message)
	}
	if e.Fields["code"] != 400 {
		t.Fatalf("expected fields.code=400, got %v", e.Fields["code"])
	}
}

func TestClassifyNilReturnsEmptyError(t *testing.T) {
	e := render.Classify(nil, nil)
	if e.Code != "" || e.Message != "" {
		t.Fatalf("expected empty Error for nil input, got %+v", e)
	}
}

func TestErrorCodeConstants(t *testing.T) {
	cases := map[string]string{
		render.CodeNotFound:     "not_found",
		render.CodeInvalidInput: "invalid_input",
		render.CodeAuth:         "authentication",
		render.CodeForbidden:    "forbidden",
		render.CodeRateLimited:  "rate_limited",
		render.CodeAmbiguous:    "ambiguous",
		render.CodeConflict:     "conflict",
		render.CodeUnavailable:  "unavailable",
		render.CodeInternal:     "internal_error",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("code constant = %q, want %q", got, want)
		}
	}
}
