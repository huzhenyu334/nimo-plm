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
} from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { srmApi, SRMProject, PRItem, PurchaseRequest, ActivityLog } from '@/api/srm';
import dayjs from 'dayjs';

// Kanban column definitions
const KANBAN_COLUMNS = [
  { key: 'pending', label: '待询价', color: '#d9d9d9' },
  { key: 'sourcing', label: '寻源中', color: '#13c2c2' },
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

const itemStatusLabels: Record<string, string> = {
  pending: '待询价', sourcing: '寻源中', ordered: '已下单', shipped: '已发货',
  received: '已收货', inspecting: '检验中', passed: '已通过', failed: '未通过',
};

const KanbanBoard: React.FC = () => {
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const projectId = searchParams.get('project') || '';
  const [drawerItem, setDrawerItem] = useState<KanbanItem | null>(null);

  // Load SRM projects for selector
  const { data: projectsData, isLoading: projectsLoading } = useQuery({
    queryKey: ['srm-projects-list'],
    queryFn: () => srmApi.listProjects({ status: 'active', page_size: 100 }),
  });

  // Load selected project details
  const { data: project } = useQuery({
    queryKey: ['srm-project', projectId],
    queryFn: () => srmApi.getProject(projectId),
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
    queryFn: () => srmApi.listSuppliers({ page_size: 200, status: 'active' }),
  });

  const supplierMap = useMemo(() => {
    const map: Record<string, string> = {};
    (supplierData?.items || []).forEach((s) => { map[s.id] = s.name; });
    return map;
  }, [supplierData]);

  // Flatten all PR items into kanban items
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
    return items;
  }, [prData, project]);

  // Group items by status into columns
  const columnData = useMemo(() => {
    const groups: Record<string, KanbanItem[]> = {};
    KANBAN_COLUMNS.forEach((col) => { groups[col.key] = []; });
    allItems.forEach((item) => {
      const status = item.status as ColumnKey;
      if (groups[status]) {
        groups[status].push(item);
      } else if (item.status === 'failed') {
        // Show failed items in inspecting column
        groups['inspecting'].push(item);
      } else {
        groups['pending'].push(item);
      }
    });
    return groups;
  }, [allItems]);

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

  const handleProjectChange = (value: string) => {
    setSearchParams({ project: value });
  };

  const projects = projectsData?.items || [];
  const isLoading = projectsLoading || prLoading;

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
            options={projects.map((p: SRMProject) => ({
              value: p.id,
              label: `${p.code} - ${p.name}`,
            }))}
          />
          <ReloadOutlined
            style={{ cursor: 'pointer', color: '#1890ff' }}
            onClick={() => {
              queryClient.invalidateQueries({ queryKey: ['srm-prs-kanban'] });
              queryClient.invalidateQueries({ queryKey: ['srm-project'] });
            }}
          />
        </div>

        {projectId && allItems.length > 0 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 16, flex: 1 }}>
            <span style={{ color: '#666', fontSize: 13 }}>
              总计: <strong>{stats.total}</strong> |
              待询价: <strong>{stats.pending}</strong> |
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
          minHeight: 'calc(100vh - 200px)',
        }}>
          {KANBAN_COLUMNS.map((col) => {
            const items = columnData[col.key] || [];
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
                    count={items.length}
                    style={{ backgroundColor: col.color }}
                    overflowCount={999}
                  />
                </div>

                {/* Cards */}
                <div style={{ flex: 1, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 8 }}>
                  {items.map((item) => (
                    <KanbanCard
                      key={item.id}
                      item={item}
                      supplierMap={supplierMap}
                      onClick={() => setDrawerItem(item)}
                    />
                  ))}
                  {items.length === 0 && (
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
    </div>
  );
};

// Individual kanban card
const KanbanCard: React.FC<{
  item: KanbanItem;
  supplierMap: Record<string, string>;
  onClick: () => void;
}> = ({ item, supplierMap, onClick }) => {
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
      countdownText = `🔴 超期${Math.abs(daysLeft)}天`;
    } else if (daysLeft <= 3) {
      borderColor = '#faad14'; // yellow - urgent
      countdownText = `⏰ 还剩${daysLeft}天`;
    } else {
      borderColor = '#52c41a'; // green - on track
      countdownText = `⏰ 还剩${daysLeft}天`;
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
      {/* Material name */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
        <span style={{ fontWeight: 600, fontSize: 13, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {item.material_name}
        </span>
        {roundNum > 1 && (
          <Tag color="purple" style={{ marginLeft: 4, fontSize: 11, lineHeight: '18px', padding: '0 4px' }}>
            R{roundNum}
          </Tag>
        )}
      </div>

      {/* Material code */}
      <div style={{ fontSize: 12, color: '#999', fontFamily: 'monospace', marginBottom: 6 }}>
        {item.material_code || '-'}
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
    </div>
  );
};

export default KanbanBoard;
