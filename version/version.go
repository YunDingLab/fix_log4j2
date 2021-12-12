package version

import "fmt"

var (
	version   string
	gitCommit string
	builtAt   string
)

func GetVersion() string {
	return version
}

func GitCommit() string {
	return gitCommit
}

func BuiltAt() string {
	return builtAt
}

func Completed() string {
	return fmt.Sprintf("%s-%s", version, gitCommit)
}
