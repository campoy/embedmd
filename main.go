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
// embedmd embeds files or fractions of files into markdown files.
// It does so by searching embedmd commands, which are a subset of the
// markdown syntax for comments. This means they are invisible when
// markdown is rendered, so they can be kept in the file as pointers
// to the origin of the embedded text.
//
// The command receives a list of markdown files, if none is given it
// reads from the standard input.
//
// The format of an embedmd command is:
//
//     [embedmd]:# (pathOrURL language /start regexp/ /end regexp/)
//
// The embedded code will be extracted from the file at pathOrURL,
// which can either be a relative path to a file in the local file
// system (using always forward slashes as directory separator) or
// a url starting with http:// or https://.
// If the pathOrURL is a url the tool will fetch the content in that url.
// The embedded content starts at the first line that matches /start regexp/
// and finishes at the first line matching /end regexp/.
//
// Omitting the the second regular expression will embed only the piece of
// text that matches /regexp/:
//
//     [embedmd]:# (pathOrURL language /regexp/)
//
// To embed the whole line matching a regular expression you can use:
//
//     [embedmd]:# (pathOrURL language /.*regexp.*\n/)
//
// If you want to embed from a point to the end you should use:
//
//     [embedmd]:# (pathOrURL language /start regexp/ $)
//
// Finally you can embed a whole file by omitting both regular expressions:
//
//     [embedmd]:# (pathOrURL language)
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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: embedmd [flags] [path ...]\n")
	flag.PrintDefaults()
}

func main() {
	rewrite := flag.Bool("w", false, "write result to (markdown) file instead of stdout")
	flag.Usage = usage
	flag.Parse()

	err := embed(flag.Args(), *rewrite)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func embed(paths []string, rewrite bool) error {
	if len(paths) == 0 {
		if rewrite {
			return fmt.Errorf("error: cannot use -w with standard input")
		}
		return process(os.Stdout, os.Stdin)
	}

	for _, path := range paths {
		if err := processFile(path, rewrite); err != nil {
			return fmt.Errorf("%s:%v", path, err)
		}
	}
	return nil
}

type file interface {
	io.ReadCloser
	io.WriterAt
	Truncate(int64) error
}

// replaced by testing functions.
var openFile = func(name string) (file, error) {
	return os.OpenFile(name, os.O_RDWR, 0666)
}

func processFile(path string, rewrite bool) error {
	if filepath.Ext(path) != ".md" {
		return fmt.Errorf("not a markdown file")
	}

	f, err := openFile(path)
	if err != nil {
		return fmt.Errorf("could not open: %v", err)
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	if err := process(buf, f); err != nil {
		return err
	}

	if rewrite {
		n, err := f.WriteAt(buf.Bytes(), 0)
		if err != nil {
			return fmt.Errorf("could not write: %v", err)
		}
		return f.Truncate(int64(n))
	}

	io.Copy(os.Stdout, buf)
	return nil
}

type textScanner interface {
	Text() string
	Scan() bool
}

type parsingState func(io.Writer, textScanner) (parsingState, error)

func parsingText(out io.Writer, s textScanner) (parsingState, error) {
	// print current line, then decide what to do based on the next one.
	fmt.Fprintln(out, s.Text())
	if !s.Scan() {
		return nil, nil // end of file, which is fine.
	}
	switch line := s.Text(); {
	case strings.HasPrefix(line, "[embedmd]:#"):
		return parsingCmd, nil
	case strings.HasPrefix(line, "```"):
		return codeParser{print: true}.parse, nil
	default:
		return parsingText, nil
	}
}

func parsingCmd(out io.Writer, s textScanner) (parsingState, error) {
	line := s.Text()
	fmt.Fprintln(out, line)
	err := extractFromFile(out, strings.Split(line, "#")[1])
	if err != nil {
		return nil, err
	}
	if !s.Scan() {
		return nil, nil // end of file, which is fine.
	}
	if strings.HasPrefix(s.Text(), "```") {
		return codeParser{print: false}.parse, nil
	}
	return parsingText, nil
}

type codeParser struct{ print bool }

func (c codeParser) parse(out io.Writer, s textScanner) (parsingState, error) {
	if c.print {
		fmt.Fprintln(out, s.Text())
	}
	if !s.Scan() {
		return nil, fmt.Errorf("unbalanced code section")
	}
	if !strings.HasPrefix(s.Text(), "```") {
		return c.parse, nil
	}

	// print the end of the code section if needed and go back to parsing text.
	if c.print {
		fmt.Fprintln(out, s.Text())
	}
	if !s.Scan() {
		return nil, nil // end of file
	}
	return parsingText, nil
}

type countingScanner struct {
	*bufio.Scanner
	line int
}

func (c *countingScanner) Scan() bool {
	b := c.Scanner.Scan()
	if b {
		c.line++
	}
	return b
}

func process(out io.Writer, in io.Reader) error {
	s := &countingScanner{bufio.NewScanner(in), 0}
	if !s.Scan() {
		return nil
	}
	state := parsingText
	var err error
	for state != nil {
		state, err = state(out, s)
		if err != nil {
			return fmt.Errorf("%d: %v", s.line, err)
		}
	}

	if err := s.Err(); err != nil {
		return fmt.Errorf("%d: %v", s.line, err)
	}
	return nil
}

func extractFromFile(w io.Writer, args string) error {
	file, lang, start, end, err := parseArgs(args)
	if err != nil {
		return err
	}

	b, err := readContents(file)
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

// replaced by testing functions.
var (
	readFile = ioutil.ReadFile
	httpGet  = http.Get
)

func readContents(path string) ([]byte, error) {
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		return readFile(filepath.FromSlash(path))
	}

	res, err := httpGet(path)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", res.Status)
	}
	return ioutil.ReadAll(res.Body)
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
