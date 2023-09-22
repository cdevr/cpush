//go:build ignore

package main

import (
  "fmt"
  "time"
  "strings"
  "os/exec"
  "bytes"

	"github.com/cdevr/cpush/utils"
)

func main() {
  var b = &bytes.Buffer{}

  fmt.Fprintf(b, "package main\n\nimport \"time\"\n\nvar (\n")
  fmt.Fprintf(b, "\tbuildTime = time.UnixMicro(%v)\n", time.Now().UnixMicro()) 
  cmd := exec.Command("git", "rev-parse", "HEAD")
  gitTag, _ := cmd.CombinedOutput()
  fmt.Fprintf(b, "\tbuildGitRevision = %q\n", strings.ReplaceAll(string(gitTag), "\n", ""))
  fmt.Fprintf(b, ")\n")

  utils.ReplaceFile("tags.go", b.String())
}
