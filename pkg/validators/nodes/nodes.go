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

package nodes

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "nodes"
	validatorDescription = "Validates node configuration including roles, taints, labels, and kubelet config"
	validatorCategory    = "Infrastructure"
)

func init() {
	_ = validator.Register(&NodesValidator{})
}

// NodesValidator checks node configuration.
type NodesValidator struct{}

// Name returns the validator name.
func (v *NodesValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *NodesValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *NodesValidator) Category() string {
	return validatorCategory
}

// Validate performs node checks.
func (v *NodesValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Get all nodes
	nodes := &corev1.NodeList{}
	if err := c.List(ctx, nodes); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Check 1: Node count
	findings = append(findings, v.checkNodeCount(nodes, profile)...)

	// Check 2: Node conditions
	findings = append(findings, v.checkNodeConditions(nodes)...)

	// Check 3: Node roles
	findings = append(findings, v.checkNodeRoles(nodes)...)

	// Check 4: OS information
	findings = append(findings, v.checkNodeOS(nodes)...)

	// Check 5: Resource pressure
	findings = append(findings, v.checkResourcePressure(nodes)...)

	return findings, nil
}

// checkNodeCount validates the number of nodes.
func (v *NodesValidator) checkNodeCount(nodes *corev1.NodeList, profile profiles.Profile) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding
	var controlPlaneCount, workerCount int

	for _, node := range nodes.Items {
		if v.hasRole(node, "master") || v.hasRole(node, "control-plane") {
			controlPlaneCount++
		}
		if v.hasRole(node, "worker") {
			workerCount++
		}
	}

	// Check control plane nodes
	if controlPlaneCount < profile.Thresholds.MinControlPlaneNodes {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "nodes-control-plane-count",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "Insufficient Control Plane Nodes",
			Description:    fmt.Sprintf("Cluster has %d control plane nodes, minimum recommended is %d for %s profile.", controlPlaneCount, profile.Thresholds.MinControlPlaneNodes, profile.Name),
			Impact:         "Fewer control plane nodes reduce the cluster's ability to tolerate failures and may impact high availability.",
			Recommendation: fmt.Sprintf("Consider adding control plane nodes to meet the minimum of %d for high availability.", profile.Thresholds.MinControlPlaneNodes),
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "nodes-control-plane-count",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Control Plane Node Count",
			Description: fmt.Sprintf("Cluster has %d control plane nodes, meeting the minimum of %d.", controlPlaneCount, profile.Thresholds.MinControlPlaneNodes),
		})
	}

	// Check worker nodes
	if workerCount < profile.Thresholds.MinWorkerNodes {
		status := assessmentv1alpha1.FindingStatusWarn
		if profile.Name == profiles.ProfileProduction {
			status = assessmentv1alpha1.FindingStatusFail
		}
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "nodes-worker-count",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         status,
			Title:          "Insufficient Worker Nodes",
			Description:    fmt.Sprintf("Cluster has %d worker nodes, minimum recommended is %d for %s profile.", workerCount, profile.Thresholds.MinWorkerNodes, profile.Name),
			Impact:         "Fewer worker nodes limit workload capacity and fault tolerance.",
			Recommendation: fmt.Sprintf("Consider adding worker nodes to meet the minimum of %d for better capacity.", profile.Thresholds.MinWorkerNodes),
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "nodes-worker-count",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Worker Node Count",
			Description: fmt.Sprintf("Cluster has %d worker nodes, meeting the minimum of %d.", workerCount, profile.Thresholds.MinWorkerNodes),
		})
	}

	return findings
}

// checkNodeConditions validates node conditions.
func (v *NodesValidator) checkNodeConditions(nodes *corev1.NodeList) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding
	var notReadyNodes []string
	var unhealthyNodes []string

	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			switch condition.Type {
			case corev1.NodeReady:
				if condition.Status != corev1.ConditionTrue {
					notReadyNodes = append(notReadyNodes, node.Name)
				}
			case corev1.NodeMemoryPressure, corev1.NodeDiskPressure, corev1.NodePIDPressure:
				if condition.Status == corev1.ConditionTrue {
					unhealthyNodes = append(unhealthyNodes, fmt.Sprintf("%s (%s)", node.Name, condition.Type))
				}
			}
		}
	}

	if len(notReadyNodes) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "nodes-not-ready",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "Nodes Not Ready",
			Description:    fmt.Sprintf("%d node(s) are not in Ready state: %s", len(notReadyNodes), strings.Join(notReadyNodes, ", ")),
			Impact:         "Nodes that are not ready cannot run workloads and may indicate infrastructure issues.",
			Recommendation: "Investigate the not-ready nodes. Check node status with 'oc describe node <node-name>' and review kubelet logs.",
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "nodes-ready",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "All Nodes Ready",
			Description: fmt.Sprintf("All %d nodes are in Ready state.", len(nodes.Items)),
		})
	}

	if len(unhealthyNodes) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "nodes-pressure",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Nodes Under Resource Pressure",
			Description:    fmt.Sprintf("Nodes experiencing resource pressure: %s", strings.Join(unhealthyNodes, ", ")),
			Impact:         "Nodes under resource pressure may evict pods and degrade workload performance.",
			Recommendation: "Review resource usage on affected nodes and consider adding capacity or rebalancing workloads.",
		})
	}

	return findings
}

