package errors

import (
	"fmt"
	"strconv"
)

var _ error = &Errorx{}

var (
	_codes = map[string]Errorx{}
)

const (
	ErrorxModDef int8 = 1 + iota
	ErrorxModCode
	ErrorxModCodePrefix
)

type Errorx struct {
	code    int
	prefix  string
	message string
	mod     int8
}

type Code string

func NewErrorx(code int, message string) Errorx {
	return add(code, message, "", ErrorxModDef)
}

func NewErrorxCode(code int, message string) Errorx {
	return add(code, message, "", errModCode)
}

func NewErrorxPrefix(code int, message string, prefix string) Errorx {
	return add(code, message, prefix, errModCode)
}

func (e *Errorx) Error() string {
	switch e.mod {
	case ErrorxModDef:
		return e.message
	case ErrorxModCode:
		return fmt.Sprintf("errors code: %d, %s", e.code, e.message)
	case ErrorxModCodePrefix:
		return fmt.Sprintf("errors code: %d, %s: %s", e.code, e.prefix, e.message)
	}
	return e.message
}

func (e *Errorx) Code() int {
	return e.code
}

func add(code int, message string, prefix string, mod int8) Errorx {
	cMessage := format(code, message)
	if _, exists := _codes[cMessage]; exists {
		panic(fmt.Sprintf("code: %d already exist", code))
	}
	_codes[cMessage] = Errorx{
		code:    code,
		prefix:  prefix,
		message: message,
		mod:     mod,
	}
	return _codes[cMessage]
}

func format(code int, message string) string {
	return strconv.FormatInt(int64(code), 10) + ":" + message
}
