package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type DocumentInfo struct {
	Title       string   `json:"title,omitempty"`
	Authors     []string `json:"authors,omitempty"`
	Date        string   `json:"date,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	Description string   `json:"description,omitempty"`
	Language    string   `json:"language,omitempty"`
}

type OutputInfo struct {
	SchemaVersion  int                    `json:"schemaVersion"`
	OutputBaseName string                 `json:"outputBaseName"`
	PreviewTitle   string                 `json:"previewTitle,omitempty"`
	Document       DocumentInfo           `json:"document,omitempty"`
	TemplateData   map[string]interface{} `json:"templateData,omitempty"`
}

// Run implements the standard template CLI protocol:
//   - --manifest → print manifestJSON
//   - --example  → print exampleMD
//   - --version  → extract and print version from manifestJSON
//   - --info     → read stdin Markdown and print output info JSON
//   - otherwise  → read stdin, call convert, print result
func Run(manifestJSON, exampleMD string, convert func(string) string, info func(string) OutputInfo) {
	manifestFlag := flag.Bool("manifest", false, "output manifest JSON")
	exampleFlag := flag.Bool("example", false, "output example markdown")
	versionFlag := flag.Bool("version", false, "output version")
	infoFlag := flag.Bool("info", false, "output document info JSON")
	flag.Parse()

	if *versionFlag {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(manifestJSON), &m); err == nil {
			if v, ok := m["version"]; ok {
				fmt.Println(v)
			}
		}
		return
	}

	if *manifestFlag {
		fmt.Print(manifestJSON)
		return
	}

	if *exampleFlag {
		fmt.Print(exampleMD)
		return
	}

	const maxInputSize = 10 << 20 // 10 MB
	input, err := io.ReadAll(io.LimitReader(os.Stdin, maxInputSize+1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}
	if len(input) > maxInputSize {
		fmt.Fprintf(os.Stderr, "error: input exceeds %d bytes\n", maxInputSize)
		os.Exit(1)
	}

	if *infoFlag {
		if info == nil {
			fmt.Print(`{"schemaVersion":1,"outputBaseName":"output"}`)
			return
		}
		data, err := json.Marshal(info(string(input)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error encoding info: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(data))
		return
	}

	fmt.Print(convert(string(input)))
}

// CleanFilenameBase returns a filesystem-safe basename while preserving the
// visible title text as much as possible.
func CleanFilenameBase(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")
	value = strings.TrimSpace(replacer.Replace(value))
	if value == "" {
		return "output"
	}
	return value
}
