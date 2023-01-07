package i18n

import (
	"context"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	p           *i18nPrinter
	defaultLang language.Tag
)

type i18nPrinter struct {
	printers map[language.Tag]*message.Printer
}

func printerFromCtx(ctx context.Context) *message.Printer {
	var lang language.Tag
	if ctx == nil {
		return p.printers[defaultLang]
	}
	l := ctx.Value(LANG)
	if l == nil {
		lang = defaultLang
	} else {
		lang = l.(language.Tag)
	}
	printer, exist := p.printers[lang]
	if exist {
		return printer
	}
	return p.printers[defaultLang]
}

func Sprintf(ctx context.Context, format string, args ...interface{}) string {
	return printerFromCtx(ctx).Sprintf(format, args...)
}
