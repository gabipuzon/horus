package check

import "testing"

func TestSummaryUptimePercentage(t *testing.T) {
	for _, test := range []struct {
		name    string
		summary Summary
		want    *float64
	}{
		{name: "no checks", summary: Summary{}},
		{name: "all failed", summary: Summary{TotalChecks: 2}, want: percentage(0)},
		{name: "mixed", summary: Summary{TotalChecks: 3, SuccessfulChecks: 2}, want: percentage(66.67)},
		{name: "all successful", summary: Summary{TotalChecks: 2, SuccessfulChecks: 2}, want: percentage(100)},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.summary.UptimePercentage()
			if (got == nil) != (test.want == nil) || (got != nil && *got != *test.want) {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func percentage(value float64) *float64 { return &value }
