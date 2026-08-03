package command_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/sairaph/mcp-wizard/command"
)

func TestRegisterAndDispatch(t *testing.T) {
	r := command.New()
	var executed bool
	r.Register(command.Handler{
		Name: "hello",
		Run: func(_ context.Context, _ []string) int {
			executed = true
			return 0
		},
	})
	handled, code := r.Dispatch(context.Background(), "hello", nil)
	if !handled {
		t.Fatal("expected handled=true")
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !executed {
		t.Fatal("handler was not executed")
	}
}

func TestDispatchUnknown(t *testing.T) {
	r := command.New()
	handled, code := r.Dispatch(context.Background(), "nonexistent", nil)
	if handled {
		t.Fatal("expected handled=false for unknown command")
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestAliasesResolve(t *testing.T) {
	r := command.New()
	var calledBy string
	r.Register(command.Handler{
		Name:    "list",
		Aliases: []string{"ls", "show"},
		Run: func(_ context.Context, args []string) int {
			calledBy = args[0]
			return 0
		},
	})
	for _, name := range []string{"list", "ls", "show"} {
		handled, code := r.Dispatch(context.Background(), name, []string{name})
		if !handled {
			t.Fatalf("expected %q to be handled", name)
		}
		if code != 0 {
			t.Fatalf("expected exit code 0 for %q", name)
		}
		if calledBy != name {
			t.Fatalf("expected calledBy=%q, got %q", name, calledBy)
		}
	}
}

func TestDuplicateNamePanics(t *testing.T) {
	r := command.New()
	r.Register(command.Handler{Name: "foo", Run: func(_ context.Context, _ []string) int { return 0 }})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for duplicate name")
		}
	}()
	r.Register(command.Handler{Name: "foo", Run: func(_ context.Context, _ []string) int { return 0 }})
}

func TestDuplicateAliasPanics(t *testing.T) {
	r := command.New()
	r.Register(command.Handler{Name: "first", Run: func(_ context.Context, _ []string) int { return 0 }})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for duplicate alias")
		}
	}()
	r.Register(command.Handler{Name: "second", Aliases: []string{"first"}, Run: func(_ context.Context, _ []string) int { return 0 }})
}

func TestEmptyNamePanics(t *testing.T) {
	r := command.New()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for empty name")
		}
	}()
	r.Register(command.Handler{Name: "", Run: func(_ context.Context, _ []string) int { return 0 }})
}

func TestNilRunPanics(t *testing.T) {
	r := command.New()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for nil Run")
		}
	}()
	r.Register(command.Handler{Name: "noop"})
}

func TestListUniqueSorted(t *testing.T) {
	r := command.New()
	r.Register(command.Handler{Name: "zebra", Run: func(_ context.Context, _ []string) int { return 0 }})
	r.Register(command.Handler{Name: "alpha", Aliases: []string{"aardvark"}, Run: func(_ context.Context, _ []string) int { return 0 }})
	r.Register(command.Handler{Name: "beta", Run: func(_ context.Context, _ []string) int { return 0 }})
	list := r.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 unique handlers, got %d", len(list))
	}
	if list[0].Name != "alpha" {
		t.Fatalf("expected first to be alpha, got %s", list[0].Name)
	}
	if list[1].Name != "beta" {
		t.Fatalf("expected second to be beta, got %s", list[1].Name)
	}
	if list[2].Name != "zebra" {
		t.Fatalf("expected third to be zebra, got %s", list[2].Name)
	}
}

func TestPrintUsage(t *testing.T) {
	r := command.New()
	r.Register(command.Handler{
		Name:        "list-boards",
		Description: "List all boards",
		Run:         func(_ context.Context, _ []string) int { return 0 },
	})
	r.Register(command.Handler{
		Name:        "get-card",
		Description: "Get a card by ID",
		Run:         func(_ context.Context, _ []string) int { return 0 },
	})
	var buf bytes.Buffer
	r.PrintUsage(&buf)
	out := buf.String()
	if !strings.Contains(out, "Commands:") {
		t.Fatal("missing Commands: header")
	}
	if !strings.Contains(out, "list-boards") || !strings.Contains(out, "List all boards") {
		t.Fatal("missing list-boards entry")
	}
	if !strings.Contains(out, "get-card") || !strings.Contains(out, "Get a card by ID") {
		t.Fatal("missing get-card entry")
	}
}

func TestPrintUsageEmpty(t *testing.T) {
	r := command.New()
	var buf bytes.Buffer
	r.PrintUsage(&buf)
	if buf.Len() != 0 {
		t.Fatalf("expected empty output, got %q", buf.String())
	}
}

func TestIsOneShot(t *testing.T) {
	r := command.New()
	r.Register(command.Handler{
		Name:    "deploy",
		Aliases: []string{"dpl"},
		Run:     func(_ context.Context, _ []string) int { return 0 },
	})
	if !r.IsOneShot("deploy") {
		t.Error("IsOneShot('deploy') should be true")
	}
	if !r.IsOneShot("dpl") {
		t.Error("IsOneShot('dpl') should be true")
	}
	if r.IsOneShot("unknown") {
		t.Error("IsOneShot('unknown') should be false")
	}
}

func TestDispatchPassesContextAndArgs(t *testing.T) {
	r := command.New()
	type key string
	r.Register(command.Handler{
		Name: "echo",
		Run: func(ctx context.Context, args []string) int {
			if v := ctx.Value(key("user")); v != "alice" {
				t.Errorf("expected ctx value 'alice', got %v", v)
			}
			if len(args) != 2 || args[0] != "hello" || args[1] != "world" {
				t.Errorf("expected args [hello world], got %v", args)
			}
			return 1
		},
	})
	ctx := context.WithValue(context.Background(), key("user"), "alice")
	handled, code := r.Dispatch(ctx, "echo", []string{"hello", "world"})
	if !handled {
		t.Fatal("expected handled=true")
	}
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}
