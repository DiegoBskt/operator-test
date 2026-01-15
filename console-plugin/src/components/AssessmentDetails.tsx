import * as React from 'react';
import { useParams } from 'react-router-dom';
import {
    Page,
    PageSection,
    Title,
    Card,
    CardTitle,
    CardBody,
    Grid,
    GridItem,
    Breadcrumb,
    BreadcrumbItem,
    Split,
    SplitItem,
    Button,
    Label,
    Flex,
    FlexItem,
    EmptyState,
    EmptyStateBody,
    EmptyStateIcon,
    Spinner,
    Tabs,
    Tab,
    TabTitleText,
} from '@patternfly/react-core';
import {
    ExclamationCircleIcon,
    SyncIcon,
} from '@patternfly/react-icons';
import { useK8sWatchResource } from '@openshift-console/dynamic-plugin-sdk';
import { Link } from 'react-router-dom';
import { ScoreGauge } from './ScoreGauge';
import { FindingsTable } from './FindingsTable';
import { ClusterAssessment } from './AssessmentDashboard';
import './styles.css';

const clusterAssessmentResource = (name: string) => ({
    groupVersionKind: {
        group: 'assessment.openshift.io',
        version: 'v1alpha1',
        kind: 'ClusterAssessment',
    },
    name,
    isList: false,
});

