import * as React from 'react';
import {
    Page,
    PageSection,
    Title,
    Grid,
    GridItem,
    Button,
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
    EmptyStateHeader,
    EmptyStateFooter,
    EmptyStateActions,
    Spinner,
    Label,
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
import { ClusterAssessment } from '../types';
import { ScoreGauge } from './ScoreGauge';
import { AssessmentsTable } from './AssessmentsTable';
import CreateAssessmentModal from './CreateAssessmentModal';
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

export default function AssessmentDashboard() {
    const [assessments, loaded, error] = useK8sWatchResource<ClusterAssessment[]>(
        clusterAssessmentResource
    );
    const [isModalOpen, setIsModalOpen] = React.useState(false);

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

    // Get score color class
    const getScoreClass = (score: number) => {
        if (score >= 80) return 'ca-plugin__score-value--good';
        if (score >= 60) return 'ca-plugin__score-value--warning';
        return 'ca-plugin__score-value--critical';
    };

    if (error) {
        return (
            <Page>
                <PageSection>
                    <EmptyState>
                        <EmptyStateIcon icon={ExclamationCircleIcon} />
                        <Title headingLevel="h4" size="lg">Error loading assessments</Title>
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
                        <Title headingLevel="h4" size="lg">Loading assessments...</Title>
                    </EmptyState>
                </PageSection>
            </Page>
        );
    }

    if (!assessments || assessments.length === 0) {
        return (
            <>
                <CreateAssessmentModal
                    isOpen={isModalOpen}
                    onClose={() => setIsModalOpen(false)}
                    onCreated={() => { }}
                />
                <Page>
                    <PageSection>
                        <EmptyState>
                            <EmptyStateHeader
                                titleText="No Assessments Found"
                                headingLevel="h4"
                                icon={<EmptyStateIcon icon={SearchIcon} />}
                            />
                            <EmptyStateBody>
                                Create your first cluster assessment to analyze your OpenShift configuration and get actionable recommendations.
                            </EmptyStateBody>
                            <EmptyStateFooter>
                                <EmptyStateActions>
                                    <Button
                                        variant="primary"
                                        icon={<PlusCircleIcon />}
                                        onClick={() => setIsModalOpen(true)}
                                    >
                                        Create Assessment
                                    </Button>
                                </EmptyStateActions>
                            </EmptyStateFooter>
                        </EmptyState>
                    </PageSection>
                </Page>
            </>
        );
    }

    const score = summary?.score ?? 0;

    return (
        <>
            <Page>
                {/* Header Section */}
                <PageSection variant="light" className="ca-plugin__page-header">
                    <Split hasGutter>
                        <SplitItem isFilled>
                            <Title headingLevel="h1" className="ca-plugin__page-title">
                                Cluster Assessment
                            </Title>
                        </SplitItem>
                        <SplitItem>
                            <Button
                                variant="primary"
                                icon={<PlusCircleIcon />}
                                onClick={() => setIsModalOpen(true)}
                            >
                                Create Assessment
                            </Button>
                        </SplitItem>
                    </Split>
                </PageSection>

                {/* Dashboard Cards Section */}
                <PageSection>
                    <Grid hasGutter lg={4} md={6} sm={12}>
                        {/* Health Score Card */}
                        <GridItem>
                            <div className="ca-plugin__summary-card ca-plugin__score-card">
                                <div className="ca-plugin__card-header">
                                    Health Score
                                </div>
                                <div className="ca-plugin__card-body">
                                    <div className="ca-plugin__score-gauge">
                                        <ScoreGauge score={score} />
                                    </div>
                                    <div className={`ca-plugin__score-value ${getScoreClass(score)}`}>
                                        {score}%
                                    </div>
                                    <div className="ca-plugin__score-label">
                                        {score >= 80 ? 'Good' : score >= 60 ? 'Warning' : 'Critical'}
                                    </div>
                                </div>
                            </div>
                        </GridItem>

                        {/* Findings Summary Card */}
                        <GridItem>
                            <div className="ca-plugin__summary-card">
                                <div className="ca-plugin__card-header">
                                    Findings Summary
                                </div>
                                <div className="ca-plugin__card-body">
                                    <ul className="ca-plugin__findings-list">
                                        <li className="ca-plugin__findings-item">
                                            <span className="ca-plugin__findings-icon ca-plugin__findings-icon--pass">
                                                <CheckCircleIcon />
                                            </span>
                                            <span className="ca-plugin__findings-label">Passed Checks</span>
                                            <span className="ca-plugin__findings-count">{summary?.passCount ?? 0}</span>
                                        </li>
                                        <li className="ca-plugin__findings-item">
                                            <span className="ca-plugin__findings-icon ca-plugin__findings-icon--warn">
                                                <ExclamationTriangleIcon />
                                            </span>
                                            <span className="ca-plugin__findings-label">Warnings</span>
                                            <span className="ca-plugin__findings-count">{summary?.warnCount ?? 0}</span>
                                        </li>
                                        <li className="ca-plugin__findings-item">
                                            <span className="ca-plugin__findings-icon ca-plugin__findings-icon--fail">
                                                <ExclamationCircleIcon />
                                            </span>
                                            <span className="ca-plugin__findings-label">Failed Checks</span>
                                            <span className="ca-plugin__findings-count">{summary?.failCount ?? 0}</span>
                                        </li>
                                        <li className="ca-plugin__findings-item">
                                            <span className="ca-plugin__findings-icon ca-plugin__findings-icon--info">
                                                <InfoCircleIcon />
                                            </span>
                                            <span className="ca-plugin__findings-label">Informational</span>
                                            <span className="ca-plugin__findings-count">{summary?.infoCount ?? 0}</span>
                                        </li>
                                    </ul>
                                </div>
                            </div>
                        </GridItem>

                        {/* Cluster Info Card */}
                        <GridItem>
                            <div className="ca-plugin__summary-card">
                                <div className="ca-plugin__card-header">
                                    Cluster Information
                                </div>
                                <div className="ca-plugin__card-body">
                                    <ul className="ca-plugin__info-list">
                                        <li className="ca-plugin__info-item">
                                            <span className="ca-plugin__info-label">Version</span>
                                            <span className="ca-plugin__info-value">
                                                {latestAssessment?.status?.clusterInfo?.clusterVersion ?? 'N/A'}
                                            </span>
                                        </li>
                                        <li className="ca-plugin__info-item">
                                            <span className="ca-plugin__info-label">Platform</span>
                                            <span className="ca-plugin__info-value">
                                                {latestAssessment?.status?.clusterInfo?.platform ?? 'N/A'}
                                            </span>
                                        </li>
                                        <li className="ca-plugin__info-item">
                                            <span className="ca-plugin__info-label">Nodes</span>
                                            <span className="ca-plugin__info-value">
                                                {latestAssessment?.status?.clusterInfo?.nodeCount ?? 'N/A'}
                                            </span>
                                        </li>
                                        <li className="ca-plugin__info-item">
                                            <span className="ca-plugin__info-label">Profile</span>
                                            <span className="ca-plugin__info-value">
                                                <Label color={latestAssessment?.spec?.profile === 'production' ? 'blue' : 'green'}>
                                                    {latestAssessment?.spec?.profile ?? 'production'}
                                                </Label>
                                            </span>
                                        </li>
                                    </ul>
                                </div>
                            </div>
                        </GridItem>
                    </Grid>

                    {/* Assessments Table */}
                    <div className="ca-plugin__summary-card ca-plugin__table-card" style={{ marginTop: '24px' }}>
                        <div className="ca-plugin__table-header">
                            <h3 className="ca-plugin__table-title">Assessment History</h3>
                        </div>
                        <div className="ca-plugin__table-wrapper">
                            <AssessmentsTable assessments={assessments} />
                        </div>
                    </div>
                </PageSection>
            </Page>
            <CreateAssessmentModal
                isOpen={isModalOpen}
                onClose={() => setIsModalOpen(false)}
                onCreated={() => { }}
            />
        </>
    );
}
