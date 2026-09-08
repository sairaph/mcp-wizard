package async_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/sairaph/mcp-wizard/async"
)

func TestLoadReturnsResult(t *testing.T) {
	cmd := async.Load(func() (string, error) {
		return "hello", nil
	})
	msg := cmd()
	res, ok := msg.(async.Result[string])
	if !ok {
		t.Fatalf("expected async.Result[string], got %T", msg)
	}
	if res.Value != "hello" {
		t.Fatalf("expected value 'hello', got %q", res.Value)
	}
	if res.Err != nil {
		t.Fatalf("expected nil error, got %v", res.Err)
	}
}

func TestLoadReturnsError(t *testing.T) {
	sentinel := errors.New("boom")
	cmd := async.Load(func() (int, error) {
		return 0, sentinel
	})
	msg := cmd()
	res, ok := msg.(async.Result[int])
	if !ok {
		t.Fatalf("expected async.Result[int], got %T", msg)
	}
	if res.Value != 0 {
		t.Fatalf("expected value 0, got %d", res.Value)
	}
	if !errors.Is(res.Err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", res.Err)
	}
}

func TestStartReturnsSequence(t *testing.T) {
	cmd := async.Start(func() (string, error) {
		return "data", nil
	})
	msg := cmd()
	// tea.Sequence returns an unexported sequenceMsg (a slice of tea.Msg)
	// Use reflect to verify it's a slice
	rv := reflect.ValueOf(msg)
	if rv.Kind() != reflect.Slice {
		t.Fatalf("expected slice from Sequence, got %T", msg)
	}
	if rv.Len() < 1 {
		t.Fatal("expected at least one message in sequence")
	}
}

func TestResultCarriesValue(t *testing.T) {
	res := async.Result[float64]{Value: 3.14, Err: nil}
	if res.Value != 3.14 {
		t.Fatalf("expected 3.14, got %f", res.Value)
	}
}

func TestResultCarriesError(t *testing.T) {
	sentinel := errors.New("fail")
	res := async.Result[bool]{Value: false, Err: sentinel}
	if !errors.Is(res.Err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", res.Err)
	}
}
