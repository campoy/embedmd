// Copyright 2014 Google Inc. All rights reserved.
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

// embed.md

// embed.md parses all the .md files in a given directory looking for embed.md
// commands. For every one of the commands it extracts some code and embeds it
// as markdown code right below the command.
//
// The command format for embed.md follows the markdown comments syntax, which
// makes it invisible while rendering. This also allows to keep a reference to
// the origin of the embedded code, therefore providing a way to update the
// embedded copy if the original file changes.

// The format of an embed.md command is:
//
//         [embed.md]:# (filename language /start regexp/ /end regexp/)
//
// The embedded code will be extracted from the file filename, starting at the
// first line that matches /start regexp/ and finishing at the first line
// matching /end regexp/.
//
// Ommiting the the second regular expression will embed only the line that
// matches /start regexp/:
//
//     [embed.md]:# (filename language /regexp/)
//
// If you want to embed from a point to the end you should use:
//
//     [embed.md]:# (filename language /start regexp/ //)
//
// Finally you can embed a whole file by omitting both regular expressions:
//
//     [embed.md]:# (filename language)
//
// You can ommit the language in any of the previous commands, and the extension
// of the file will be used for the snippet syntax highlighting. Note that while
// this works Go files, it will fail with most language such as .md vs markdown.
//
//     [embed.md]:# (file.ext)
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
	"unicode"
)

func main() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, ".")
	}

	for _, path := range os.Args[1:] {
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
		if filepath.Ext(f.Name()) == ".md" {
			names = append(names, f.Name())
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

		// Ommit any code snippets right after a godown command.
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

			if strings.HasPrefix(text, "[embed.md]:#") {
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

	fmt.Fprintln(w, "```"+lang)
	w.Write(b)
	fmt.Fprintln(w, "```")
	return nil
}

func parseArgs(args string) (file, lang string, start, end *string, err error) {
	args = strings.TrimSpace(args)
	if args[0] != '(' || args[len(args)-1] != ')' {
		return "", "", nil, nil, errors.New("argument list should be in parenthesis")
	}

	args = strings.TrimSpace(args[1 : len(args)-1])
	if sep := strings.IndexFunc(args, unicode.IsSpace); sep >= 0 {
		file, args = args[:sep], strings.TrimSpace(args[sep+1:])
	} else {
		return args, filepath.Ext(args)[1:], nil, nil, nil
	}

	rem := strings.Split(args, "/")
	lang = strings.TrimSpace(rem[0])
	if lang == "" {
		lang = filepath.Ext(file)[1:]
	}
	switch len(rem) {
	case 5:
		end = &rem[3]
		fallthrough
	case 3:
		start = &rem[1]
	case 1:
	default:
		return "", "", nil, nil, errors.New("malformed regular expression")
	}
	return file, lang, start, end, nil
}

func extract(b []byte, start, end *string) ([]byte, error) {
	if start == nil && end == nil {
		return b, nil
	}

	match := func(s string) ([]int, error) {
		re, err := regexp.Compile(s)
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
		b = b[loc[0]:]
	}

	if end == nil {
		if eol := bytes.IndexByte(b, '\n'); eol >= 0 {
			b = b[:eol]
		}
		return b, nil
	}

	if *end != "" {
		loc, err := match(*end)
		if err != nil {
			return nil, err
		}
		b = b[:loc[1]]
	}

	return b, nil
}
