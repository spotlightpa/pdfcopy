package app

import (
	"context"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	"github.com/carlmjohnson/csv"
	"github.com/carlmjohnson/flagx"
	"github.com/carlmjohnson/flagx/lazyio"
	"github.com/carlmjohnson/flowmatic"
	"github.com/carlmjohnson/versioninfo"
)

const AppName = "pdfcopy"

func CLI(args []string) error {
	var app appEnv
	err := app.ParseArgs(args)
	if err != nil {
		return err
	}
	if err = app.Exec(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	return err
}

func (app *appEnv) ParseArgs(args []string) error {
	fl := flag.NewFlagSet(AppName, flag.ContinueOnError)
	src := lazyio.FileOrURL(lazyio.StdIO, nil)
	app.src = src
	fl.Var(src, "src", "source file or URL")
	fl.StringVar(&app.dst, "dst", "output.pdf", "destination `filepath`")
	fl.StringVar(&app.temp, "temp", "", "temporary `filepath` for downloads and intermediate PDFs")
	fl.IntVar(&app.maxProcs, "workers", 10, "number of workers")
	app.Logger = log.New(os.Stderr, AppName+" ", log.LstdFlags)
	flagx.BoolFunc(fl, "silent", "log debug output", func() error {
		app.Logger.SetOutput(io.Discard)
		return nil
	})
	fl.Usage = func() {
		fmt.Fprintf(fl.Output(), `pdfcopy - %s

Download stuff and screenshot it

Usage:

	pdfcopy [options]

Options:
`, versioninfo.Version)
		fl.PrintDefaults()
	}
	versioninfo.AddFlag(fl)
	if err := fl.Parse(args); err != nil {
		return err
	}
	if err := flagx.ParseEnv(fl, AppName); err != nil {
		return err
	}
	return nil
}

type appEnv struct {
	src       io.ReadCloser
	temp, dst string
	maxProcs  int
	*log.Logger
}

func (app *appEnv) Exec() (err error) {
	// Open list of URLs
	urls, err := app.readURLs()
	if err != nil {
		return err
	}
	if app.temp == "" {
		// Make temp directory
		tempdir, err := os.MkdirTemp("", "")
		if err != nil {
			return err
		}
		app.temp = tempdir
	}
	app.Logger.Printf("tempdir %q", app.temp)

	// Start some Flowmatic groups
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err = flowmatic.Each(app.maxProcs, urls, func(url string) error {
		return app.buildPDF(ctx, url)
	})
	if err != nil {
		return err
	}

	// Once they're all done'
	// pdftk ./*.pdf cat output merged.pdf
	// TODO: The order of PDFs is random. Fix that somehow
	args, err := filepath.Glob(filepath.Join(app.temp, "*.pdf"))
	if err != nil {
		return err
	}
	args = append(args, "cat", "output", app.dst)
	cmd := exec.CommandContext(ctx, "pdftk", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	app.Logger.Printf("pdftk cat %s", app.dst)
	if err := cmd.Run(); err != nil {
		return err
	}

	return err
}

func (app *appEnv) readURLs() ([]string, error) {
	var urls []string
	fr := csv.NewFieldReader(app.src)
	for fr.Scan() {
		urls = append(urls, fr.Field("url"))
	}
	return urls, fr.Err()
}

func (app *appEnv) buildPDF(ctx context.Context, url string) error {
	hash := md5.Sum([]byte(url))
	png := fmt.Sprintf("%0x.png", hash)
	pdf := fmt.Sprintf("%0x.pdf", hash)

	// Skip if stat file
	_, err := os.Stat(filepath.Join(app.temp, png))
	switch {
	case err == nil:
		app.Logger.Printf("have %s", png)
	case !errors.Is(err, fs.ErrNotExist):
		return err
	default:
		app.Logger.Printf("start %0x from %q", hash, url)
		// TODO retry in loop
		cmd := exec.CommandContext(ctx, "shot-scraper", "--reduced-motion",
			// TODO figure out whether to use #content or not
			"-s", "#content",
			"-p", "16", "--output", png, url)
		cmd.Dir = app.temp
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			// mark this up
			return fmt.Errorf("problem with %q from %q: %w", png, url, err)
		}
	}
	// Skip if stat file
	_, err = os.Stat(filepath.Join(app.temp, pdf))
	switch {
	case err == nil:
		app.Logger.Printf("have %s", pdf)
		return nil
	case !errors.Is(err, fs.ErrNotExist):
		return err
	default:
		cmd := exec.CommandContext(ctx, "convert", png, pdf)
		cmd.Dir = app.temp
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	app.Logger.Printf("done %0x from %q", hash, url)
	return nil
}
