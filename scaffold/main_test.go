package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubstituteBraced(t *testing.T) {
	subs := map[string]string{"Name": "demo", "Owner": "acme"}
	cases := map[string]string{
		"${Name}":                "demo",
		"${Owner}/${Name}":       "acme/demo",
		"${Unknown}":             "${Unknown}",
		"$Name":                  "$Name",
		"${{ secrets.TOKEN }}":   "${{ secrets.TOKEN }}",
		"${URL%/*}":              "${URL%/*}",
		"${":                     "${",
		"trailing $":             "trailing $",
		"${Name}-${os}-${arch}":  "demo-${os}-${arch}",
		"\"$Bin-$os-$arch\"":     "\"$Bin-$os-$arch\"",
		"${{ x }} then ${Owner}": "${{ x }} then acme",
	}
	for in, want := range cases {
		if got := substituteBraced(in, subs); got != want {
			t.Errorf("substituteBraced(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateInputs(t *testing.T) {
	good := [][2]string{{"myserver", "acme"}, {"my_server", "a-b"}, {"_x", "x"}, {"Srv2", "Org-Name"}}
	for _, c := range good {
		if err := validateInputs(c[0], c[1]); err != nil {
			t.Errorf("validateInputs(%q, %q) unexpected error: %v", c[0], c[1], err)
		}
	}
	bad := [][2]string{{"my-server", "acme"}, {"1abc", "acme"}, {"func", "acme"}, {"ok", "-acme"}, {"ok", "acme-"}, {"ok", "a--b"}, {"ok", ""}, {"", "acme"}}
	for _, c := range bad {
		if err := validateInputs(c[0], c[1]); err == nil {
			t.Errorf("validateInputs(%q, %q) expected error", c[0], c[1])
		}
	}
}

func TestScaffoldWritesTemplatesWithSubstitution(t *testing.T) {
	dir := t.TempDir()
	subs := map[string]string{
		"Name": "demo", "Owner": "Acme", "OwnerLower": "acme",
		"ModulePath": "github.com/acme/demo", "BinaryName": "demo", "ServerName": "demo",
	}
	if err := scaffoldProject(dir, subs); err != nil {
		t.Fatal(err)
	}
	if err := scaffoldScripts(dir, subs); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"main.go", "README.md", ".goreleaser.yml", "install.sh", "install.ps1",
		".github/workflows/ci.yml", ".github/workflows/release.yml",
		"internal/domain/domain.go", "internal/mcpserver/server.go", "internal/mcpserver/config.go",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("expected %s to be generated: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "main.go.tmpl")); !os.IsNotExist(err) {
		t.Error(".tmpl suffix must be stripped")
	}

	main, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if strings.Contains(string(main), "${") {
		t.Errorf("main.go still contains unsubstituted placeholders")
	}
	if !strings.Contains(string(main), `"github.com/acme/demo/internal/domain"`) {
		t.Errorf("main.go should import the generated module path")
	}

	sh, _ := os.ReadFile(filepath.Join(dir, "install.sh"))
	if !strings.Contains(string(sh), `OWNER="Acme"`) || !strings.Contains(string(sh), "${BIN}-${os}-${arch}") {
		t.Errorf("install.sh substitution wrong:\n%s", sh)
	}
	info, _ := os.Stat(filepath.Join(dir, "install.sh"))
	if info.Mode()&0111 == 0 {
		t.Error("install.sh must be executable")
	}

	release, _ := os.ReadFile(filepath.Join(dir, ".github/workflows/release.yml"))
	if !strings.Contains(string(release), "${{ secrets.GITHUB_TOKEN }}") {
		t.Errorf("GitHub Actions expressions must survive substitution:\n%s", release)
	}
	gor, _ := os.ReadFile(filepath.Join(dir, ".goreleaser.yml"))
	if !strings.Contains(string(gor), "install.sh") || !strings.Contains(string(gor), "project_name: demo") {
		t.Errorf("goreleaser config must publish install scripts under the binary name:\n%s", gor)
	}
}

func TestGenerateRefusesNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "existing"), []byte("x"), 0644)
	err := generate(dir, "demo", "acme")
	if err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("expected not-empty error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "existing")); statErr != nil {
		t.Fatal("existing content must be untouched")
	}
}

func TestGenerateCleansUpOnFailure(t *testing.T) {
	// Point go at a module path that cannot be tidied by making the go
	// binary unavailable: PATH without go makes `go mod init` fail after the
	// templates are written, which must remove them again.
	t.Setenv("PATH", t.TempDir())
	parent := t.TempDir()
	target := filepath.Join(parent, "newproj")
	err := generate(target, "demo", "acme")
	if err == nil {
		t.Fatal("expected failure without go on PATH")
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("failed generation must remove the directory it created, stat err=%v", statErr)
	}

	// A pre-existing empty directory is kept but emptied.
	existing := t.TempDir()
	if err := generate(existing, "demo", "acme"); err == nil {
		t.Fatal("expected failure")
	}
	entries, _ := os.ReadDir(existing)
	if len(entries) != 0 {
		t.Fatalf("pre-existing directory must be emptied on failure, has %d entries", len(entries))
	}
}
