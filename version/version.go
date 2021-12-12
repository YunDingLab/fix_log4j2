package version

import (
	"fmt"
	"io"
)

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

func Fprint(w io.Writer) {
	fmt.Fprintf(w, "Version:\t%s\n", version)
	fmt.Fprintf(w, "Git Commit:\t%s\n", gitCommit)
	fmt.Fprintf(w, "Built At:\t%s\n", builtAt)
}
