package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/accountant"
	"github.com/chamzzzzzz/accounting/amount"
	"github.com/chamzzzzzz/accounting/book"
	"github.com/chamzzzzzz/accounting/book/provider/ssfs"
	"github.com/chamzzzzzz/accounting/journal"
	"github.com/chamzzzzzz/accounting/voucher"
)

type CLI struct {
	args        []string
	dir         string
	provider    book.Provider
	cmd         string
	subcmd      string
	title       string
	date        string
	catalog     string
	description string
	entries     []string
}

func main() {
	cli := &CLI{
		args: os.Args[1:],
		dir:  ".",
	}
	if err := cli.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func (c *CLI) Run() error {
	if len(c.args) == 0 || c.args[0] == "-h" || c.args[0] == "--help" {
		c.printHelp()
		return nil
	}

	c.parseFlags()

	if len(c.args) == 0 {
		c.printHelp()
		return nil
	}

	c.cmd = c.args[0]
	if len(c.args) > 1 {
		c.subcmd = c.args[1]
	}

	if c.cmd == "help" {
		if len(c.args) > 1 {
			c.cmd, c.subcmd = c.args[1], "help"
		} else {
			c.cmd, c.subcmd = "", ""
		}
	}
	if c.subcmd == "-h" || c.subcmd == "--help" {
		c.subcmd = "help"
	}
	if c.subcmd == "" || c.subcmd == "help" {
		switch c.cmd {
		case "book":
			c.printBookHelp()
			return nil
		case "account":
			c.printAccountHelp()
			return nil
		case "journal":
			c.printJournalHelp()
			return nil
		case "voucher":
			c.printVoucherHelp()
			return nil
		default:
			c.printHelp()
			return nil
		}
	}

	c.dir, _ = filepath.Abs(c.dir)
	c.provider = &ssfs.Provider{Dir: c.dir}

	switch c.cmd {
	case "book":
		return c.runBook()
	case "account":
		return c.runAccount()
	case "journal":
		return c.runJournal()
	case "voucher":
		return c.runVoucher()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", c.cmd)
		c.printHelp()
		os.Exit(1)
	}
	return nil
}

func (c *CLI) runBook() error {
	switch c.subcmd {
	case "create":
		return c.cmdBookCreate()
	default:
		c.printBookHelp()
		return nil
	}
}

func (c *CLI) runAccount() error {
	switch c.subcmd {
	case "list":
		return c.cmdAccountList()
	case "add":
		return c.cmdAccountAdd()
	case "delete":
		return c.cmdAccountDelete()
	default:
		c.printAccountHelp()
		return nil
	}
}

func (c *CLI) runJournal() error {
	switch c.subcmd {
	case "list":
		return c.cmdJournalList()
	case "add":
		return c.cmdJournalAdd()
	case "delete":
		return c.cmdJournalDelete()
	default:
		c.printJournalHelp()
		return nil
	}
}

func (c *CLI) runVoucher() error {
	switch c.subcmd {
	case "add":
		return c.cmdVoucherAdd()
	default:
		c.printVoucherHelp()
		return nil
	}
}

func (c *CLI) loadBook() (*book.Book, error) {
	bk, err := c.provider.Load(context.Background(), ".")
	if err != nil {
		return nil, err
	}
	if bk == nil {
		return nil, fmt.Errorf("book not found: %s", c.dir)
	}
	return bk, nil
}

func (c *CLI) saveBook(bk *book.Book) error {
	return c.provider.Save(context.Background(), bk)
}

func (c *CLI) parseFlags() {
	var args []string
	for i := 0; i < len(c.args); i++ {
		switch c.args[i] {
		case "--book", "-b":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.dir = c.args[i+1]
				i++
			}
		case "--title", "-t":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.title = c.args[i+1]
				i++
			}
		case "--date", "-d":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.date = c.args[i+1]
				i++
			}
		case "--catalog", "-c":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.catalog = c.args[i+1]
				i++
			}
		case "--description":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.description = c.args[i+1]
				i++
			}
		case "--entry", "-e":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.entries = append(c.entries, c.args[i+1])
				i++
			}
		default:
			args = append(args, c.args[i])
		}
	}
	c.args = args
}

func (c *CLI) printHelp() {
	fmt.Println(`accounting - Accounting CLI

Usage:
  accounting [--book <dir>] <command> [options]

Commands:
  book      Book management
  account   Account management
  journal   Journal management
  voucher   Voucher management

Options:
  --book, -b  Book directory (default: ".")

Use "accounting <command> help" for more information.`)
}

func (c *CLI) printBookHelp() {
	fmt.Println(`Usage: accounting book <subcommand> [options]

Subcommands:
  create    Create a new book
  help      Show this help`)
}

func (c *CLI) printAccountHelp() {
	fmt.Println(`Usage: accounting account <subcommand> [options]

Subcommands:
  list      List all accounts
  add       Add an account
  delete    Delete an account
  help      Show this help

Options:
  --title, -t  Account title (required)`)
}

func (c *CLI) printJournalHelp() {
	fmt.Println(`Usage: accounting journal <subcommand> [options]

Subcommands:
  list      List all journals
  add       Add a journal
  delete    Delete a journal
  help      Show this help

Options:
  --date, -d      Date (YYYY-MM-DD, required)
  --catalog, -c   Catalog (required)`)
}

