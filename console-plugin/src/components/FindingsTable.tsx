import * as React from 'react';
import {
    Table,
    Thead,
    Tbody,
    Tr,
    Th,
    Td,
    ExpandableRowContent,
} from '@patternfly/react-table';
import {
    Label,
    TextContent,
    Text,
    TextVariants,
    Button,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    InputGroup,
    InputGroupItem,
    TextInput,
    FormSelect,
    FormSelectOption,
    Pagination,
    PaginationVariant,
} from '@patternfly/react-core';
import {
    CheckCircleIcon,
    ExclamationTriangleIcon,
    ExclamationCircleIcon,
    InfoCircleIcon,
    ExternalLinkAltIcon,
} from '@patternfly/react-icons';
import { Finding } from '../types';

interface FindingsTableProps {
    findings: Finding[];
}

const getStatusIcon = (status: string) => {
    switch (status) {
        case 'PASS':
            return <CheckCircleIcon color="var(--pf-v5-global--success-color--100)" />;
        case 'WARN':
            return <ExclamationTriangleIcon color="var(--pf-v5-global--warning-color--100)" />;
        case 'FAIL':
            return <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />;
        case 'INFO':
        default:
            return <InfoCircleIcon color="var(--pf-v5-global--info-color--100)" />;
    }
};

const getStatusLabel = (status: string) => {
    switch (status) {
        case 'PASS':
            return <Label color="green">{status}</Label>;
        case 'WARN':
            return <Label color="orange">{status}</Label>;
        case 'FAIL':
            return <Label color="red">{status}</Label>;
        case 'INFO':
        default:
            return <Label color="blue">{status}</Label>;
    }
};

