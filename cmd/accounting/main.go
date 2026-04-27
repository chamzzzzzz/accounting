package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/accountant"
	"github.com/chamzzzzzz/accounting/amount"
	"github.com/chamzzzzzz/accounting/book"
	"github.com/chamzzzzzz/accounting/book/provider/ledger"
	"github.com/chamzzzzzz/accounting/book/provider/ssfs"
	"github.com/chamzzzzzz/accounting/journal"
	"github.com/chamzzzzzz/accounting/report"
	"github.com/chamzzzzzz/accounting/rule"
	"github.com/chamzzzzzz/accounting/sourcedocument"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner/ai"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner/ai/openai"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner/ocr"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner/ocr/normalizer"
	"github.com/chamzzzzzz/accounting/strategy"
	"github.com/chamzzzzzz/accounting/strategy/basic"
	"github.com/chamzzzzzz/accounting/voucher"
	"github.com/chamzzzzzz/gocr"
	"github.com/chamzzzzzz/gocr/macocr"
)

type option struct {
	args        []string
	dir         string
	to          string
	from        string
	format      string
	cmd         string
	subcmd      string
	titles      []string
	name        string
	date        string
	id          string
	catalog     string
	description string
	entries     []string
	spec        string
	scanner     string
	normalize   bool
	labels      []string
	strategy    string
	orderNumber string
	merchant    string
}

type CLI struct {
	option
	provider book.Provider
	strategy strategy.Strategy
}

func main() {
	cli := &CLI{
		option: option{
			args:      os.Args[1:],
			dir:       ".",
			format:    "ssfs",
			strategy:  "basic",
			normalize: true,
		},
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
		case "rule":
			c.printRuleHelp()
			return nil
		case "report":
			c.printReportHelp()
			return nil
		case "sourcedocument":
			c.printSourceDocumentHelp()
			return nil
		default:
			c.printHelp()
			return nil
		}
	}

	c.dir, _ = filepath.Abs(c.dir)
	switch c.format {
	case "ssfs":
		c.provider = &ssfs.Provider{Dir: c.dir}
	case "ledger":
		c.provider = &ledger.Provider{Dir: c.dir}
	default:
		return fmt.Errorf("unknown provider: %s", c.format)
	}

	switch c.option.strategy {
	case "basic":
		c.strategy = &basic.Strategy{}
	default:
		return fmt.Errorf("unknown strategy: %s", c.option.strategy)
	}

	switch c.cmd {
	case "book":
		return c.runBook()
	case "account":
		return c.runAccount()
	case "journal":
		return c.runJournal()
	case "voucher":
		return c.runVoucher()
	case "rule":
		return c.runRule()
	case "report":
		return c.runReport()
	case "sourcedocument":
		return c.runSourceDocument()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", c.cmd)
		c.printHelp()
		os.Exit(1)
	}
	return nil
}

func (c *CLI) runSourceDocument() error {
	switch c.subcmd {
	case "scan":
		return c.cmdSourceDocumentScan()
	default:
		c.printSourceDocumentHelp()
		return nil
	}
}

