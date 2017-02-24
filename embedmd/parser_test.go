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

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	tc := []struct {
		name string
		in   string
		out  string
		run  commandRunner
		err  string
	}{
		{
			name: "empty file",
			in:   "",
			out:  "",
		},
		{
			name: "just text",
			in:   "one\ntwo\nthree\n",
			out:  "one\ntwo\nthree\n",
		},
		{
			name: "a command",
			in:   "one\n[embedmd]:# (code.go)",
			out:  "one\n[embedmd]:# (code.go)\nOK\n",
			run: func(w io.Writer, cmd *command) error {
				if cmd.path != "code.go" {
					return fmt.Errorf("bad command")
				}
				fmt.Fprint(w, "OK\n")
				return nil
			},
		},
		{
			name: "a command then some text",
			in:   "one\n[embedmd]:# (code.go)\nYay\n",
			out:  "one\n[embedmd]:# (code.go)\nOK\nYay\n",
			run: func(w io.Writer, cmd *command) error {
				if cmd.path != "code.go" {
					return fmt.Errorf("bad command")
				}
				fmt.Fprint(w, "OK\n")
				return nil
			},
		},
		{
			name: "a bad command",
			in:   "one\n[embedmd]:# (code\n",
			err:  "2: argument list should be in parenthesis",
		},
		{
			name: "an ignored command",
			in:   "one\n```\n[embedmd]:# (code.go)\n```\n",
			out:  "one\n```\n[embedmd]:# (code.go)\n```\n",
		},
		{
			name: "unbalanced code section",
			in:   "one\n```\nsome code\n",
			err:  "3: unbalanced code section",
		},
		{
			name: "two contiguous code sections",
			in:   "\n```go\nhello\n```\n```go\nbye\n```\n",
			out:  "\n```go\nhello\n```\n```go\nbye\n```\n",
		},
		{
			name: "two non contiguous code sections",
			in:   "```go\nhello\n```\n\n```go\nbye\n```\n",
			out:  "```go\nhello\n```\n\n```go\nbye\n```\n",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := process(&out, strings.NewReader(tt.in), tt.run)
			if !eqErr(t, tt.name, err, tt.err) {
				return
			}
			if got := out.String(); got != tt.out {
				t.Errorf("case [%s] expected %q; got %q", tt.name, tt.out, got)
			}
		})
	}
}
