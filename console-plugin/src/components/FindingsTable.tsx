import * as React from 'react';
import {
    Table,
    Thead,
    Tr,
    Th,
    Tbody,
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
    ToolbarFilter,
    SearchInput,
    MenuToggle,
    MenuToggleElement,
    Select,
    SelectOption,
    SelectList,
} from '@patternfly/react-core';
import {
    CheckCircleIcon,
    ExclamationTriangleIcon,
    ExclamationCircleIcon,
    InfoCircleIcon,
    ExternalLinkAltIcon,
} from '@patternfly/react-icons';

export interface Finding {
    id: string;
    validator: string;
    category: string;
    resource?: string;
    namespace?: string;
    status: 'PASS' | 'WARN' | 'FAIL' | 'INFO';
    title: string;
    description: string;
    impact?: string;
    recommendation?: string;
    references?: string[];
}

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
            return <Label color="gold">{status}</Label>;
        case 'FAIL':
            return <Label color="red">{status}</Label>;
        case 'INFO':
        default:
            return <Label color="blue">{status}</Label>;
    }
};

export const FindingsTable: React.FC<FindingsTableProps> = ({ findings }) => {
    const [expandedRows, setExpandedRows] = React.useState<Set<string>>(new Set());
    const [searchValue, setSearchValue] = React.useState('');
    const [severityFilter, setSeverityFilter] = React.useState<string>('All');
    const [categoryFilter, setCategoryFilter] = React.useState<string>('All');
    const [isSeverityOpen, setIsSeverityOpen] = React.useState(false);
    const [isCategoryOpen, setIsCategoryOpen] = React.useState(false);

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

    const toggleRow = (id: string) => {
        const newExpanded = new Set(expandedRows);
        if (newExpanded.has(id)) {
            newExpanded.delete(id);
        } else {
            newExpanded.add(id);
        }
        setExpandedRows(newExpanded);
    };

    const columnNames = {
        status: 'Status',
        category: 'Category',
        title: 'Finding',
        resource: 'Resource',
    };

    return (
        <>
            <Toolbar id="findings-toolbar">
                <ToolbarContent>
                    <ToolbarItem variant="search-filter">
                        <SearchInput
                            placeholder="Search findings..."
                            value={searchValue}
                            onChange={(_, value) => setSearchValue(value)}
                            onClear={() => setSearchValue('')}
                        />
                    </ToolbarItem>
                    <ToolbarFilter
                        chips={severityFilter !== 'All' ? [severityFilter] : []}
                        deleteChip={() => setSeverityFilter('All')}
                        categoryName="Severity"
                    >
                        <Select
                            isOpen={isSeverityOpen}
                            onOpenChange={setIsSeverityOpen}
                            onSelect={(_, selection) => {
                                setSeverityFilter(selection as string);
                                setIsSeverityOpen(false);
                            }}
                            selected={severityFilter}
                            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                                <MenuToggle ref={toggleRef} onClick={() => setIsSeverityOpen(!isSeverityOpen)}>
                                    Severity: {severityFilter}
                                </MenuToggle>
                            )}
                        >
                            <SelectList>
                                {['All', 'PASS', 'WARN', 'FAIL', 'INFO'].map((s) => (
                                    <SelectOption key={s} value={s}>
                                        {s}
                                    </SelectOption>
                                ))}
                            </SelectList>
                        </Select>
                    </ToolbarFilter>
                    <ToolbarFilter
                        chips={categoryFilter !== 'All' ? [categoryFilter] : []}
                        deleteChip={() => setCategoryFilter('All')}
                        categoryName="Category"
                    >
                        <Select
                            isOpen={isCategoryOpen}
                            onOpenChange={setIsCategoryOpen}
                            onSelect={(_, selection) => {
                                setCategoryFilter(selection as string);
                                setIsCategoryOpen(false);
                            }}
                            selected={categoryFilter}
                            toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                                <MenuToggle ref={toggleRef} onClick={() => setIsCategoryOpen(!isCategoryOpen)}>
                                    Category: {categoryFilter}
                                </MenuToggle>
                            )}
                        >
                            <SelectList>
                                {categories.map((c) => (
                                    <SelectOption key={c} value={c}>
                                        {c}
                                    </SelectOption>
                                ))}
                            </SelectList>
                        </Select>
                    </ToolbarFilter>
                    <ToolbarItem>
                        <Text component={TextVariants.small}>
                            {filteredFindings.length} of {findings.length} findings
                        </Text>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>

            <Table aria-label="Findings table">
                <Thead>
                    <Tr>
                        <Th screenReaderText="Row expansion" />
                        <Th>{columnNames.status}</Th>
                        <Th>{columnNames.category}</Th>
                        <Th>{columnNames.title}</Th>
                        <Th>{columnNames.resource}</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {filteredFindings.map((finding, index) => (
                        <React.Fragment key={finding.id}>
                            <Tr>
                                <Td
                                    expand={{
                                        rowIndex: index,
                                        isExpanded: expandedRows.has(finding.id),
                                        onToggle: () => toggleRow(finding.id),
                                    }}
                                />
                                <Td dataLabel={columnNames.status}>
                                    {getStatusIcon(finding.status)} {getStatusLabel(finding.status)}
                                </Td>
                                <Td dataLabel={columnNames.category}>
                                    <Label>{finding.category}</Label>
                                </Td>
                                <Td dataLabel={columnNames.title}>{finding.title}</Td>
                                <Td dataLabel={columnNames.resource}>
                                    {finding.resource ? (
                                        <>
                                            {finding.namespace && `${finding.namespace}/`}
                                            {finding.resource}
                                        </>
                                    ) : (
                                        '-'
                                    )}
                                </Td>
                            </Tr>
                            <Tr isExpanded={expandedRows.has(finding.id)}>
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
                                                            iconPosition="right"
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
                        </React.Fragment>
                    ))}
                </Tbody>
            </Table>
        </>
    );
};

export default FindingsTable;
