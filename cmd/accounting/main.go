package main

import (
	"fmt"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

var (
	logger = log.New(os.Stdout, "accounting: ", log.Ldate|log.Lmicroseconds)
)

type App struct {
	cli *cli.App
}

func (app *App) Run() error {
	app.cli = &cli.App{
		Usage: "accounting cli",
		Commands: []*cli.Command{
			{
				Name:  "sourcedocument",
				Usage: "sourcedocument",
				Subcommands: []*cli.Command{
					{
						Name:   "scan",
						Usage:  "scan source file to sourcedocument",
						Action: app.scan,
					},
				},
			},
		},
	}
	return app.cli.Run(os.Args)
}

func (app *App) scan(c *cli.Context) error {
	scanner, err := scanner.NewScanner(nil)
	if err != nil {
		return err
	}

	for _, file := range c.Args().Slice() {
		sourcedocument, err := scanner.Scan(file)
		if err != nil {
			return err
		}
		fmt.Printf("%+v\n", sourcedocument)
	}
	return nil
}

func main() {
	if err := (&App{}).Run(); err != nil {
		logger.Printf("run, err='%s'\n", err)
	}
}
