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

package embedmd

import "testing"

func TestParseCommand(t *testing.T) {
	tc := []struct {
		name string
		in   string
		cmd  command
		err  string
	}{
		{name: "start to end",
			in:  "(code.go /start/ /end/)",
			cmd: command{path: "code.go", lang: "go", start: ptr("/start/"), end: ptr("/end/")}},
		{name: "only start",
			in:  "(code.go     /start/)",
			cmd: command{path: "code.go", lang: "go", start: ptr("/start/")}},
		{name: "empty list",
			in:  "()",
			err: "missing file name"},
		{name: "file with no extension and no lang",
			in:  "(test)",
			err: "language is required when file has no extension"},
		{name: "surrounding blanks",
			in:  "   \t  (code.go)  \t  ",
			cmd: command{path: "code.go", lang: "go"}},
		{name: "no parenthesis",
			in:  "{code.go}",
			err: "argument list should be in parenthesis"},
		{name: "only left parenthesis",
			in:  "(code.go",
			err: "argument list should be in parenthesis"},
		{name: "regexp not closed",
			in:  "(code.go /start)",
			err: "unbalanced /"},
		{name: "end regexp not closed",
			in:  "(code.go /start/ /end)",
			err: "unbalanced /"},
		{name: "file name and language",
			in:  "(test.md markdown)",
			cmd: command{path: "test.md", lang: "markdown"}},
		{name: "multi-line comments",
			in:  `(doc.go /\/\*/ /\*\//)`,
			cmd: command{path: "doc.go", lang: "go", start: ptr(`/\/\*/`), end: ptr(`/\*\//`)}},
		{name: "using $ as end",
			in:  "(foo.go /start/ $)",
			cmd: command{path: "foo.go", lang: "go", start: ptr("/start/"), end: ptr("$")}},
		{name: "extra arguments",
			in: "(foo.go /start/ $ extra)", err: "too many arguments"},
		{name: "file name with directories",
			in:  "(foo/bar.go)",
			cmd: command{path: "foo/bar.go", lang: "go"}},
		{name: "url",
			in:  "(http://golang.org/sample.go)",
			cmd: command{path: "http://golang.org/sample.go", lang: "go"}},
		{name: "bad url",
			in:  "(http://golang:org:sample.go)",
			cmd: command{path: "http://golang:org:sample.go", lang: "go"}},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parseCommand(tt.in)
			if !eqErr(t, tt.name, err, tt.err) {
				return
			}

			want, got := tt.cmd, *cmd
			if want.path != got.path {
				t.Errorf("case [%s]: expected file %q; got %q", tt.name, want.path, got.path)
			}
			if want.lang != got.lang {
				t.Errorf("case [%s]: expected language %q; got %q", tt.name, want.lang, got.lang)
			}
			if !eqPtr(want.start, got.start) {
				t.Errorf("case [%s]: expected start %v; got %v", tt.name, str(want.start), str(got.start))
			}
			if !eqPtr(want.end, got.end) {
				t.Errorf("case [%s]: expected end %v; got %v", tt.name, str(want.end), str(got.end))
			}
		})
	}
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
