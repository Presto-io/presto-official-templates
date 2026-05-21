package cli

import (
	"strings"
	"testing"
)

func TestCleanFilenameBase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "replaces unsafe characters", input: ` A/B\C:D*E?F"G<H>I|J `, want: `A_B_C_D_E_F_G_H_I_J`},
		{name: "defaults blank title", input: " \t\n", want: "output"},
		{name: "keeps safe unicode title", input: "电气设备控制线路安装与调试", want: "电气设备控制线路安装与调试"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanFilenameBase(tt.input); got != tt.want {
				t.Fatalf("CleanFilenameBase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSafeConvertRejectsBlankOutput(t *testing.T) {
	_, err := safeConvert(func(string) string { return " \n\t" }, "input")
	if err == nil || !strings.Contains(err.Error(), "empty Typst output") {
		t.Fatalf("expected empty Typst output error, got %v", err)
	}
}

func TestSafeConvertRecoversPanic(t *testing.T) {
	_, err := safeConvert(func(string) string { panic("boom") }, "input")
	if err == nil || !strings.Contains(err.Error(), "panic: boom") {
		t.Fatalf("expected panic error, got %v", err)
	}
}

func TestSafeConvertReturnsNonBlankOutput(t *testing.T) {
	got, err := safeConvert(func(string) string { return "#set page()\n" }, "input")
	if err != nil {
		t.Fatalf("expected successful conversion, got %v", err)
	}
	if got != "#set page()\n" {
		t.Fatalf("unexpected output %q", got)
	}
}
