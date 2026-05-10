package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chamzzzzzz/accounting/accountant"
	"github.com/chamzzzzzz/accounting/book"
	"github.com/chamzzzzzz/accounting/book/provider/ledger"
	"github.com/chamzzzzzz/accounting/service/protocol"
	"github.com/chamzzzzzz/accounting/sourcedocument/scanner"
	"github.com/chamzzzzzz/accounting/strategy"
	"github.com/chamzzzzzz/accounting/strategy/basic"
)

type Option struct {
	PathPrefix string
	LogLevel   slog.Level
	Dir        string
	Book       string
}

type Service struct {
	Option   Option
	Scanner  scanner.Scanner
	mux      http.ServeMux
	provider book.Provider
	strategy strategy.Strategy
}

func (s *Service) Init(ctx context.Context) error {
	dir := s.Option.Dir
	if dir == "" {
		dir = "data"
	}
	s.provider = &ledger.Provider{Dir: dir}
	s.strategy = &basic.Strategy{}

	s.HandlePathPrefixFunc("/CreateBook/", s.HandleCreateBook)
	s.HandlePathPrefixFunc("/GetAccount/", s.HandleGetAccount)
	s.HandlePathPrefixFunc("/AddAccount/", s.HandleAddAccount)
	s.HandlePathPrefixFunc("/DeleteAccount/", s.HandleDeleteAccount)
	s.HandlePathPrefixFunc("/ScanSourceDocument/", s.HandleScanSourceDocument)
	s.HandlePathPrefixFunc("/PrepareVoucher/", s.HandlePrepareVoucher)
	s.HandlePathPrefixFunc("/AddVoucher/", s.HandleAddVoucher)
	s.HandlePathPrefixFunc("/GetRule/", s.HandleGetRule)
	s.HandlePathPrefixFunc("/AddRule/", s.HandleAddRule)
	s.HandlePathPrefixFunc("/DeleteRule/", s.HandleDeleteRule)
	s.HandlePathPrefixFunc("/ReportBalance/", s.HandleReportBalance)
	s.HandlePathPrefixFunc("/ReportRegister/", s.HandleReportRegister)
	return nil
}

func (s *Service) Shutdown(ctx context.Context) error {
	return nil
}

func (s *Service) Uninit(ctx context.Context) error {
	return nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Service) PathPrefix() string {
	pp := s.Option.PathPrefix
	if pp == "" {
		pp = "/accounting"
	} else if pp == "/" {
		pp = ""
	} else if !strings.HasPrefix(pp, "/") {
		pp = "/" + pp
	}
	return pp
}

func (s *Service) HandlePathPrefixFunc(pattern string, handler func(w http.ResponseWriter, r *http.Request)) {
	s.HandlePathPrefix(pattern, http.HandlerFunc(handler))
}

func (s *Service) HandlePathPrefix(pattern string, handler http.Handler) {
	s.mux.Handle(s.PathPrefix()+pattern, handler)
}

func (s *Service) HandleCreateBook(w http.ResponseWriter, r *http.Request) {
	q := &protocol.CreateBookRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.Id)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}
	title := q.Title
	if title == "" {
		title = bookId
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk != nil {
		http.Error(w, "duplicate book", http.StatusForbidden)
		return
	}

	bk = &book.Book{
		Id:   bookId,
		Info: &book.Info{Title: title},
	}
	if err := s.provider.Save(r.Context(), bk); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.CreateBookResponse{Id: bookId}
	s.MustWrite(w, p)
}

func (s *Service) HandleGetAccount(w http.ResponseWriter, r *http.Request) {
	q := &protocol.GetAccountRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	p := &protocol.GetAccountResponse{Accounts: bk.Accounts}
	s.MustWrite(w, p)
}

