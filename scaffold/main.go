// Command mcp-wizard scaffolds new MCP server projects.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sairaph/mcp-wizard/app"
	"github.com/sairaph/mcp-wizard/app/form"
	"github.com/sairaph/mcp-wizard/app/menu"
	"github.com/sairaph/mcp-wizard/tui"
)

//go:embed all:templates
var templateFS embed.FS

var validOwner = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]?[A-Za-z0-9])*$|^[A-Za-z0-9]$`)
var validName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

func main() {
	name := flag.String("name", "", "MCP server name (Go identifier)")
	owner := flag.String("owner", "", "GitHub owner (username or org)")
	dir := flag.String("dir", "", "target directory (default: ./<name>)")
	flag.Parse()

	if (*name == "") != (*owner == "") {
		fmt.Fprintf(os.Stderr, "Error: --name and --owner must be provided together\n")
		os.Exit(2)
	}

	if *dir != "" && (*name == "" || *owner == "") {
		fmt.Fprintf(os.Stderr, "Error: --dir requires --name and --owner\n")
		os.Exit(2)
	}

	// Flags provided — run CLI mode.
	if *name != "" && *owner != "" {
		runCLI(*name, *owner, *dir)
		return
	}

	// No flags — show usage or open TUI.
	if tui.IsInteractive() {
		os.Exit(runTUI())
	}
	fmt.Fprintf(os.Stderr, "Usage: mcp-wizard new --name <name> --owner <owner> [--dir <dir>]\n")
	os.Exit(2)
}

// --- CLI mode ---

func runCLI(name, owner, dir string) {
	if !validOwner.MatchString(owner) {
		fmt.Fprintf(os.Stderr, "Error: --owner must match %s\n", validOwner.String())
		os.Exit(2)
	}
	if !validName.MatchString(name) {
		fmt.Fprintf(os.Stderr, "Error: --name must be a valid Go identifier\n")
		os.Exit(2)
	}
	if goKeywords[name] {
		fmt.Fprintf(os.Stderr, "Error: --name %q is a Go keyword and cannot be used\n", name)
		os.Exit(2)
	}

	targetDir := dir
	if targetDir == "" {
		targetDir = "./" + name
	}

	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: go is not installed. Install Go from https://go.dev/dl/\n")
		os.Exit(1)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		os.Exit(1)
	}
	if len(entries) > 0 {
		fmt.Fprintf(os.Stderr, "Error: %s is not empty\n", targetDir)
		os.Exit(1)
	}

	modulePath := fmt.Sprintf("github.com/%s/%s", strings.ToLower(owner), strings.ToLower(name))

	subs := map[string]string{
		"Name":       name,
		"Owner":      owner,
		"OwnerLower": strings.ToLower(owner),
		"ModulePath": modulePath,
		"BinaryName": name,
		"ServerName": name,
	}

	if err := scaffoldProject(targetDir, subs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := scaffoldScripts(targetDir, subs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	initMod := exec.Command("go", "mod", "init", modulePath)
	initMod.Dir = targetDir
	if out, err := initMod.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing Go module: %v\n%s\n", err, out)
		os.Exit(1)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = targetDir
	if out, err := tidy.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error tidying module: %v\n%s\n", err, out)
		os.Exit(1)
	}

	fmt.Printf("Created MCP server project at %s\n", targetDir)
}

// --- TUI mode ---

type tuiState struct {
	app.AppModel
	name         string
	owner        string
	dir          string
	menu         *menu.Model
	form         *form.Model
	done         bool
	runAfterQuit func()
}

func runTUI() int {
	s := &tuiState{}
	s.menu = menu.New("mcp-wizard", func() []menu.Item {
		return []menu.Item{
			{Label: "Create new project", Action: "new"},
			{Label: "Help", Action: "help"},
			{Label: "Quit", Action: "quit"},
		}
	})
	exitCode := app.Run(context.Background(), s, app.Options{Title: "mcp-wizard"})
	if s.runAfterQuit != nil {
		s.runAfterQuit()
	}
	return exitCode
}

func (s *tuiState) Init() tea.Cmd {
	return s.menu.Init()
}

func (s *tuiState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s.HandleGlobalKeys(msg) {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.Step == 99 {
			s.Step = 0
			return s, nil
		}
	case app.ActionMsg:
		switch msg.Source {
		case "menu":
			if msg.Value == "select" {
				action, _ := msg.Data.(string)
				switch action {
				case "new":
					s.Step = 1
					s.form = form.New("New Project", []form.Field{
						{Label: "Project name", Validate: func(v string) error {
							if !validName.MatchString(v) {
								return fmt.Errorf("must be a valid Go identifier (letters, digits, underscores)")
							}
							if goKeywords[v] {
								return fmt.Errorf("%q is a Go keyword", v)
							}
							return nil
						}},
						{Label: "GitHub owner", Validate: func(v string) error {
							if !validOwner.MatchString(v) {
								return fmt.Errorf("must match GitHub username or org pattern")
							}
							return nil
						}},
					})
					return s, s.form.Init()
				case "help":
					s.Step = 99 // help screen
					return s, nil
				case "quit":
					s.Quit = true
					return s, tea.Quit
				}
			}
		case "form":
			switch msg.Value {
			case "submitted":
				vals := s.form.Values()
				s.name = vals["Project name"]
				s.owner = vals["GitHub owner"]
				s.Quit = true
				s.runAfterQuit = func() {
					runCLI(s.name, s.owner, "")
				}
				return s, tea.Quit
			case "cancelled":
				s.Step = 0
				return s, nil
			}
		}
		return s, nil
	}

	if s.Step == 0 && s.menu != nil {
		cmd := s.menu.Update(msg)
		return s, cmd
	}
	if s.Step == 1 && s.form != nil {
		cmd := s.form.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *tuiState) View() string {
	if s.Step == 99 {
		return "mcp-wizard new --name <name> --owner <owner> [--dir <dir>]\n\nPress any key to return."
	}
	if s.Step == 0 && s.menu != nil {
		return s.menu.View()
	}
	if s.Step == 1 && s.form != nil {
		return s.form.View()
	}
	if s.Status != "" {
		return s.Status
	}
	return ""
}

// --- Scaffold functions ---

func scaffoldProject(targetDir string, subs map[string]string) error {
	return fs.WalkDir(templateFS, "templates/project", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("templates/project", path)
		if err != nil {
			return fmt.Errorf("compute relative path: %w", err)
		}
		if rel == "." {
			return nil
		}
		targetPath := filepath.Join(targetDir, rel)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		content := string(data)
		if filepath.Ext(path) == ".tmpl" {
			targetPath = strings.TrimSuffix(targetPath, ".tmpl")
			content = substituteBraced(content, subs)
		}
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", targetPath, err)
		}
		return nil
	})
}

func scaffoldScripts(targetDir string, subs map[string]string) error {
	return fs.WalkDir(templateFS, "templates/scripts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("templates/scripts", path)
		if err != nil {
			return fmt.Errorf("compute relative path: %w", err)
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(targetDir, rel), 0755)
		}
		targetPath := filepath.Join(targetDir, rel)
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		content := substituteBraced(string(data), subs)
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", targetPath, err)
		}
		if filepath.Ext(targetPath) == ".sh" || filepath.Ext(targetPath) == ".ps1" {
			if err := os.Chmod(targetPath, 0755); err != nil {
				return fmt.Errorf("chmod %s: %w", targetPath, err)
			}
		}
		return nil
	})
}

// substituteBraced replaces ${key} patterns in s using subs, leaving
// $var (unbraced) and ${{ ... }} (GitHub Actions / PowerShell) intact.
func substituteBraced(s string, subs map[string]string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '$' && i+2 < len(s) && s[i+1] == '{' {
			if s[i+2] == '{' {
				out.WriteString("${{")
				i += 2
				continue
			}
			end := strings.IndexByte(s[i+2:], '}')
			if end < 0 {
				out.WriteByte(s[i])
				continue
			}
			key := s[i+2 : i+2+end]
			if v, ok := subs[key]; ok {
				out.WriteString(v)
			} else {
				out.WriteString("${" + key + "}")
			}
			i += 2 + end
		} else {
			out.WriteByte(s[i])
		}
	}
	return out.String()
}
