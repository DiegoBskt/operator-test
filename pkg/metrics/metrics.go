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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// AssessmentScore is a gauge that tracks the overall assessment score (0-100)
	AssessmentScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_assessment_score",
			Help: "Overall cluster assessment score (0-100)",
		},
		[]string{"assessment_name", "profile"},
	)

	// FindingsTotal is a gauge that tracks the number of findings by status
	FindingsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_assessment_findings_total",
			Help: "Total number of findings by status",
		},
		[]string{"assessment_name", "status"},
	)

	// FindingsByCategory is a gauge that tracks findings by category and status
	FindingsByCategory = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_assessment_findings_by_category",
			Help: "Number of findings by category and status",
		},
		[]string{"assessment_name", "category", "status"},
	)

	// LastRunTimestamp is a gauge that tracks when the last assessment ran
	LastRunTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_assessment_last_run_timestamp",
			Help: "Unix timestamp of the last assessment run",
		},
		[]string{"assessment_name"},
	)

	// AssessmentDuration is a gauge that tracks how long the assessment took
	AssessmentDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_assessment_duration_seconds",
			Help: "Duration of the last assessment in seconds",
		},
		[]string{"assessment_name"},
	)

	// ValidatorFindings is a gauge that tracks findings per validator
	ValidatorFindings = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_assessment_validator_findings",
			Help: "Number of findings per validator",
		},
		[]string{"assessment_name", "validator", "status"},
	)

	// ClusterInfo is a gauge that provides cluster metadata as labels
	ClusterInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_assessment_cluster_info",
			Help: "Cluster information (always 1, use labels for metadata)",
		},
		[]string{"cluster_id", "cluster_version", "platform", "channel"},
	)
)

func init() {
	// Register metrics with the controller-runtime metrics registry
	metrics.Registry.MustRegister(
		AssessmentScore,
		FindingsTotal,
		FindingsByCategory,
		LastRunTimestamp,
		AssessmentDuration,
		ValidatorFindings,
		ClusterInfo,
	)
}

// RecordAssessmentMetrics records all metrics for an assessment
func RecordAssessmentMetrics(
	assessmentName string,
	profile string,
	score int,
	passCount, warnCount, failCount, infoCount int,
	lastRunUnix float64,
	durationSeconds float64,
) {
	// Record score
	AssessmentScore.WithLabelValues(assessmentName, profile).Set(float64(score))

	// Record findings by status
	FindingsTotal.WithLabelValues(assessmentName, "PASS").Set(float64(passCount))
	FindingsTotal.WithLabelValues(assessmentName, "WARN").Set(float64(warnCount))
	FindingsTotal.WithLabelValues(assessmentName, "FAIL").Set(float64(failCount))
	FindingsTotal.WithLabelValues(assessmentName, "INFO").Set(float64(infoCount))

	// Record timestamp and duration
	LastRunTimestamp.WithLabelValues(assessmentName).Set(lastRunUnix)
	AssessmentDuration.WithLabelValues(assessmentName).Set(durationSeconds)
}

// RecordClusterInfo records cluster metadata as a metric
func RecordClusterInfo(clusterID, clusterVersion, platform, channel string) {
	ClusterInfo.WithLabelValues(clusterID, clusterVersion, platform, channel).Set(1)
}

// RecordValidatorMetrics records findings for a specific validator
func RecordValidatorMetrics(assessmentName, validator string, passCount, warnCount, failCount, infoCount int) {
	ValidatorFindings.WithLabelValues(assessmentName, validator, "PASS").Set(float64(passCount))
	ValidatorFindings.WithLabelValues(assessmentName, validator, "WARN").Set(float64(warnCount))
	ValidatorFindings.WithLabelValues(assessmentName, validator, "FAIL").Set(float64(failCount))
	ValidatorFindings.WithLabelValues(assessmentName, validator, "INFO").Set(float64(infoCount))
}

// RecordCategoryMetrics records findings for a category
func RecordCategoryMetrics(assessmentName, category string, passCount, warnCount, failCount, infoCount int) {
	FindingsByCategory.WithLabelValues(assessmentName, category, "PASS").Set(float64(passCount))
	FindingsByCategory.WithLabelValues(assessmentName, category, "WARN").Set(float64(warnCount))
	FindingsByCategory.WithLabelValues(assessmentName, category, "FAIL").Set(float64(failCount))
	FindingsByCategory.WithLabelValues(assessmentName, category, "INFO").Set(float64(infoCount))
}
