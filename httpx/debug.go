package httpx

import (
	"fmt"
	"io"
	"os"
	"pkgx/env"
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

func init() {
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

func debugPrint(format string, values ...any) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		fmt.Fprintf(DefaultWriter, "["+ModeName()+"] "+format, values...)
	}
}
