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

// embedmd
//
// embedmd parses all the .md files in a given directory looking for embedmd
// commands. For every one of the commands it extracts the code and embeds it
// as markdown code right below the command.
//
// The command format for embedmd follows the markdown comments syntax, which
// makes it invisible while rendering. This also allows to keep a reference to
// the origin of the embedded code, therefore providing a way to update the
// embedded copy if the original file changes.
//
// The format of an embedmd command is:
//
//     [embedmd]:# (filename language /start regexp/ /end regexp/)
//
// The embedded code will be extracted from the file filename, starting at the
// piece of text matching /start regexp/ and finishing at the match of
// /end regexp/.
//
// Ommiting the the second regular expression will embed only the piece of
// text that matches /regexp/:
//
//     [embedmd]:# (filename language /regexp/)
//
// To embed the whole line matching a regular expression you can use:
//
//     [embedmd]:# (filename language /.*regexp.*\n/)
//
// If you want to embed from a point to the end you should use:
//
//     [embedmd]:# (filename language /start regexp/ $)
//
// Finally you can embed a whole file by omitting both regular expressions:
//
//     [embedmd]:# (filename language)
//
// You can ommit the language in any of the previous commands, and the extension
// of the file will be used for the snippet syntax highlighting. Note that while
// this works Go files, since the file extension .go matches the name of the language
// go, this will fail with other files like .md whose language name is markdown.
//
//     [embedmd]:# (file.ext)
//
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	paths := os.Args[1:]
	if len(paths) == 0 {
		paths = append(paths, ".")
	}

	for _, path := range paths {
		fs, err := markdownFiles(path)
		if err != nil {
			log.Printf("could not list files in %s: %v", path, err)
		}

		for _, f := range fs {
			if err := process(f); err != nil {
				log.Printf("could not process %s: %v", f, err)
			}
		}
	}
}

func markdownFiles(path string) ([]string, error) {
	d, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	fs, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, f := range fs {
		if name := f.Name(); f.Mode().IsRegular() && filepath.Ext(name) == ".md" {
			names = append(names, name)
		}
	}
	return names, nil
}

type file interface {
	io.ReadCloser
	io.WriterAt
}

// replaced by testing functions.
var openFile = func(name string) (file, error) {
	return os.OpenFile(name, os.O_RDWR, 0666)
}

func process(path string) error {
	f, err := openFile(path)
	if err != nil {
		return fmt.Errorf("could not open %s: %v", path, err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	out := new(bytes.Buffer)
	nextToCmd, skippingCode := false, false
	for line := 1; s.Scan(); line++ {
		text := s.Text()

		// Ommit any code snippets right after a embedmd command.
		if nextToCmd || skippingCode {
			if strings.HasPrefix(text, "```") {
				if skippingCode {
					skippingCode = false
					continue
				}
				skippingCode = true
			}
			nextToCmd = false
		}
		if !skippingCode {
			fmt.Fprintln(out, text)

			if strings.HasPrefix(text, "[embedmd]:#") {
				err := extractFromFile(out, strings.Split(text, "#")[1])
				if err != nil {
					return fmt.Errorf("%v at line %d", err, line)
				}
				nextToCmd = true
			}
		}
	}

	if err := s.Err(); err != nil {
		return fmt.Errorf("could not scan: %v", err)
	}

	_, err = f.WriteAt(out.Bytes(), 0)
	return nil
}

// replaced by testing functions.
var readFile = ioutil.ReadFile

func extractFromFile(w io.Writer, args string) error {
	file, lang, start, end, err := parseArgs(args)
	if err != nil {
		return err
	}

	b, err := readFile(file)
	if err != nil {
		return fmt.Errorf("could not read %s: %v", file, err)
	}

	b, err = extract(b, start, end)
	if err != nil {
		return fmt.Errorf("could not extract content from %s: %v", file, err)
	}

	if len(b) > 0 && b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}

	fmt.Fprintln(w, "```"+lang)
	w.Write(b)
	fmt.Fprintln(w, "```")
	return nil
}

// fields returns a list of the groups of text separated by blanks,
// keeping all text surrounded by / as a group.
func fields(s string) ([]string, error) {
	var args []string

	for s = strings.TrimSpace(s); len(s) > 0; s = strings.TrimSpace(s) {
		if s[0] == '/' {
			sep := strings.IndexByte(s[1:], '/')
			if sep < 0 {
				return nil, errors.New("unbalanced /")
			}
			args, s = append(args, s[:sep+2]), s[sep+2:]
		} else {
			sep := strings.IndexByte(s[1:], ' ')
			if sep < 0 {
				return append(args, s), nil
			}
			args, s = append(args, s[:sep+1]), s[sep+1:]
		}
	}

	return args, nil
}

func parseArgs(s string) (file, lang string, start, end *string, err error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return "", "", nil, nil, errors.New("argument list should be in parenthesis")
	}

	args, err := fields(s[1 : len(s)-1])
	if err != nil {
		return "", "", nil, nil, err
	}
	if len(args) == 0 {
		return "", "", nil, nil, errors.New("missing file name")
	}

	file, args = args[0], args[1:]
	if len(args) > 0 && args[0][0] != '/' {
		lang, args = args[0], args[1:]
	} else {
		ext := filepath.Ext(file[1:])
		if len(ext) == 0 {
			return "", "", nil, nil, errors.New("language is required when file has no extension")
		}
		lang = ext[1:]
	}

	switch {
	case len(args) == 1:
		start = &args[0]
	case len(args) == 2:
		start, end = &args[0], &args[1]
	case len(args) > 2:
		return "", "", nil, nil, errors.New("too many arguments")
	}

	return file, lang, start, end, nil
}

func extract(b []byte, start, end *string) ([]byte, error) {
	if start == nil && end == nil {
		return b, nil
	}

	match := func(s string) ([]int, error) {
		if len(s) <= 2 || s[0] != '/' || s[len(s)-1] != '/' {
			return nil, fmt.Errorf("missing slashes (/) around %q", s)
		}
		re, err := regexp.Compile(s[1 : len(s)-1])
		if err != nil {
			return nil, err
		}
		loc := re.FindIndex(b)
		if loc == nil {
			return nil, fmt.Errorf("could not match %q", s)
		}
		return loc, nil
	}

	if *start != "" {
		loc, err := match(*start)
		if err != nil {
			return nil, err
		}
		if end == nil {
			return b[loc[0]:loc[1]], nil
		}
		b = b[loc[0]:]
	}

	if *end != "$" {
		loc, err := match(*end)
		if err != nil {
			return nil, err
		}
		b = b[:loc[1]]
	}

	return b, nil
}
