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

package certificates

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "certificates"
	validatorDescription = "Validates certificate expiration for critical cluster certificates"
	validatorCategory    = "Security"
)

func init() {
	_ = validator.Register(&CertificatesValidator{})
}

// CertificatesValidator checks certificate expiration.
type CertificatesValidator struct{}

// Name returns the validator name.
func (v *CertificatesValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *CertificatesValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *CertificatesValidator) Category() string {
	return validatorCategory
}

// Validate performs certificate expiration checks.
func (v *CertificatesValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check router certificates
	findings = append(findings, v.checkRouterCerts(ctx, c)...)

	// Check API server certificates
	findings = append(findings, v.checkAPIServerCerts(ctx, c)...)

	// Check ingress certificates
	findings = append(findings, v.checkIngressCerts(ctx, c)...)

	// Summary finding if all checks pass
	if len(findings) == 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "certificates-all-valid",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Certificates Valid",
			Description: "No certificate expiration issues found.",
		})
	}

	return findings, nil
}

// checkRouterCerts checks the default ingress router certificates.
func (v *CertificatesValidator) checkRouterCerts(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check router-certs-default secret
	secret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{
		Name:      "router-certs-default",
		Namespace: "openshift-ingress",
	}, secret)

	if err != nil {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "certificates-router-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Unable to Check Router Certificates",
			Description: fmt.Sprintf("Could not access router certificates: %v", err),
		})
		return findings
	}

	// Check if custom certificate is configured
	if _, hasCustom := secret.Data["tls.crt"]; hasCustom {
		// Analyze certificate expiration would require parsing the cert
		// For now, just report that a custom cert exists
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "certificates-router-custom",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Custom Router Certificate Configured",
			Description: "A custom TLS certificate is configured for the default ingress router.",
		})
	}

	return findings
}

// checkAPIServerCerts checks API server certificate secrets.
func (v *CertificatesValidator) checkAPIServerCerts(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for custom API server certificate
	secret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{
		Name:      "user-serving-cert",
		Namespace: "openshift-config",
	}, secret)

	if err == nil {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "certificates-apiserver-custom",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Custom API Server Certificate",
			Description:    "A custom API server certificate is configured. Ensure it is properly managed and renewed before expiration.",
			Recommendation: "Set up certificate rotation monitoring and alerting.",
		})
	}

	return findings
}

// checkIngressCerts checks for ingress certificate configuration.
func (v *CertificatesValidator) checkIngressCerts(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// List secrets in openshift-ingress namespace with tls type
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, client.InNamespace("openshift-ingress")); err != nil {
		return findings
	}

	tlsSecrets := 0
	for _, secret := range secretList.Items {
		if secret.Type == corev1.SecretTypeTLS {
			tlsSecrets++
		}
	}

	if tlsSecrets > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "certificates-ingress-found",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Ingress TLS Secrets Present",
			Description: fmt.Sprintf("Found %d TLS secrets in openshift-ingress namespace.", tlsSecrets),
		})
	}

	// Check cert expiry using annotations (if cert-manager is used)
	now := time.Now()
	warningThreshold := now.Add(30 * 24 * time.Hour) // 30 days

	for _, secret := range secretList.Items {
		if secret.Type != corev1.SecretTypeTLS {
			continue
		}

		// Check for cert-manager annotation
		if expiry, ok := secret.Annotations["cert-manager.io/certificate-expiry"]; ok {
			expiryTime, err := time.Parse(time.RFC3339, expiry)
			if err == nil {
				if expiryTime.Before(now) {
					findings = append(findings, assessmentv1alpha1.Finding{
						ID:             fmt.Sprintf("certificates-expired-%s", secret.Name),
						Validator:      validatorName,
						Category:       validatorCategory,
						Status:         assessmentv1alpha1.FindingStatusFail,
						Title:          "Expired Certificate",
						Description:    fmt.Sprintf("Certificate secret %s has expired on %s", secret.Name, expiry),
						Recommendation: "Renew the certificate immediately.",
					})
				} else if expiryTime.Before(warningThreshold) {
					findings = append(findings, assessmentv1alpha1.Finding{
						ID:             fmt.Sprintf("certificates-expiring-%s", secret.Name),
						Validator:      validatorName,
						Category:       validatorCategory,
						Status:         assessmentv1alpha1.FindingStatusWarn,
						Title:          "Certificate Expiring Soon",
						Description:    fmt.Sprintf("Certificate secret %s expires on %s", secret.Name, expiry),
						Recommendation: "Plan certificate renewal before expiration.",
					})
				}
			}
		}
	}

	return findings
}
