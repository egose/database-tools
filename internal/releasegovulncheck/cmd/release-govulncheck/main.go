package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/egose/database-tools/internal/releasegovulncheck"
)

func main() {
	var scannerStatus int
	var stderrPath string
	flag.IntVar(&scannerStatus, "scanner-status", 0, "govulncheck process exit status")
	flag.StringVar(&stderrPath, "stderr-output", "", "file containing govulncheck stderr")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: release-govulncheck -scanner-status <status> -stderr-output <file> <json-output>")
		os.Exit(2)
	}

	stdout, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open govulncheck JSON output: %v\n", err)
		os.Exit(1)
	}
	defer stdout.Close()

	var stderr []byte
	if stderrPath != "" {
		stderr, err = os.ReadFile(stderrPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read govulncheck stderr output: %v\n", err)
			os.Exit(1)
		}
	}

	result, err := releasegovulncheck.Evaluate(stdout, scannerStatus, string(stderr), time.Now().UTC(), releasegovulncheck.Exceptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "release vulnerability gate failed: %v\n", err)
		os.Exit(1)
	}

	if result.AllowedReachable == 0 {
		fmt.Println("release vulnerability gate passed: no reachable findings")
		return
	}
	fmt.Printf("release vulnerability gate passed: %d reachable findings matched documented unexpired exceptions; %d imported/module-only findings were not reachable at symbol level\n", result.AllowedReachable, result.IgnoredImported)
}
