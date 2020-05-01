package main

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegration(t *testing.T) {
	cmd := exec.Command("embedmd", "docs.md")
	cmd.Dir = "sample"
	got, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("could not process file (%v): %s", err, got)
	}
	wants, err := ioutil.ReadFile(filepath.Join("sample", "result.md"))
	if err != nil {
		t.Fatalf("could not read result: %v", err)
	}
	if string(got) != string(wants) {
		t.Fatalf("got bad result (compared to result.md):\n%s", got)
	}
}

func TestRunTwice(t *testing.T) {
	cmd := exec.Command("embedmd", "docs.md")
	cmd.Dir = "sample"
	got, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("could not process file (%v): %s", err, got)
	}

	cmdAgain := exec.Command("embedmd")
	cmdAgain.Dir = "sample"
	cmdAgain.Stdin = strings.NewReader(string(got))
	gotAgain, err := cmdAgain.CombinedOutput()
	if err != nil {
		t.Fatalf("could not process file (%v): %s", err, got)
	}
	wants, err := ioutil.ReadFile(filepath.Join("sample", "result.md"))
	if err != nil {
		t.Fatalf("could not read result: %v", err)
	}
	if string(gotAgain) != string(wants) {
		t.Fatalf("got bad result (compared to result.md):\n%s", gotAgain)
	}
}
