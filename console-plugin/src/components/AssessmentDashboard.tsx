import * as React from 'react';
import {
    Page,
    PageSection,
    Title,
    Card,
    CardTitle,
    CardBody,
    Grid,
    GridItem,
    Button,
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
    EmptyStateHeader,
    EmptyStateFooter,
    Spinner,
    Label,
    Flex,
    FlexItem,
    Split,
    SplitItem,
} from '@patternfly/react-core';
import {
    CheckCircleIcon,
    ExclamationTriangleIcon,
    ExclamationCircleIcon,
    InfoCircleIcon,
    SearchIcon,
    PlusCircleIcon,
} from '@patternfly/react-icons';
import { useK8sWatchResource } from '@openshift-console/dynamic-plugin-sdk';
import { Link } from 'react-router-dom';
import { ScoreGauge } from './ScoreGauge';
import { AssessmentsTable } from './AssessmentsTable';
import './styles.css';

// ClusterAssessment resource type
const clusterAssessmentResource = {
    groupVersionKind: {
        group: 'assessment.openshift.io',
        version: 'v1alpha1',
        kind: 'ClusterAssessment',
    },
    isList: true,
};

export interface ClusterAssessment {
    metadata: {
        name: string;
        creationTimestamp: string;
    };
    spec: {
        profile?: string;
        schedule?: string;
    };
    status?: {
        phase?: string;
        lastRunTime?: string;
        summary?: {
            score?: number;
            passCount: number;
            warnCount: number;
            failCount: number;
            infoCount: number;
            totalChecks: number;
        };
        clusterInfo?: {
            clusterVersion?: string;
            platform?: string;
            nodeCount?: number;
        };
    };
}

const AssessmentDashboard: React.FC = () => {
    const [assessments, loaded, error] = useK8sWatchResource<ClusterAssessment[]>(
        clusterAssessmentResource
    );

    // Get the most recent assessment for summary stats
    const latestAssessment = React.useMemo(() => {
        if (!assessments || assessments.length === 0) return null;
        return assessments.sort((a, b) => {
            const timeA = a.status?.lastRunTime || a.metadata.creationTimestamp;
            const timeB = b.status?.lastRunTime || b.metadata.creationTimestamp;
            return new Date(timeB).getTime() - new Date(timeA).getTime();
        })[0];
    }, [assessments]);

    const summary = latestAssessment?.status?.summary;

    if (error) {
        return (
            <Page>
                <PageSection>
                    <EmptyState>
                        <EmptyStateHeader
                            titleText="Error loading assessments"
                            icon={<EmptyStateIcon icon={ExclamationCircleIcon} />}
                        />
                        <EmptyStateBody>{String(error)}</EmptyStateBody>
                    </EmptyState>
                </PageSection>
            </Page>
        );
    }

    if (!loaded) {
        return (
            <Page>
                <PageSection>
                    <EmptyState>
                        <Spinner size="xl" />
                        <EmptyStateBody>Loading assessments...</EmptyStateBody>
                    </EmptyState>
                </PageSection>
            </Page>
        );
    }

    if (!assessments || assessments.length === 0) {
        return (
            <Page>
                <PageSection>
                    <EmptyState>
                        <EmptyStateHeader
                            titleText="No assessments found"
                            icon={<EmptyStateIcon icon={SearchIcon} />}
                        />
                        <EmptyStateBody>
                            Create your first cluster assessment to analyze your OpenShift configuration.
                        </EmptyStateBody>
                        <EmptyStateFooter>
                            <Button variant="primary" icon={<PlusCircleIcon />}>
                                Create Assessment
                            </Button>
                        </EmptyStateFooter>
                    </EmptyState>
                </PageSection>
            </Page>
        );
    }

    return (
        <Page>
            <PageSection variant="light">
                <Split hasGutter>
                    <SplitItem isFilled>
                        <Title headingLevel="h1">Cluster Assessment</Title>
                    </SplitItem>
                    <SplitItem>
                        <Button variant="primary" icon={<PlusCircleIcon />}>
                            Create Assessment
                        </Button>
                    </SplitItem>
                </Split>
            </PageSection>

            <PageSection>
                <Grid hasGutter>
                    {/* Score Card */}
                    <GridItem md={4}>
                        <Card className="ca-plugin__summary-card">
                            <CardTitle>Health Score</CardTitle>
                            <CardBody>
                                <ScoreGauge score={summary?.score ?? 0} />
                            </CardBody>
                        </Card>
                    </GridItem>

                    {/* Findings Summary Card */}
                    <GridItem md={4}>
                        <Card className="ca-plugin__summary-card">
                            <CardTitle>Findings Summary</CardTitle>
                            <CardBody>
                                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                                    <FlexItem>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <CheckCircleIcon color="var(--pf-v5-global--success-color--100)" />
                                            </FlexItem>
                                            <FlexItem>Pass: {summary?.passCount ?? 0}</FlexItem>
                                        </Flex>
                                    </FlexItem>
                                    <FlexItem>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <ExclamationTriangleIcon color="var(--pf-v5-global--warning-color--100)" />
                                            </FlexItem>
                                            <FlexItem>Warn: {summary?.warnCount ?? 0}</FlexItem>
                                        </Flex>
                                    </FlexItem>
                                    <FlexItem>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />
                                            </FlexItem>
                                            <FlexItem>Fail: {summary?.failCount ?? 0}</FlexItem>
                                        </Flex>
                                    </FlexItem>
                                    <FlexItem>
                                        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
                                            <FlexItem>
                                                <InfoCircleIcon color="var(--pf-v5-global--info-color--100)" />
                                            </FlexItem>
                                            <FlexItem>Info: {summary?.infoCount ?? 0}</FlexItem>
                                        </Flex>
                                    </FlexItem>
                                </Flex>
                            </CardBody>
                        </Card>
                    </GridItem>

                    {/* Cluster Info Card */}
                    <GridItem md={4}>
                        <Card className="ca-plugin__summary-card">
                            <CardTitle>Cluster Info</CardTitle>
                            <CardBody>
                                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
                                    <FlexItem>
                                        <strong>Version:</strong>{' '}
                                        {latestAssessment?.status?.clusterInfo?.clusterVersion ?? 'N/A'}
                                    </FlexItem>
                                    <FlexItem>
                                        <strong>Platform:</strong>{' '}
                                        {latestAssessment?.status?.clusterInfo?.platform ?? 'N/A'}
                                    </FlexItem>
                                    <FlexItem>
                                        <strong>Nodes:</strong>{' '}
                                        {latestAssessment?.status?.clusterInfo?.nodeCount ?? 'N/A'}
                                    </FlexItem>
                                    <FlexItem>
                                        <strong>Profile:</strong>{' '}
                                        <Label color={latestAssessment?.spec?.profile === 'production' ? 'blue' : 'green'}>
                                            {latestAssessment?.spec?.profile ?? 'production'}
                                        </Label>
                                    </FlexItem>
                                </Flex>
                            </CardBody>
                        </Card>
                    </GridItem>

                    {/* Assessments Table */}
                    <GridItem span={12}>
                        <Card>
                            <CardTitle>Assessments</CardTitle>
                            <CardBody>
                                <AssessmentsTable assessments={assessments} />
                            </CardBody>
                        </Card>
                    </GridItem>
                </Grid>
            </PageSection>
        </Page>
    );
};

export default AssessmentDashboard;
