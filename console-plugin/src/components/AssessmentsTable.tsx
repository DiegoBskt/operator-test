import * as React from 'react';
import {
    Table,
    TableHeader,
    TableBody,
    IRow,
    ICell,
} from '@patternfly/react-table';
import {
    Label,
    Button,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom';
import { ClusterAssessment } from './AssessmentDashboard';

interface AssessmentsTableProps {
    assessments: ClusterAssessment[];
}

const getPhaseLabel = (phase?: string) => {
    switch (phase) {
        case 'Completed':
            return <Label color="green">{phase}</Label>;
        case 'Running':
            return <Label color="blue">{phase}</Label>;
        case 'Failed':
            return <Label color="red">{phase}</Label>;
        case 'Pending':
        default:
            return <Label color="grey">{phase || 'Pending'}</Label>;
    }
};

const getProfileLabel = (profile?: string) => {
    switch (profile) {
        case 'production':
            return <Label color="blue">Production</Label>;
        case 'development':
            return <Label color="green">Development</Label>;
        default:
            return <Label>{profile || 'production'}</Label>;
    }
};

const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleString();
};

export const AssessmentsTable: React.FC<AssessmentsTableProps> = ({ assessments }) => {
    const columns: ICell[] = [
        { title: 'Name' },
        { title: 'Profile' },
        { title: 'Phase' },
        { title: 'Score' },
        { title: 'Pass' },
        { title: 'Warn' },
        { title: 'Fail' },
        { title: 'Last Run' },
        { title: '' },
    ];

    const sortedAssessments = React.useMemo(() => {
        return [...assessments].sort((a, b) => {
            const timeA = a.status?.lastRunTime || a.metadata.creationTimestamp;
            const timeB = b.status?.lastRunTime || b.metadata.creationTimestamp;
            return new Date(timeB).getTime() - new Date(timeA).getTime();
        });
    }, [assessments]);

    const rows: IRow[] = sortedAssessments.map((assessment) => ({
        cells: [
            {
                title: (
                    <Link to={`/cluster-assessment/${assessment.metadata.name}`}>
                        {assessment.metadata.name}
                    </Link>
                ),
            },
            { title: getProfileLabel(assessment.spec?.profile) },
            { title: getPhaseLabel(assessment.status?.phase) },
            { title: assessment.status?.summary?.score ?? '-' },
            {
                title: (
                    <span style={{ color: 'var(--pf-global--success-color--100)' }}>
                        {assessment.status?.summary?.passCount ?? 0}
                    </span>
                ),
            },
            {
                title: (
                    <span style={{ color: 'var(--pf-global--warning-color--100)' }}>
                        {assessment.status?.summary?.warnCount ?? 0}
                    </span>
                ),
            },
            {
                title: (
                    <span style={{ color: 'var(--pf-global--danger-color--100)' }}>
                        {assessment.status?.summary?.failCount ?? 0}
                    </span>
                ),
            },
            { title: formatDate(assessment.status?.lastRunTime) },
            {
                title: (
                    <Button
                        variant="link"
                        component={(props: any) => (
                            <Link {...props} to={`/cluster-assessment/${assessment.metadata.name}`} />
                        )}
                    >
                        View
                    </Button>
                ),
            },
        ],
    }));

    return (
        <Table aria-label="Cluster assessments table" cells={columns} rows={rows}>
            <TableHeader />
            <TableBody />
        </Table>
    );
};

export default AssessmentsTable;
