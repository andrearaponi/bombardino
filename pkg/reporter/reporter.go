package reporter

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"

	"github.com/andrearaponi/bombardino/internal/models"
)

//go:embed templates/report.html
var htmlTemplate string

type Reporter struct {
	verbose bool
}

func New(verbose bool) *Reporter {
	return &Reporter{
		verbose: verbose,
	}
}

func (r *Reporter) GenerateReport(summary *models.Summary) {
	r.printHeader()
	r.printSummary(summary)
	r.printStatusCodes(summary)
	if len(summary.EndpointResults) > 0 {
		r.printEndpointResults(summary)
	}
	if len(summary.Errors) > 0 {
		r.printErrors(summary)
	}
	r.printFooter()
}

type JSONReport struct {
	Summary   JSONSummary             `json:"summary"`
	Endpoints map[string]JSONEndpoint `json:"endpoints"`
	DebugLogs []models.DebugLog       `json:"debug_logs,omitempty"`
	Success   bool                    `json:"success"`
}

type JSONSummary struct {
	TotalRequests    int            `json:"total_requests"`
	SuccessfulReqs   int            `json:"successful_requests"`
	FailedReqs       int            `json:"failed_requests"`
	SuccessRate      float64        `json:"success_rate_percent"`
	TotalTime        string         `json:"total_time"`
	AvgResponseTime  string         `json:"avg_response_time"`
	MinResponseTime  string         `json:"min_response_time"`
	MaxResponseTime  string         `json:"max_response_time"`
	P50ResponseTime  string         `json:"p50_response_time"`
	P95ResponseTime  string         `json:"p95_response_time"`
	P99ResponseTime  string         `json:"p99_response_time"`
	RequestsPerSec   float64        `json:"requests_per_sec"`
	StatusCodes      map[string]int `json:"status_codes"`
	Errors           map[string]int `json:"errors"`
	TotalAssertions  int            `json:"total_assertions,omitempty"`
	AssertionsPassed int            `json:"assertions_passed,omitempty"`
	AssertionsFailed int            `json:"assertions_failed,omitempty"`
}

type JSONEndpoint struct {
	Name             string         `json:"name"`
	URL              string         `json:"url"`
	TotalRequests    int            `json:"total_requests"`
	SuccessfulReqs   int            `json:"successful_requests"`
	FailedReqs       int            `json:"failed_requests"`
	SuccessRate      float64        `json:"success_rate_percent"`
	AvgResponseTime  string         `json:"avg_response_time"`
	P50ResponseTime  string         `json:"p50_response_time"`
	P95ResponseTime  string         `json:"p95_response_time"`
	P99ResponseTime  string         `json:"p99_response_time"`
	StatusCodes      map[string]int `json:"status_codes"`
	Errors           []string       `json:"errors"`
	Success          bool           `json:"success"`
	TotalAssertions  int            `json:"total_assertions,omitempty"`
	AssertionsPassed int            `json:"assertions_passed,omitempty"`
	AssertionsFailed int            `json:"assertions_failed,omitempty"`
}

func (r *Reporter) GenerateJSONReport(summary *models.Summary) error {
	jsonReport := r.createJSONReport(summary)
	output, err := json.MarshalIndent(jsonReport, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(output))
	return nil
}

func (r *Reporter) createJSONReport(summary *models.Summary) JSONReport {
	var successRate float64
	if summary.TotalRequests > 0 {
		successRate = float64(summary.SuccessfulReqs) / float64(summary.TotalRequests) * 100
	}

	statusCodes := make(map[string]int)
	for code, count := range summary.StatusCodes {
		statusCodes[fmt.Sprintf("%d", code)] = count
	}

	endpoints := make(map[string]JSONEndpoint)
	for name, ep := range summary.EndpointResults {
		var epSuccessRate float64
		if ep.TotalRequests > 0 {
			epSuccessRate = float64(ep.SuccessfulReqs) / float64(ep.TotalRequests) * 100
		}
		epStatusCodes := make(map[string]int)
		for code, count := range ep.StatusCodes {
			epStatusCodes[fmt.Sprintf("%d", code)] = count
		}

		endpoints[name] = JSONEndpoint{
			Name:             ep.Name,
			URL:              ep.URL,
			TotalRequests:    ep.TotalRequests,
			SuccessfulReqs:   ep.SuccessfulReqs,
			FailedReqs:       ep.FailedReqs,
			SuccessRate:      epSuccessRate,
			AvgResponseTime:  ep.AvgResponseTime.Round(1000).String(),
			P50ResponseTime:  ep.P50ResponseTime.Round(1000).String(),
			P95ResponseTime:  ep.P95ResponseTime.Round(1000).String(),
			P99ResponseTime:  ep.P99ResponseTime.Round(1000).String(),
			StatusCodes:      epStatusCodes,
			Errors:           ep.Errors,
			Success:          ep.FailedReqs == 0,
			TotalAssertions:  ep.TotalAssertions,
			AssertionsPassed: ep.AssertionsPassed,
			AssertionsFailed: ep.AssertionsFailed,
		}
	}

	jsonReport := JSONReport{
		Summary: JSONSummary{
			TotalRequests:    summary.TotalRequests,
			SuccessfulReqs:   summary.SuccessfulReqs,
			FailedReqs:       summary.FailedReqs,
			SuccessRate:      successRate,
			TotalTime:        summary.TotalTime.Round(1000).String(),
			AvgResponseTime:  summary.AvgResponseTime.Round(1000).String(),
			MinResponseTime:  summary.MinResponseTime.Round(1000).String(),
			MaxResponseTime:  summary.MaxResponseTime.Round(1000).String(),
			P50ResponseTime:  summary.P50ResponseTime.Round(1000).String(),
			P95ResponseTime:  summary.P95ResponseTime.Round(1000).String(),
			P99ResponseTime:  summary.P99ResponseTime.Round(1000).String(),
			RequestsPerSec:   summary.RequestsPerSec,
			StatusCodes:      statusCodes,
			Errors:           summary.Errors,
			TotalAssertions:  summary.TotalAssertions,
			AssertionsPassed: summary.AssertionsPassed,
			AssertionsFailed: summary.AssertionsFailed,
		},
		Endpoints: endpoints,
		Success:   summary.FailedReqs == 0,
	}
	
	// Include debug logs if verbose mode is enabled and there are logs
	if r.verbose && len(summary.DebugLogs) > 0 {
		jsonReport.DebugLogs = summary.DebugLogs
	}
	
	return jsonReport
}