// checkNodeRoles validates node role configuration.
func (v *NodesValidator) checkNodeRoles(nodes *corev1.NodeList) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding
	var noRoleNodes []string
	var mixedRoleNodes []string

	for _, node := range nodes.Items {
		hasControlPlane := v.hasRole(node, "master") || v.hasRole(node, "control-plane")
		hasWorker := v.hasRole(node, "worker")

		if !hasControlPlane && !hasWorker {
			// Check for infra role
			if !v.hasRole(node, "infra") {
				noRoleNodes = append(noRoleNodes, node.Name)
			}
		}

		if hasControlPlane && hasWorker {
			mixedRoleNodes = append(mixedRoleNodes, node.Name)
		}
	}

	if len(noRoleNodes) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "nodes-no-role",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Nodes Without Recognized Role",
			Description:    fmt.Sprintf("%d node(s) do not have a recognized role: %s", len(noRoleNodes), strings.Join(noRoleNodes, ", ")),
			Impact:         "Nodes without proper roles may not be included in MachineConfigPools and could have inconsistent configuration.",
			Recommendation: "Ensure nodes have appropriate role labels (worker, master, infra).",
		})
	}

	if len(mixedRoleNodes) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "nodes-mixed-role",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Nodes With Mixed Roles",
			Description:    fmt.Sprintf("%d node(s) have both control-plane and worker roles: %s", len(mixedRoleNodes), strings.Join(mixedRoleNodes, ", ")),
			Impact:         "Mixed-role nodes run both control plane and workloads, which is typical for compact clusters but may affect isolation.",
			Recommendation: "For production workloads, consider using dedicated worker nodes separate from control plane.",
		})
	}

	return findings
}

// checkNodeOS validates node OS information.
func (v *NodesValidator) checkNodeOS(nodes *corev1.NodeList) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding
	osVersions := make(map[string][]string)

	for _, node := range nodes.Items {
		osImage := node.Status.NodeInfo.OSImage
		osVersions[osImage] = append(osVersions[osImage], node.Name)
	}

	// Check for OS consistency
	if len(osVersions) > 1 {
		var osInfo []string
		for os, nodeList := range osVersions {
			osInfo = append(osInfo, fmt.Sprintf("%s (%d nodes)", os, len(nodeList)))
		}
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "nodes-os-mixed",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Mixed OS Versions",
			Description:    fmt.Sprintf("Nodes are running different OS versions: %s", strings.Join(osInfo, ", ")),
			Impact:         "Mixed OS versions can complicate troubleshooting and may indicate incomplete updates.",
			Recommendation: "Ensure all nodes are updated to the same OS version. Check MachineConfigPool status.",
		})
	} else {
		for os := range osVersions {
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:          "nodes-os-consistent",
				Validator:   validatorName,
				Category:    validatorCategory,
				Status:      assessmentv1alpha1.FindingStatusPass,
				Title:       "Consistent Node OS",
				Description: fmt.Sprintf("All nodes are running: %s", os),
			})
			// Check for RHCOS
			if !strings.Contains(strings.ToLower(os), "red hat") && !strings.Contains(strings.ToLower(os), "rhcos") {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:             "nodes-os-not-rhcos",
					Validator:      validatorName,
					Category:       validatorCategory,
					Status:         assessmentv1alpha1.FindingStatusWarn,
					Title:          "Non-RHCOS Operating System",
					Description:    fmt.Sprintf("Nodes are not running Red Hat CoreOS: %s", os),
					Impact:         "Non-RHCOS nodes may have different behavior and reduced support scope.",
					Recommendation: "Consider using Red Hat CoreOS for full OpenShift support coverage.",
				})
			}
		}
	}

	return findings
}

// checkResourcePressure checks for resource constraints.
func (v *NodesValidator) checkResourcePressure(nodes *corev1.NodeList) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding
	var lowMemoryNodes []string
	var lowCPUNodes []string

	for _, node := range nodes.Items {
		allocatable := node.Status.Allocatable
		capacity := node.Status.Capacity

		// Check memory utilization based on allocatable vs capacity
		if allocatable.Memory().Value() > 0 && capacity.Memory().Value() > 0 {
			memoryRatio := float64(allocatable.Memory().Value()) / float64(capacity.Memory().Value())
			if memoryRatio < 0.5 {
				lowMemoryNodes = append(lowMemoryNodes, node.Name)
			}
		}

		// Check CPU
		if allocatable.Cpu().MilliValue() > 0 && capacity.Cpu().MilliValue() > 0 {
			cpuRatio := float64(allocatable.Cpu().MilliValue()) / float64(capacity.Cpu().MilliValue())
			if cpuRatio < 0.5 {
				lowCPUNodes = append(lowCPUNodes, node.Name)
			}
		}
	}

	if len(lowMemoryNodes) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "nodes-low-allocatable-memory",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Low Allocatable Memory",
			Description:    fmt.Sprintf("%d node(s) have less than 50%% memory allocatable: %s", len(lowMemoryNodes), strings.Join(lowMemoryNodes, ", ")),
			Impact:         "Nodes with low allocatable resources have limited capacity for workloads.",
			Recommendation: "Review system reserved resources and consider if nodes need more memory.",
		})
	}

	if len(lowCPUNodes) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "nodes-low-allocatable-cpu",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Low Allocatable CPU",
			Description:    fmt.Sprintf("%d node(s) have less than 50%% CPU allocatable: %s", len(lowCPUNodes), strings.Join(lowCPUNodes, ", ")),
			Impact:         "Nodes with low allocatable CPU have limited capacity for workloads.",
			Recommendation: "Review system reserved resources and kubelet configuration.",
		})
	}

	return findings
}

// hasRole checks if a node has a specific role.
func (v *NodesValidator) hasRole(node corev1.Node, role string) bool {
	_, ok := node.Labels[fmt.Sprintf("node-role.kubernetes.io/%s", role)]
	return ok
}
