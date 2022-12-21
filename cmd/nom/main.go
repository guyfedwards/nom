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

var opts struct {
	Verbose bool   `short:"v" long:"verbose" description:"Show verbose logging"`
	Number  int    `short:"n" long:"number" description:"Number of results to show"`
	Pager   string `short:"p" long:"pager" description:"Pager to use for longer output. Set to false for no pager"`
	NoCache bool   `long:"no-cache" description:"Do not use the cache"`
}

var parser = flags.NewParser(&opts, flags.Default)
var ErrNotEnoughArgs = errors.New("not enough args")

func main() {
	args, err := parser.Parse()
	handleError(err, opts.Verbose)

	if len(args) == 0 {
		handleError(ErrNotEnoughArgs, opts.Verbose)
	}

	cfg, err := config.New("", opts.Pager)
	if err != nil {
		handleError(err, opts.Verbose)
	}

	err = cfg.Load()
	if err != nil {
		handleError(err, opts.Verbose)
	}

	cash := cache.New(cache.DefaultPath, cache.DefaultExpiry)

	cmds := commands.New(cfg, cash)

	switch args[0] {
	case "list":
		handleError(cmds.List(opts.Number, !opts.NoCache), opts.Verbose)
	case "add":
		if len(args) != 2 {
			handleError(ErrNotEnoughArgs, opts.Verbose)
		}
		handleError(cmds.Add(args[1]), opts.Verbose)
	case "read":
		if len(args) < 2 {
			handleError(ErrNotEnoughArgs, opts.Verbose)
		}
		handleError(cmds.Read(args[1:]...), opts.Verbose)
	}
}

func handleError(err error, verbose bool) {
	if err != nil {
		if verbose {
			fmt.Println(err)
		}
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}
}
