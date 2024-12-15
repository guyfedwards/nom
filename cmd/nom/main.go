package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/guyfedwards/nom/v2/internal/commands"
	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
)

type Options struct {
	Verbose      bool     `short:"v" long:"verbose" description:"Show verbose logging"`
	Number       int      `short:"n" long:"number" description:"Number of results to show"`
	Pager        string   `short:"p" long:"pager" description:"Pager to use for longer output. Set to false for no pager"`
	ConfigPath   string   `long:"config-path" description:"Location of config.yml"`
	PreviewFeeds []string `short:"f" long:"feed" description:"Feed(s) URL(s) for preview"`
	Version      bool     `long:"version" description:"Display version information"`
}

var (
	version          = "dev"
	ErrNotEnoughArgs = errors.New("not enough args")
)

func run(args []string, opts Options) error {
	cfg, err := config.New(opts.ConfigPath, opts.Pager, opts.PreviewFeeds, version)
	if err != nil {
		return err
	}

	if err = cfg.Load(); err != nil {
		return err
	}

	s, err := store.NewSQLiteStore(cfg.ConfigDir)
	if err != nil {
		return fmt.Errorf("main.go: %w", err)
	}
	cmds := commands.New(cfg, s)

	if opts.Version || (len(args) > 0 && args[0] == "version") {
		fmt.Printf("%s\n", version)
		return nil
	}

	// no subcommand, run the TUI
	if len(args) == 0 {
		return cmds.TUI()
	}

	switch args[0] {
	case "list":
		return cmds.List(opts.Number)
	case "refresh":
		return cmds.Refresh()
	case "config":
		return cmds.ShowConfig()
	case "add":
		if len(args) != 2 {
			return ErrNotEnoughArgs
		}

		return cmds.Add(args[1])
	}

	return nil
}

func main() {
	var opts Options

	parser := flags.NewParser(&opts, flags.Default)

	args, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := run(args, opts); err != nil {
		if opts.Verbose || errors.Is(err, config.ErrFeedAlreadyExists) {
			fmt.Printf("%v\n", err)
		}

		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}
}
