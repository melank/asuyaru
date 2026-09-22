package main

import (
	"strings"
	"testing"
)

type stringAddr string

func (a stringAddr) Network() string { return "tcp" }
func (a stringAddr) String() string  { return string(a) }

func TestListenURL(t *testing.T) {
	t.Parallel()
	if got := listenURL(stringAddr("127.0.0.1:8080")); got != "http://127.0.0.1:8080/" {
		t.Fatalf("url = %s", got)
	}
}

func TestRequireLoopback(t *testing.T) {
	t.Parallel()
	if err := requireLoopback("127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	if err := requireLoopback("0.0.0.0:8080"); err == nil {
		t.Fatal("0.0.0.0 を許可した")
	}
}

func TestDefaultDBPath(t *testing.T) {
	t.Parallel()
	path, err := defaultDBPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "/asuyaru/asuyaru.db") {
		t.Fatalf("path = %s", path)
	}
}
