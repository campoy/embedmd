// Copyright 2016 Google Inc. All rights reserved.
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
	cases := []struct {
		in   string
		f, l string
		s, e *string
		err  string
	}{
		{in: "(code.go /start/ /end/)", f: "code.go", l: "go", s: ptr("start"), e: ptr("end")},
		{in: "(code.go     /start/)", f: "code.go", l: "go", s: ptr("start")},
		{in: "   \t  (code.go)  \t  ", f: "code.go", l: "go"},
		{in: "{code.go}", err: "argument list should be in parenthesis"},
		{in: "(code.go", err: "argument list should be in parenthesis"},
		{in: "(code.go /start)", err: "malformed regular expression"},
		{in: "(code.go /start/ /end)", err: "malformed regular expression"},
		{in: "(code.go /start/ /end/ /)", err: "malformed regular expression"},
		{in: "(test.md markdown)", f: "test.md", l: "markdown"},
	}

	for i, c := range cases {
		f, l, s, e, err := parseArgs(c.in)
		if !eqErr(t, i, err, c.err) {
			continue
		}
		if f != c.f {
			t.Errorf("case %d: expected file %q from %s; got %q", i, c.f, c.in, f)
		}
		if l != c.l {
			t.Errorf("case %d: expected language %q from %s; got %q", i, c.l, c.in, l)
		}
		if !eqPtr(s, c.s) {
			t.Errorf("case %d: expected start %v from %s; got %v", i, str(c.s), c.in, str(s))
		}
		if !eqPtr(e, c.e) {
			t.Errorf("case %d: expected end %v from %s; got %v", i, str(c.e), c.in, str(e))
		}
	}
}

var content = []byte(`
package main

import "fmt"

func main() {
        fmt.Println("hello, test")
}
`)

func TestExtract(t *testing.T) {
	cases := []struct {
		start, end *string
		out        string
		err        string
	}{
		{out: string(content)},
		{start: ptr("func main"), out: "func main() {"},
		{start: ptr("package main"), end: ptr(""), out: string(content[1:])},
		{start: ptr("("), err: "error parsing regexp: missing closing ): `(`"},
		{start: ptr("gopher"), err: "could not match \"gopher\""},

		{start: ptr("fmt.P"), end: ptr("hello"), out: "fmt.Println(\"hello"},
		{start: ptr("fmt.P"), end: ptr("hello.*"), out: "fmt.Println(\"hello, test\")"},
		{start: ptr("func main"), end: ptr("}"), out: "func main() {\n        fmt.Println(\"hello, test\")\n}"},
		{start: ptr("fmt.P"), end: ptr(")"), err: "error parsing regexp: unexpected ): `)`"},
	}

	for i, c := range cases {
		b, err := extract(content, c.start, c.end)
		if !eqErr(t, i, err, c.err) {
			continue
		}
		if string(b) != c.out {
			t.Errorf("case %d: expected extracting %q; got %q", i, c.out, b)
		}
	}
}

func TestExtractFromFile(t *testing.T) {
	defer func(f func(string) ([]byte, error)) { readFile = f }(readFile)

	cases := []struct {
		in    string
		files map[string][]byte
		out   string
		err   string
	}{
		{
			in:    "(code.go)",
			files: map[string][]byte{"code.go": content},
			out:   "```go\n" + string(content) + "```\n",
		},
		{
			in:  "(code.go)",
			err: "could not read code.go: file does not exist",
		},
		{
			in:  "wrong",
			err: "argument list should be in parenthesis",
		},
		{
			in:    "(code.go /potato/)",
			files: map[string][]byte{"code.go": content},
			err:   "could not extract content from code.go: could not match \"potato\"",
		},
	}

	for i, c := range cases {
		readFile = fakeReadFile(c.files)
		w := new(bytes.Buffer)
		err := extractFromFile(w, c.in)
		if !eqErr(t, i, err, c.err) {
			continue
		}
		if w.String() != c.out {
			t.Errorf("case %d: expected output %q; got %q", i, w.String(), c.out)
		}

	}
}

func TestProcess(t *testing.T) {
	defer func(f func(string) ([]byte, error)) { readFile = f }(readFile)
	defer func(f func(string) (file, error)) { openFile = f }(openFile)

	openFile = func(string) (file, error) { return nil, os.ErrNotExist }
	err := process("something.md")
	eqErr(t, -1, err, "could not open something.md: file does not exist")

	cases := []struct {
		in    string
		files map[string][]byte
		out   string
		err   string
	}{
		{
			in: "# This is some markdown\n" +
				"[embed.md]:# (code.go)\n" +
				"Yay!\n",
			err: "could not read code.go: file does not exist",
		},
		{
			in: "# This is some markdown\n" +
				"[embed.md]:# (code.go)\n" +
				"Yay!\n",
			files: map[string][]byte{"code.go": content},
			out: "# This is some markdown\n" +
				"[embed.md]:# (code.go)\n" +
				"```go\n" +
				string(content) +
				"```\n" +
				"Yay!\n",
		},
		{
			in: "# This is some markdown\n" +
				"[embed.md]:# (code.go)\n" +
				"```go\n" +
				string(content) +
				"```\n" +
				"Yay!\n",
			files: map[string][]byte{"code.go": content},
			out: "# This is some markdown\n" +
				"[embed.md]:# (code.go)\n" +
				"```go\n" +
				string(content) +
				"```\n" +
				"Yay!\n",
		},
	}

	for i, c := range cases {
		readFile = fakeReadFile(c.files)
		f := newFakeFile(c.in)
		openFile = func(name string) (file, error) { return f, nil }

		err := process("anyfile.md")
		if !eqErr(t, i, err, c.err) {
			continue
		}
		if out := f.buf.String(); c.out != out {
			t.Errorf("case %d: expected output:\n###\n%s\n###; got###\n%s\n###", i, c.out, out)
		}
	}
}

func eqErr(t *testing.T, id int, err error, msg string) bool {
	if err == nil && msg != "" {
		t.Errorf("case %d: expected error message %q; but got nothing", id, msg)
		return false
	}
	if err != nil && msg != err.Error() {
		t.Errorf("case %d: expected error message %q; but got %v", id, msg, err)
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
