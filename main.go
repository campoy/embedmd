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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/campoy/embedmd/embedmd"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: embedmd [flags] [path ...]\n")
	flag.PrintDefaults()
}

func main() {
	rewrite := flag.Bool("w", false, "write result to (markdown) file instead of stdout")
	doDiff := flag.Bool("d", false, "display diffs instead of rewriting files")
	flag.Usage = usage
	flag.Parse()

	diff, err := embed(flag.Args(), *rewrite, *doDiff)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if diff && *doDiff {
		os.Exit(2)
	}
}

var (
	stdout io.Writer = os.Stdout
	stdin  io.Reader = os.Stdin
)

func embed(paths []string, rewrite, doDiff bool) (foundDiff bool, err error) {
	if rewrite && doDiff {
		return false, fmt.Errorf("error: cannot use -w and -d simulatenously")
	}

	if len(paths) == 0 {
		if rewrite {
			return false, fmt.Errorf("error: cannot use -w with standard input")
		}
		if !doDiff {
			return false, embedmd.Process(stdout, stdin)
		}

		var out, in bytes.Buffer
		if err := embedmd.Process(&out, io.TeeReader(stdin, &in)); err != nil {
			return false, err
		}
		d, err := diff(in.Bytes(), out.Bytes())
		if err != nil || len(d) == 0 {
			return false, err
		}
		fmt.Fprintf(stdout, "%s", d)
		return true, nil
	}

	for _, path := range paths {
		d, err := processFile(path, rewrite, doDiff)
		if err != nil {
			return false, fmt.Errorf("%s:%v", path, err)
		}
		doDiff = doDiff || d
	}
	return doDiff, nil
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

func readFile(path string) ([]byte, error) {
	f, err := openFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

func processFile(path string, rewrite, doDiff bool) (foundDiff bool, err error) {
	if filepath.Ext(path) != ".md" {
		return false, fmt.Errorf("not a markdown file")
	}

	f, err := openFile(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	if err := embedmd.Process(buf, f, embedmd.WithBaseDir(filepath.Dir(path))); err != nil {
		return false, err
	}

	if doDiff {
		f, err := readFile(path)
		if err != nil {
			return false, fmt.Errorf("could not read %s for diff: %v", path, err)
		}
		data, err := diff(f, buf.Bytes())
		if err != nil || len(data) == 0 {
			return false, err
		}
		fmt.Fprintf(stdout, "%s", data)
		return true, nil
	}

	if rewrite {
		n, err := f.WriteAt(buf.Bytes(), 0)
		if err != nil {
			return false, fmt.Errorf("could not write: %v", err)
		}
		return false, f.Truncate(int64(n))
	}

	io.Copy(stdout, buf)
	return false, nil
}

func diff(b1, b2 []byte) ([]byte, error) {
	f1, err := ioutil.TempFile("", "embedmd")
	if err != nil {
		return nil, fmt.Errorf("could not create tmp file: %v", err)
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "embedmd")
	if err != nil {
		return nil, fmt.Errorf("could not create tmp file: %v", err)
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	data, err := exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) == 0 && err == nil {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		return nil, err
	}

	// drop the first two lines of the output, since the paths shown
	// correspond to files that have been already removed.
	lines := bytes.SplitN(data, []byte{'\n'}, 3)
	if len(lines) != 3 {
		return nil, fmt.Errorf("unexpected format for diff output: %s", data)
	}
	return lines[2], nil
}
