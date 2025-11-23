package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"helm.sh/helm/v3/pkg/chart/loader"

	"helm-values-check/internal/checker"
)

func main() {
	log.SetFlags(0)

	includeSubcharts := flag.Bool("include-subcharts", false, "also inspect templates of subcharts in charts/ directory")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] PATH_TO_CHART\n\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		os.Exit(2)
	}

	chartPath := flag.Arg(0)

	ch, err := loader.Load(chartPath)
	if err != nil {
		log.Fatalf("failed to load chart from %q: %v", chartPath, err)
	}

	result, err := checker.CheckChart(ch, checker.Config{
		IncludeSubcharts: *includeSubcharts,
	})
	if err != nil {
		log.Fatalf("check failed: %v", err)
	}

	exitCode := 0

	if len(result.UsedNotDefined) > 0 {
		fmt.Println("Used in templates but not defined in values.yaml:")
		for _, k := range result.UsedNotDefined {
			fmt.Printf("  - %s\n", k)
		}
		fmt.Println()
		exitCode = 1
	}

	if len(result.DefinedNotUsed) > 0 {
		fmt.Println("Defined in values.yaml but not used in templates:")
		for _, k := range result.DefinedNotUsed {
			fmt.Printf("  - %s\n", k)
		}
		fmt.Println()
		if exitCode == 0 {
			exitCode = 1
		}
	}

	if exitCode == 0 {
		fmt.Println("OK: no mismatches between values.yaml and template usage.")
	}

	os.Exit(exitCode)
}
