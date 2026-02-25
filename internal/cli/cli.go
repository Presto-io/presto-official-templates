package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

// Run implements the standard template CLI protocol:
//   - --manifest → print manifestJSON
//   - --example  → print exampleMD
//   - --version  → extract and print version from manifestJSON
//   - otherwise  → read stdin, call convert, print result
func Run(manifestJSON, exampleMD string, convert func(string) string) {
	manifestFlag := flag.Bool("manifest", false, "output manifest JSON")
	exampleFlag := flag.Bool("example", false, "output example markdown")
	versionFlag := flag.Bool("version", false, "output version")
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

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(convert(string(input)))
}
