import * as React from 'react';
import {
    Table,
    TableHeader,
    TableBody,
    IRow,
    ICell,
    expandable,
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
    TextInput,
    Select,
    SelectOption,
    SelectVariant,
} from '@patternfly/react-core';
import {
    CheckCircleIcon,
    ExclamationTriangleIcon,
    ExclamationCircleIcon,
    InfoCircleIcon,
    ExternalLinkAltIcon,
    SearchIcon,
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
            return <CheckCircleIcon color="var(--pf-global--success-color--100)" />;
        case 'WARN':
            return <ExclamationTriangleIcon color="var(--pf-global--warning-color--100)" />;
        case 'FAIL':
            return <ExclamationCircleIcon color="var(--pf-global--danger-color--100)" />;
        case 'INFO':
        default:
            return <InfoCircleIcon color="var(--pf-global--info-color--100)" />;
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

export const FindingsTable: React.FC<FindingsTableProps> = ({ findings }) => {
    const [expandedRows, setExpandedRows] = React.useState<{ [key: number]: boolean }>({});
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

    const columns: ICell[] = [
        { title: 'Status', cellFormatters: [expandable] },
        { title: 'Category' },
        { title: 'Finding' },
        { title: 'Resource' },
    ];

    const rows: IRow[] = [];
    filteredFindings.forEach((finding, index) => {
        // Parent row
        rows.push({
            isOpen: expandedRows[index * 2] || false,
            cells: [
                {
                    title: (
                        <>
                            {getStatusIcon(finding.status)} {getStatusLabel(finding.status)}
                        </>
                    ),
                },
                { title: <Label>{finding.category}</Label> },
                { title: finding.title },
                {
                    title: finding.resource
                        ? `${finding.namespace ? `${finding.namespace}/` : ''}${finding.resource}`
                        : '-',
                },
            ],
        });
        // Child row (expandable content)
        rows.push({
            parent: index * 2,
            fullWidth: true,
            cells: [
                {
                    title: (
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
                    ),
                },
            ],
        });
    });

    const onCollapse = (_event: any, rowIndex: number, isOpen: boolean) => {
        setExpandedRows((prev) => ({
            ...prev,
            [rowIndex]: isOpen,
        }));
    };

    return (
        <>
            <Toolbar id="findings-toolbar">
                <ToolbarContent>
                    <ToolbarItem>
                        <InputGroup>
                            <TextInput
                                name="search"
                                id="search-input"
                                type="text"
                                aria-label="Search findings"
                                placeholder="Search findings..."
                                value={searchValue}
                                onChange={(val) => setSearchValue(val)}
                            />
                        </InputGroup>
                    </ToolbarItem>
                    <ToolbarItem>
                        <Select
                            variant={SelectVariant.single}
                            aria-label="Filter by severity"
                            onToggle={() => setIsSeverityOpen(!isSeverityOpen)}
                            onSelect={(_, selection) => {
                                setSeverityFilter(selection as string);
                                setIsSeverityOpen(false);
                            }}
                            selections={severityFilter}
                            isOpen={isSeverityOpen}
                            placeholderText="Severity"
                        >
                            {['All', 'PASS', 'WARN', 'FAIL', 'INFO'].map((s) => (
                                <SelectOption key={s} value={s}>
                                    {s}
                                </SelectOption>
                            ))}
                        </Select>
                    </ToolbarItem>
                    <ToolbarItem>
                        <Select
                            variant={SelectVariant.single}
                            aria-label="Filter by category"
                            onToggle={() => setIsCategoryOpen(!isCategoryOpen)}
                            onSelect={(_, selection) => {
                                setCategoryFilter(selection as string);
                                setIsCategoryOpen(false);
                            }}
                            selections={categoryFilter}
                            isOpen={isCategoryOpen}
                            placeholderText="Category"
                        >
                            {categories.map((c) => (
                                <SelectOption key={c} value={c}>
                                    {c}
                                </SelectOption>
                            ))}
                        </Select>
                    </ToolbarItem>
                    <ToolbarItem>
                        <Text component={TextVariants.small}>
                            {filteredFindings.length} of {findings.length} findings
                        </Text>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>

            <Table
                aria-label="Findings table"
                onCollapse={onCollapse}
                cells={columns}
                rows={rows}
            >
                <TableHeader />
                <TableBody />
            </Table>
        </>
    );
};

export default FindingsTable;
