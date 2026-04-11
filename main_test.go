// Copyright 2016 Google Intt. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/campoy/embedmd/internal/testutil"
)

// fakeInfo is a minimal fs.FileInfo whose Mode returns a fixed value.
type fakeInfo struct{ mode fs.FileMode }

func (f fakeInfo) Name() string      { return "" }
func (f fakeInfo) Size() int64       { return 0 }
func (f fakeInfo) Mode() fs.FileMode { return f.mode }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool       { return false }
func (f fakeInfo) Sys() interface{}  { return nil }

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
		if !testutil.EqErr(t, tt.name, err, tt.err) {
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
			out: "@@ -1,3 +1,4 @@\n one\n two\n three\n+\n",
		},
	}

	defer func(f func(string) (file, error)) { openFile = f }(openFile)
	defer func(f func(string) (fs.FileInfo, error)) { statFile = f }(statFile)
	defer func(f func(string, fs.FileMode) error) { chmodFile = f }(chmodFile)

	statFile = func(string) (fs.FileInfo, error) { return fakeInfo{mode: 0644}, nil }
	chmodFile = func(string, fs.FileMode) error { return nil }

	for _, tt := range tc {
		f := newFakeFile(tt.in)
		openFile = func(path string) (file, error) { return f, nil }
		stdout = os.Stdout
		if tt.d {
			stdout = &f.buf
		}

		_, err := embed([]string{"docs.md"}, tt.w, tt.d)
		if !testutil.EqErr(t, tt.name, err, tt.err) {
			continue
		}
		if got := f.buf.String(); tt.out != got {
			t.Errorf("case [%s]: expected output \n%q; got\n%q", tt.name, tt.out, got)
		}

	}
}

// TestFilePermissionsPreserved verifies that processFile restores the original
// file permissions after a rewrite, even if the underlying write mechanism
// were to change (e.g. to a temp-file + rename pattern).
func TestFilePermissionsPreserved(t *testing.T) {
	// Create a real temp file with restrictive permissions (0600).
	f, err := os.CreateTemp(t.TempDir(), "embedmd-perm-*.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("hello\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	want := fs.FileMode(0600)
	if err := os.Chmod(f.Name(), want); err != nil {
		t.Fatal(err)
	}

	defer func(fn func(string) (file, error)) { openFile = fn }(openFile)
	openFile = func(name string) (file, error) {
		return os.OpenFile(name, os.O_RDWR, 0)
	}

	if _, err := processFile(f.Name(), true, false); err != nil {
		t.Fatalf("processFile: %v", err)
	}

	info, err := os.Stat(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode(); got != want {
		t.Errorf("permissions changed: got %v, want %v", got, want)
	}
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