func (c *CLI) cmdSourceDocumentScan() error {
	if c.from == "" {
		return fmt.Errorf("--from is required")
	}

	paths, err := walk(c.from)
	if err != nil {
		return err
	}

	s, err := c.createScanner()
	if err != nil {
		return err
	}

	if c.to != "" {
		if err := os.MkdirAll(c.to, 0755); err != nil {
			return err
		}
	}

	for _, path := range paths {
		doc := &scanner.Document{Path: path}
		sd, err := s.Scan(context.Background(), doc)
		if err != nil {
			return err
		}

		if c.catalog != "" {
			sd.Catalog = c.catalog
		}

		if c.to != "" {
			rel, err := filepath.Rel(c.from, path)
			if err != nil || rel == "." {
				rel = filepath.Base(path)
			}
			name := strings.TrimSuffix(rel, filepath.Ext(rel)) + ".json"
			path := filepath.Join(c.to, name)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			b, err := json.MarshalIndent(sd, "", "  ")
			if err != nil {
				return err
			}
			b = append(b, '\n')
			if err := os.WriteFile(path, b, 0644); err != nil {
				return err
			}
		} else {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(sd); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *CLI) createScanner() (scanner.Scanner, error) {
	sc := c.scanner
	if sc == "" {
		sc = "ocr"
	}

	switch sc {
	case "ocr":
		var spec gocr.Option
		if c.spec != "" {
			b, err := os.ReadFile(c.spec)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(b, &spec); err != nil {
				return nil, err
			}
		}
		if spec.Type == "" {
			spec.Type = "macocr"
		}

		ws := gocr.NewWorkspace()
		ws.RegisterCreator(&macocr.Creator{})
		r, err := ws.CreateRecognizer(&spec)
		if err != nil {
			return nil, err
		}
		var n *normalizer.Normalizer
		if c.normalize {
			n = &normalizer.Normalizer{}
		}
		return &ocr.Scanner{Recognizer: r, Normalizer: n}, nil
	case "openai":
		var spec ai.Spec
		if c.spec != "" {
			b, err := os.ReadFile(c.spec)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(b, &spec); err != nil {
				return nil, err
			}
		}
		return &openai.Scanner{Spec: spec}, nil
	default:
		return nil, fmt.Errorf("unknown scanner: %s", c.scanner)
	}
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
	case "prepare":
		return c.cmdVoucherPrepare()
	default:
		c.printVoucherHelp()
		return nil
	}
}

func (c *CLI) runRule() error {
	switch c.subcmd {
	case "list":
		return c.cmdRuleList()
	case "add":
		return c.cmdRuleAdd()
	case "delete":
		return c.cmdRuleDelete()
	default:
		c.printRuleHelp()
		return nil
	}
}

func (c *CLI) runReport() error {
	switch c.subcmd {
	case "balance":
		return c.cmdReportBalance()
	case "register":
		return c.cmdReportRegister()
	default:
		c.printReportHelp()
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

func (c *CLI) accounting(save bool, fn func(bk *book.Book, acc *accountant.Accountant) error) error {
	bk, err := c.loadBook()
	if err != nil {
		return err
	}
	acc := &accountant.Accountant{Book: bk, Strategy: c.strategy}
	if err := fn(bk, acc); err != nil {
		return err
	}
	if save {
		return c.saveBook(bk)
	}
	return nil
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
			continue
		case "--format", "-f":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.format = c.args[i+1]
				i++
			}
			continue
		case "--title", "-t":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.titles = append(c.titles, c.args[i+1])
				i++
			}
			continue
		case "--name", "-n":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.name = c.args[i+1]
				i++
			}
			continue
		case "--date", "-d":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.date = c.args[i+1]
				i++
			}
			continue
		case "--id":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.id = c.args[i+1]
				i++
			}
			continue
		case "--catalog", "-c":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.catalog = c.args[i+1]
				i++
			}
			continue
		case "--description":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.description = c.args[i+1]
				i++
			}
			continue
		case "--order-number":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.orderNumber = c.args[i+1]
				i++
			}
			continue
		case "--merchant":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.merchant = c.args[i+1]
				i++
			}
			continue
		case "--entry", "-e":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.entries = append(c.entries, c.args[i+1])
				i++
			}
			continue
		case "--spec":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.spec = c.args[i+1]
				i++
			}
			continue
		case "--scanner":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.scanner = c.args[i+1]
				i++
			}
			continue
		case "--normalize":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.normalize, _ = strconv.ParseBool(c.args[i+1])
				i++
			}
			continue
		case "--strategy":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.option.strategy = c.args[i+1]
				i++
			}
			continue
		case "--to":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.to = c.args[i+1]
				i++
			}
			continue
		case "--from":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.from = c.args[i+1]
				i++
			}
			continue
		case "--label":
			if i+1 < len(c.args) && c.args[i+1][0] != '-' {
				c.labels = append(c.labels, c.args[i+1])
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
  accounting [--book <dir>] [--format <name>] <command> [options]

Commands:
  book      Book management
  account   Account management
  journal   Journal management
  voucher   Voucher management
  rule      Rule management
  report    Report management
  sourcedocument  Source document management

Options:
  --book, -b      Book directory (default: ".")
  --format, -f    Book format (ssfs, ledger; default: ssfs)

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
  --title, -t    Account title (required)
  --catalog, -c  Account catalog (required)
  --label        Account label, space-separated-words (repeatable)`)
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
  prepare   Prepare vouchers
  help      Show this help

Options:
  --id               Voucher ID
  --date, -d         Date (RFC3339, required)
  --catalog, -c      Catalog (required)
  --description      Description
  --order-number     Order number
  --merchant         Merchant
  --entry, -e        Entry: "account amount currency" (repeatable)
  --from             Source document or voucher file path
  --to               Voucher output directory
  --strategy         Voucher prepare strategy (default: basic)`)
}

