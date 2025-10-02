package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/guyfedwards/nom/v2/internal/commands"
	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
)

type Options struct {
	Verbose      bool     `short:"v" long:"verbose" description:"Show verbose logging"`
	Pager        string   `short:"p" long:"pager" description:"Pager to use for longer output. Set to false for no pager"`
	ConfigPath   string   `short:"c" long:"config-path" description:"Location of config.yml"`
	PreviewFeeds []string `short:"f" long:"feed" description:"Feed(s) URL(s) for preview"`
}

var (
	options Options
	version = "dev"
)

// Setup subcommands

type Add struct {
	Positional struct {
		Url  string `positional-arg-name:"URL" required:"yes"`
		Name string `positional-arg-name:"NAME"`
	} `positional-args:"yes"`
}

func (r *Add) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}
	return cmds.Add(r.Positional.Url, r.Positional.Name)
}

type Config struct{}

func (r *Config) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}
	return cmds.ShowConfig()
}

type List struct{}

func (r *List) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}

	return cmds.List()
}

type Version struct{}

func (r *Version) Execute(args []string) error {
	_, err := getCmds()
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", version)
	return nil
}

type Refresh struct{}

func (r *Refresh) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}
	return cmds.Refresh()
}

type Unread struct{}

func (r *Unread) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}
	count := cmds.CountUnread()
	fmt.Printf("%d\n", count)
	return nil
}

func getCmds() (*commands.Commands, error) {
	cfg, err := config.New(options.ConfigPath, options.Pager, options.PreviewFeeds, version)
	if err != nil {
		return nil, err
	}

	if err = cfg.Load(); err != nil {
		return nil, err
	}

	s, err := store.NewSQLiteStore(cfg.ConfigDir, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("main.go: %w", err)
	}
	cmds := commands.New(cfg, s)
	return cmds, nil
}

func main() {
	parser := flags.NewParser(&options, flags.Default)
	// allow nom to be run without any subcommands
	parser.SubcommandsOptional = true

	// add commands
	parser.AddCommand("add", "Add feed", "Add a new feed", &Add{})
	parser.AddCommand("config", "Show config", "Show configuration", &Config{})
	parser.AddCommand("list", "List feeds", "List all feeds", &List{})
	parser.AddCommand("version", "Show Version", "Display version information", &Version{})
	parser.AddCommand("refresh", "Refresh feeds", "refresh feed(s) without opening TUI", &Refresh{})
	parser.AddCommand("unread", "Count unread", "Get count of unread items", &Unread{})

	// parse the command line arguments
	_, err := parser.Parse()

	// check for help flag
	if err != nil {
		if flagErr, ok := err.(*flags.Error); ok && flagErr.Type != flags.ErrHelp {
			parser.WriteHelp(os.Stdout)
		}

		os.Exit(0)
	}

	// no subcommand or help flag, run the TUI
	if parser.Active == nil {
		cmds, err := getCmds()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = cmds.TUI()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}
}
