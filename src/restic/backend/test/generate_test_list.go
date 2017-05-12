// +build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"text/template"
	"unicode"
	"unicode/utf8"
)

var data struct {
	Package string
	Funcs   []string
}

var testTemplate = `
// DO NOT EDIT, AUTOMATICALLY GENERATED
package {{ .Package }}

import (
	"testing"
)

var testFunctions = []struct {
	Name string
	Fn   func(t testing.TB, suite *Suite)
}{
{{ range $f := .Funcs -}}
	{"{{ $f }}", BackendTest{{ $f }},},
{{ end }}
}
`

var testFile = flag.String("testfile", "tests.go", "file to search test functions in")
var outputFile = flag.String("output", "funcs.go", "output file to write generated code to")
var packageName = flag.String("package", "", "the package name to use")
var prefix = flag.String("prefix", "", "test function prefix")
var quiet = flag.Bool("quiet", false, "be quiet")

func errx(err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

var funcRegex = regexp.MustCompile(`^func\s+BackendTest(.+)\s*\(`)

func findTestFunctions() (funcs []string) {
	f, err := os.Open(*testFile)
	errx(err)

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		match := funcRegex.FindStringSubmatch(sc.Text())
		if len(match) > 0 {
			funcs = append(funcs, match[1])
		}
	}

	if err := sc.Err(); err != nil {
		log.Fatalf("Error scanning file: %v", err)
	}

	errx(f.Close())
	return funcs
}

func generateOutput(wr io.Writer, data interface{}) {
	t := template.Must(template.New("backendtest").Parse(testTemplate))

	cmd := exec.Command("gofmt")
	cmd.Stdout = wr
	in, err := cmd.StdinPipe()
	errx(err)
	errx(cmd.Start())
	errx(t.Execute(in, data))
	errx(in.Close())
	errx(cmd.Wait())
}

func packageTestFunctionPrefix(pkg string) string {
	if pkg == "" {
		return ""
	}

	r, n := utf8.DecodeRuneInString(pkg)
	return string(unicode.ToUpper(r)) + pkg[n:]
}

func init() {
	flag.Parse()
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Getwd() %v\n", err)
		os.Exit(1)
	}

	pkg := *packageName
	if pkg == "" {
		pkg = filepath.Base(dir)
	}

	f, err := os.Create(*outputFile)
	errx(err)

	data.Package = pkg
	data.Funcs = findTestFunctions()
	generateOutput(f, data)

	errx(f.Close())

	if !*quiet {
		fmt.Printf("wrote backend tests for package %v to %v\n", data.Package, *outputFile)
	}
}