func (c *CLI) printRuleHelp() {
	fmt.Println(`Usage: accounting rule <subcommand> [options]

Subcommands:
  list      List all rules
  add       Add a rule
  delete    Delete a rule
  help      Show this help

Options:
  --catalog, -c     Rule catalog (optional for add, required for delete)
  --name, -n        Rule name
  --description     Rule description
  --from            Rule file path`)
}

func (c *CLI) printReportHelp() {
	fmt.Println(`Usage: accounting report <subcommand> [options]

Subcommands:
  balance   Show account balance report
  register  Show account register report
  help      Show this help

Options:
  --title, -t  Account title (repeatable)`)
}

func (c *CLI) printSourceDocumentHelp() {
	fmt.Println(`Usage: accounting sourcedocument <subcommand> [options]

Subcommands:
  scan      Scan a source document
  help      Show this help

Options:
  --from      Path to the document or directory (required)
  --to        Path to the source document output directory (optional)
  --spec      Path to the scanner spec file (optional)
  --scanner   Scanner type (openai, ocr, default: ocr)
  --normalize Normalize the scanned source document (true, false, default: true)`)
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
	return c.accounting(false, func(bk *book.Book, acc *accountant.Accountant) error {
		if len(bk.Accounts) == 0 {
			fmt.Println("No accounts")
			return nil
		}
		fmt.Println("Accounts:")
		for _, a := range bk.Accounts {
			fmt.Printf("  %s\n", a.Title)
		}
		return nil
	})
}

