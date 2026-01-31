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
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    SearchInput,
    FormSelect,
    FormSelectOption,
    Pagination,
    PaginationVariant,
    EmptyState,
    EmptyStateBody,
    EmptyStateIcon,
    EmptyStateHeader,
    EmptyStateFooter,
    Button,
} from '@patternfly/react-core';
import { SearchIcon, InfoCircleIcon } from '@patternfly/react-icons';
import { Finding } from '../types';
import { FindingsTableRow } from './FindingsTableRow';

interface FindingsTableProps {
    findings: Finding[];
}

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

    const handleToggle = React.useCallback((finding: Finding, isExpanding: boolean) => {
        const key = finding.id || (finding.title + finding.category);
        setExpandedRows((prev) => ({
            ...prev,
            [key]: isExpanding,
        }));
    }, []);

    const isRowExpanded = (finding: Finding) => {
        const key = finding.id || (finding.title + finding.category);
        return !!expandedRows[key];
    };

    const clearFilters = () => {
        setSearchValue('');
        setSeverityFilter('All');
        setCategoryFilter('All');
    };

    const severityOptions = ['All', 'PASS', 'WARN', 'FAIL', 'INFO'];

    // Early return if no findings at all (and thus no filters needed)
    if (findings.length === 0) {
        return (
            <EmptyState variant="full">
                <EmptyStateHeader titleText="No findings" icon={<EmptyStateIcon icon={InfoCircleIcon} />} headingLevel="h4" />
                <EmptyStateBody>
                    There are no findings to display.
                </EmptyStateBody>
            </EmptyState>
        );
    }

    return (
        <>
            <Toolbar id="findings-toolbar">
                <ToolbarContent>
                    <ToolbarItem>
                        <SearchInput
                            aria-label="Search findings"
                            placeholder="Search findings..."
                            value={searchValue}
                            onChange={(_event, value) => setSearchValue(value)}
                            onClear={() => setSearchValue('')}
                        />
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
                {paginatedFindings.length > 0 ? (
                    paginatedFindings.map((finding, rowIndex) => (
                        <FindingsTableRow
                            key={finding.id || rowIndex}
                            finding={finding}
                            rowIndex={rowIndex}
                            isExpanded={isRowExpanded(finding)}
                            onToggle={handleToggle}
                        />
                    ))
                ) : (
                    <Tbody>
                        <Tr>
                            <Td colSpan={5}>
                                <EmptyState variant="sm">
                                    <EmptyStateHeader titleText="No matching findings" icon={<EmptyStateIcon icon={SearchIcon} />} headingLevel="h4" />
                                    <EmptyStateBody>
                                        No findings match the current filters.
                                    </EmptyStateBody>
                                    <EmptyStateFooter>
                                        <Button variant="link" onClick={clearFilters}>Clear all filters</Button>
                                    </EmptyStateFooter>
                                </EmptyState>
                            </Td>
                        </Tr>
                    </Tbody>
                )}
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
