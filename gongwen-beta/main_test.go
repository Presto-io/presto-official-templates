package main

import (
	"strings"
	"testing"
)

func TestConvertProducesTypstCanary(t *testing.T) {
	output := convert("hello ] #world")

	if !strings.HasPrefix(output, "#set page(") {
		t.Fatal("expected Typst output to start with a directive")
	}
	if strings.Contains(output, "] #world") {
		t.Fatal("expected Typst content control characters to be escaped")
	}
	if !strings.Contains(output, "gongwen-beta canary") {
		t.Fatal("expected canary marker")
	}
}