func (c *CLI) cmdAccountAdd() error {
	if len(c.titles) == 0 {
		return fmt.Errorf("--title is required")
	}
	if c.catalog == "" {
		return fmt.Errorf("--catalog is required")
	}
	labels := parseLabels(c.labels)
	err := c.accounting(true, func(bk *book.Book, acc *accountant.Accountant) error {
		for _, title := range c.titles {
			if err := acc.AddAccount(&account.Account{Title: title, Catalog: c.catalog, Labels: labels}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, title := range c.titles {
		fmt.Printf("Account added: %s (Catalog: %s)\n", title, c.catalog)
	}
	return nil
}

func parseLabels(ss []string) []*account.Label {
	var labels []*account.Label
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		words := strings.Fields(s)
		if len(words) > 0 {
			labels = append(labels, &account.Label{Words: words})
		}
	}
	return labels
}

func (c *CLI) cmdAccountDelete() error {
	if len(c.titles) == 0 {
		return fmt.Errorf("--title is required")
	}
	err := c.accounting(true, func(bk *book.Book, acc *accountant.Accountant) error {
		for _, title := range c.titles {
			if err := acc.DelAccount(title); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, title := range c.titles {
		fmt.Printf("Account deleted: %s\n", title)
	}
	return nil
}

func (c *CLI) cmdJournalList() error {
	return c.accounting(false, func(bk *book.Book, acc *accountant.Accountant) error {
		if len(bk.Journals) == 0 {
			fmt.Println("No journals")
			return nil
		}
		fmt.Println("Journals:")
		for _, j := range bk.Journals {
			fmt.Printf("  %s  %s  (%d vouchers)\n", j.Date, j.Catalog, len(j.Vouchers))
		}
		return nil
	})
}

func (c *CLI) cmdJournalAdd() error {
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
	err = c.accounting(true, func(bk *book.Book, acc *accountant.Accountant) error {
		return acc.AddJournal(&journal.Journal{Date: date, Catalog: c.catalog})
	})
	if err != nil {
		return err
	}
	fmt.Printf("Journal added: %s %s\n", date, c.catalog)
	return nil
}

func (c *CLI) cmdJournalDelete() error {
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
	err = c.accounting(true, func(bk *book.Book, acc *accountant.Accountant) error {
		return acc.DelJournal(date, c.catalog)
	})
	if err != nil {
		return err
	}
	fmt.Printf("Journal deleted: %s %s\n", date, c.catalog)
	return nil
}

func (c *CLI) cmdVoucherAdd() error {
	var vouchers []*voucher.Voucher

	if c.from != "" {
		paths, err := walk(c.from)
		if err != nil {
			return err
		}

		for _, path := range paths {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			v := &voucher.Voucher{}
			if err := json.Unmarshal(b, v); err != nil {
				return fmt.Errorf("load voucher %s: %w", path, err)
			}
			vouchers = append(vouchers, v)
		}
	} else {
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
		vouchers = append(vouchers, &voucher.Voucher{
			Id:          c.id,
			Date:        date,
			Catalog:     c.catalog,
			Entries:     entries,
			Description: c.description,
			OrderNumber: c.orderNumber,
			Merchant:    c.merchant,
		})
	}

	err := c.accounting(true, func(bk *book.Book, acc *accountant.Accountant) error {
		for _, v := range vouchers {
			if err := acc.AddVoucher(v); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, v := range vouchers {
		id := v.Id
		if id == "" {
			id = "-"
		}
		fmt.Printf("Voucher added: %s %s %s %d entries\n", id, v.Date, v.Catalog, len(v.Entries))
	}
	return nil
}

func (c *CLI) cmdVoucherPrepare() error {
	if c.from == "" {
		return fmt.Errorf("--from is required")
	}

	paths, err := walk(c.from)
	if err != nil {
		return err
	}

	if c.to != "" {
		if err := os.MkdirAll(c.to, 0755); err != nil {
			return err
		}
	}

	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var sd sourcedocument.SourceDocument
		if err := json.Unmarshal(b, &sd); err != nil {
			return fmt.Errorf("load source document %s: %w", path, err)
		}

		var vouchers []*voucher.Voucher
		err = c.accounting(false, func(bk *book.Book, acc *accountant.Accountant) error {
			vouchers, err = acc.PrepareVoucher(context.Background(), &sd)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}

		if c.to != "" {
			multiple := len(vouchers) > 1
			for i, v := range vouchers {
				name := strings.TrimSpace(v.Id)
				if name == "" {
					name = time.Now().Format("20060102150405")
				}
				if multiple {
					name = fmt.Sprintf("%s.%d", name, i+1)
				}
				path := filepath.Join(c.to, name)
				b, err := json.MarshalIndent(v, "", "  ")
				if err != nil {
					return err
				}
				b = append(b, '\n')
				if err := os.WriteFile(path, b, 0644); err != nil {
					return err
				}
			}
			continue
		} else {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(vouchers); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *CLI) cmdRuleList() error {
	return c.accounting(false, func(bk *book.Book, acc *accountant.Accountant) error {
		var rules []*rule.Rule
		for _, r := range bk.Rules {
			if c.catalog != "" && r.Catalog != c.catalog {
				continue
			}
			rules = append(rules, r)
		}

		if len(rules) == 0 {
			fmt.Println("No rules")
			return nil
		}

		sort.Slice(rules, func(i, j int) bool {
			return rules[i].Catalog < rules[j].Catalog
		})

		fmt.Println("Rules:")
		for _, r := range rules {
			fmt.Printf("  %s  %s\n", r.Catalog, r.Name)
		}
		return nil
	})
}

func (c *CLI) cmdRuleAdd() error {
	if c.from == "" {
		return fmt.Errorf("--from is required")
	}

	paths, err := walk(c.from)
	if err != nil {
		return err
	}

	var rules []*rule.Rule
	for _, path := range paths {
		rule := &rule.Rule{}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, rule); err != nil {
			return fmt.Errorf("load rule %s: %w", path, err)
		}
		if rule.Catalog == "" {
			rule.Catalog = c.catalog
		}
		if rule.Catalog == "" {
			return fmt.Errorf("--catalog is required (or provide catalog in %s)", path)
		}
		if rule.Name == "" {
			rule.Name = c.name
		}
		if rule.Description == "" {
			rule.Description = c.description
		}
		rules = append(rules, rule)
	}

	err = c.accounting(true, func(bk *book.Book, acc *accountant.Accountant) error {
		for _, rule := range rules {
			if err := acc.AddRule(rule); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, rule := range rules {
		fmt.Printf("Rule added: %s (Name: %s)\n", rule.Catalog, rule.Name)
	}
	return nil
}

func (c *CLI) cmdRuleDelete() error {
	if c.catalog == "" {
		return fmt.Errorf("--catalog is required")
	}
	err := c.accounting(true, func(bk *book.Book, acc *accountant.Accountant) error {
		return acc.DelRule(c.catalog)
	})
	if err != nil {
		return err
	}
	fmt.Printf("Rule deleted: %s\n", c.catalog)
	return nil
}

func walk(name string) ([]string, error) {
	if strings.HasSuffix(name, "/") {
		var paths []string
		err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return paths, nil
	}
	return []string{name}, nil
}

const (
	amountColWidth = 11
	lineSeparator  = "--------------------"
)

func (c *CLI) cmdReportBalance() error {
	titles := c.titles
	titles = append(titles, c.args[2:]...)

	return c.accounting(false, func(bk *book.Book, acc *accountant.Accountant) error {
		r, err := acc.ReportAccountBalance(&report.ReportParameters{
			Titles: titles,
		})
		if err != nil {
			return err
		}

		c.printAccountBalanceReport(r)
		return nil
	})
}

func (c *CLI) printAccountBalanceReport(r *report.AccountBalanceReport) {
	for _, b := range r.Balance {
		c.printAccountBalance(b, 0)
	}
	c.printAccountBalanceReportTotal(r)
}

func (c *CLI) printAccountBalanceReportTotal(r *report.AccountBalanceReport) {
	totals := make(map[string]float64)
	for _, b := range r.Balance {
		for _, a := range b.Amounts {
			q, _ := strconv.ParseFloat(a.Quantity, 64)
			totals[a.Currency] += q
		}
	}

	var currencies []string
	for cur, total := range totals {
		if math.Abs(total) > 0.001 {
			currencies = append(currencies, cur)
		}
	}

	if len(currencies) == 0 {
		fmt.Println(lineSeparator)
		fmt.Printf("%*s\n", len(lineSeparator), "0")
		return
	}

	sort.Strings(currencies)
	fmt.Println(lineSeparator)
	for _, cur := range currencies {
		s := fmt.Sprintf("%.2f %s", totals[cur], cur)
		fmt.Printf("%*s\n", len(lineSeparator), s)
	}
}

func (c *CLI) printAccountBalance(b *report.AccountBalance, indent int) {
	// Filter children to only those with non-zero balances in their subtree
	var hasNonZeroBalance func(*report.AccountBalance) bool
	hasNonZeroBalance = func(n *report.AccountBalance) bool {
		for _, a := range n.Amounts {
			if q, _ := strconv.ParseFloat(a.Quantity, 64); math.Abs(q) > 0.001 {
				return true
			}
		}
		for _, child := range n.Children {
			if hasNonZeroBalance(child) {
				return true
			}
		}
		return false
	}

	if !hasNonZeroBalance(b) {
		return
	}

	var activeChildren []*report.AccountBalance
	for _, child := range b.Children {
		if hasNonZeroBalance(child) {
			activeChildren = append(activeChildren, child)
		}
	}

	node := b
	title := b.Title
	if indent > 0 {
		parts := strings.Split(title, ":")
		title = parts[len(parts)-1]
	}

	// Consolidate paths if there's only one active child with the same non-zero amounts
	for len(activeChildren) == 1 && reflect.DeepEqual(node.Amounts, activeChildren[0].Amounts) {
		child := activeChildren[0]
		childParts := strings.Split(child.Title, ":")
		title += ":" + childParts[len(childParts)-1]
		node = child

		// Update active children for the next iteration
		activeChildren = nil
		for _, nextChild := range node.Children {
			if hasNonZeroBalance(nextChild) {
				activeChildren = append(activeChildren, nextChild)
			}
		}
	}

	// Filter and format non-zero amounts
	var nonZeroLines []string
	for _, a := range node.Amounts {
		if q, _ := strconv.ParseFloat(a.Quantity, 64); math.Abs(q) > 0.001 {
			nonZeroLines = append(nonZeroLines, fmt.Sprintf("%*s %s", amountColWidth, a.Quantity, a.Currency))
		}
	}

	// Output non-zero amounts and account title
	indentStr := strings.Repeat("  ", indent)
	if len(nonZeroLines) > 0 {
		for i, line := range nonZeroLines {
			if i == 0 {
				fmt.Printf("%s  %s%s\n", line, indentStr, title)
			} else {
				fmt.Printf("%s\n", line)
			}
		}
	} else if len(activeChildren) > 0 {
		// Parent node with zero balance but non-zero children
		fmt.Printf("%*s  %s%s\n", amountColWidth+1+3, "", indentStr, title)
	}

	for _, child := range activeChildren {
		c.printAccountBalance(child, indent+1)
	}
}

func (c *CLI) cmdReportRegister() error {
	titles := c.titles
	if len(c.args) > 2 {
		titles = append(titles, c.args[2:]...)
	}

	return c.accounting(false, func(bk *book.Book, acc *accountant.Accountant) error {
		r, err := acc.ReportAccountRegister(&report.ReportParameters{
			Titles: titles,
		})
		if err != nil {
			return err
		}

		c.printAccountRegisterReport(r)
		return nil
	})
}

func (c *CLI) printAccountRegisterReport(r *report.AccountRegisterReport) {
	dateWidth := len("2006-01-02 15:04")
	descWidth := 0
	titleWidth := 0
	amtWidth := 0
	balWidth := 0

	formatQuantity := func(q string) string {
		f, _ := strconv.ParseFloat(q, 64)
		s := fmt.Sprintf("%.2f", f)
		if f < 0 {
			s = s[1:]
		}
		parts := strings.Split(s, ".")
		integer := parts[0]
		fraction := parts[1]
		var result []string
		for i := len(integer); i > 0; i -= 3 {
			start := i - 3
			if start < 0 {
				start = 0
			}
			result = append([]string{integer[start:i]}, result...)
		}
		res := strings.Join(result, ",") + "." + fraction
		if f < 0 {
			res = "-" + res
		}
		return res
	}

	displayWidth := func(s string) int {
		width := 0
		for _, r := range s {
			switch {
			case r == '\t':
				width += 4
			case unicode.IsControl(r):
				continue
			case r >= 0x1100 && (r <= 0x115F ||
				r == 0x2329 || r == 0x232A ||
				(r >= 0x2E80 && r <= 0xA4CF && r != 0x303F) ||
				(r >= 0xAC00 && r <= 0xD7A3) ||
				(r >= 0xF900 && r <= 0xFAFF) ||
				(r >= 0xFE10 && r <= 0xFE19) ||
				(r >= 0xFE30 && r <= 0xFE6F) ||
				(r >= 0xFF00 && r <= 0xFF60) ||
				(r >= 0xFFE0 && r <= 0xFFE6) ||
				(r >= 0x1F300 && r <= 0x1FAFF)):
				width += 2
			case unicode.In(r,
				unicode.Han,
				unicode.Hangul,
				unicode.Hiragana,
				unicode.Katakana,
				unicode.Bopomofo):
				width += 2
			default:
				width++
			}
		}
		return width
	}

	padRight := func(s string, width int) string {
		padding := width - displayWidth(s)
		if padding <= 0 {
			return s
		}
		return s + strings.Repeat(" ", padding)
	}

	padLeft := func(s string, width int) string {
		padding := width - displayWidth(s)
		if padding <= 0 {
			return s
		}
		return strings.Repeat(" ", padding) + s
	}

	for _, reg := range r.Register {
		desc := reg.Voucher.Description
		if desc == "" {
			desc = reg.Voucher.Catalog
		}
		if displayWidth(desc) > descWidth {
			descWidth = displayWidth(desc)
		}
		if displayWidth(reg.Title) > titleWidth {
			titleWidth = displayWidth(reg.Title)
		}
		for _, amt := range reg.Amounts {
			l := displayWidth(formatQuantity(amt.Quantity) + " " + amt.Currency)
			if l > amtWidth {
				amtWidth = l
			}
		}
		for _, bal := range reg.Balances {
			if q, _ := strconv.ParseFloat(bal.Quantity, 64); math.Abs(q) > 0.001 {
				l := displayWidth(formatQuantity(bal.Quantity) + " " + bal.Currency)
				if l > balWidth {
					balWidth = l
				}
			}
		}
	}

	for _, reg := range r.Register {
		t, err := time.ParseInLocation(time.RFC3339, reg.Voucher.Date, time.Local)
		if err != nil {
			t, _ = time.ParseInLocation("2006-01-02", reg.Voucher.Date, time.Local)
		}
		dateStr := t.Format("2006-01-02 15:04")
		desc := reg.Voucher.Description
		if desc == "" {
			desc = reg.Voucher.Catalog
		}

		var nonZeroBalances []*amount.Amount
		for _, bal := range reg.Balances {
			if q, _ := strconv.ParseFloat(bal.Quantity, 64); math.Abs(q) > 0.001 {
				nonZeroBalances = append(nonZeroBalances, bal)
			}
		}

		firstAmtStr := ""
		if len(reg.Amounts) > 0 {
			firstAmtStr = formatQuantity(reg.Amounts[0].Quantity) + " " + reg.Amounts[0].Currency
		}

		firstBalStr := ""
		if len(nonZeroBalances) > 0 {
			firstBalStr = formatQuantity(nonZeroBalances[0].Quantity) + " " + nonZeroBalances[0].Currency
		}

		fmt.Println(
			padRight(dateStr, dateWidth) + "  " +
				padRight(desc, descWidth) + "  " +
				padRight(reg.Title, titleWidth) + "  " +
				padLeft(firstAmtStr, amtWidth) + "  " +
				padLeft(firstBalStr, balWidth),
		)

		maxLen := len(reg.Amounts)
		if len(nonZeroBalances) > maxLen {
			maxLen = len(nonZeroBalances)
		}

		for i := 1; i < maxLen; i++ {
			amtStr := ""
			balStr := ""
			if i < len(reg.Amounts) {
				amtStr = formatQuantity(reg.Amounts[i].Quantity) + " " + reg.Amounts[i].Currency
			}
			if i < len(nonZeroBalances) {
				balStr = formatQuantity(nonZeroBalances[i].Quantity) + " " + nonZeroBalances[i].Currency
			}
			fmt.Println(
				padRight("", dateWidth) + "  " +
					padRight("", descWidth) + "  " +
					padRight("", titleWidth) + "  " +
					padLeft(amtStr, amtWidth) + "  " +
					padLeft(balStr, balWidth),
			)
		}
	}
}

func parseDate(s, layout string) (string, error) {
	t, err := time.ParseInLocation(layout, s, time.Local)
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
