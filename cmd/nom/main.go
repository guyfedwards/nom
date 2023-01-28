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
	cfg, err := config.New(opts.ConfigPath, opts.Pager, opts.NoCache, opts.PreviewFeeds)
	if err != nil {
		return err
	}

	if err := cfg.Load(); err != nil {
		return err
	}

	var cash cache.CacheInterface
	if cfg.IsPreviewMode() {
		cash = cache.NewMemoryCache()
	} else {
		cash = cache.NewFileCache(cache.DefaultPath, cache.DefaultExpiry)
	}

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
	// disable http2 client as causing issues with reddit rss feed requests
	// https://github.com/guyfedwards/nom/issues/7
	os.Setenv("GODEBUG", "http2client=0")

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