func (c *CLI) printVoucherHelp() {
	fmt.Println(`Usage: accounting voucher <subcommand> [options]

Subcommands:
  add       Add a voucher
  help      Show this help

Options:
  --date, -d         Date (RFC3339, required)
  --catalog, -c      Catalog (required)
  --description      Description
  --entry, -e        Entry: "account amount currency" (repeatable)`)
}

func (c *CLI) cmdBookCreate() error {
	os.MkdirAll(c.dir, 0755)
	bk := &book.Book{
		Id:       ".",
		Info:     &book.Info{Title: "New Book"},
		Accounts: []*account.Account{},
		Journals: []*journal.Journal{},
	}
	if err := c.provider.Save(context.Background(), bk); err != nil {
		return err
	}
	fmt.Printf("Book created: %s\n", c.dir)
	return nil
}

func (c *CLI) cmdAccountList() error {
	bk, err := c.loadBook()
	if err != nil {
		return err
	}
	if len(bk.Accounts) == 0 {
		fmt.Println("No accounts")
		return nil
	}
	fmt.Println("Accounts:")
	for _, a := range bk.Accounts {
		fmt.Printf("  %s\n", a.Title)
	}
	return nil
}

func (c *CLI) cmdAccountAdd() error {
	bk, err := c.loadBook()
	if err != nil {
		return err
	}
	if c.title == "" {
		return fmt.Errorf("--title is required")
	}
	acc := &accountant.Accountant{Book: bk}
	acc.AddAccount(&account.Account{Title: c.title})
	if err := c.saveBook(bk); err != nil {
		return err
	}
	fmt.Printf("Account added: %s\n", c.title)
	return nil
}

func (c *CLI) cmdAccountDelete() error {
	bk, err := c.loadBook()
	if err != nil {
		return err
	}
	if c.title == "" {
		return fmt.Errorf("--title is required")
	}
	acc := &accountant.Accountant{Book: bk}
	acc.DelAccount(c.title)
	if err := c.saveBook(bk); err != nil {
		return err
	}
	fmt.Printf("Account deleted: %s\n", c.title)
	return nil
}

func (c *CLI) cmdJournalList() error {
	bk, err := c.loadBook()
	if err != nil {
		return err
	}
	if len(bk.Journals) == 0 {
		fmt.Println("No journals")
		return nil
	}
	fmt.Println("Journals:")
	for _, j := range bk.Journals {
		fmt.Printf("  %s  %s  (%d vouchers)\n", j.Date, j.Catalog, len(j.Vouchers))
	}
	return nil
}

func (c *CLI) cmdJournalAdd() error {
	bk, err := c.loadBook()
	if err != nil {
		return err
	}
	if c.date == "" {
		return fmt.Errorf("--date is required")
	}
	if c.catalog == "" {
		return fmt.Errorf("--catalog is required")
	}
	date, err := parseDate(c.date, "2006-01")
	if err != nil {
		return err
	}
	acc := &accountant.Accountant{Book: bk}
	acc.AddJournal(&journal.Journal{Date: date, Catalog: c.catalog})
	if err := c.saveBook(bk); err != nil {
		return err
	}
	fmt.Printf("Journal added: %s %s\n", date, c.catalog)
	return nil
}

func (c *CLI) cmdJournalDelete() error {
	bk, err := c.loadBook()
	if err != nil {
		return err
	}
	if c.date == "" {
		return fmt.Errorf("--date is required")
	}
	if c.catalog == "" {
		return fmt.Errorf("--catalog is required")
	}
	date, err := parseDate(c.date, "2006-01")
	if err != nil {
		return err
	}
	acc := &accountant.Accountant{Book: bk}
	acc.DelJournal(date, c.catalog)
	if err := c.saveBook(bk); err != nil {
		return err
	}
	fmt.Printf("Journal deleted: %s %s\n", date, c.catalog)
	return nil
}

func (c *CLI) cmdVoucherAdd() error {
	bk, err := c.loadBook()
	if err != nil {
		return err
	}
	if c.date == "" {
		return fmt.Errorf("--date is required")
	}
	if c.catalog == "" {
		return fmt.Errorf("--catalog is required")
	}
	date, err := parseDate(c.date, time.RFC3339)
	if err != nil {
		return err
	}
	entries, err := parseEntries(c.entries)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("at least one --entry is required")
	}
	acc := &accountant.Accountant{Book: bk}
	acc.AddVoucher(&voucher.Voucher{
		Date:        date,
		Catalog:     c.catalog,
		Entries:     entries,
		Description: c.description,
	})
	if err := c.saveBook(bk); err != nil {
		return err
	}
	fmt.Printf("Voucher added: %s %s (%d entries)\n", date, c.catalog, len(entries))
	return nil
}

func parseDate(s, layout string) (string, error) {
	t, err := time.Parse(layout, s)
	if err != nil {
		return "", fmt.Errorf("invalid date format (expected %s): %v", layout, err)
	}
	return t.Format(layout), nil
}

func parseEntries(args []string) ([]*voucher.Entry, error) {
	var entries []*voucher.Entry
	for _, e := range args {
		parts := strings.Fields(e)
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid entry: %s (expected: account amount currency)", e)
		}
		entries = append(entries, &voucher.Entry{
			Account: parts[0],
			Amount:  &amount.Amount{Quantity: parts[1], Currency: parts[2]},
		})
	}
	return entries, nil
}