export default function FindingsTable({ findings }: FindingsTableProps) {
    const [expandedRows, setExpandedRows] = React.useState<{ [key: string]: boolean }>({});
    const [searchValue, setSearchValue] = React.useState('');
    const [severityFilter, setSeverityFilter] = React.useState<string>('All');
    const [categoryFilter, setCategoryFilter] = React.useState<string>('All');

    // Pagination state
    const [page, setPage] = React.useState(1);
    const [perPage, setPerPage] = React.useState(10);

    const categories = React.useMemo(() => {
        const cats = new Set(findings.map((f) => f.category));
        return ['All', ...Array.from(cats)];
    }, [findings]);

    const filteredFindings = React.useMemo(() => {
        return findings.filter((finding) => {
            const matchesSearch =
                searchValue === '' ||
                finding.title.toLowerCase().includes(searchValue.toLowerCase()) ||
                finding.description.toLowerCase().includes(searchValue.toLowerCase());
            const matchesSeverity = severityFilter === 'All' || finding.status === severityFilter;
            const matchesCategory = categoryFilter === 'All' || finding.category === categoryFilter;
            return matchesSearch && matchesSeverity && matchesCategory;
        });
    }, [findings, searchValue, severityFilter, categoryFilter]);

    // Reset page when filters change
    React.useEffect(() => {
        setPage(1);
    }, [searchValue, severityFilter, categoryFilter]);

    // Calculate paginated findings
    const paginatedFindings = React.useMemo(() => {
        const start = (page - 1) * perPage;
        const end = start + perPage;
        return filteredFindings.slice(start, end);
    }, [filteredFindings, page, perPage]);

    const onSetPage = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPage: number) => {
        setPage(newPage);
    };

    const onPerPageSelect = (_event: React.MouseEvent | React.KeyboardEvent | MouseEvent, newPerPage: number) => {
        setPerPage(newPerPage);
        setPage(1);
    };

    const setRowExpanded = (finding: Finding, isExpanding: boolean) => {
        const key = finding.title + finding.category;
        setExpandedRows((prev) => ({
            ...prev,
            [key]: isExpanding,
        }));
    };

    const isRowExpanded = (finding: Finding) => {
        const key = finding.title + finding.category;
        return !!expandedRows[key];
    };

    const severityOptions = ['All', 'PASS', 'WARN', 'FAIL', 'INFO'];

    return (
        <>
            <Toolbar id="findings-toolbar">
                <ToolbarContent>
                    <ToolbarItem>
                        <InputGroup>
                            <InputGroupItem isFill>
                                <TextInput
                                    name="search"
                                    id="search-input"
                                    type="text"
                                    aria-label="Search findings"
                                    placeholder="Search findings..."
                                    value={searchValue}
                                    onChange={(_event, value) => setSearchValue(value)}
                                />
                            </InputGroupItem>
                        </InputGroup>
                    </ToolbarItem>
                    <ToolbarItem>
                        <FormSelect
                            aria-label="Filter by severity"
                            value={severityFilter}
                            onChange={(_event, value) => setSeverityFilter(value)}
                        >
                            {severityOptions.map((s) => (
                                <FormSelectOption key={s} value={s} label={s} />
                            ))}
                        </FormSelect>
                    </ToolbarItem>
                    <ToolbarItem>
                        <FormSelect
                            aria-label="Filter by category"
                            value={categoryFilter}
                            onChange={(_event, value) => setCategoryFilter(value)}
                        >
                            {categories.map((c) => (
                                <FormSelectOption key={c} value={c} label={c} />
                            ))}
                        </FormSelect>
                    </ToolbarItem>
                    <ToolbarItem variant="pagination">
                        <Pagination
                            itemCount={filteredFindings.length}
                            perPage={perPage}
                            page={page}
                            onSetPage={onSetPage}
                            onPerPageSelect={onPerPageSelect}
                            isCompact
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>

            <Table aria-label="Findings table" variant="compact">
                <Thead>
                    <Tr>
                        <Th screenReaderText="Row expansion" />
                        <Th>Status</Th>
                        <Th>Category</Th>
                        <Th>Finding</Th>
                        <Th>Resource</Th>
                    </Tr>
                </Thead>
                {paginatedFindings.map((finding, rowIndex) => (
                    <Tbody key={rowIndex} isExpanded={isRowExpanded(finding)}>
                        <Tr>
                            <Td
                                expand={{
                                    rowIndex,
                                    isExpanded: isRowExpanded(finding),
                                    onToggle: () => setRowExpanded(finding, !isRowExpanded(finding)),
                                }}
                            />
                            <Td dataLabel="Status">
                                {getStatusIcon(finding.status)} {getStatusLabel(finding.status)}
                            </Td>
                            <Td dataLabel="Category"><Label>{finding.category}</Label></Td>
                            <Td dataLabel="Finding">{finding.title}</Td>
                            <Td dataLabel="Resource">
                                {finding.resource
                                    ? `${finding.namespace ? `${finding.namespace}/` : ''}${finding.resource}`
                                    : '-'}
                            </Td>
                        </Tr>
                        <Tr isExpanded={isRowExpanded(finding)}>
                            <Td colSpan={5}>
                                <ExpandableRowContent>
                                    <TextContent>
                                        <Text component={TextVariants.h4}>Description</Text>
                                        <Text>{finding.description}</Text>
                                        {finding.impact && (
                                            <>
                                                <Text component={TextVariants.h4}>Impact</Text>
                                                <Text>{finding.impact}</Text>
                                            </>
                                        )}
                                        {finding.recommendation && (
                                            <>
                                                <Text component={TextVariants.h4}>Recommendation</Text>
                                                <Text>{finding.recommendation}</Text>
                                            </>
                                        )}
                                        {finding.references && finding.references.length > 0 && (
                                            <>
                                                <Text component={TextVariants.h4}>References</Text>
                                                {finding.references.map((ref, i) => (
                                                    <Button
                                                        key={i}
                                                        variant="link"
                                                        isInline
                                                        component="a"
                                                        href={ref}
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        icon={<ExternalLinkAltIcon />}
                                                        iconPosition="end"
                                                    >
                                                        {ref}
                                                    </Button>
                                                ))}
                                            </>
                                        )}
                                    </TextContent>
                                </ExpandableRowContent>
                            </Td>
                        </Tr>
                    </Tbody>
                ))}
            </Table>
            <Pagination
                itemCount={filteredFindings.length}
                perPage={perPage}
                page={page}
                onSetPage={onSetPage}
                onPerPageSelect={onPerPageSelect}
                variant={PaginationVariant.bottom}
            />
        </>
    );
}

export { FindingsTable };
