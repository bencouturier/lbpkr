package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func TestLbpkrSelfBdist(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	cmd := newCommand("lbpkr", "self", "bdist")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running bdist: %v", err)
	}
}

func TestLbpkrSelfBdistRpm(t *testing.T) {
	if _, err := exec.LookPath("rpmbuild"); err != nil {
		t.Skip("no rpmbuild installed")
	}

	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	cmd := newCommand("lbpkr", "self", "bdist-rpm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running bdist-rpm: %v", err)
	}
}

func TestLbpkrInstallLbpkr(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "lbpkr")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running install: %v", err)
	}
}
