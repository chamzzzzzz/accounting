package protocol

import (
	"github.com/chamzzzzzz/accounting/account"
	"github.com/chamzzzzzz/accounting/report"
	"github.com/chamzzzzzz/accounting/rule"
	"github.com/chamzzzzzz/accounting/sourcedocument"
	"github.com/chamzzzzzz/accounting/voucher"
)

type Document struct {
	Type string `json:"type,omitempty"`
	Data string `json:"data,omitempty"`
}

type CreateBookRequest struct {
	Id    string `json:"id,omitempty"`
	Title string `json:"title,omitempty"`
}

type CreateBookResponse struct {
	Id string `json:"id,omitempty"`
}

type GetAccountRequest struct {
	BookId string `json:"book_id,omitempty"`
}

type GetAccountResponse struct {
	Accounts []*account.Account `json:"accounts,omitempty"`
}

type AddAccountRequest struct {
	BookId  string           `json:"book_id,omitempty"`
	Account *account.Account `json:"account,omitempty"`
}

type AddAccountResponse struct {
}

type UpdateAccountRequest struct {
	BookId  string           `json:"book_id,omitempty"`
	Account *account.Account `json:"account,omitempty"`
}

type UpdateAccountResponse struct {
}

type DeleteAccountRequest struct {
	BookId string `json:"book_id,omitempty"`
	Title  string `json:"title,omitempty"`
}

type DeleteAccountResponse struct {
}

type ScanSourceDocumentRequest struct {
	Document *Document `json:"document,omitempty"`
}

type ScanSourceDocumentResponse struct {
	SourceDocument *sourcedocument.SourceDocument `json:"source_document,omitempty"`
}

type PrepareVoucherRequest struct {
	BookId         string                         `json:"book_id,omitempty"`
	SourceDocument *sourcedocument.SourceDocument `json:"source_document,omitempty"`
}

type PrepareVoucherResponse struct {
	Vouchers []*voucher.Voucher `json:"vouchers,omitempty"`
}

type AddVoucherRequest struct {
	BookId  string           `json:"book_id,omitempty"`
	Voucher *voucher.Voucher `json:"voucher,omitempty"`
}

type AddVoucherResponse struct {
}

type GetRuleRequest struct {
	BookId string `json:"book_id,omitempty"`
}

type GetRuleResponse struct {
	Rules []*rule.Rule `json:"rules,omitempty"`
}

type AddRuleRequest struct {
	BookId string     `json:"book_id,omitempty"`
	Rule   *rule.Rule `json:"rule,omitempty"`
}

type AddRuleResponse struct {
}

type DeleteRuleRequest struct {
	BookId  string `json:"book_id,omitempty"`
	Catalog string `json:"catalog,omitempty"`
}

type DeleteRuleResponse struct {
}

type ReportBalanceRequest struct {
	BookId     string                   `json:"book_id,omitempty"`
	Parameters *report.ReportParameters `json:"parameters,omitempty"`
}

type ReportBalanceResponse struct {
	Report *report.AccountBalanceReport `json:"report,omitempty"`
}

type ReportRegisterRequest struct {
	BookId     string                   `json:"book_id,omitempty"`
	Parameters *report.ReportParameters `json:"parameters,omitempty"`
}

type ReportRegisterResponse struct {
	Report *report.AccountRegisterReport `json:"report,omitempty"`
}
