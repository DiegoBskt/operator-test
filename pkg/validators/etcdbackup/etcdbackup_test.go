package etcdbackup

import (
	"context"
	"fmt"
	"testing"
	"time"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mockClient struct {
	client.Client
	listCalls        int
	getCalls         int
	listType         string // "UnstructuredList" or "PartialObjectMetadataList"
	getCronJobCalled []string
}

func (m *mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	m.listCalls++

	// Handle ConfigMapList
	if _, ok := list.(*corev1.ConfigMapList); ok {
		return nil
	}

	// Handle OADP DataProtectionApplicationList (called in checkOADP but we test checkBackupCronJobs specifically)
	// But if we tested Validate(), we'd need to handle it.
	// Here we stick to checkBackupCronJobs.

	// Populate data for CronJobs
	items := []struct {
		Name      string
		Namespace string
	}{
		{"etcd-backup-job", "default"},
		{"random-job", "default"},
	}

	if l, ok := list.(*unstructured.UnstructuredList); ok {
		// Check GVK to make sure it's asking for CronJobs
		gvk := l.GroupVersionKind()
		if gvk.Kind == "CronJobList" {
			m.listType = "UnstructuredList"
			for _, item := range items {
				u := unstructured.Unstructured{}
				u.SetName(item.Name)
				u.SetNamespace(item.Namespace)
				u.SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"})
				// Add status for UnstructuredList
				u.Object = map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      item.Name,
						"namespace": item.Namespace,
					},
					"status": map[string]interface{}{
						"lastScheduleTime": time.Now().Format(time.RFC3339),
					},
				}
				l.Items = append(l.Items, u)
			}
			return nil
		}
	} else if l, ok := list.(*metav1.PartialObjectMetadataList); ok {
		gvk := l.GroupVersionKind()
		if gvk.Kind == "CronJobList" {
			m.listType = "PartialObjectMetadataList"
			for _, item := range items {
				p := metav1.PartialObjectMetadata{}
				p.Name = item.Name
				p.Namespace = item.Namespace
				p.SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"})
				l.Items = append(l.Items, p)
			}
			return nil
		}
	} else if l, ok := list.(*unstructured.UnstructuredList); ok {
		// Fallback for other UnstructuredLists if any
		_ = l
	}

	return nil // Return empty for others or unexpected
}

func (m *mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {

	if u, ok := obj.(*unstructured.Unstructured); ok {
		// Only count calls for CronJobs
		if u.GetKind() == "CronJob" || (u.GetObjectKind().GroupVersionKind().Kind == "CronJob") {
			m.getCalls++
			m.getCronJobCalled = append(m.getCronJobCalled, key.Name)

			u.SetName(key.Name)
			u.SetNamespace(key.Namespace)
			u.SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"})
			u.Object = map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      key.Name,
					"namespace": key.Namespace,
				},
				"status": map[string]interface{}{
					"lastScheduleTime": time.Now().Format(time.RFC3339),
				},
			}
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func TestCheckBackupCronJobs_Logic(t *testing.T) {
	c := &mockClient{}
	v := &EtcdBackupValidator{}

	findings := v.checkBackupCronJobs(context.Background(), c)

	found := false
	for _, f := range findings {
		if f.ID == "etcdbackup-cronjob-default-etcd-backup-job" {
			found = true
			if f.Status != assessmentv1alpha1.FindingStatusPass {
				t.Errorf("Expected Pass, got %s", f.Status)
			}
		}
	}
	if !found {
		t.Errorf("Did not find etcd-backup-job finding")
	}

	// Verify correct implementation usage
	if c.listType == "" {
		t.Log("Warning: listType was not captured (maybe List() returned before CronJob list?)")
	}

	// Optimized: Should use PartialObjectMetadataList
	if c.listType != "PartialObjectMetadataList" {
		t.Errorf("Expected PartialObjectMetadataList, got %s", c.listType)
	}
	// Optimized: Should have 1 Get call (for etcd-backup-job only)
	if c.getCalls != 1 {
		t.Errorf("Expected 1 Get call, got %d", c.getCalls)
	}
}
