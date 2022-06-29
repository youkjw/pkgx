package env

import (
	"flag"
	"os"
	"strings"
)

const (
	DeployEnvDev  = "develop"    //开发
	DeployEnvTest = "test"       //测试
	DeployEnvDemo = "demo"       //demo
	DeployEnvProd = "production" //生产
)

var (
	DeployEnv string // 环境标识 develop-开发、test-测试、prod-生产
)

func init() {
	addFlag(flag.CommandLine)
}

func IsDevelop() bool {
	return Value() == DeployEnvDev
}

func IsProduct() bool {
	return Value() == DeployEnvProd
}

func IsTest() bool {
	return Value() == DeployEnvTest
}

func IsDemo() bool {
	return Value() == DeployEnvDemo
}

func Value() string {
	return strings.ToLower(DeployEnv)
}

func addFlag(fs *flag.FlagSet) {
	fs.StringVar(&DeployEnv, "deploy_env", defaultString("DEPLOY_ENV", DeployEnvDev), "deploy_env identifies the runtime environment")
}

func defaultString(key, value string) string {
	v := os.Getenv(key)
	if v == "" {
		return value
	}

	return v
}
