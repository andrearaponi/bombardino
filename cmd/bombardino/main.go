package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/andrearaponi/bombardino/pkg/config"
	"github.com/andrearaponi/bombardino/pkg/engine"
	"github.com/andrearaponi/bombardino/pkg/progress"
	"github.com/andrearaponi/bombardino/pkg/reporter"
)

// Build-time variables (set via ldflags)
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	var (
		configFile   = flag.String("config", "", "Path to JSON configuration file")
		workers      = flag.Int("workers", 10, "Number of concurrent workers")
		verbose      = flag.Bool("verbose", false, "Enable verbose output")
		showVersion  = flag.Bool("version", false, "Show version information")
		outputFormat = flag.String("output", "text", "Output format: text or json")
	)
	flag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	if *configFile == "" {
		fmt.Println("❌ Error: Configuration file is required")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  bombardino -config=<config.json> [options]")
		fmt.Println()
		fmt.Println("Required:")
		fmt.Println("  -config string    Path to JSON configuration file")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -workers int      Number of concurrent workers (default: 10)")
		fmt.Println("  -verbose          Enable verbose output (default: false)")
		fmt.Println("  -output string    Output format: text or json (default: text)")
		fmt.Println("  -version          Show version information")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  bombardino -config=test.json")
		fmt.Println("  bombardino -config=test.json -workers=20 -output=json")
		fmt.Println("  bombardino -version")
		os.Exit(1)
	}

	cfg, err := config.LoadFromFile(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Only show progress bar for text output
	var progressBar *progress.ProgressBar
	if *outputFormat != "json" {
		progressBar = progress.New(cfg.GetTotalRequests())
	}
	testEngine := engine.New(*workers, progressBar, *verbose)

	results := testEngine.Run(cfg)

	// Generate report
	if *outputFormat == "json" {
		reporter := reporter.New(*verbose)
		if err := reporter.GenerateJSONReport(results); err != nil {
			log.Fatalf("Failed to generate JSON report: %v", err)
		}
	} else {
		reporter := reporter.New(*verbose)
		reporter.GenerateReport(results)
	}

	// Exit with appropriate code based on test results
	if results.FailedReqs > 0 {
		os.Exit(1) // Exit with error code if any tests failed
	}
}

func printVersion() {
	fmt.Printf("Bombardino %s\n", version)
	fmt.Printf("Commit: %s\n", commit)
	fmt.Printf("Built: %s\n", buildTime)
	fmt.Println()
	fmt.Println("A powerful REST API stress testing tool written in Go")
}
