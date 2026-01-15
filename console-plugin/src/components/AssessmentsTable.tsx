import * as React from 'react';
import {
    Table,
    Thead,
    Tr,
    Th,
    Tbody,
    Td,
} from '@patternfly/react-table';
import {
    Label,
    Button,
    Timestamp,
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

export const AssessmentsTable: React.FC<AssessmentsTableProps> = ({ assessments }) => {
    const columnNames = {
        name: 'Name',
        profile: 'Profile',
        phase: 'Phase',
        score: 'Score',
        pass: 'Pass',
        warn: 'Warn',
        fail: 'Fail',
        lastRun: 'Last Run',
        actions: '',
    };

    const sortedAssessments = React.useMemo(() => {
        return [...assessments].sort((a, b) => {
            const timeA = a.status?.lastRunTime || a.metadata.creationTimestamp;
            const timeB = b.status?.lastRunTime || b.metadata.creationTimestamp;
            return new Date(timeB).getTime() - new Date(timeA).getTime();
        });
    }, [assessments]);

    return (
        <Table aria-label="Cluster assessments table">
            <Thead>
                <Tr>
                    <Th>{columnNames.name}</Th>
                    <Th>{columnNames.profile}</Th>
                    <Th>{columnNames.phase}</Th>
                    <Th>{columnNames.score}</Th>
                    <Th>{columnNames.pass}</Th>
                    <Th>{columnNames.warn}</Th>
                    <Th>{columnNames.fail}</Th>
                    <Th>{columnNames.lastRun}</Th>
                    <Th>{columnNames.actions}</Th>
                </Tr>
            </Thead>
            <Tbody>
                {sortedAssessments.map((assessment) => (
                    <Tr key={assessment.metadata.name}>
                        <Td dataLabel={columnNames.name}>
                            <Link to={`/cluster-assessment/${assessment.metadata.name}`}>
                                {assessment.metadata.name}
                            </Link>
                        </Td>
                        <Td dataLabel={columnNames.profile}>
                            {getProfileLabel(assessment.spec?.profile)}
                        </Td>
                        <Td dataLabel={columnNames.phase}>
                            {getPhaseLabel(assessment.status?.phase)}
                        </Td>
                        <Td dataLabel={columnNames.score}>
                            {assessment.status?.summary?.score ?? '-'}
                        </Td>
                        <Td dataLabel={columnNames.pass}>
                            <span style={{ color: 'var(--pf-v5-global--success-color--100)' }}>
                                {assessment.status?.summary?.passCount ?? 0}
                            </span>
                        </Td>
                        <Td dataLabel={columnNames.warn}>
                            <span style={{ color: 'var(--pf-v5-global--warning-color--100)' }}>
                                {assessment.status?.summary?.warnCount ?? 0}
                            </span>
                        </Td>
                        <Td dataLabel={columnNames.fail}>
                            <span style={{ color: 'var(--pf-v5-global--danger-color--100)' }}>
                                {assessment.status?.summary?.failCount ?? 0}
                            </span>
                        </Td>
                        <Td dataLabel={columnNames.lastRun}>
                            {assessment.status?.lastRunTime ? (
                                <Timestamp date={new Date(assessment.status.lastRunTime)} />
                            ) : (
                                '-'
                            )}
                        </Td>
                        <Td dataLabel={columnNames.actions}>
                            <Button
                                variant="link"
                                component={(props) => (
                                    <Link {...props} to={`/cluster-assessment/${assessment.metadata.name}`} />
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
};

export default AssessmentsTable;
