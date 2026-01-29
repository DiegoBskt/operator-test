import * as React from 'react';
import {
    Modal,
    ModalVariant,
    Button,
    Form,
    FormGroup,
    TextInput,
    FormSelect,
    FormSelectOption,
    Checkbox,
    Alert,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';
import { k8sCreate, K8sModel } from '@openshift-console/dynamic-plugin-sdk';

interface CreateAssessmentModalProps {
    isOpen: boolean;
    onClose: () => void;
    onCreated: () => void;
}

const clusterAssessmentModel: K8sModel = {
    apiVersion: 'v1alpha1',
    apiGroup: 'assessment.openshift.io',
    kind: 'ClusterAssessment',
    plural: 'clusterassessments',
    abbr: 'CA',
    label: 'Cluster Assessment',
    labelPlural: 'Cluster Assessments',
    namespaced: false,
};

export default function CreateAssessmentModal({
    isOpen,
    onClose,
    onCreated,
}: CreateAssessmentModalProps) {
    const [name, setName] = React.useState('');
    const [profile, setProfile] = React.useState('production');
    const [enableHtml, setEnableHtml] = React.useState(true);
    const [enableJson, setEnableJson] = React.useState(true);
    const [isSubmitting, setIsSubmitting] = React.useState(false);
    const [error, setError] = React.useState<string | null>(null);

    const handleSubmit = async () => {
        if (!name.trim()) {
            setError('Name is required');
            return;
        }

        setIsSubmitting(true);
        setError(null);

        const formats: string[] = [];
        if (enableHtml) formats.push('html');
        if (enableJson) formats.push('json');

        const resource = {
            apiVersion: 'assessment.openshift.io/v1alpha1',
            kind: 'ClusterAssessment',
            metadata: {
                name: name.trim(),
            },
            spec: {
                profile,
                reportStorage: {
                    configMap: {
                        enabled: true,
                        format: formats.join(','),
                    },
                },
            },
        };

        try {
            await k8sCreate({ model: clusterAssessmentModel, data: resource });
            setIsSubmitting(false);
            setName('');
            setProfile('production');
            onCreated();
            onClose();
        } catch (err) {
            setIsSubmitting(false);
            setError(err instanceof Error ? err.message : String(err));
        }
    };

    const handleClose = () => {
        setName('');
        setProfile('production');
        setError(null);
        onClose();
    };

    const profileOptions = [
        { value: 'production', label: 'Production (Strict)' },
        { value: 'development', label: 'Development (Relaxed)' },
    ];

    return (
        <Modal
            variant={ModalVariant.medium}
            title="Create Cluster Assessment"
            isOpen={isOpen}
            onClose={handleClose}
            actions={[
                <Button
                    key="create"
                    variant="primary"
                    onClick={handleSubmit}
                    isDisabled={isSubmitting || !name.trim()}
                    isLoading={isSubmitting}
                >
                    Create
                </Button>,
                <Button key="cancel" variant="link" onClick={handleClose}>
                    Cancel
                </Button>,
            ]}
        >
            <Form>
                {error && (
                    <Alert variant="danger" isInline title="Error creating assessment">
                        {error}
                    </Alert>
                )}
                <FormGroup label="Name" isRequired fieldId="assessment-name">
                    <TextInput
                        isRequired
                        autoFocus
                        id="assessment-name"
                        value={name}
                        onChange={(_event, value) => setName(value)}
                        placeholder="my-assessment"
                    />
                </FormGroup>
                <FormGroup label="Profile" fieldId="assessment-profile">
                    <FormSelect
                        id="assessment-profile"
                        value={profile}
                        onChange={(_event, value) => setProfile(value)}
                    >
                        {profileOptions.map((option) => (
                            <FormSelectOption
                                key={option.value}
                                value={option.value}
                                label={option.label}
                            />
                        ))}
                    </FormSelect>
                    <FormHelperText>
                        <HelperText>
                            <HelperTextItem variant="default">
                                {profile === 'production'
                                    ? 'Strict checks suitable for production environments.'
                                    : 'Relaxed checks suitable for development or test environments.'}
                            </HelperTextItem>
                        </HelperText>
                    </FormHelperText>
                </FormGroup>
                <FormGroup label="Report Formats" role="group">
                    <Checkbox
                        id="format-html"
                        label="HTML Report"
                        isChecked={enableHtml}
                        onChange={(_event, checked) => setEnableHtml(checked)}
                    />
                    <Checkbox
                        id="format-json"
                        label="JSON Report"
                        isChecked={enableJson}
                        onChange={(_event, checked) => setEnableJson(checked)}
                    />
                </FormGroup>
            </Form>
        </Modal>
    );
}
