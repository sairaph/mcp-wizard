// Command mcp-wizard scaffolds new MCP server projects.
//
// Usage:
//
//	mcp-wizard [new] --name <name> --owner <owner> [--dir <dir>]
//	mcp-wizard              (interactive, when run from a terminal)
package main

import (
	"context"
	"embed"
	"errors"
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

var version = "dev"

var validOwner = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]?[A-Za-z0-9])*$|^[A-Za-z0-9]$`)
var validName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// libraryModule is the module path generated projects import.
const libraryModule = "github.com/sairaph/mcp-wizard"

const usageText = "Usage: mcp-wizard [new] --name <name> --owner <owner> [--dir <dir>]\n" +
	"       mcp-wizard version\n\n" +
	"Run without arguments in a terminal for the interactive wizard.\n" +
	"Set MCP_WIZARD_LIBRARY=<path> to build the project against a local checkout of the library.\n"

func main() {
	args := os.Args[1:]
	// "new" is the documented subcommand; it is optional because the flags
	// alone identify the request.
	if len(args) > 0 {
		switch args[0] {
		case "new":
			args = args[1:]
		case "version", "--version", "-v":
			fmt.Println(version)
			return
		case "help", "--help", "-h":
			fmt.Print(usageText)
			return
		}
	}

	fs := flag.NewFlagSet("mcp-wizard", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	name := fs.String("name", "", "MCP server name (Go identifier)")
	owner := fs.String("owner", "", "GitHub owner (username or org)")
	dir := fs.String("dir", "", "target directory (default: ./<name>)")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, usageText, "\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(2)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "Error: unexpected argument %q\n\n%s", fs.Arg(0), usageText)
		os.Exit(2)
	}

	if (*name == "") != (*owner == "") {
		fmt.Fprintf(os.Stderr, "Error: --name and --owner must be provided together\n\n%s", usageText)
		os.Exit(2)
	}
	if *dir != "" && *name == "" {
		fmt.Fprintf(os.Stderr, "Error: --dir requires --name and --owner\n\n%s", usageText)
		os.Exit(2)
	}

	// Flags provided - run CLI mode.
	if *name != "" {
		os.Exit(runCLI(*name, *owner, *dir))
	}

	// No flags - open the TUI or show usage.
	if tui.IsInteractive() {
		os.Exit(runTUI())
	}
	fmt.Fprint(os.Stderr, usageText)
	os.Exit(2)
}

// --- CLI mode ---

// runCLI validates the inputs and generates the project, printing any error.
func runCLI(name, owner, dir string) int {
	if err := validateInputs(name, owner); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: go is not installed or not on PATH. Install Go from https://go.dev/dl/\n")
		return 1
	}
	targetDir := dir
	if targetDir == "" {
		targetDir = "./" + name
	}
	if err := generate(targetDir, name, owner); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("Created MCP server project at %s\n", targetDir)
	return 0
}

func validateInputs(name, owner string) error {
	if !validOwner.MatchString(owner) {
		return fmt.Errorf("--owner must be a GitHub username or organisation (letters, digits, single hyphens), got %q", owner)
	}
	if !validName.MatchString(name) {
		return fmt.Errorf("--name must be a valid Go identifier (letters, digits, underscores), got %q", name)
	}
	if goKeywords[name] {
		return fmt.Errorf("--name %q is a Go keyword and cannot be used", name)
	}
	return nil
}

// generate writes the project into targetDir and initialises its Go module.
// targetDir must not exist or must be empty. If any step fails, everything
// generate created is removed again so a retry starts clean.
func generate(targetDir, name, owner string) (err error) {
	existed := true
	if _, statErr := os.Stat(targetDir); statErr != nil {
		if !os.IsNotExist(statErr) {
			return fmt.Errorf("inspect %s: %w", targetDir, statErr)
		}
		existed = false
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s is not empty", targetDir)
	}

	defer func() {
		if err == nil {
			return
		}
		if existed {
			// Leave the directory itself, remove what we put in it.
			if entries, readErr := os.ReadDir(targetDir); readErr == nil {
				for _, e := range entries {
					os.RemoveAll(filepath.Join(targetDir, e.Name()))
				}
			}
			return
		}
		os.RemoveAll(targetDir)
	}()

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
		return err
	}
	if err := scaffoldScripts(targetDir, subs); err != nil {
		return err
	}

	initMod := exec.Command("go", "mod", "init", modulePath)
	initMod.Dir = targetDir
	if out, runErr := initMod.CombinedOutput(); runErr != nil {
		return fmt.Errorf("go mod init: %v\n%s", runErr, out)
	}

	// MCP_WIZARD_LIBRARY points the generated module at a local checkout of
	// the library instead of the published module (used by CI and when
	// developing the library itself).
	if lib := os.Getenv("MCP_WIZARD_LIBRARY"); lib != "" {
		abs, absErr := filepath.Abs(lib)
		if absErr != nil {
			return fmt.Errorf("MCP_WIZARD_LIBRARY: %w", absErr)
		}
		replace := exec.Command("go", "mod", "edit", "-replace", libraryModule+"="+abs)
		replace.Dir = targetDir
		if out, runErr := replace.CombinedOutput(); runErr != nil {
			return fmt.Errorf("go mod edit -replace: %v\n%s", runErr, out)
		}
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = targetDir
	if out, runErr := tidy.CombinedOutput(); runErr != nil {
		return fmt.Errorf("go mod tidy (network access to fetch dependencies is required): %v\n%s", runErr, out)
	}
	return nil
}

// --- TUI mode ---

const (
	stepMenu app.Step = iota
	stepForm
	stepHelp
)

type tuiState struct {
	app.AppModel
	name         string
	owner        string
	dir          string
	menu         *menu.Model
	form         *form.Model
	runAfterQuit func() int
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
	exitCode := app.Run(context.Background(), s, app.Options{Title: "mcp-wizard", Version: version})
	if exitCode == 0 && s.runAfterQuit != nil {
		// Generation runs after the TUI has released the terminal so its
		// output (and go's) is visible.
		return s.runAfterQuit()
	}
	return exitCode
}

func (s *tuiState) Init() tea.Cmd {
	return s.menu.Init()
}

func (s *tuiState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if handled, cmd := s.HandleGlobalKeys(msg); handled {
		return s, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.Step == stepHelp {
			s.Step = stepMenu
			return s, nil
		}
	case app.ActionMsg:
		switch msg.Source {
		case "menu":
			switch msg.Value {
			case "quit":
				s.Quit = true
				return s, tea.Quit
			case "select":
				action, _ := msg.Data.(string)
				switch action {
				case "new":
					s.Step = stepForm
					s.form = newProjectForm()
					return s, s.form.Init()
				case "help":
					s.Step = stepHelp
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
				s.dir = strings.TrimSpace(vals["Directory (optional)"])
				s.Quit = true
				s.runAfterQuit = func() int {
					return runCLI(s.name, s.owner, s.dir)
				}
				return s, tea.Quit
			case "cancelled":
				s.Step = stepMenu
				return s, nil
			}
		}
		return s, nil
	}

	switch s.Step {
	case stepMenu:
		return s, s.menu.Update(msg)
	case stepForm:
		if s.form != nil {
			return s, s.form.Update(msg)
		}
	}
	return s, nil
}

func newProjectForm() *form.Model {
	return form.New("New Project", []form.Field{
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
		{Label: "Directory (optional)"},
	})
}

func (s *tuiState) View() string {
	switch s.Step {
	case stepHelp:
		return "\n  " + strings.ReplaceAll(usageText, "\n", "\n  ") + "\n  Press any key to return."
	case stepMenu:
		return s.menu.View()
	case stepForm:
		if s.form != nil {
			return s.form.View()
		}
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
// $var (unbraced), unknown ${other} keys, and ${{ ... }} (GitHub Actions /
// PowerShell) intact.
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
