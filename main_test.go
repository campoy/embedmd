// Copyright 2016 Google Intt. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to writing, software distributed
// under the License is distributed on a "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tc := []struct {
		name string
		in   string
		f, l string
		s, e *string
		err  string
	}{
		{name: "start to end",
			in: "(code.go /start/ /end/)", f: "code.go", l: "go", s: ptr("/start/"), e: ptr("/end/")},
		{name: "only start",
			in: "(code.go     /start/)", f: "code.go", l: "go", s: ptr("/start/")},
		{name: "empty list",
			in: "()", err: "missing file name"},
		{name: "file with no extension and no lang",
			in: "(test)", err: "language is required when file has no extension"},
		{name: "surrounding blanks",
			in: "   \t  (code.go)  \t  ", f: "code.go", l: "go"},
		{name: "no parenthesis",
			in: "{code.go}", err: "argument list should be in parenthesis"},
		{name: "only left parenthesis",
			in: "(code.go", err: "argument list should be in parenthesis"},
		{name: "regexp not closed",
			in: "(code.go /start)", err: "unbalanced /"},
		{name: "end regexp not closed",
			in: "(code.go /start/ /end)", err: "unbalanced /"},
		{name: "file name and language",
			in: "(test.md markdown)", f: "test.md", l: "markdown"},
		{name: "using $ as end",
			in: "(foo.go /start/ $)", f: "foo.go", l: "go", s: ptr("/start/"), e: ptr("$")},
		{name: "extra arguments",
			in: "(foo.go /start/ $ extra)", err: "too many arguments"},
		{name: "file name with directories",
			in: "(foo/bar.go)", f: "foo/bar.go", l: "go"},
	}

	for _, tt := range tc {
		f, l, s, e, err := parseArgs(tt.in)
		if !eqErr(t, tt.name, err, tt.err) {
			continue
		}
		if f != tt.f {
			t.Errorf("case [%s]: expected file %q; got %q", tt.name, tt.f, f)
		}
		if l != tt.l {
			t.Errorf("case [%s]: expected language %q; got %q", tt.name, tt.l, l)
		}
		if !eqPtr(s, tt.s) {
			t.Errorf("case [%s]: expected start %v; got %v", tt.name, str(tt.s), str(s))
		}
		if !eqPtr(e, tt.e) {
			t.Errorf("case [%s]: expected end %v; got %v", tt.name, str(tt.e), str(e))
		}
	}
}

const content = `
package main

import "fmt"

func main() {
        fmt.Println("hello, test")
}
`

func TestExtract(t *testing.T) {
	tc := []struct {
		name       string
		start, end *string
		out        string
		err        string
	}{
		{name: "no limits",
			out: string(content)},
		{name: "only one line",
			start: ptr("/func main.*\n/"), out: "func main() {\n"},
		{name: "from package to end",
			start: ptr("/package main/"), end: ptr("$"), out: string(content[1:])},
		{name: "not matching",
			start: ptr("/gopher/"), err: "could not match \"/gopher/\""},
		{name: "part of a line",
			start: ptr("/fmt.P/"), end: ptr("/hello/"), out: "fmt.Println(\"hello"},
		{name: "function call",
			start: ptr("/fmt\\.[^()]*/"), out: "fmt.Println"},
		{name: "from fmt to end of line",
			start: ptr("/fmt.P.*\n/"), out: "fmt.Println(\"hello, test\")\n"},
		{name: "from func to end of next line",
			start: ptr("/func/"), end: ptr("/Println.*\n/"), out: "func main() {\n        fmt.Println(\"hello, test\")\n"},
		{name: "from func to }",
			start: ptr("/func main/"), end: ptr("/}/"), out: "func main() {\n        fmt.Println(\"hello, test\")\n}"},

		{name: "bad start regexp",
			start: ptr("/(/"), err: "error parsing regexp: missing closing ): `(`"},
		{name: "bad regexp",
			start: ptr("something"), err: "missing slashes (/) around \"something\""},
		{name: "bad end regexp",
			start: ptr("/fmt.P/"), end: ptr("/)/"), err: "error parsing regexp: unexpected ): `)`"},
	}

	for _, tt := range tc {
		b, err := extract([]byte(content), tt.start, tt.end)
		if !eqErr(t, tt.name, err, tt.err) {
			continue
		}
		if string(b) != tt.out {
			t.Errorf("case [%s]: expected extracting %q; got %q", tt.name, tt.out, b)
		}
	}
}

