package cli_test

import (
	"errors"
	"testing"

	"github.com/sairaph/mcp-wizard/cli"
)

func TestParseBare(t *testing.T) {
	cmd, err := cli.Parse(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "mcp" {
		t.Fatalf("expected name=mcp, got %q", cmd.Name)
	}

	cmd, err = cli.Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "mcp" {
		t.Fatalf("expected name=mcp, got %q", cmd.Name)
	}
}

func TestParseGlobalHelp(t *testing.T) {
	for _, a := range []string{"--help", "-h"} {
		cmd, err := cli.Parse([]string{a})
		if !errors.Is(err, cli.ErrUsage) {
			t.Errorf("args=%q: expected ErrUsage, got %v", a, err)
		}
		if cmd.Name != "help" {
			t.Errorf("args=%q: expected name=help, got %q", a, cmd.Name)
		}
	}
}

func TestParseGlobalVersion(t *testing.T) {
	for _, a := range []string{"--version", "-v"} {
		cmd, err := cli.Parse([]string{a})
		if err != nil {
			t.Fatalf("args=%q: unexpected error: %v", a, err)
		}
		if cmd.Name != "version" {
			t.Errorf("args=%q: expected name=version, got %q", a, cmd.Name)
		}
	}
}

func TestParseHelpSubcommand(t *testing.T) {
	cmd, err := cli.Parse([]string{"help"})
	if !errors.Is(err, cli.ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}
	if cmd.Name != "help" {
		t.Errorf("expected name=help, got %q", cmd.Name)
	}
}

func TestParseVersionSubcommand(t *testing.T) {
	cmd, err := cli.Parse([]string{"version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "version" {
		t.Errorf("expected name=version, got %q", cmd.Name)
	}
}

func TestParseInstall(t *testing.T) {
	cmd, err := cli.Parse([]string{"install", "--yes", "--all", "--email", "a@b.com", "--token", "sekret", "--name", "my-srv", "--clients", "claude, windsurf", "extra"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "install" {
		t.Errorf("expected name=install, got %q", cmd.Name)
	}
	if !cmd.Yes {
		t.Error("expected Yes=true")
	}
	if !cmd.All {
		t.Error("expected All=true")
	}
	if cmd.ServerName != "my-srv" {
		t.Errorf("expected ServerName=my-srv, got %q", cmd.ServerName)
	}
	if cmd.Credentials["email"] != "a@b.com" {
		t.Errorf("expected email=a@b.com, got %q", cmd.Credentials["email"])
	}
	if cmd.Credentials["token"] != "sekret" {
		t.Errorf("expected token=sekret, got %q", cmd.Credentials["token"])
	}
	if len(cmd.Clients) != 2 || cmd.Clients[0] != "claude" || cmd.Clients[1] != "windsurf" {
		t.Errorf("expected Clients=[claude, windsurf], got %v", cmd.Clients)
	}
	if len(cmd.Args) != 1 || cmd.Args[0] != "extra" {
		t.Errorf("expected Args=[extra], got %v", cmd.Args)
	}
}

func TestParseInstallDryRun(t *testing.T) {
	cmd, err := cli.Parse([]string{"configure", "--dry-run"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "configure" {
		t.Errorf("expected name=configure, got %q", cmd.Name)
	}
	if !cmd.DryRun {
		t.Error("expected DryRun=true")
	}
}

func TestParseUninstall(t *testing.T) {
	cmd, err := cli.Parse([]string{"uninstall", "--yes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "uninstall" {
		t.Errorf("expected name=uninstall, got %q", cmd.Name)
	}
	if !cmd.Yes {
		t.Error("expected Yes=true")
	}
}

func TestParseInstallHelp(t *testing.T) {
	cmd, err := cli.Parse([]string{"install", "--help"})
	if !errors.Is(err, cli.ErrUsage) {
		t.Errorf("expected ErrUsage, got %v", err)
	}
	if cmd.Name != "install" {
		t.Errorf("expected name=install, got %q", cmd.Name)
	}
}

func TestParseMCP(t *testing.T) {
	cmd, err := cli.Parse([]string{"mcp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "mcp" {
		t.Errorf("expected name=mcp, got %q", cmd.Name)
	}
}

func TestParseServerAliases(t *testing.T) {
	for _, a := range []string{"server", "serve"} {
		cmd, err := cli.Parse([]string{a})
		if err != nil {
			t.Fatalf("args=%q: unexpected error: %v", a, err)
		}
		if cmd.Name != "mcp" {
			t.Errorf("args=%q: expected name=mcp, got %q", a, cmd.Name)
		}
	}
}

func TestParseLogin(t *testing.T) {
	cmd, err := cli.Parse([]string{"login", "--email", "u@example.com", "--token", "tkn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "login" {
		t.Errorf("expected name=login, got %q", cmd.Name)
	}
	if cmd.Credentials["email"] != "u@example.com" {
		t.Errorf("expected email=u@example.com, got %q", cmd.Credentials["email"])
	}
	if cmd.Credentials["token"] != "tkn" {
		t.Errorf("expected token=tkn, got %q", cmd.Credentials["token"])
	}
}

func TestParseLoginNoFlags(t *testing.T) {
	cmd, err := cli.Parse([]string{"login"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "login" {
		t.Errorf("expected name=login, got %q", cmd.Name)
	}
	if len(cmd.Credentials) != 0 {
		t.Errorf("expected empty Credentials, got %v", cmd.Credentials)
	}
}

func TestParseDoctor(t *testing.T) {
	cmd, err := cli.Parse([]string{"doctor"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "doctor" {
		t.Errorf("expected name=doctor, got %q", cmd.Name)
	}
}

func TestParseUpdate(t *testing.T) {
	cmd, err := cli.Parse([]string{"update", "--from", "/tmp/new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "update" {
		t.Errorf("expected name=update, got %q", cmd.Name)
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "--from" || cmd.Args[1] != "/tmp/new" {
		t.Errorf("expected Args=[--from, /tmp/new], got %v", cmd.Args)
	}
}

func TestParseUpdateNoFlags(t *testing.T) {
	cmd, err := cli.Parse([]string{"update"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "update" {
		t.Errorf("expected name=update, got %q", cmd.Name)
	}
}

func TestParseUnknownSubcommand(t *testing.T) {
	cmd, err := cli.Parse([]string{"my-cmd", "arg1", "--flag", "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Name != "my-cmd" {
		t.Errorf("expected name=my-cmd, got %q", cmd.Name)
	}
	if len(cmd.Args) != 3 || cmd.Args[0] != "arg1" || cmd.Args[1] != "--flag" || cmd.Args[2] != "val" {
		t.Errorf("expected Args=[arg1, --flag, val], got %v", cmd.Args)
	}
}

func TestParseClientsCommaSeparated(t *testing.T) {
	cmd, err := cli.Parse([]string{"install", "--clients", "claude , windsurf, cursor "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmd.Clients) != 3 {
		t.Fatalf("expected 3 clients, got %v", cmd.Clients)
	}
	if cmd.Clients[0] != "claude" {
		t.Errorf("expected clients[0]=claude, got %q", cmd.Clients[0])
	}
	if cmd.Clients[1] != "windsurf" {
		t.Errorf("expected clients[1]=windsurf, got %q", cmd.Clients[1])
	}
	if cmd.Clients[2] != "cursor" {
		t.Errorf("expected clients[2]=cursor, got %q", cmd.Clients[2])
	}
}

func TestParseEmptyCredentials(t *testing.T) {
	cmd, err := cli.Parse([]string{"install"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Credentials == nil {
		t.Fatal("expected non-nil Credentials map")
	}
	if len(cmd.Credentials) != 0 {
		t.Errorf("expected empty Credentials, got %v", cmd.Credentials)
	}
}
