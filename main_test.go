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
	"os"
	"strings"
	"testing"
)

func TestEmbedStreams(t *testing.T) {
	tc := []struct {
		name      string
		in, out   string
		err       string
		d, w      bool
		foundDiff bool
	}{
		{name: "just some text",
			in:  "# hello\ntest\n",
			out: "# hello\ntest\n",
		},
		{name: "can't rewrite stdin",
			w:   true,
			err: "error: cannot use -w with standard input",
		},
		{name: "can't diff and rewrite",
			w: true, d: true,
			err: "error: cannot use -w and -d simultaneously",
		},
		{name: "empty diff",
			d:         true,
			in:        "# hello\ntest\n",
			foundDiff: false,
		},
		{name: "non empty diff",
			d:  true,
			in: "# hello\ntest",
			out: `@@ -1,2 +1,3 @@
 # hello
 test
+
`,
			foundDiff: true,
		},
	}

	defer func(r io.Reader, w io.Writer) { stdin, stdout = r, w }(stdin, stdout)

	for _, tt := range tc {
		stdin = strings.NewReader(tt.in)
		buf := &bytes.Buffer{}
		stdout = buf
		foundDiff, err := embed(nil, tt.w, tt.d)
		if !eqErr(t, tt.name, err, tt.err) {
			continue
		}
		if got := buf.String(); tt.out != got {
			t.Errorf("case [%s] expected output\n%q\n; got\n%q", tt.name, tt.out, got)
		}
		if tt.d && foundDiff != tt.foundDiff {
			if foundDiff {
				t.Errorf("case [%s] expected to find a diff, but didn't", tt.name)
			} else {
				t.Errorf("case [%s] didn't expect to find a diff, but did", tt.name)
			}
		}
	}
}

func TestEmbedFiles(t *testing.T) {
	tc := []struct {
		name string
		in   string
		out  string
		err  string
		d, w bool
	}{
		{name: "rewriting a single file",
			in:  "one\ntwo\nthree",
			w:   true,
			out: "one\ntwo\nthree\n",
		},
		{name: "diffing a single file",
			in:  "one\ntwo\nthree",
			d:   true,
			out: "@@ -1 +1,4 @@\n+one\n+two\n+three\n \n",
		},
	}

	defer func(f func(string) (file, error)) { openFile = f }(openFile)

	for _, tt := range tc {
		f := newFakeFile(tt.in)
		openFile = func(path string) (file, error) { return f, nil }
		stdout = os.Stdout
		if tt.d {
			stdout = &f.buf
		}

		_, err := embed([]string{"docs.md"}, tt.w, tt.d)
		if !eqErr(t, tt.name, err, tt.err) {
			continue
		}
		if got := f.buf.String(); tt.out != got {
			t.Errorf("case [%s]: expected output \n%q; got\n%q", tt.name, tt.out, got)
		}

	}
}

func eqErr(t *testing.T, id string, err error, msg string) bool {
	if err == nil && msg == "" {
		return true
	}
	if err == nil && msg != "" {
		t.Errorf("case [%s]: expected error message %q; but got nothing", id, msg)
		return false
	}
	if err != nil && msg != err.Error() {
		t.Errorf("case [%s]: expected error message %q; but got %q", id, msg, err)
	}
	return false
}

type fakeFile struct {
	io.ReadCloser
	buf bytes.Buffer
}

func (f *fakeFile) WriteAt(b []byte, offset int64) (int, error) { return f.buf.Write(b) }
func (f *fakeFile) Truncate(int64) error                        { return nil }

func newFakeFile(s string) *fakeFile {
	return &fakeFile{ReadCloser: io.NopCloser(strings.NewReader(s))}
}
