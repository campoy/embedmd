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
	"net/http"
	"net/url"
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
		{name: "url",
			in: "(http://golang.org/sample.go)", f: "http://golang.org/sample.go", l: "go"},
		{name: "bad url",
			in: "(http://golang:org:sample.go)", f: "http://golang:org:sample.go", l: "go"},
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
		dir   string
	}{
		{
			name:  "extract the whole file",
			in:    "(code.go)",
			files: map[string][]byte{"code.go": []byte(content)},
			out:   "```go\n" + string(content) + "```\n",
		},
		{
			name:  "extract the whole from a different directory",
			in:    "(code.go)",
			dir:   "sample",
			files: map[string][]byte{"sample/code.go": []byte(content)},
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
		err := extractFromFile(w, tt.in, tt.dir)
		if !eqErr(t, tt.name, err, tt.err) {
			continue
		}
		if w.String() != tt.out {
			t.Errorf("case [%s]: expected output %q; got %q", tt.name, tt.out, w.String())
		}

	}
}

func TestEmbed(t *testing.T) {
	defer func(f func(string) ([]byte, error)) { readFile = f }(readFile)
	defer func(f func(string) (file, error)) { openFile = f }(openFile)
	defer func(f func(string) (*http.Response, error)) { httpGet = f }(httpGet)

	openFile = func(string) (file, error) { return nil, os.ErrNotExist }
	err := processFile("something.md", true, false)
	eqErr(t, "no files", err, "could not open: file does not exist")

	tc := []struct {
		name  string
		in    string
		files map[string][]byte
		urls  map[string][]byte
		out   string
		err   string
		diff  bool
	}{
		{
			name: "missing file",
			in: "# This is some markdown\n" +
				"[embedmd]:# (code.go)\n" +
				"Yay!\n",
			err: "file.md:2: could not read code.go: file does not exist",
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
		{
			name: "embedding code from a URL",
			in: "# This is some markdown\n" +
				"[embedmd]:# (https://fakeurl.com/main.go)\n" +
				"Yay!\n",
			urls: map[string][]byte{"https://fakeurl.com/main.go": []byte(content)},
			out: "# This is some markdown\n" +
				"[embedmd]:# (https://fakeurl.com/main.go)\n" +
				"```go\n" +
				string(content) +
				"```\n" +
				"Yay!\n",
		},
		{
			name: "embedding code from a URL not found",
			in: "# This is some markdown\n" +
				"[embedmd]:# (https://fakeurl.com/main.go)\n" +
				"Yay!\n",
			err: "file.md:2: could not read https://fakeurl.com/main.go: status Not Found",
		},
		{
			name: "embedding code from a bad URL",
			in: "# This is some markdown\n" +
				"[embedmd]:# (https://fakeurl.com\\main.go)\n" +
				"Yay!\n",
			err: "file.md:2: could not read https://fakeurl.com\\main.go: parse https://fakeurl.com\\main.go: invalid character \"\\\\\" in host name",
		},
		{
			name: "ignore commands in code blocks",
			in: "# This is some markdown\n" +
				"```markdown\n" +
				"[embedmd]:# (nothing.md)\n" +
				"```\n" +
				"Yay!\n",
			out: "# This is some markdown\n" +
				"```markdown\n" +
				"[embedmd]:# (nothing.md)\n" +
				"```\n" +
				"Yay!\n",
		},
		{
			name: "diff generating code for first time",
			in: "# This is some markdown\n" +
				"[embedmd]:# (code.go)\n" +
				"Yay!\n",
			files: map[string][]byte{"code.go": []byte(content)},
			out: "@@ -1,3 +1,13 @@\n" +
				" # This is some markdown\n" +
				" [embedmd]:# (code.go)\n" +
				"+```go\n" +
				"+\n" +
				"+package main\n" +
				"+\n" +
				"+import \"fmt\"\n" +
				"+\n" +
				"+func main() {\n" +
				"+        fmt.Println(\"hello, test\")\n" +
				"+}\n" +
				"+```\n" +
				" Yay!\n",
			diff: true,
		},
	}

	for _, tt := range tc {
		readFile = fakeReadFile(tt.files)
		f := newFakeFile(tt.in)
		openFile = func(name string) (file, error) { return f, nil }
		httpGet = fakeHTTPGet(tt.urls)
		if tt.diff {
			tt.files["file.md"] = []byte(tt.in)
			defer func(w io.Writer) { stdout = w }(stdout)
			stdout = &bytes.Buffer{}
		}
		err := embed([]string{"file.md"}, true, tt.diff)
		if !eqErr(t, tt.name, err, tt.err) {
			continue
		}
		out := f.buf.String()
		if tt.diff {
			out = stdout.(*bytes.Buffer).String()
		}
		if tt.out != out {
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
		t.Errorf("case [%s]: expected error message %q; but got %q", id, msg, err)
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
func (f *fakeFile) Truncate(int64) error                        { return nil }

func newFakeFile(s string) *fakeFile {
	return &fakeFile{ReadCloser: ioutil.NopCloser(strings.NewReader(s))}
}

func fakeHTTPGet(urls map[string][]byte) func(string) (*http.Response, error) {
	return func(path string) (*http.Response, error) {
		_, err := url.Parse(path)
		if err != nil {
			return nil, err
		}

		// I could use httptest.ResponseRecorder but the method Result is only
		// available since go1.7.
		b, ok := urls[path]
		if !ok {
			return &http.Response{
				Status:     "Not Found",
				StatusCode: http.StatusNotFound,
				Body:       ioutil.NopCloser(strings.NewReader("Not Found")),
			}, nil
		}
		return &http.Response{
			Status:     "OK",
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewReader(b)),
		}, nil
	}
}
