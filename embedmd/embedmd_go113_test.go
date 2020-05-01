// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// +build !go1.14

package embedmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestProcessGo113(t *testing.T) {
	tc := []struct {
		name  string
		in    string
		dir   string
		files map[string][]byte
		urls  map[string][]byte
		out   string
		err   string
		diff  bool
	}{
		{
			name: "embedding code from a bad URL",
			in: "# This is some markdown\n" +
				"[embedmd]:# (https://fakeurl.com\\main.go)\n" +
				"Yay!\n",
			err: "2: could not read https://fakeurl.com\\main.go: parse https://fakeurl.com\\main.go: invalid character \"\\\\\" in host name",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cp := mixedContentProvider{tt.files, tt.urls}
			if tt.diff {
				cp.files["file.md"] = []byte(tt.in)
			}
			opts := []Option{WithFetcher(cp)}
			if tt.dir != "" {
				opts = append(opts, WithBaseDir(tt.dir))
			}
			err := Process(&out, strings.NewReader(tt.in), opts...)
			if !eqErr(t, tt.name, err, tt.err) {
				return
			}
			if tt.out != out.String() {
				t.Errorf("case [%s]: expected output:\n###\n%s\n###; got###\n%s\n###", tt.name, tt.out, out.String())
			}
		})
	}
}
