package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/chamzzzzzz/accounting"
	"github.com/chamzzzzzz/accounting/analyzer"
)

func main() {
	recognizer, err := accounting.NewRecognizer("macOCR")
	if err != nil {
		panic(err)
	}

	analyzers := []accounting.Analyzer{
		&analyzer.Alipay{},
		&analyzer.WechatPay{},
		&analyzer.UnionPay{},
	}

	accountant := accounting.Accountant{
		Recognizer: recognizer,
		Analyzers:  analyzers,
	}

	source, err := accountant.Recognize(os.Args[1])
	if err != nil {
		slog.Error("recognize source", "error", err)
		return
	}

	sourcedocument, err := accountant.Review(source)
	if err != nil {
		slog.Error("review source", "error", err)
		return
	}

	for _, item := range sourcedocument.Source.Items {
		fmt.Printf("%+v\n", item)
	}
	fmt.Printf("%+v\n", sourcedocument)
}