func (r *Reporter) printHeader() {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                              BOMBARDINO RESULTS                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func (r *Reporter) printSummary(summary *models.Summary) {
	fmt.Println("📊 SUMMARY")
	fmt.Println(strings.Repeat("─", 80))

	successRate := float64(0)
	failedRate := float64(0)
	skippedRate := float64(0)
	if summary.TotalRequests > 0 {
		successRate = float64(summary.SuccessfulReqs) / float64(summary.TotalRequests) * 100
		failedRate = float64(summary.FailedReqs) / float64(summary.TotalRequests) * 100
		skippedRate = float64(summary.SkippedReqs) / float64(summary.TotalRequests) * 100
	}

	fmt.Printf("Total Requests:      %d\n", summary.TotalRequests)
	fmt.Printf("Successful:          %d (%.1f%%)\n", summary.SuccessfulReqs, successRate)
	fmt.Printf("Failed:              %d (%.1f%%)\n", summary.FailedReqs, failedRate)
	if summary.SkippedReqs > 0 {
		fmt.Printf("Skipped:             %d (%.1f%%)\n", summary.SkippedReqs, skippedRate)
	}
	fmt.Printf("Requests/sec:        %.2f\n", summary.RequestsPerSec)
	fmt.Printf("Total Duration:      %v\n", summary.TotalTime.Round(1000))
	fmt.Println()

	// Print assertions summary if any assertions were evaluated
	if summary.TotalAssertions > 0 {
		fmt.Println("✅ ASSERTIONS")
		fmt.Println(strings.Repeat("─", 80))
		assertionRate := float64(summary.AssertionsPassed) / float64(summary.TotalAssertions) * 100
		fmt.Printf("Total Assertions:    %d\n", summary.TotalAssertions)
		fmt.Printf("Passed:              %d (%.1f%%)\n", summary.AssertionsPassed, assertionRate)
		fmt.Printf("Failed:              %d (%.1f%%)\n", summary.AssertionsFailed, 100-assertionRate)
		fmt.Println()
	}

	fmt.Println("⏱️  RESPONSE TIMES")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("Average:             %v\n", summary.AvgResponseTime.Round(1000))
	fmt.Printf("Minimum:             %v\n", summary.MinResponseTime.Round(1000))
	fmt.Printf("Maximum:             %v\n", summary.MaxResponseTime.Round(1000))
	fmt.Printf("P50 (median):        %v\n", summary.P50ResponseTime.Round(1000))
	fmt.Printf("P95:                 %v\n", summary.P95ResponseTime.Round(1000))
	fmt.Printf("P99:                 %v\n", summary.P99ResponseTime.Round(1000))
	fmt.Println()
}

func (r *Reporter) printStatusCodes(summary *models.Summary) {
	if len(summary.StatusCodes) == 0 {
		return
	}

	fmt.Println("📈 STATUS CODES")
	fmt.Println(strings.Repeat("─", 80))

	type statusCount struct {
		code  int
		count int
	}

	var statuses []statusCount
	for code, count := range summary.StatusCodes {
		statuses = append(statuses, statusCount{code, count})
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].code < statuses[j].code
	})

	for _, sc := range statuses {
		percentage := float64(sc.count) / float64(summary.TotalRequests) * 100
		emoji := r.getStatusEmoji(sc.code)
		fmt.Printf("%s %d:              %d (%.1f%%)\n", emoji, sc.code, sc.count, percentage)
	}
	fmt.Println()
}