func TestExtractFromFile(t *testing.T) {
	defer func(f func(string) ([]byte, error)) { readFile = f }(readFile)

	tc := []struct {
		name  string
		in    string
		files map[string][]byte
		out   string
		err   string
	}{
		{
			name:  "extract the whole file",
			in:    "(code.go)",
			files: map[string][]byte{"code.go": []byte(content)},
			out:   "```go\n" + string(content) + "```\n",
		},
		{
			name:  "added line break",
			in:    "(code.go /fmt\\.Println/)",
			files: map[string][]byte{"code.go": []byte(content)},
			out:   "```go\nfmt.Println\n```\n",
		},
		{
			name: "missing file",
			in:   "(code.go)",
			err:  "could not read code.go: file does not exist",
		},
		{
			name: "bad argument",
			in:   "wrong",
			err:  "argument list should be in parenthesis",
		},
		{
			name:  "unmatched regexp",
			in:    "(code.go /potato/)",
			files: map[string][]byte{"code.go": []byte(content)},
			err:   "could not extract content from code.go: could not match \"/potato/\"",
		},
	}

	for _, tt := range tc {
		readFile = fakeReadFile(tt.files)
		w := new(bytes.Buffer)
		err := extractFromFile(w, tt.in)
		if !eqErr(t, tt.name, err, tt.err) {
			continue
		}
		if w.String() != tt.out {
			t.Errorf("case [%s]: expected output %q; got %q", tt.name, tt.out, w.String())
		}

	}
}

func TestProcess(t *testing.T) {
	defer func(f func(string) ([]byte, error)) { readFile = f }(readFile)
	defer func(f func(string) (file, error)) { openFile = f }(openFile)

	openFile = func(string) (file, error) { return nil, os.ErrNotExist }
	err := processFile("something.md", true)
	eqErr(t, "no files", err, "could not open: file does not exist")

	tc := []struct {
		name  string
		in    string
		files map[string][]byte
		out   string
		err   string
	}{
		{
			name: "missing file",
			in: "# This is some markdown\n" +
				"[embedmd]:# (code.go)\n" +
				"Yay!\n",
			err: "could not read code.go: file does not exist at line 2",
		},
		{
			name: "generating code for first time",
			in: "# This is some markdown\n" +
				"[embedmd]:# (code.go)\n" +
				"Yay!\n",
			files: map[string][]byte{"code.go": []byte(content)},
			out: "# This is some markdown\n" +
				"[embedmd]:# (code.go)\n" +
				"```go\n" +
				string(content) +
				"```\n" +
				"Yay!\n",
		},
		{
			name: "replacing existing code",
			in: "# This is some markdown\n" +
				"[embedmd]:# (code.go)\n" +
				"```go\n" +
				string(content) +
				"```\n" +
				"Yay!\n",
			files: map[string][]byte{"code.go": []byte(content)},
			out: "# This is some markdown\n" +
				"[embedmd]:# (code.go)\n" +
				"```go\n" +
				string(content) +
				"```\n" +
				"Yay!\n",
		},
	}

	for _, tt := range tc {
		readFile = fakeReadFile(tt.files)
		f := newFakeFile(tt.in)
		openFile = func(name string) (file, error) { return f, nil }

		err := processFile("anyfile.md", true)
		if !eqErr(t, tt.name, err, tt.err) {
			continue
		}
		if out := f.buf.String(); tt.out != out {
			t.Errorf("case [%s]: expected output:\n###\n%s\n###; got###\n%s\n###", tt.name, tt.out, out)
		}
	}
}

func eqErr(t *testing.T, id string, err error, msg string) bool {
	if err == nil && msg != "" {
		t.Errorf("case [%s]: expected error message %q; but got nothing", id, msg)
		return false
	}
	if err != nil && msg != err.Error() {
		t.Errorf("case [%s]: expected error message %q; but got %v", id, msg, err)
		return false
	}
	return true
}

func ptr(s string) *string { return &s }
func str(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
func eqPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func fakeReadFile(files map[string][]byte) func(string) ([]byte, error) {
	return func(name string) ([]byte, error) {
		if f, ok := files[name]; ok {
			return f, nil
		}
		return nil, os.ErrNotExist
	}
}

type fakeFile struct {
	io.ReadCloser
	buf bytes.Buffer
}

func (f *fakeFile) WriteAt(b []byte, offset int64) (int, error) { return f.buf.Write(b) }

func newFakeFile(s string) *fakeFile {
	return &fakeFile{ReadCloser: ioutil.NopCloser(strings.NewReader(s))}
}
