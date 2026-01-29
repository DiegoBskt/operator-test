package resourcequotas

import (
	"context"
	"testing"

	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mockClient struct {
	client.Client
	listCalls              int
	listNamespaceCalls     int
	listPartialCalls       int
	listResourceQuotaCalls int
	listLimitRangeCalls    int
}

func (m *mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	m.listCalls++

	if _, ok := list.(*corev1.NamespaceList); ok {
		m.listNamespaceCalls++
		// Return dummy namespaces
		nl := list.(*corev1.NamespaceList)
		nl.Items = []corev1.Namespace{
			{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "openshift-logging"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "my-app"}},
		}
		return nil
	}

	if _, ok := list.(*metav1.PartialObjectMetadataList); ok {
		m.listPartialCalls++
		pl := list.(*metav1.PartialObjectMetadataList)
		pl.Items = []metav1.PartialObjectMetadata{
			{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: ""}},
			{ObjectMeta: metav1.ObjectMeta{Name: "openshift-logging", Namespace: ""}},
			{ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: ""}},
		}
		return nil
	}

	if _, ok := list.(*corev1.ResourceQuotaList); ok {
		m.listResourceQuotaCalls++
		return nil
	}

	if _, ok := list.(*corev1.LimitRangeList); ok {
		m.listLimitRangeCalls++
		return nil
	}

	return nil
}

func (m *mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return nil
}

func TestValidate_Optimization(t *testing.T) {
	c := &mockClient{}
	v := &ResourceQuotasValidator{}
	profile := profiles.Profile{}

	_, _ = v.Validate(context.Background(), c, profile)

	t.Logf("NamespaceList calls: %d", c.listNamespaceCalls)
	t.Logf("PartialObjectMetadataList calls: %d", c.listPartialCalls)

	// Optimization Assertions
	if c.listPartialCalls != 1 {
		t.Errorf("Expected 1 PartialObjectMetadataList call, got %d", c.listPartialCalls)
	}

	if c.listNamespaceCalls != 0 {
		t.Errorf("Expected 0 NamespaceList calls, got %d", c.listNamespaceCalls)
	}
}
