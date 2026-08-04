package cli

import (
	"errors"
	"flag"
	"strings"
)

// ErrUsage is returned when the user asks for help or provides an unknown flag.
var ErrUsage = errors.New("usage")

// Command carries the parsed subcommand and its flags.
type Command struct {
	Name    string            // subcommand name, "help", or "version"
	Args    []string          // remaining positional args after subcommand + flags
	Clients []string          // --clients (for install/configure)
	All     bool              // --all
	Yes     bool              // --yes
	DryRun  bool              // --dry-run
	ServerName string         // --name

	// Credentials carries --email, --token, etc. for unattended login.
	Credentials map[string]string
}

// Parse reads args (typically os.Args[1:]) and returns a Command.
//
// Standard subcommands: install, configure, uninstall, mcp, doctor, help, version.
// Unknown subcommands return Command{Name: args[0], Args: args[1:]} so the
// consumer's main.go switch can handle project-specific commands.
//
// --help/-h and --version/-v without a subcommand return help/version.
func Parse(args []string) (Command, error) {
	var cmd Command
	cmd.Credentials = make(map[string]string)

	if len(args) == 0 {
		cmd.Name = "mcp"
		return cmd, nil
	}

	// Global --help and --version.
	switch args[0] {
	case "--help", "-h":
		cmd.Name = "help"
		return cmd, ErrUsage
	case "--version", "-v":
		cmd.Name = "version"
		return cmd, nil
	}

	cmd.Name = args[0]
	tail := args[1:]

	switch cmd.Name {
	case "help":
		cmd.Name = "help"
		cmd.Args = tail
		return cmd, ErrUsage

	case "version":
		cmd.Name = "version"
		cmd.Args = tail
		return cmd, nil

	case "install", "configure", "uninstall":
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.Usage = func() {}
		fs.BoolVar(&cmd.All, "all", false, "register with all known clients")
		fs.BoolVar(&cmd.Yes, "yes", false, "non-interactive mode")
		fs.BoolVar(&cmd.DryRun, "dry-run", false, "show what would change")
		fs.StringVar(&cmd.ServerName, "name", "", "server name in client configs")
		// Credential flags are captured as key=value pairs.
		email := fs.String("email", "", "email for authentication")
		token := fs.String("token", "", "API token for authentication")
		// Accept --clients as comma-separated list.
		clientsStr := fs.String("clients", "", "comma-separated client list")

		if err := fs.Parse(tail); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return cmd, ErrUsage
			}
			return cmd, err
		}

		if *email != "" {
			cmd.Credentials["email"] = *email
		}
		if *token != "" {
			cmd.Credentials["token"] = *token
		}
		if *clientsStr != "" {
			cmd.Clients = strings.Split(*clientsStr, ",")
			for i := range cmd.Clients {
				cmd.Clients[i] = strings.TrimSpace(cmd.Clients[i])
			}
		}
		cmd.Args = fs.Args()
		return cmd, nil

	case "mcp", "server", "serve":
		cmd.Name = "mcp"
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.Usage = func() {}
		if err := fs.Parse(tail); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return cmd, ErrUsage
			}
			return cmd, err
		}
		cmd.Args = fs.Args()
		return cmd, nil

	case "login":
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.Usage = func() {}
		email := fs.String("email", "", "email for authentication")
		token := fs.String("token", "", "API token for authentication")
		if err := fs.Parse(tail); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return cmd, ErrUsage
			}
			return cmd, err
		}
		if *email != "" {
			cmd.Credentials["email"] = *email
		}
		if *token != "" {
			cmd.Credentials["token"] = *token
		}
		cmd.Args = fs.Args()
		return cmd, nil

	case "doctor":
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.Usage = func() {}
		if err := fs.Parse(tail); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return cmd, ErrUsage
			}
			return cmd, err
		}
		cmd.Args = fs.Args()
		return cmd, nil

	case "update":
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		fs.Usage = func() {}
		from := fs.String("from", "", "swap from a pre-downloaded temp file")
		if err := fs.Parse(tail); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return cmd, ErrUsage
			}
			return cmd, err
		}
		cmd.Args = fs.Args()
		if *from != "" {
			cmd.Args = append([]string{"--from", *from}, cmd.Args...)
		}
		return cmd, nil

	default:
		// Unknown subcommand — pass through for consumer to handle.
		cmd.Args = tail
		return cmd, nil
	}
}
