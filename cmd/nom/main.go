package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/guyfedwards/nom/internal/cache"
	"github.com/guyfedwards/nom/internal/commands"
	"github.com/guyfedwards/nom/internal/config"
)

type Options struct {
	Verbose      bool     `short:"v" long:"verbose" description:"Show verbose logging"`
	Number       int      `short:"n" long:"number" description:"Number of results to show"`
	Pager        string   `short:"p" long:"pager" description:"Pager to use for longer output. Set to false for no pager"`
	NoCache      bool     `long:"no-cache" description:"Do not use the cache"`
	ConfigPath   string   `long:"config-path" description:"Location of config.yml"`
	PreviewFeeds []string `short:"f" long:"feed" description:"Feed(s) URL(s) for preview"`
}

var ErrNotEnoughArgs = errors.New("not enough args")

func run(args []string, opts Options) error {
	if len(opts.PreviewFeeds) > 0 {
		// Don't mess up the cache of configured feeds in the preview mode.
		// Preview mode should use short-lived in-momory cache, but to make it
		// happen we should support that through a cache interface.
		// Another solution is to cache in different directory and wipe it out once nom is closed, i.e.
		// defer wipeOutPreviewCache()
		opts.NoCache = true
	}

	cfg, err := config.New(opts.ConfigPath, opts.Pager, opts.NoCache, opts.PreviewFeeds)
	if err != nil {
		return err
	}

	if err := cfg.Load(); err != nil {
		return err
	}

	cash := cache.New(cache.DefaultPath, cache.DefaultExpiry)

	cmds := commands.New(cfg, cash)

	// no subcommand, run the TUI
	if len(args) == 0 {
		return cmds.TUI()
	}

	switch args[0] {
	case "list":
		return cmds.List(opts.Number, !opts.NoCache)
	case "add":
		if len(args) != 2 {
			return ErrNotEnoughArgs
		}

		return cmds.Add(args[1])
	case "read":
		if len(args) < 2 {
			return ErrNotEnoughArgs
		}

		return cmds.Read(args[1:]...)
	}

	return nil
}

func main() {
	var opts Options

	parser := flags.NewParser(&opts, flags.Default)

	args, err := parser.Parse()
	if err != nil {
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	if err := run(args, opts); err != nil {
		if opts.Verbose {
			fmt.Printf("%v\n", err)
		}

		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}
}