const AssessmentDetails: React.FC = () => {
    const { name } = useParams<{ name: string }>();
    const [activeTabKey, setActiveTabKey] = React.useState<string | number>(0);

    const [assessment, loaded, error] = useK8sWatchResource<ClusterAssessment>(
        clusterAssessmentResource(name)
    );

    if (error) {
        return (
            <Page>
                <PageSection>
                    <EmptyState>
                        <EmptyStateIcon icon={ExclamationCircleIcon} />
                        <Title headingLevel="h4" size="lg">Error loading assessment</Title>
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
                        <Title headingLevel="h4" size="lg">Loading assessment...</Title>
                    </EmptyState>
                </PageSection>
            </Page>
        );
    }

    const summary = assessment?.status?.summary;
    const clusterInfo = assessment?.status?.clusterInfo;
    const findings = assessment?.status?.findings || [];

    return (
        <Page>
            <PageSection variant="light">
                <Breadcrumb>
                    <BreadcrumbItem>
                        <Link to="/cluster-assessment">Cluster Assessment</Link>
                    </BreadcrumbItem>
                    <BreadcrumbItem isActive>{name}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>

            <PageSection variant="light">
                <Split hasGutter>
                    <SplitItem isFilled>
                        <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                            <FlexItem>
                                <Title headingLevel="h1">{name}</Title>
                            </FlexItem>
                            <FlexItem>
                                {assessment?.status?.phase === 'Completed' && (
                                    <Label color="green">Completed</Label>
                                )}
                                {assessment?.status?.phase === 'Running' && (
                                    <Label color="blue">Running</Label>
                                )}
                                {assessment?.status?.phase === 'Failed' && (
                                    <Label color="red">Failed</Label>
                                )}
                                {assessment?.status?.phase === 'Pending' && (
                                    <Label color="grey">Pending</Label>
                                )}
                            </FlexItem>
                        </Flex>
                    </SplitItem>
                    <SplitItem>
                        <Button variant="secondary" icon={<SyncIcon />}>
                            Re-run Assessment
                        </Button>
                    </SplitItem>
                </Split>
            </PageSection>

            <PageSection>
                <Grid hasGutter>
                    {/* Score Card */}
                    <GridItem md={3}>
                        <Card className="ca-plugin__summary-card">
                            <CardTitle>Health Score</CardTitle>
                            <CardBody>
                                <ScoreGauge score={summary?.score ?? 0} />
                            </CardBody>
                        </Card>
                    </GridItem>

                    {/* Cluster Info */}
                    <GridItem md={3}>
                        <Card className="ca-plugin__summary-card">
                            <CardTitle>Cluster Info</CardTitle>
                            <CardBody>
                                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                    <FlexItem>
                                        <strong>Version:</strong> {clusterInfo?.clusterVersion ?? 'N/A'}
                                    </FlexItem>
                                    <FlexItem>
                                        <strong>Platform:</strong> {clusterInfo?.platform ?? 'N/A'}
                                    </FlexItem>
                                    <FlexItem>
                                        <strong>Nodes:</strong> {clusterInfo?.nodeCount ?? 'N/A'}
                                    </FlexItem>
                                </Flex>
                            </CardBody>
                        </Card>
                    </GridItem>

                    {/* Assessment Config */}
                    <GridItem md={3}>
                        <Card className="ca-plugin__summary-card">
                            <CardTitle>Configuration</CardTitle>
                            <CardBody>
                                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                    <FlexItem>
                                        <strong>Profile:</strong>{' '}
                                        <Label color={assessment?.spec?.profile === 'production' ? 'blue' : 'green'}>
                                            {assessment?.spec?.profile ?? 'production'}
                                        </Label>
                                    </FlexItem>
                                    <FlexItem>
                                        <strong>Schedule:</strong> {assessment?.spec?.schedule || 'One-time'}
                                    </FlexItem>
                                    <FlexItem>
                                        <strong>Last Run:</strong>{' '}
                                        {assessment?.status?.lastRunTime
                                            ? new Date(assessment.status.lastRunTime).toLocaleString()
                                            : 'Never'}
                                    </FlexItem>
                                </Flex>
                            </CardBody>
                        </Card>
                    </GridItem>

                    {/* Summary Stats */}
                    <GridItem md={3}>
                        <Card className="ca-plugin__summary-card">
                            <CardTitle>Results Summary</CardTitle>
                            <CardBody>
                                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsSm' }}>
                                    <FlexItem>
                                        <strong>Total Checks:</strong> {summary?.totalChecks ?? 0}
                                    </FlexItem>
                                    <FlexItem style={{ color: 'var(--pf-global--success-color--100)' }}>
                                        <strong>Pass:</strong> {summary?.passCount ?? 0}
                                    </FlexItem>
                                    <FlexItem style={{ color: 'var(--pf-global--warning-color--100)' }}>
                                        <strong>Warn:</strong> {summary?.warnCount ?? 0}
                                    </FlexItem>
                                    <FlexItem style={{ color: 'var(--pf-global--danger-color--100)' }}>
                                        <strong>Fail:</strong> {summary?.failCount ?? 0}
                                    </FlexItem>
                                </Flex>
                            </CardBody>
                        </Card>
                    </GridItem>

                    {/* Findings Table */}
                    <GridItem span={12}>
                        <Card>
                            <CardBody>
                                <Tabs
                                    activeKey={activeTabKey}
                                    onSelect={(_event, tabIndex) => setActiveTabKey(tabIndex)}
                                >
                                    <Tab eventKey={0} title={<TabTitleText>All Findings ({findings.length})</TabTitleText>}>
                                        <div style={{ paddingTop: '16px' }}>
                                            <FindingsTable findings={findings} />
                                        </div>
                                    </Tab>
                                    <Tab
                                        eventKey={1}
                                        title={
                                            <TabTitleText>
                                                Issues ({findings.filter((f) => f.status === 'FAIL' || f.status === 'WARN').length})
                                            </TabTitleText>
                                        }
                                    >
                                        <div style={{ paddingTop: '16px' }}>
                                            <FindingsTable
                                                findings={findings.filter((f) => f.status === 'FAIL' || f.status === 'WARN')}
                                            />
                                        </div>
                                    </Tab>
                                </Tabs>
                            </CardBody>
                        </Card>
                    </GridItem>
                </Grid>
            </PageSection>
        </Page>
    );
};

export default AssessmentDetails;
