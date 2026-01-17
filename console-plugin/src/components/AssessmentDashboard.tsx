import * as React from 'react';
import {
    Page,
    PageSection,
    Title,
} from '@patternfly/react-core';

// Minimal test component to debug React error #306
const AssessmentDashboard: React.FC = () => {
    return (
        <Page>
            <PageSection variant="light">
                <Title headingLevel="h1">Cluster Assessment</Title>
                <p>Plugin is working! This is a minimal test.</p>
            </PageSection>
        </Page>
    );
};

export default AssessmentDashboard;
