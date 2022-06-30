package httpx

import (
	"fmt"
	"io"
	"os"
	"pkgx/env"
	"reflect"
	"runtime"
	"strings"
)

const (
	debugCode = iota
	testCode
	demoCode
	productionCode
)

var (
	mode     = debugCode
	modeName = env.DeployEnvDev
)

func InitDebug() {
	switch {
	case env.IsTest():
		mode = testCode
	case env.IsDemo():
		mode = demoCode
	case env.IsProduct():
		mode = productionCode
	default:
		mode = debugCode
	}

	modeName = env.DeployEnv
}

// Mode returns current mode.
func Mode() int {
	return mode
}

// ModeName returns current modeName.
func ModeName() string {
	return modeName
}

var (
	DefaultWriter      io.Writer = os.Stdout
	DefaultErrorWriter io.Writer = os.Stderr
)

func IsDebugging() bool {
	return Mode() == debugCode
}

// DebugPrintRouterFunc indicates debug log output format.
var DebugPrintRouterFunc func(httpMethod, absolutePath, handlerName string, nuHandlers int)

func debugPrintRouter(httpMethod, absolutePath string, handlers HandlersChain) {
	if IsDebugging() {
		nuHandlers := len(handlers)
		handlerName := nameOfFunction(handlers.Last())
		if DebugPrintRouterFunc == nil {
			debugPrint("%-6s %-25s --> %s (%d handlers)\n", httpMethod, absolutePath, handlerName, nuHandlers)
		} else {
			DebugPrintRouterFunc(httpMethod, absolutePath, handlerName, nuHandlers)
		}
	}
}

func debugPrintError(err error) {
	if err != nil && IsDebugging() {
		fmt.Fprintf(DefaultErrorWriter, "["+ModeName()+"] [ERROR] %v\n", err)
	}
}

func debugPrint(format string, values ...any) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		fmt.Fprintf(DefaultWriter, "["+ModeName()+"]"+format, values...)
	}
}

func nameOfFunction(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
