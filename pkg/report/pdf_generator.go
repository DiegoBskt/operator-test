/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package report

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
)

// Colors for status badges
var (
	colorPass = []int{34, 139, 34}  // Forest Green
	colorWarn = []int{255, 165, 0}  // Orange
	colorFail = []int{220, 20, 60}  // Crimson
	colorInfo = []int{70, 130, 180} // Steel Blue
)

// GeneratePDF creates a PDF report from the assessment.
func GeneratePDF(assessment *assessmentv1alpha1.ClusterAssessment) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)

	// Add first page
	pdf.AddPage()

	// Title
	pdf.SetFont("Helvetica", "B", 24)
	pdf.SetTextColor(0, 51, 102)
	pdf.CellFormat(0, 15, "OpenShift Cluster Assessment Report", "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Subtitle with date
	pdf.SetFont("Helvetica", "", 12)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(0, 8, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006 at 15:04 MST")), "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Cluster Info Box
	addSectionTitle(pdf, "Cluster Information")
	addClusterInfoTable(pdf, assessment)
	pdf.Ln(10)

	// Summary Section
	addSectionTitle(pdf, "Assessment Summary")
	addSummarySection(pdf, assessment)
	pdf.Ln(10)

	// Score visualization
	if assessment.Status.Summary.Score != nil {
		addScoreVisualization(pdf, *assessment.Status.Summary.Score)
		pdf.Ln(10)
	}

	// Findings by Category
	addSectionTitle(pdf, "Findings by Category")
	addFindingsByCategory(pdf, assessment)

	// Detailed Findings
	pdf.AddPage()
	addSectionTitle(pdf, "Detailed Findings")
	addDetailedFindings(pdf, assessment)

	// Output to bytes
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func addSectionTitle(pdf *gofpdf.Fpdf, title string) {
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(0, 51, 102)
	pdf.SetFillColor(240, 240, 245)
	pdf.CellFormat(0, 10, title, "", 1, "L", true, 0, "")
	pdf.Ln(3)
}

func addClusterInfoTable(pdf *gofpdf.Fpdf, assessment *assessmentv1alpha1.ClusterAssessment) {
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(0, 0, 0)

	info := assessment.Status.ClusterInfo

	// Two column layout
	colWidth := 85.0
	rowHeight := 7.0

	rows := [][]string{
		{"Cluster ID:", info.ClusterID},
		{"OpenShift Version:", info.ClusterVersion},
		{"Platform:", info.Platform},
		{"Update Channel:", info.Channel},
		{"Total Nodes:", fmt.Sprintf("%d", info.NodeCount)},
		{"Control Plane Nodes:", fmt.Sprintf("%d", info.ControlPlaneNodes)},
		{"Worker Nodes:", fmt.Sprintf("%d", info.WorkerNodes)},
		{"Assessment Profile:", assessment.Spec.Profile},
	}

	for _, row := range rows {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(colWidth, rowHeight, row[0], "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(colWidth, rowHeight, row[1], "", 1, "L", false, 0, "")
	}
}

func addSummarySection(pdf *gofpdf.Fpdf, assessment *assessmentv1alpha1.ClusterAssessment) {
	summary := assessment.Status.Summary

	// Summary boxes
	boxWidth := 40.0
	boxHeight := 20.0
	startX := 15.0
	y := pdf.GetY()

	summaryItems := []struct {
		label string
		count int
		color []int
	}{
		{"PASS", summary.PassCount, colorPass},
		{"WARN", summary.WarnCount, colorWarn},
		{"FAIL", summary.FailCount, colorFail},
		{"INFO", summary.InfoCount, colorInfo},
	}

	for i, item := range summaryItems {
		x := startX + float64(i)*(boxWidth+5)

		// Box background
		pdf.SetFillColor(item.color[0], item.color[1], item.color[2])
		pdf.RoundedRect(x, y, boxWidth, boxHeight, 3, "1234", "F")

		// Count
		pdf.SetFont("Helvetica", "B", 16)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetXY(x, y+2)
		pdf.CellFormat(boxWidth, 10, fmt.Sprintf("%d", item.count), "", 0, "C", false, 0, "")

		// Label
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetXY(x, y+12)
		pdf.CellFormat(boxWidth, 6, item.label, "", 0, "C", false, 0, "")
	}

	pdf.SetY(y + boxHeight + 5)
	pdf.SetTextColor(0, 0, 0)

	// Total checks
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Total Checks: %d", summary.TotalChecks), "", 1, "L", false, 0, "")
}

func addScoreVisualization(pdf *gofpdf.Fpdf, score int) {
	y := pdf.GetY()

	// Score label
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(30, 10, "Score:", "", 0, "L", false, 0, "")

	// Progress bar background
	barWidth := 120.0
	barHeight := 10.0
	barX := 45.0

	pdf.SetFillColor(220, 220, 220)
	pdf.RoundedRect(barX, y, barWidth, barHeight, 2, "1234", "F")

	// Progress bar fill
	fillWidth := barWidth * float64(score) / 100.0
	if score >= 80 {
		pdf.SetFillColor(colorPass[0], colorPass[1], colorPass[2])
	} else if score >= 60 {
		pdf.SetFillColor(colorWarn[0], colorWarn[1], colorWarn[2])
	} else {
		pdf.SetFillColor(colorFail[0], colorFail[1], colorFail[2])
	}
	if fillWidth > 0 {
		pdf.RoundedRect(barX, y, fillWidth, barHeight, 2, "1234", "F")
	}

	// Score text
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(barX, y)
	pdf.CellFormat(barWidth, barHeight, fmt.Sprintf("%d%%", score), "", 0, "C", false, 0, "")

	pdf.SetY(y + barHeight + 2)
}

func addFindingsByCategory(pdf *gofpdf.Fpdf, assessment *assessmentv1alpha1.ClusterAssessment) {
	// Group findings by category
	categories := make(map[string][]assessmentv1alpha1.Finding)
	for _, f := range assessment.Status.Findings {
		categories[f.Category] = append(categories[f.Category], f)
	}

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(0, 0, 0)

	for category, findings := range categories {
		pass, warn, fail, info := 0, 0, 0, 0
		for _, f := range findings {
			switch f.Status {
			case assessmentv1alpha1.FindingStatusPass:
				pass++
			case assessmentv1alpha1.FindingStatusWarn:
				warn++
			case assessmentv1alpha1.FindingStatusFail:
				fail++
			case assessmentv1alpha1.FindingStatusInfo:
				info++
			}
		}

		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(50, 6, category+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)

		statusStr := fmt.Sprintf("%d pass, %d warn, %d fail, %d info", pass, warn, fail, info)
		pdf.CellFormat(0, 6, statusStr, "", 1, "L", false, 0, "")
	}
}

func addDetailedFindings(pdf *gofpdf.Fpdf, assessment *assessmentv1alpha1.ClusterAssessment) {
	// Group findings by status for better organization
	statusOrder := []assessmentv1alpha1.FindingStatus{
		assessmentv1alpha1.FindingStatusFail,
		assessmentv1alpha1.FindingStatusWarn,
		assessmentv1alpha1.FindingStatusInfo,
		assessmentv1alpha1.FindingStatusPass,
	}

	// Optimization: Group findings by status in a single pass (O(N)) instead of repeated filtering (O(4N))
	findingsByStatus := make(map[assessmentv1alpha1.FindingStatus][]assessmentv1alpha1.Finding)
	for _, f := range assessment.Status.Findings {
		findingsByStatus[f.Status] = append(findingsByStatus[f.Status], f)
	}

	for _, status := range statusOrder {
		findings := findingsByStatus[status]
		if len(findings) == 0 {
			continue
		}

		// Status header
		addStatusHeader(pdf, status, len(findings))

		for _, f := range findings {
			addFindingCard(pdf, f)
		}
		pdf.Ln(5)
	}
}

func addStatusHeader(pdf *gofpdf.Fpdf, status assessmentv1alpha1.FindingStatus, count int) {
	var color []int
	var label string

	switch status {
	case assessmentv1alpha1.FindingStatusPass:
		color = colorPass
		label = "PASS"
	case assessmentv1alpha1.FindingStatusWarn:
		color = colorWarn
		label = "WARNING"
	case assessmentv1alpha1.FindingStatusFail:
		color = colorFail
		label = "FAILED"
	case assessmentv1alpha1.FindingStatusInfo:
		color = colorInfo
		label = "INFO"
	}

	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetTextColor(color[0], color[1], color[2])
	pdf.CellFormat(0, 8, fmt.Sprintf("%s (%d)", label, count), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

func addFindingCard(pdf *gofpdf.Fpdf, f assessmentv1alpha1.Finding) {
	// Check if we need a new page
	if pdf.GetY() > 250 {
		pdf.AddPage()
	}

	startY := pdf.GetY()

	// Card background
	pdf.SetFillColor(248, 248, 250)
	pdf.RoundedRect(15, startY, 180, 25, 2, "1234", "F")

	// Status badge
	var color []int
	switch f.Status {
	case assessmentv1alpha1.FindingStatusPass:
		color = colorPass
	case assessmentv1alpha1.FindingStatusWarn:
		color = colorWarn
	case assessmentv1alpha1.FindingStatusFail:
		color = colorFail
	case assessmentv1alpha1.FindingStatusInfo:
		color = colorInfo
	}

	pdf.SetFillColor(color[0], color[1], color[2])
	pdf.RoundedRect(17, startY+2, 8, 8, 1, "1234", "F")

	// Title
	pdf.SetXY(28, startY+2)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(0, 0, 0)

	title := f.Title
	if len(title) > 70 {
		title = title[:67] + "..."
	}
	pdf.CellFormat(0, 5, title, "", 1, "L", false, 0, "")

	// Description
	pdf.SetXY(28, startY+8)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(80, 80, 80)

	desc := f.Description
	if len(desc) > 120 {
		desc = desc[:117] + "..."
	}
	pdf.MultiCell(165, 4, desc, "", "L", false)

	// Category and Validator
	pdf.SetXY(28, startY+18)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(0, 4, fmt.Sprintf("Category: %s | Validator: %s", f.Category, f.Validator), "", 1, "L", false, 0, "")

	// Add recommendation if FAIL or WARN
	if (f.Status == assessmentv1alpha1.FindingStatusFail || f.Status == assessmentv1alpha1.FindingStatusWarn) && f.Recommendation != "" {
		pdf.SetY(startY + 25)
		pdf.SetFillColor(255, 250, 240)
		pdf.RoundedRect(15, pdf.GetY(), 180, 12, 2, "1234", "F")

		pdf.SetXY(17, pdf.GetY()+2)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(100, 80, 60)

		rec := f.Recommendation
		if len(rec) > 150 {
			rec = rec[:147] + "..."
		}
		pdf.MultiCell(176, 4, "Recommendation: "+rec, "", "L", false)
		pdf.Ln(2)
	} else {
		pdf.SetY(startY + 28)
	}

	pdf.Ln(2)
}

// GenerateHTML creates an HTML report that can be easily converted to PDF.
func GenerateHTML(assessment *assessmentv1alpha1.ClusterAssessment) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>OpenShift Cluster Assessment Report</title>
    <style>
        body { font-family: 'Segoe UI', Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 900px; margin: 0 auto; background: white; padding: 40px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #003366; border-bottom: 3px solid #003366; padding-bottom: 10px; }
        h2 { color: #003366; margin-top: 30px; }
        .summary-box { display: inline-block; padding: 15px 25px; margin: 5px; border-radius: 8px; color: white; text-align: center; min-width: 80px; }
        .pass { background: #228B22; }
        .warn { background: #FFA500; }
        .fail { background: #DC143C; }
        .info { background: #4682B4; }
        .count { font-size: 24px; font-weight: bold; }
        .label { font-size: 12px; }
        .finding { background: #f8f8fa; padding: 15px; margin: 10px 0; border-radius: 5px; border-left: 4px solid #ccc; }
        .finding.status-FAIL { border-left-color: #DC143C; }
        .finding.status-WARN { border-left-color: #FFA500; }
        .finding.status-PASS { border-left-color: #228B22; }
        .finding.status-INFO { border-left-color: #4682B4; }
        .finding-title { font-weight: bold; margin-bottom: 5px; }
        .finding-desc { color: #555; margin-bottom: 5px; }
        .finding-meta { font-size: 11px; color: #888; }
        .recommendation { background: #fffaef; padding: 10px; margin-top: 10px; border-radius: 3px; font-style: italic; }
        .info-table { width: 100%; border-collapse: collapse; }
        .info-table td { padding: 8px; border-bottom: 1px solid #eee; }
        .info-table td:first-child { font-weight: bold; width: 200px; }
        .score-bar { background: #ddd; height: 30px; border-radius: 15px; overflow: hidden; margin: 10px 0; }
        .score-fill { height: 100%; display: flex; align-items: center; justify-content: center; color: white; font-weight: bold; }
    </style>
</head>
<body>
<div class="container">
`)

	// Title
	buf.WriteString(fmt.Sprintf(`<h1>OpenShift Cluster Assessment Report</h1>
<p style="color: #888;">Generated: %s</p>
`, time.Now().Format("January 2, 2006 at 15:04 MST")))

	// Cluster Info
	info := assessment.Status.ClusterInfo
	buf.WriteString(`<h2>Cluster Information</h2>
<table class="info-table">`)
	buf.WriteString(fmt.Sprintf(`<tr><td>Cluster ID</td><td>%s</td></tr>`, html.EscapeString(info.ClusterID)))
	buf.WriteString(fmt.Sprintf(`<tr><td>OpenShift Version</td><td>%s</td></tr>`, html.EscapeString(info.ClusterVersion)))
	buf.WriteString(fmt.Sprintf(`<tr><td>Platform</td><td>%s</td></tr>`, html.EscapeString(info.Platform)))
	buf.WriteString(fmt.Sprintf(`<tr><td>Update Channel</td><td>%s</td></tr>`, html.EscapeString(info.Channel)))
	buf.WriteString(fmt.Sprintf(`<tr><td>Total Nodes</td><td>%d</td></tr>`, info.NodeCount))
	buf.WriteString(fmt.Sprintf(`<tr><td>Control Plane Nodes</td><td>%d</td></tr>`, info.ControlPlaneNodes))
	buf.WriteString(fmt.Sprintf(`<tr><td>Worker Nodes</td><td>%d</td></tr>`, info.WorkerNodes))
	buf.WriteString(fmt.Sprintf(`<tr><td>Assessment Profile</td><td>%s</td></tr>`, html.EscapeString(assessment.Spec.Profile)))
	buf.WriteString(`</table>`)

	// Summary
	summary := assessment.Status.Summary
	buf.WriteString(`<h2>Assessment Summary</h2>
<div style="margin: 20px 0;">`)
	buf.WriteString(fmt.Sprintf(`<div class="summary-box pass"><div class="count">%d</div><div class="label">PASS</div></div>`, summary.PassCount))
	buf.WriteString(fmt.Sprintf(`<div class="summary-box warn"><div class="count">%d</div><div class="label">WARN</div></div>`, summary.WarnCount))
	buf.WriteString(fmt.Sprintf(`<div class="summary-box fail"><div class="count">%d</div><div class="label">FAIL</div></div>`, summary.FailCount))
	buf.WriteString(fmt.Sprintf(`<div class="summary-box info"><div class="count">%d</div><div class="label">INFO</div></div>`, summary.InfoCount))
	buf.WriteString(`</div>`)
	buf.WriteString(fmt.Sprintf(`<p>Total Checks: %d</p>`, summary.TotalChecks))

	// Score bar
	if summary.Score != nil {
		scoreColor := "#228B22"
		if *summary.Score < 60 {
			scoreColor = "#DC143C"
		} else if *summary.Score < 80 {
			scoreColor = "#FFA500"
		}
		buf.WriteString(fmt.Sprintf(`<div class="score-bar"><div class="score-fill" style="width: %d%%; background: %s;">%d%%</div></div>`, *summary.Score, scoreColor, *summary.Score))
	}

	// Detailed Findings
	buf.WriteString(`<h2>Detailed Findings</h2>`)

	statusOrder := []assessmentv1alpha1.FindingStatus{
		assessmentv1alpha1.FindingStatusFail,
		assessmentv1alpha1.FindingStatusWarn,
		assessmentv1alpha1.FindingStatusInfo,
		assessmentv1alpha1.FindingStatusPass,
	}

	// Group findings by status
	findingsByStatus := make(map[assessmentv1alpha1.FindingStatus][]assessmentv1alpha1.Finding)
	for _, f := range assessment.Status.Findings {
		findingsByStatus[f.Status] = append(findingsByStatus[f.Status], f)
	}

	for _, status := range statusOrder {
		for _, f := range findingsByStatus[status] {
			buf.WriteString(fmt.Sprintf(`<div class="finding status-%s">`, f.Status))
			buf.WriteString(fmt.Sprintf(`<div class="finding-title">[%s] %s</div>`, f.Status, html.EscapeString(f.Title)))
			buf.WriteString(fmt.Sprintf(`<div class="finding-desc">%s</div>`, html.EscapeString(f.Description)))
			buf.WriteString(fmt.Sprintf(`<div class="finding-meta">Category: %s | Validator: %s</div>`, html.EscapeString(f.Category), html.EscapeString(f.Validator)))
			if f.Recommendation != "" && (f.Status == assessmentv1alpha1.FindingStatusFail || f.Status == assessmentv1alpha1.FindingStatusWarn) {
				buf.WriteString(fmt.Sprintf(`<div class="recommendation">💡 %s</div>`, html.EscapeString(f.Recommendation)))
			}
			if len(f.References) > 0 {
				buf.WriteString(`<div class="finding-meta" style="margin-top: 5px;">References: `)
				for i, ref := range f.References {
					if i > 0 {
						buf.WriteString(", ")
					}
					// Only allow http and https schemes for links to prevent XSS (e.g., javascript:)
					lowerRef := strings.ToLower(ref)
					if strings.HasPrefix(lowerRef, "http://") || strings.HasPrefix(lowerRef, "https://") {
						buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(ref), html.EscapeString(truncateURL(ref))))
					} else {
						// Render unsafe URLs as plain text
						buf.WriteString(html.EscapeString(ref))
					}
				}
				buf.WriteString(`</div>`)
			}
			buf.WriteString(`</div>`)
		}
	}

	buf.WriteString(`</div></body></html>`)

	return buf.Bytes(), nil
}

func truncateURL(url string) string {
	if len(url) > 50 {
		return url[:47] + "..."
	}
	return url
}