func (r *Reporter) printEndpointResults(summary *models.Summary) {
	fmt.Println("🎯 ENDPOINT RESULTS")
	fmt.Println(strings.Repeat("─", 80))

	type endpointResult struct {
		name     string
		endpoint *models.EndpointSummary
	}

	var endpoints []endpointResult
	for name, ep := range summary.EndpointResults {
		endpoints = append(endpoints, endpointResult{name, ep})
	}

	// Sort by execution order (first executed first)
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].endpoint.FirstExecutedAt.Before(endpoints[j].endpoint.FirstExecutedAt)
	})

	for _, ep := range endpoints {
		// Determine status icon
		status := "✅"
		if ep.endpoint.SkippedReqs > 0 && ep.endpoint.SuccessfulReqs == 0 && ep.endpoint.FailedReqs == 0 {
			status = "⏭️"
		} else if ep.endpoint.FailedReqs > 0 {
			status = "❌"
		}

		fmt.Printf("%s %s\n", status, ep.endpoint.Name)
		fmt.Printf("   URL: %s\n", ep.endpoint.URL)

		// If entirely skipped, show skip info
		if ep.endpoint.SkippedReqs > 0 && ep.endpoint.SuccessfulReqs == 0 && ep.endpoint.FailedReqs == 0 {
			fmt.Printf("   Skipped: %d (dependency failed)\n", ep.endpoint.SkippedReqs)
		} else {
			successRate := float64(0)
			if ep.endpoint.TotalRequests > 0 {
				successRate = float64(ep.endpoint.SuccessfulReqs) / float64(ep.endpoint.TotalRequests) * 100
			}
			fmt.Printf("   Requests: %d | Success: %d (%.1f%%) | Failed: %d\n",
				ep.endpoint.TotalRequests, ep.endpoint.SuccessfulReqs, successRate, ep.endpoint.FailedReqs)
			fmt.Printf("   Response Times: Avg=%v | P50=%v | P95=%v | P99=%v\n",
				ep.endpoint.AvgResponseTime.Round(1000),
				ep.endpoint.P50ResponseTime.Round(1000),
				ep.endpoint.P95ResponseTime.Round(1000),
				ep.endpoint.P99ResponseTime.Round(1000))
		}

		if ep.endpoint.TotalAssertions > 0 {
			assertionRate := float64(ep.endpoint.AssertionsPassed) / float64(ep.endpoint.TotalAssertions) * 100
			fmt.Printf("   Assertions: %d total | Passed: %d (%.1f%%) | Failed: %d\n",
				ep.endpoint.TotalAssertions, ep.endpoint.AssertionsPassed, assertionRate, ep.endpoint.AssertionsFailed)
		}

		if len(ep.endpoint.StatusCodes) > 0 {
			fmt.Printf("   Status Codes: ")
			var codes []string
			for code, count := range ep.endpoint.StatusCodes {
				codes = append(codes, fmt.Sprintf("%d (%d)", code, count))
			}
			fmt.Printf("%s\n", strings.Join(codes, ", "))
		}

		if len(ep.endpoint.Errors) > 0 && r.verbose {
			fmt.Printf("   Errors: %d unique\n", len(ep.endpoint.Errors))
		}
		fmt.Println()
	}
}

func (r *Reporter) printErrors(summary *models.Summary) {
	fmt.Println("❌ ERRORS")
	fmt.Println(strings.Repeat("─", 80))

	type errorCount struct {
		error string
		count int
	}

	var errors []errorCount
	for err, count := range summary.Errors {
		errors = append(errors, errorCount{err, count})
	}

	sort.Slice(errors, func(i, j int) bool {
		return errors[i].count > errors[j].count
	})

	for _, ec := range errors {
		percentage := float64(ec.count) / float64(summary.TotalRequests) * 100
		fmt.Printf("• %s: %d (%.1f%%)\n", ec.error, ec.count, percentage)
	}
	fmt.Println()
}

func (r *Reporter) printFooter() {
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println("🚀 Test completed successfully!")
	fmt.Println()
}

func (r *Reporter) getStatusEmoji(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "✅"
	case statusCode >= 300 && statusCode < 400:
		return "🔄"
	case statusCode >= 400 && statusCode < 500:
		return "⚠️"
	case statusCode >= 500 && statusCode < 600:
		return "❌"
	default:
		return "❓"
	}
}

func (r *Reporter) GenerateHTMLReport(summary *models.Summary) error {
	jsonReport := r.createJSONReport(summary)
	
	funcMap := template.FuncMap{
		"percentage": func(part, total int) float64 {
			if total == 0 {
				return 0
			}
			return float64(part) / float64(total) * 100
		},
		"statusClass": func(status string) string {
			if len(status) >= 1 {
				switch status[0] {
				case '2':
					return "status-2xx"
				case '3':
					return "status-3xx"
				case '4':
					return "status-4xx"
				case '5':
					return "status-5xx"
				}
			}
			return ""
		},
		"sub": func(a, b float64) float64 {
			return a - b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
	}
	
	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}
	
	err = tmpl.Execute(os.Stdout, jsonReport)
	if err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}
	
	return nil
}

