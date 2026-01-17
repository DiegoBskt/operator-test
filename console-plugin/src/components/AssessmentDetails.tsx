import * as React from 'react';
import {
    Page,
    PageSection,
    Title,
} from '@patternfly/react-core';

// Minimal test component for AssessmentDetails
const AssessmentDetails: React.FC = () => {
    return (
        <Page>
            <PageSection variant="light">
                <Title headingLevel="h1">Assessment Details</Title>
                <p>Details view is working!</p>
            </PageSection>
        </Page>
    );
};

export default AssessmentDetails;
