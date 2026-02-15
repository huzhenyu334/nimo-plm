import React, { useState, useMemo } from 'react';
import {
  Card,
  Select,
  Tag,
  Badge,
  Progress,
  Drawer,
  Descriptions,
  Timeline,
  Spin,
  Empty,
  Button,
  Popconfirm,
  Modal,
  Form,
  InputNumber,
  DatePicker,
  Divider,
  App,
  Table,
  Checkbox,
} from 'antd';
import { ReloadOutlined, AppstoreOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { srmApi, SRMProject, PRItem, PurchaseRequest, ActivityLog, Supplier, SamplingRequest } from '@/api/srm';
import { projectApi } from '@/api/projects';
import { Input } from 'antd';
import dayjs from 'dayjs';

// Kanban column definitions
const KANBAN_COLUMNS = [
  { key: 'pending', label: '寻源中', color: '#d9d9d9' },
  { key: 'quoting', label: '报价中', color: '#faad14' },
  { key: 'sourcing', label: '待下单', color: '#13c2c2' },
  { key: 'ordered', label: '已下单', color: '#1890ff' },
  { key: 'shipped', label: '已发货', color: '#722ed1' },
  { key: 'received', label: '已收货', color: '#2f54eb' },
  { key: 'inspecting', label: '检验中', color: '#fa8c16' },
  { key: 'passed', label: '已通过', color: '#52c41a' },
] as const;

type ColumnKey = typeof KANBAN_COLUMNS[number]['key'];

// Extended item with PR context
interface KanbanItem extends PRItem {
  pr_code: string;
  pr_title: string;
  project_target_date?: string;
}

// Passive component categories
const PASSIVE_CATEGORIES = new Set(['电容', '电阻', '电感', '晶振', '晶体管', '测试点', 'electronic']);

const isPassiveComponent = (category: string) => PASSIVE_CATEGORIES.has(category);

// Category sort order: IC → 被动元件(aggregated) → 结构件 → 治具 → 辅料(组装)
const CATEGORY_ORDER: Record<string, number> = {
  'IC': 0,
  '_passive_': 1,
  '结构件': 2,
  'structural': 2,
  '治具': 3,
  '组装': 4,
};

const getCategorySortKey = (category: string): number => {
  if (isPassiveComponent(category)) return CATEGORY_ORDER['_passive_'];
  return CATEGORY_ORDER[category] ?? 2.5;
};

// Category filter options for demand classification
const CATEGORY_FILTERS = [
  { value: 'all', label: '全部需求' },
  { value: 'EBOM', label: '电子EBOM' },
  { value: 'PBOM', label: '工艺PBOM' },
  { value: 'PBOM', label: '包装PBOM' },
  { value: 'TOOLING', label: '工装治具' },
  { value: 'AUXILIARY', label: '生产辅料' },
  { value: 'LICENSE', label: '软件License' },
];
const ALL_CATEGORY_VALUES = CATEGORY_FILTERS.filter(c => c.value !== 'all').map(c => c.value);

// Aggregated card representation
interface PassiveGroup {
  type: 'passive_group';
  items: KanbanItem[];
  status: string;
}

type ColumnEntry = { type: 'single'; item: KanbanItem } | PassiveGroup;

const itemStatusLabels: Record<string, string> = {
  pending: '寻源中', quoting: '报价中', sourcing: '待下单',
  ordered: '已下单', shipped: '已发货', received: '已收货', inspecting: '检验中',
  passed: '已通过', failed: '未通过',
};

// Action definitions per status
const STATUS_ACTIONS: Record<string, Array<{ label: string; toStatus: string; danger?: boolean; primary?: boolean; special?: string }>> = {
  pending: [
    { label: '发起询价', toStatus: 'quoting', primary: true },
  ],
  quoting: [
    { label: '确认报价', toStatus: 'sourcing', primary: true },
  ],
  sourcing: [
    { label: '确认下单', toStatus: 'ordered', primary: true },
  ],
  ordered: [
    { label: '标记发货', toStatus: 'shipped', primary: true },
  ],
  shipped: [
    { label: '确认收货', toStatus: 'received', primary: true },
  ],
  received: [
    { label: '发起检验', toStatus: 'inspecting', primary: true },
  ],
  inspecting: [
    { label: '标记通过', toStatus: 'passed', primary: true },
    { label: '标记不通过', toStatus: 'failed', danger: true },
  ],
};

const KanbanBoard: React.FC = () => {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const projectId = searchParams.get('project') || '';
  const [drawerItem, setDrawerItem] = useState<KanbanItem | null>(null);
  const [assignModalItem, setAssignModalItem] = useState<KanbanItem | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [assignForm] = Form.useForm();

  // Passive group drawer state
  const [passiveDrawer, setPassiveDrawer] = useState<PassiveGroup | null>(null);
  const [selectedPassiveIds, setSelectedPassiveIds] = useState<string[]>([]);
  const [batchAssignModal, setBatchAssignModal] = useState(false);
  const [batchStatusModal, setBatchStatusModal] = useState(false);
  const [batchAssignForm] = Form.useForm();
  const [batchTargetStatus, setBatchTargetStatus] = useState('');

  // Sampling state
  const [samplingModalItem, setSamplingModalItem] = useState<KanbanItem | null>(null);
  const [samplingForm] = Form.useForm();

  // Category filter state
  const [selectedCategories, setSelectedCategories] = useState<string[]>(['all', ...ALL_CATEGORY_VALUES]);

  // Load SRM projects for selector
  const { data: projectsData, isLoading: projectsLoading } = useQuery({
    queryKey: ['srm-projects-list'],
    queryFn: () => srmApi.listProjects({ page_size: 100 }),
  });

  // Load PLM projects for selector (covers projects without SRM project)
  const { data: plmProjectsData } = useQuery({
    queryKey: ['plm-projects-kanban'],
    queryFn: () => projectApi.list({ page_size: 100 }),
  });

  // Load selected project details (may fail if projectId is a PLM project ID)
  const { data: project } = useQuery({
    queryKey: ['srm-project', projectId],
    queryFn: () => srmApi.getProject(projectId).catch(() => null),
    enabled: !!projectId,
  });

  // Load PRs for the selected project
  const { data: prData, isLoading: prLoading } = useQuery({
    queryKey: ['srm-prs-kanban', projectId],
    queryFn: () => srmApi.listPRs({ project_id: projectId, page_size: 200 }),
    enabled: !!projectId,
  });

  // Load supplier map
  const { data: supplierData } = useQuery({
    queryKey: ['srm-suppliers-select'],
    queryFn: () => srmApi.listSuppliers({ page_size: 200 }),
  });

  const supplierMap = useMemo(() => {
    const map: Record<string, string> = {};
    (supplierData?.items || []).forEach((s) => { map[s.id] = s.name; });
    return map;
  }, [supplierData]);

  const supplierList: Supplier[] = useMemo(() => supplierData?.items || [], [supplierData]);

  // Flatten all PR items into kanban items, applying category filter
  const allItems: KanbanItem[] = useMemo(() => {
    const prs = prData?.items || [];
    const items: KanbanItem[] = [];
    prs.forEach((pr: PurchaseRequest) => {
      (pr.items || []).forEach((item) => {
        items.push({
          ...item,
          pr_code: pr.pr_code,
          pr_title: pr.title,
          project_target_date: project?.target_date,
        });
      });
    });

    // Apply category filter
    const activeCats = selectedCategories.filter(v => v !== 'all');
    if (activeCats.length === ALL_CATEGORY_VALUES.length) return items; // all selected, no filtering

    return items.filter((item) => {
      const bomType = (item.source_bom_type || '').toUpperCase();
      const matGroup = (item.material_group || '').toLowerCase();

      if (bomType === 'EBOM' && activeCats.includes('EBOM')) return true;
      if (bomType === 'PBOM' && activeCats.includes('PBOM')) return true;
      if (bomType === 'PBOM' && activeCats.includes('PBOM')) return true;
      if (matGroup === 'tooling' && activeCats.includes('TOOLING')) return true;
      if (matGroup === 'auxiliary' && activeCats.includes('AUXILIARY')) return true;
      if (matGroup === 'license' && activeCats.includes('LICENSE')) return true;

      // Items with no bom_type and no material_group: show if 'EBOM' is selected (default bucket)
      if (!bomType && !matGroup && activeCats.includes('EBOM')) return true;

      return false;
    });
  }, [prData, project, selectedCategories]);

  // Group items by status into columns
  const columnData = useMemo(() => {
    const groups: Record<string, KanbanItem[]> = {};
    KANBAN_COLUMNS.forEach((col) => { groups[col.key] = []; });
    allItems.forEach((item) => {
      const status = item.status as ColumnKey;
      if (groups[status]) {
        groups[status].push(item);
      } else if (item.status === 'failed') {
        groups['inspecting'].push(item);
      } else if (item.status === 'sampling') {
        groups['quoting'].push(item);
      } else {
        groups['pending'].push(item);
      }
    });
    return groups;
  }, [allItems]);

  // Build sorted column entries with passive aggregation
  const columnEntries = useMemo(() => {
    const result: Record<string, ColumnEntry[]> = {};
    KANBAN_COLUMNS.forEach((col) => {
      const items = columnData[col.key] || [];
      const passiveItems: KanbanItem[] = [];
      const nonPassiveItems: KanbanItem[] = [];

      items.forEach((item) => {
        if (isPassiveComponent(item.category)) {
          passiveItems.push(item);
        } else {
          nonPassiveItems.push(item);
        }
      });

      const entries: ColumnEntry[] = nonPassiveItems.map((item) => ({
        type: 'single' as const,
        item,
      }));

      if (passiveItems.length > 0) {
        entries.push({
          type: 'passive_group' as const,
          items: passiveItems,
          status: col.key,
        });
      }

      // Sort by category order
      entries.sort((a, b) => {
        const catA = a.type === 'single' ? getCategorySortKey(a.item.category) : CATEGORY_ORDER['_passive_'];
        const catB = b.type === 'single' ? getCategorySortKey(b.item.category) : CATEGORY_ORDER['_passive_'];
        return catA - catB;
      });

      result[col.key] = entries;
    });
    return result;
  }, [columnData]);

  // Summary stats
  const stats = useMemo(() => {
    const total = allItems.length;
    const pending = columnData['pending']?.length || 0;
    const ordered = columnData['ordered']?.length || 0;
    const received = columnData['received']?.length || 0;
    const passed = columnData['passed']?.length || 0;
    const pct = total > 0 ? Math.round((passed / total) * 100) : 0;
    return { total, pending, ordered, received, passed, pct };
  }, [allItems, columnData]);

  // Activity logs for drawer
  const { data: activityData } = useQuery({
    queryKey: ['srm-activities', 'pr_item', drawerItem?.id],
    queryFn: () => srmApi.listActivities('pr_item', drawerItem!.id, { page_size: 20 }),
    enabled: !!drawerItem?.id,
  });

  // Sampling records for drawer
  const { data: samplingData } = useQuery({
    queryKey: ['srm-sampling', drawerItem?.id],
    queryFn: () => srmApi.listSampling(drawerItem!.id),
    enabled: !!drawerItem?.id,
  });

  const handleProjectChange = (value: string) => {
    setSearchParams({ project: value });
  };

  const handleCategoryChange = (values: string[]) => {
    const prevHadAll = selectedCategories.includes('all');
    const nowHasAll = values.includes('all');
    const subValues = values.filter(v => v !== 'all');

    if (nowHasAll && !prevHadAll) {
      // User just checked "all" → select everything
      setSelectedCategories(['all', ...ALL_CATEGORY_VALUES]);
    } else if (!nowHasAll && prevHadAll) {
      // User just unchecked "all" → clear everything
      setSelectedCategories([]);
    } else {
      // Individual toggle
      if (subValues.length === ALL_CATEGORY_VALUES.length) {
        setSelectedCategories(['all', ...subValues]);
      } else {
        setSelectedCategories(subValues);
      }
    }
  };

  const refreshKanban = () => {
    queryClient.invalidateQueries({ queryKey: ['srm-prs-kanban'] });
    queryClient.invalidateQueries({ queryKey: ['srm-project'] });
    if (drawerItem) {
      queryClient.invalidateQueries({ queryKey: ['srm-activities', 'pr_item', drawerItem.id] });
      queryClient.invalidateQueries({ queryKey: ['srm-sampling', drawerItem.id] });
    }
  };

  const handleStatusChange = async (item: KanbanItem, toStatus: string) => {
    setActionLoading(true);
    try {
      await srmApi.updatePRItemStatus(item.id, toStatus);
      message.success('操作成功');
      setDrawerItem(null);
      refreshKanban();
    } catch {
      message.error('操作失败');
    } finally {
      setActionLoading(false);
    }
  };

  const handleAssignSupplier = async () => {
    if (!assignModalItem) return;
    try {
      const values = await assignForm.validateFields();
      setActionLoading(true);
      await srmApi.assignSupplier(assignModalItem.pr_id, assignModalItem.id, {
        supplier_id: values.supplier_id,
        unit_price: values.unit_price,
        expected_date: values.expected_date ? values.expected_date.toISOString() : undefined,
      });
      message.success('供应商分配成功');
      setAssignModalItem(null);
      assignForm.resetFields();
      setDrawerItem(null);
      refreshKanban();
    } catch {
      message.error('分配失败');
    } finally {
      setActionLoading(false);
    }
  };

  // Batch assign supplier for passive group
  const handleBatchAssignSupplier = async () => {
    if (selectedPassiveIds.length === 0) return;
    try {
      const values = await batchAssignForm.validateFields();
      setActionLoading(true);
      const items = passiveDrawer?.items.filter((i) => selectedPassiveIds.includes(i.id)) || [];
      await Promise.all(
        items.map((item) =>
          srmApi.assignSupplier(item.pr_id, item.id, {
            supplier_id: values.supplier_id,
            unit_price: values.unit_price,
            expected_date: values.expected_date ? values.expected_date.toISOString() : undefined,
          })
        )
      );
      message.success(`已批量分配 ${items.length} 颗物料`);
      setBatchAssignModal(false);
      batchAssignForm.resetFields();
      setSelectedPassiveIds([]);
      setPassiveDrawer(null);
      refreshKanban();
    } catch {
      message.error('批量分配失败');
    } finally {
      setActionLoading(false);
    }
  };

  // Batch status change for passive group
  const handleBatchStatusChange = async () => {
    if (selectedPassiveIds.length === 0 || !batchTargetStatus) return;
    setActionLoading(true);
    try {
      const items = passiveDrawer?.items.filter((i) => selectedPassiveIds.includes(i.id)) || [];
      await Promise.all(
        items.map((item) => srmApi.updatePRItemStatus(item.id, batchTargetStatus))
      );
      message.success(`已批量变更 ${items.length} 颗物料状态`);
      setBatchStatusModal(false);
      setBatchTargetStatus('');
      setSelectedPassiveIds([]);
      setPassiveDrawer(null);
      refreshKanban();
    } catch {
      message.error('批量变更失败');
    } finally {
      setActionLoading(false);
    }
  };

  // Create sampling request
  const handleCreateSampling = async () => {
    if (!samplingModalItem) return;
    try {
      const values = await samplingForm.validateFields();
      setActionLoading(true);
      await srmApi.createSampling(samplingModalItem.id, {
        supplier_id: values.supplier_id,
        sample_qty: values.sample_qty,
        notes: values.notes,
      });
      message.success('打样请求已创建');
      setSamplingModalItem(null);
      samplingForm.resetFields();
      setDrawerItem(null);
      refreshKanban();
    } catch {
      message.error('创建打样失败');
    } finally {
      setActionLoading(false);
    }
  };

  // Update sampling status
  const handleUpdateSamplingStatus = async (samplingId: string, status: string) => {
    setActionLoading(true);
    try {
      await srmApi.updateSamplingStatus(samplingId, status);
      message.success('打样状态已更新');
      refreshKanban();
      queryClient.invalidateQueries({ queryKey: ['srm-sampling'] });
    } catch {
      message.error('更新打样状态失败');
    } finally {
      setActionLoading(false);
    }
  };

  // Merge SRM projects + PLM projects for selector (deduplicate by PLM project ID)
  const projectOptions = useMemo(() => {
    const srmProjects = projectsData?.items || [];
    const plmProjects = plmProjectsData?.items || [];
    const options: { value: string; label: string }[] = [];
    const seen = new Set<string>();

    // SRM projects first (use SRM project ID)
    srmProjects.forEach((p: SRMProject) => {
      options.push({ value: p.id, label: `${p.code} - ${p.name}` });
      seen.add(p.id);
      if (p.plm_project_id) seen.add(p.plm_project_id);
    });

    // PLM projects (use PLM project ID, skip if already covered by SRM project)
    plmProjects.forEach((p: any) => {
      if (!seen.has(p.id)) {
        options.push({ value: p.id, label: `${p.code} - ${p.name}` });
      }
    });

    return options;
  }, [projectsData, plmProjectsData]);

  const isLoading = projectsLoading || prLoading;

  // Render action buttons for a given item (used in both card and drawer)
  const renderActions = (item: KanbanItem, size: 'small' | 'middle' = 'small') => {
    const actions = STATUS_ACTIONS[item.status] || [];
    const showAssign = item.status === 'pending' && !item.supplier_id;
    if (actions.length === 0 && !showAssign) return null;

    return (
      <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
        {showAssign && (
          <Button
            size={size}
            type="link"
            style={{ padding: '0 4px', fontSize: size === 'small' ? 12 : 13, height: 'auto' }}
            onClick={(e) => { e.stopPropagation(); setAssignModalItem(item); }}
          >
            分配供应商
          </Button>
        )}
        {actions.map((action) =>
          action.special === 'sampling' ? (
            <Button
              key={action.toStatus}
              size={size}
              type="link"
              style={{ padding: '0 4px', fontSize: size === 'small' ? 12 : 13, height: 'auto', color: '#eb2f96' }}
              onClick={(e) => { e.stopPropagation(); setSamplingModalItem(item); }}
            >
              {action.label}
            </Button>
          ) : action.danger ? (
            <Popconfirm
              key={action.toStatus}
              title="确认操作"
              description={`确定要${action.label}吗？`}
              onConfirm={(e) => { e?.stopPropagation(); handleStatusChange(item, action.toStatus); }}
              onCancel={(e) => e?.stopPropagation()}
              okText="确定"
              cancelText="取消"
              okButtonProps={{ danger: true }}
            >
              <Button
                size={size}
                type="link"
                danger
                style={{ padding: '0 4px', fontSize: size === 'small' ? 12 : 13, height: 'auto' }}
                loading={actionLoading}
                onClick={(e) => e.stopPropagation()}
              >
                {action.label}
              </Button>
            </Popconfirm>
          ) : (
            <Button
              key={action.toStatus}
              size={size}
              type="link"
              style={{ padding: '0 4px', fontSize: size === 'small' ? 12 : 13, height: 'auto', color: action.primary ? '#1890ff' : undefined }}
              loading={actionLoading}
              onClick={(e) => { e.stopPropagation(); handleStatusChange(item, action.toStatus); }}
            >
              {action.label}
            </Button>
          )
        )}
      </div>
    );
  };

  // Get available batch status actions for a passive group
  const getBatchStatusActions = (group: PassiveGroup) => {
    return STATUS_ACTIONS[group.status] || [];
  };

  // Column card count (accounting for aggregation: passive group = 1 visual card, but badge shows real count)
  const getColumnItemCount = (colKey: string) => columnData[colKey]?.length || 0;

  // Passive group table columns for drawer
  const passiveTableColumns = [
    { title: '物料名称', dataIndex: 'material_name', key: 'material_name', ellipsis: true },
    { title: '规格', dataIndex: 'specification', key: 'specification', ellipsis: true, render: (v: string) => v || '-' },
    { title: '分类', dataIndex: 'category', key: 'category', width: 70 },
    { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 60, render: (v: number, r: KanbanItem) => `${v} ${r.unit || ''}` },
    {
      title: '供应商', key: 'supplier', width: 100,
      render: (_: unknown, r: KanbanItem) => r.supplier_id ? (supplierMap[r.supplier_id] || '-') : <Tag color="orange">未分配</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 80,
      render: (v: string) => <Tag>{itemStatusLabels[v] || v}</Tag>,
    },
  ];

  return (
    <div>
      {/* Top bar: project selector + stats */}
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontWeight: 600, fontSize: 16 }}>采购看板</span>
          <Select
            placeholder="选择采购项目"
            style={{ width: 260 }}
            value={projectId || undefined}
            onChange={handleProjectChange}
            loading={projectsLoading}
            showSearch
            optionFilterProp="label"
            options={projectOptions}
          />
          <ReloadOutlined
            style={{ cursor: 'pointer', color: '#1890ff' }}
            onClick={() => refreshKanban()}
          />
          <Select
            mode="multiple"
            placeholder="需求分类"
            style={{ width: 320 }}
            value={selectedCategories}
            onChange={handleCategoryChange}
            maxTagCount={2}
            maxTagPlaceholder={(omitted) => `+${omitted.length}`}
            options={CATEGORY_FILTERS}
          />
        </div>

        {projectId && allItems.length > 0 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 16, flex: 1 }}>
            <span style={{ color: '#666', fontSize: 13 }}>
              总计: <strong>{stats.total}</strong> |
              寻源中: <strong>{stats.pending}</strong> |
              已下单: <strong>{stats.ordered}</strong> |
              已收货: <strong>{stats.received}</strong> |
              已通过: <strong>{stats.passed}/{stats.total}</strong> ({stats.pct}%)
            </span>
            <Progress
              percent={stats.pct}
              size="small"
              style={{ width: 160, margin: 0 }}
              strokeColor="#52c41a"
            />
          </div>
        )}
      </div>

      {/* Kanban columns */}
      {!projectId ? (
        <Card>
          <Empty description="请选择一个采购项目以查看看板" />
        </Card>
      ) : isLoading ? (
        <div style={{ textAlign: 'center', padding: 80 }}>
          <Spin size="large" />
        </div>
      ) : allItems.length === 0 ? (
        <Card>
          <Empty description="该项目暂无采购物料" />
        </Card>
      ) : (
        <div style={{
          display: 'flex',
          gap: 12,
          overflowX: 'auto',
          paddingBottom: 16,
          height: 'calc(100vh - 200px)',
        }}>
          {KANBAN_COLUMNS.map((col) => {
            const entries = columnEntries[col.key] || [];
            const itemCount = getColumnItemCount(col.key);
            return (
              <div
                key={col.key}
                style={{
                  minWidth: 220,
                  maxWidth: 280,
                  flex: '1 0 220px',
                  background: '#fafafa',
                  borderRadius: 8,
                  padding: 12,
                  display: 'flex',
                  flexDirection: 'column',
                  minHeight: 0,
                }}
              >
                {/* Column header */}
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: 12,
                  paddingBottom: 8,
                  borderBottom: `3px solid ${col.color}`,
                }}>
                  <span style={{ fontWeight: 600, fontSize: 14 }}>{col.label}</span>
                  <Badge
                    count={itemCount}
                    style={{ backgroundColor: col.color }}
                    overflowCount={999}
                  />
                </div>

                {/* Cards */}
                <div style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 8, minHeight: 0 }}>
                  {entries.map((entry) => {
                    if (entry.type === 'passive_group') {
                      return (
                        <PassiveGroupCard
                          key={`passive-${col.key}`}
                          group={entry}
                          supplierMap={supplierMap}
                          onClick={() => {
                            setPassiveDrawer(entry);
                            setSelectedPassiveIds([]);
                          }}
                        />
                      );
                    }
                    return (
                      <KanbanCard
                        key={entry.item.id}
                        item={entry.item}
                        supplierMap={supplierMap}
                        onClick={() => setDrawerItem(entry.item)}
                        actions={renderActions(entry.item)}
                      />
                    );
                  })}
                  {entries.length === 0 && (
                    <div style={{ color: '#bbb', textAlign: 'center', padding: 24, fontSize: 13 }}>
                      暂无物料
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Item detail drawer */}
      <Drawer
        title={drawerItem?.material_name || '物料详情'}
        open={!!drawerItem}
        onClose={() => setDrawerItem(null)}
        width={520}
      >
        {drawerItem && (
          <>
            <Descriptions column={1} bordered size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label="物料名称">{drawerItem.material_name}</Descriptions.Item>
              <Descriptions.Item label="物料编码">
                <span style={{ fontFamily: 'monospace' }}>{drawerItem.material_code || '-'}</span>
              </Descriptions.Item>
              <Descriptions.Item label="规格">{drawerItem.specification || '-'}</Descriptions.Item>
              <Descriptions.Item label="分类">{drawerItem.category || '-'}</Descriptions.Item>
              <Descriptions.Item label="数量">{drawerItem.quantity} {drawerItem.unit}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag>{itemStatusLabels[drawerItem.status] || drawerItem.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="供应商">
                {drawerItem.supplier_id ? (supplierMap[drawerItem.supplier_id] || '-') : '未分配'}
              </Descriptions.Item>
              <Descriptions.Item label="单价">
                {drawerItem.unit_price != null ? `¥${drawerItem.unit_price.toFixed(2)}` : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="预计交期">
                {drawerItem.expected_date ? dayjs(drawerItem.expected_date).format('YYYY-MM-DD') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="实际到货">
                {drawerItem.actual_date ? dayjs(drawerItem.actual_date).format('YYYY-MM-DD') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="来源PR">{drawerItem.pr_code}</Descriptions.Item>
              <Descriptions.Item label="备注">{drawerItem.notes || '-'}</Descriptions.Item>
            </Descriptions>

            {/* Action area in drawer */}
            {renderActions(drawerItem, 'middle') && (
              <>
                <Divider style={{ margin: '16px 0 12px' }} />
                <div style={{ marginBottom: 16 }}>
                  <h4 style={{ marginBottom: 8 }}>操作</h4>
                  {renderActions(drawerItem, 'middle')}
                </div>
              </>
            )}

            {/* Sampling records */}
            {(samplingData || []).length > 0 && (
              <>
                <Divider style={{ margin: '16px 0 12px' }} />
                <h4 style={{ marginBottom: 12 }}>打样记录</h4>
                {(samplingData || []).map((sr: SamplingRequest) => (
                  <Card key={sr.id} size="small" style={{ marginBottom: 8 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                      <span style={{ fontWeight: 600, fontSize: 13 }}>R{sr.round}</span>
                      <Tag color={
                        sr.status === 'passed' ? 'green' :
                        sr.status === 'failed' ? 'red' :
                        sr.status === 'verifying' ? 'orange' :
                        sr.status === 'arrived' ? 'blue' :
                        sr.status === 'shipping' ? 'purple' : 'default'
                      }>
                        {sr.status === 'preparing' ? '制样中' :
                         sr.status === 'shipping' ? '运输中' :
                         sr.status === 'arrived' ? '已到货' :
                         sr.status === 'verifying' ? '验证中' :
                         sr.status === 'passed' ? '已通过' :
                         sr.status === 'failed' ? '不通过' : sr.status}
                      </Tag>
                    </div>
                    <div style={{ fontSize: 12, color: '#666' }}>
                      供应商: {sr.supplier_name || '-'} | 样品数: {sr.sample_qty}
                    </div>
                    {sr.notes && <div style={{ fontSize: 12, color: '#999', marginTop: 2 }}>{sr.notes}</div>}
                    {sr.verify_result && (
                      <div style={{ fontSize: 12, marginTop: 2 }}>
                        验证结果: <Tag color={sr.verify_result === 'passed' ? 'green' : 'red'} style={{ fontSize: 11 }}>
                          {sr.verify_result === 'passed' ? '通过' : '不通过'}
                        </Tag>
                        {sr.reject_reason && <span style={{ color: '#999' }}>{sr.reject_reason}</span>}
                      </div>
                    )}
                    <div style={{ fontSize: 11, color: '#bbb', marginTop: 4 }}>
                      {dayjs(sr.created_at).format('YYYY-MM-DD HH:mm')}
                    </div>
                    {/* Sampling status action buttons */}
                    {(sr.status === 'preparing' || sr.status === 'shipping') && (
                      <div style={{ marginTop: 6, borderTop: '1px solid #f0f0f0', paddingTop: 6 }}>
                        {sr.status === 'preparing' && (
                          <Button size="small" type="link" loading={actionLoading}
                            onClick={() => handleUpdateSamplingStatus(sr.id, 'shipping')}>
                            标记发货
                          </Button>
                        )}
                        {sr.status === 'shipping' && (
                          <Button size="small" type="link" loading={actionLoading}
                            onClick={() => handleUpdateSamplingStatus(sr.id, 'arrived')}>
                            确认到货
                          </Button>
                        )}
                      </div>
                    )}
                  </Card>
                ))}
              </>
            )}

            <Divider style={{ margin: '16px 0 12px' }} />
            <h4 style={{ marginBottom: 12 }}>操作记录</h4>
            {(activityData?.items || []).length === 0 ? (
              <Empty description="暂无操作记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            ) : (
              <Timeline
                items={(activityData?.items || []).map((log: ActivityLog) => ({
                  children: (
                    <div>
                      <div style={{ fontSize: 13 }}>{log.content}</div>
                      <div style={{ fontSize: 12, color: '#999' }}>
                        {log.operator_name} · {dayjs(log.created_at).format('MM-DD HH:mm')}
                      </div>
                    </div>
                  ),
                }))}
              />
            )}
          </>
        )}
      </Drawer>

      {/* Passive group detail drawer */}
      <Drawer
        title={`被动元件 (${passiveDrawer?.items.length || 0}颗)`}
        open={!!passiveDrawer}
        onClose={() => { setPassiveDrawer(null); setSelectedPassiveIds([]); }}
        width={720}
      >
        {passiveDrawer && (
          <>
            <Table
              dataSource={passiveDrawer.items}
              columns={passiveTableColumns}
              rowKey="id"
              size="small"
              pagination={false}
              rowSelection={{
                selectedRowKeys: selectedPassiveIds,
                onChange: (keys) => setSelectedPassiveIds(keys as string[]),
              }}
            />

            {/* Batch action bar */}
            {selectedPassiveIds.length > 0 && (
              <div style={{
                position: 'sticky',
                bottom: 0,
                background: '#fff',
                borderTop: '1px solid #f0f0f0',
                padding: '12px 0',
                marginTop: 16,
                display: 'flex',
                alignItems: 'center',
                gap: 12,
              }}>
                <Checkbox
                  checked={selectedPassiveIds.length === passiveDrawer.items.length}
                  indeterminate={selectedPassiveIds.length > 0 && selectedPassiveIds.length < passiveDrawer.items.length}
                  onChange={(e) => {
                    setSelectedPassiveIds(e.target.checked ? passiveDrawer.items.map((i) => i.id) : []);
                  }}
                >
                  全选
                </Checkbox>
                <span style={{ color: '#666', fontSize: 13 }}>已选 {selectedPassiveIds.length} 项</span>
                <div style={{ flex: 1 }} />
                {passiveDrawer.status === 'pending' && (
                  <Button
                    type="primary"
                    size="small"
                    onClick={() => setBatchAssignModal(true)}
                  >
                    批量分配供应商
                  </Button>
                )}
                {getBatchStatusActions(passiveDrawer).map((action) => (
                  <Button
                    key={action.toStatus}
                    size="small"
                    type={action.primary ? 'primary' : 'default'}
                    danger={action.danger}
                    onClick={() => {
                      setBatchTargetStatus(action.toStatus);
                      setBatchStatusModal(true);
                    }}
                  >
                    批量{action.label}
                  </Button>
                ))}
              </div>
            )}
          </>
        )}
      </Drawer>

      {/* Assign Supplier Modal (single item) */}
      <Modal
        title="分配供应商"
        open={!!assignModalItem}
        onCancel={() => { setAssignModalItem(null); assignForm.resetFields(); }}
        onOk={handleAssignSupplier}
        confirmLoading={actionLoading}
        okText="确认分配"
        cancelText="取消"
        destroyOnClose
      >
        {assignModalItem && (
          <div style={{ marginBottom: 16, color: '#666', fontSize: 13 }}>
            物料: <strong>{assignModalItem.material_name}</strong> ({assignModalItem.material_code || '-'})
          </div>
        )}
        <Form form={assignForm} layout="vertical">
          <Form.Item
            name="supplier_id"
            label="供应商"
            rules={[{ required: true, message: '请选择供应商' }]}
          >
            <Select
              placeholder="请选择供应商"
              showSearch
              optionFilterProp="label"
              options={supplierList.map((s) => ({
                value: s.id,
                label: `${s.code} - ${s.name}`,
              }))}
            />
          </Form.Item>
          <Form.Item name="unit_price" label="单价 (¥)">
            <InputNumber style={{ width: '100%' }} min={0} precision={2} placeholder="请输入单价" />
          </Form.Item>
          <Form.Item name="expected_date" label="预计交期">
            <DatePicker style={{ width: '100%' }} placeholder="请选择预计交期" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Batch Assign Supplier Modal */}
      <Modal
        title={`批量分配供应商 (${selectedPassiveIds.length}颗)`}
        open={batchAssignModal}
        onCancel={() => { setBatchAssignModal(false); batchAssignForm.resetFields(); }}
        onOk={handleBatchAssignSupplier}
        confirmLoading={actionLoading}
        okText="确认分配"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={batchAssignForm} layout="vertical">
          <Form.Item
            name="supplier_id"
            label="供应商"
            rules={[{ required: true, message: '请选择供应商' }]}
          >
            <Select
              placeholder="请选择供应商"
              showSearch
              optionFilterProp="label"
              options={supplierList.map((s) => ({
                value: s.id,
                label: `${s.code} - ${s.name}`,
              }))}
            />
          </Form.Item>
          <Form.Item name="unit_price" label="单价 (¥)">
            <InputNumber style={{ width: '100%' }} min={0} precision={2} placeholder="请输入单价" />
          </Form.Item>
          <Form.Item name="expected_date" label="预计交期">
            <DatePicker style={{ width: '100%' }} placeholder="请选择预计交期" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Batch Status Change Confirm Modal */}
      <Modal
        title="批量状态变更"
        open={batchStatusModal}
        onCancel={() => { setBatchStatusModal(false); setBatchTargetStatus(''); }}
        onOk={handleBatchStatusChange}
        confirmLoading={actionLoading}
        okText="确认变更"
        cancelText="取消"
      >
        <p>
          确定要将选中的 <strong>{selectedPassiveIds.length}</strong> 颗被动元件状态变更为
          「<strong>{itemStatusLabels[batchTargetStatus] || batchTargetStatus}</strong>」吗？
        </p>
      </Modal>

      {/* Sampling Modal */}
      <Modal
        title="发起打样"
        open={!!samplingModalItem}
        onCancel={() => { setSamplingModalItem(null); samplingForm.resetFields(); }}
        onOk={handleCreateSampling}
        confirmLoading={actionLoading}
        okText="确认发起"
        cancelText="取消"
        destroyOnClose
      >
        {samplingModalItem && (
          <div style={{ marginBottom: 16, color: '#666', fontSize: 13 }}>
            物料: <strong>{samplingModalItem.material_name}</strong> ({samplingModalItem.material_code || '-'})
          </div>
        )}
        <Form form={samplingForm} layout="vertical">
          <Form.Item
            name="supplier_id"
            label="供应商"
            rules={[{ required: true, message: '请选择供应商' }]}
          >
            <Select
              placeholder="请选择供应商"
              showSearch
              optionFilterProp="label"
              options={supplierList.map((s) => ({
                value: s.id,
                label: `${s.code} - ${s.name}`,
              }))}
            />
          </Form.Item>
          <Form.Item
            name="sample_qty"
            label="样品数量"
            rules={[{ required: true, message: '请输入样品数量' }]}
          >
            <InputNumber style={{ width: '100%' }} min={1} placeholder="请输入样品数量" />
          </Form.Item>
          <Form.Item name="notes" label="备注">
            <Input.TextArea rows={2} placeholder="备注信息" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

// Passive group aggregated card
const PassiveGroupCard: React.FC<{
  group: PassiveGroup;
  supplierMap: Record<string, string>;
  onClick: () => void;
}> = ({ group, onClick }) => {
  const total = group.items.length;
  const assignedCount = group.items.filter((i) => !!i.supplier_id).length;
  const pct = total > 0 ? Math.round((assignedCount / total) * 100) : 0;

  return (
    <div
      onClick={onClick}
      data-testid="passive-group-card"
      style={{
        background: 'linear-gradient(135deg, #e6f7ff 0%, #f0f5ff 100%)',
        borderRadius: 6,
        padding: '10px 12px',
        borderLeft: '4px solid #1890ff',
        boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
        cursor: 'pointer',
        transition: 'box-shadow 0.2s',
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.boxShadow = '0 2px 8px rgba(0,0,0,0.15)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.boxShadow = '0 1px 3px rgba(0,0,0,0.08)';
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
        <AppstoreOutlined style={{ color: '#1890ff', fontSize: 14 }} />
        <span style={{ fontWeight: 600, fontSize: 13 }}>被动元件</span>
        <Tag color="blue" style={{ fontSize: 11, lineHeight: '18px', padding: '0 4px', margin: 0 }}>
          {total}颗
        </Tag>
      </div>

      {/* Category breakdown */}
      <div style={{ fontSize: 11, color: '#666', marginBottom: 6 }}>
        {Array.from(new Set(group.items.map((i) => i.category)))
          .map((cat) => {
            const count = group.items.filter((i) => i.category === cat).length;
            return `${cat}(${count})`;
          })
          .join(' / ')}
      </div>

      {/* Progress bar */}
      <Progress
        percent={pct}
        size="small"
        strokeColor="#1890ff"
        format={() => `${assignedCount}/${total}`}
        style={{ margin: 0 }}
      />
    </div>
  );
};

// BOM type → material category tag config
const getBomCategoryTag = (item: KanbanItem): { label: string; color: string } | null => {
  const bomType = (item.source_bom_type || '').toUpperCase();
  const matGroup = (item.material_group || '').toLowerCase();

  if (bomType === 'EBOM') return { label: '电子类', color: 'blue' };
  if (bomType === 'PBOM') return { label: '工艺类', color: 'green' };
  if (bomType === 'PBOM') return { label: '包装类', color: 'orange' };
  if (matGroup === 'tooling' || bomType === 'TOOLING') return { label: '工装治具类', color: 'purple' };
  if (matGroup === 'auxiliary' || matGroup === 'consumable' || bomType === 'CONSUMABLE') return { label: '辅料类', color: 'default' };
  return null;
};

// Individual kanban card
const KanbanCard: React.FC<{
  item: KanbanItem;
  supplierMap: Record<string, string>;
  onClick: () => void;
  actions?: React.ReactNode;
}> = ({ item, supplierMap, onClick, actions }) => {
  // Calculate urgency based on expected_date or project target date
  const deadline = item.expected_date || item.project_target_date;
  let borderColor = '#e8e8e8'; // gray - no deadline
  let countdownText = '';

  if (deadline && item.status !== 'passed') {
    const now = dayjs();
    const target = dayjs(deadline);
    const daysLeft = target.diff(now, 'day');

    if (daysLeft < 0) {
      borderColor = '#ff4d4f'; // red - overdue
      countdownText = `超期${Math.abs(daysLeft)}天`;
    } else if (daysLeft <= 3) {
      borderColor = '#faad14'; // yellow - urgent
      countdownText = `还剩${daysLeft}天`;
    } else {
      borderColor = '#52c41a'; // green - on track
      countdownText = `还剩${daysLeft}天`;
    }
  }

  // Round badge (sampling round)
  const roundMatch = item.notes?.match(/R(\d+)/);
  const roundNum = roundMatch ? parseInt(roundMatch[1]) : 0;

  return (
    <div
      onClick={onClick}
      style={{
        background: '#fff',
        borderRadius: 6,
        padding: '10px 12px',
        borderLeft: `4px solid ${borderColor}`,
        boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
        cursor: 'pointer',
        transition: 'box-shadow 0.2s',
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.boxShadow = '0 2px 8px rgba(0,0,0,0.15)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.boxShadow = '0 1px 3px rgba(0,0,0,0.08)';
      }}
    >
      {/* Material name + BOM category tag */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
        <span style={{ fontWeight: 600, fontSize: 13, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', marginRight: 4 }}>
          {item.material_name}
        </span>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4, flexShrink: 0 }}>
          {(() => {
            const tagInfo = getBomCategoryTag(item);
            return tagInfo ? (
              <Tag color={tagInfo.color} style={{ fontSize: 11, lineHeight: '18px', padding: '0 4px', margin: 0 }}>
                {tagInfo.label}
              </Tag>
            ) : null;
          })()}
          {roundNum > 1 && (
            <Tag color="purple" style={{ fontSize: 11, lineHeight: '18px', padding: '0 4px', margin: 0 }}>
              R{roundNum}
            </Tag>
          )}
        </div>
      </div>

      {/* Material code */}
      <div style={{ marginBottom: 6 }}>
        <span style={{ fontSize: 12, color: '#999', fontFamily: 'monospace' }}>
          {item.material_code || '-'}
        </span>
      </div>

      {/* Supplier */}
      {item.supplier_id && (
        <div style={{ fontSize: 12, color: '#666', marginBottom: 4 }}>
          {supplierMap[item.supplier_id] || '供应商'}
        </div>
      )}

      {/* Bottom row: countdown + category */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 4 }}>
        {countdownText && (
          <span style={{ fontSize: 11 }}>{countdownText}</span>
        )}
        {item.category && (
          <Tag style={{ fontSize: 11, lineHeight: '18px', padding: '0 4px', margin: 0 }}>
            {item.category}
          </Tag>
        )}
      </div>

      {/* Action buttons on card */}
      {actions && (
        <div style={{ marginTop: 6, borderTop: '1px solid #f0f0f0', paddingTop: 6 }}>
          {actions}
        </div>
      )}
    </div>
  );
};

export default KanbanBoard;
