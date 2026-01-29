import * as React from 'react';
import {
    Table,
    Thead,
    Tbody,
    Tr,
    Th,
    Td,
} from '@patternfly/react-table';
import {
    Label,
    Button,
} from '@patternfly/react-core';
import { Link } from 'react-router-dom';
import { ClusterAssessment } from '../types';

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

export default function AssessmentsTable({ assessments }: AssessmentsTableProps) {
    const sortedAssessments = React.useMemo(() => {
        return [...assessments].sort((a, b) => {
            const timeA = a.status?.lastRunTime || a.metadata.creationTimestamp;
            const timeB = b.status?.lastRunTime || b.metadata.creationTimestamp;
            return new Date(timeB).getTime() - new Date(timeA).getTime();
        });
    }, [assessments]);

    return (
        <Table aria-label="Cluster assessments table" variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Profile</Th>
                    <Th>Phase</Th>
                    <Th>Score</Th>
                    <Th>Pass</Th>
                    <Th>Warn</Th>
                    <Th>Fail</Th>
                    <Th>Last Run</Th>
                    <Th></Th>
                </Tr>
            </Thead>
            <Tbody>
                {sortedAssessments.map((assessment) => (
                    <Tr key={assessment.metadata.name}>
                        <Td dataLabel="Name">
                            <Link to={`/cluster-assessment/${assessment.metadata.name}`}>
                                {assessment.metadata.name}
                            </Link>
                        </Td>
                        <Td dataLabel="Profile">{getProfileLabel(assessment.spec?.profile)}</Td>
                        <Td dataLabel="Phase">{getPhaseLabel(assessment.status?.phase)}</Td>
                        <Td dataLabel="Score">{assessment.status?.summary?.score ?? '-'}</Td>
                        <Td dataLabel="Pass">
                            <span style={{ color: 'var(--pf-v5-global--success-color--100)' }}>
                                {assessment.status?.summary?.passCount ?? 0}
                            </span>
                        </Td>
                        <Td dataLabel="Warn">
                            <span style={{ color: 'var(--pf-v5-global--warning-color--100)' }}>
                                {assessment.status?.summary?.warnCount ?? 0}
                            </span>
                        </Td>
                        <Td dataLabel="Fail">
                            <span style={{ color: 'var(--pf-v5-global--danger-color--100)' }}>
                                {assessment.status?.summary?.failCount ?? 0}
                            </span>
                        </Td>
                        <Td dataLabel="Last Run">{formatDate(assessment.status?.lastRunTime)}</Td>
                        <Td>
                            <Button
                                variant="link"
                                component={(props: React.HTMLProps<HTMLAnchorElement>) => (
                                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                                    <Link {...props as any} to={`/cluster-assessment/${assessment.metadata.name}`} />
                                )}
                            >
                                View
                            </Button>
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </Table>
    );
}

export { AssessmentsTable };
