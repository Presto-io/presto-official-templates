package cli

import "testing"

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
