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
// embedmd supports two flags:
// -d: will print the difference of the input file with what the output
//     would have been if executed.
// -w: rewrites the given files rather than writing the output to the standard
//     output.
//
// For more information on the format of the commands, read the documentation
// of the github.com/campoy/embedmd/embedmd package.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/campoy/embedmd/embedmd"
	"github.com/pmezard/go-difflib/difflib"
)

// modified while building by -ldflags.
var version = "unknown"

func usage() {
	fmt.Fprintf(os.Stderr, "usage: embedmd [flags] [path ...]\n")
	flag.PrintDefaults()
}

type app struct {
	stdout   io.Writer
	stdin    io.Reader
	openFile func(string) (file, error)
}

func defaultOpenFile(name string) (file, error) {
	return os.OpenFile(name, os.O_RDWR, 0666)
}

func main() {
	rewrite := flag.Bool("w", false, "write result to (markdown) file instead of stdout")
	doDiff := flag.Bool("d", false, "display diffs instead of rewriting files")
	printVersion := flag.Bool("v", false, "display embedmd version")
	flag.Usage = usage
	flag.Parse()

	if *printVersion {
		fmt.Println("embedmd version: " + version)
		return
	}

	a := &app{
		stdout:   os.Stdout,
		stdin:    os.Stdin,
		openFile: defaultOpenFile,
	}

	diff, err := a.embed(flag.Args(), *rewrite, *doDiff)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if diff && *doDiff {
		os.Exit(2)
	}
}

func (a *app) embed(paths []string, rewrite, doDiff bool) (foundDiff bool, err error) {
	if rewrite && doDiff {
		return false, fmt.Errorf("error: cannot use -w and -d simultaneously")
	}

	if len(paths) == 0 {
		if rewrite {
			return false, fmt.Errorf("error: cannot use -w with standard input")
		}
		if !doDiff {
			return false, embedmd.Process(a.stdout, a.stdin)
		}

		var out, in bytes.Buffer
		if err := embedmd.Process(&out, io.TeeReader(a.stdin, &in)); err != nil {
			return false, err
		}
		d, err := diff(in.String(), out.String())
		if err != nil || len(d) == 0 {
			return false, err
		}
		fmt.Fprintf(a.stdout, "%s", d)
		return true, nil
	}

	for _, path := range paths {
		d, err := a.processFile(path, rewrite, doDiff)
		if err != nil {
			return false, fmt.Errorf("%s:%v", path, err)
		}
		foundDiff = foundDiff || d
	}
	return foundDiff, nil
}

type file interface {
	io.ReadCloser
	io.WriterAt
	Truncate(int64) error
}

func (a *app) readFile(path string) ([]byte, error) {
	f, err := a.openFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (a *app) processFile(path string, rewrite, doDiff bool) (foundDiff bool, err error) {
	if filepath.Ext(path) != ".md" {
		return false, fmt.Errorf("not a markdown file")
	}

	f, err := a.openFile(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	if err := embedmd.Process(buf, f, embedmd.WithBaseDir(filepath.Dir(path))); err != nil {
		return false, err
	}

	if doDiff {
		f, err := a.readFile(path)
		if err != nil {
			return false, fmt.Errorf("could not read %s for diff: %v", path, err)
		}
		data, err := diff(string(f), buf.String())
		if err != nil || len(data) == 0 {
			return false, err
		}
		fmt.Fprintf(a.stdout, "%s", data)
		return true, nil
	}

	if rewrite {
		n, err := f.WriteAt(buf.Bytes(), 0)
		if err != nil {
			return false, fmt.Errorf("could not write: %v", err)
		}
		return false, f.Truncate(int64(n))
	}

	_, err = io.Copy(a.stdout, buf)
	return false, err
}

func diff(a, b string) (string, error) {
	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:       difflib.SplitLines(a),
		B:       difflib.SplitLines(b),
		Context: 3,
	})
}