func (s *Service) HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	q := &protocol.AddAccountRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}
	if q.Account == nil {
		http.Error(w, "no account", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	acc := &accountant.Accountant{Book: bk}
	if err := acc.AddAccount(q.Account); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.provider.Save(r.Context(), bk); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.AddAccountResponse{}
	s.MustWrite(w, p)
}

func (s *Service) HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	q := &protocol.DeleteAccountRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}
	if q.Title == "" {
		http.Error(w, "no title", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	acc := &accountant.Accountant{Book: bk}
	if err := acc.DelAccount(q.Title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.provider.Save(r.Context(), bk); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.DeleteAccountResponse{}
	s.MustWrite(w, p)
}

func (s *Service) HandleScanSourceDocument(w http.ResponseWriter, r *http.Request) {
	q := &protocol.ScanSourceDocumentRequest{}
	s.MustRead(r, q)

	if q.Document == nil {
		http.Error(w, "no document", http.StatusBadRequest)
		return
	}

	if s.Scanner == nil {
		http.Error(w, "no scanner", http.StatusInternalServerError)
		return
	}

	data, err := base64.StdEncoding.DecodeString(q.Document.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	temp := filepath.Join(os.TempDir(), "accounting")
	if err := os.MkdirAll(temp, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f, err := os.CreateTemp(temp, "document-*.jpg")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	name := f.Name()
	defer os.Remove(name)

	if _, err := f.Write(data); err != nil {
		f.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f.Close()

	sd, err := s.Scanner.Scan(r.Context(), &scanner.Document{
		Path: name,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.ScanSourceDocumentResponse{SourceDocument: sd}
	s.MustWrite(w, p)
}

func (s *Service) HandlePrepareVoucher(w http.ResponseWriter, r *http.Request) {
	q := &protocol.PrepareVoucherRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}
	if q.SourceDocument == nil {
		http.Error(w, "no source document", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	acc := &accountant.Accountant{Book: bk, Strategy: s.strategy}
	vouchers, err := acc.PrepareVoucher(r.Context(), q.SourceDocument)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.PrepareVoucherResponse{Vouchers: vouchers}
	s.MustWrite(w, p)
}

func (s *Service) HandleAddVoucher(w http.ResponseWriter, r *http.Request) {
	q := &protocol.AddVoucherRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}
	if q.Voucher == nil {
		http.Error(w, "no voucher", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	acc := &accountant.Accountant{Book: bk}
	if err := acc.AddVoucher(q.Voucher); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.provider.Save(r.Context(), bk); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.AddVoucherResponse{}
	s.MustWrite(w, p)
}

func (s *Service) HandleGetRule(w http.ResponseWriter, r *http.Request) {
	q := &protocol.GetRuleRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	p := &protocol.GetRuleResponse{Rules: bk.Rules}
	s.MustWrite(w, p)
}

func (s *Service) HandleAddRule(w http.ResponseWriter, r *http.Request) {
	q := &protocol.AddRuleRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}
	if q.Rule == nil {
		http.Error(w, "no rule", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	acc := &accountant.Accountant{Book: bk}
	if err := acc.AddRule(q.Rule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.provider.Save(r.Context(), bk); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.AddRuleResponse{}
	s.MustWrite(w, p)
}

func (s *Service) HandleDeleteRule(w http.ResponseWriter, r *http.Request) {
	q := &protocol.DeleteRuleRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}
	if q.Catalog == "" {
		http.Error(w, "no catalog", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	acc := &accountant.Accountant{Book: bk}
	if err := acc.DelRule(q.Catalog); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.provider.Save(r.Context(), bk); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.DeleteRuleResponse{}
	s.MustWrite(w, p)
}

func (s *Service) HandleReportBalance(w http.ResponseWriter, r *http.Request) {
	q := &protocol.ReportBalanceRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	acc := &accountant.Accountant{Book: bk}
	report, err := acc.ReportAccountBalance(q.Parameters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.ReportBalanceResponse{Report: report}
	s.MustWrite(w, p)
}

func (s *Service) HandleReportRegister(w http.ResponseWriter, r *http.Request) {
	q := &protocol.ReportRegisterRequest{}
	s.MustRead(r, q)

	bookId := s.getBookId(q.BookId)
	if bookId == "" {
		http.Error(w, "no book id", http.StatusBadRequest)
		return
	}

	bk, err := s.provider.Load(r.Context(), bookId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bk == nil {
		http.Error(w, "no book", http.StatusNotFound)
		return
	}

	acc := &accountant.Accountant{Book: bk}
	report, err := acc.ReportAccountRegister(q.Parameters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := &protocol.ReportRegisterResponse{Report: report}
	s.MustWrite(w, p)
}

func (s *Service) Read(r *http.Request, v any) error {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}

func (s *Service) Write(w http.ResponseWriter, v any) error {
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	return err
}

func (s *Service) MustRead(r *http.Request, v any) {
	if err := s.Read(r, v); err != nil {
		panic(err)
	}
}

func (s *Service) MustWrite(w http.ResponseWriter, v any) {
	if err := s.Write(w, v); err != nil {
		panic(err)
	}
}

func (s *Service) getBookId(bookId string) string {
	if s.Option.Book != "" {
		return s.Option.Book
	}
	return bookId
}
