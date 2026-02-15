import { useState, useEffect } from 'react';
import { Card, Table, Tag, Space, Button, Input, Select, Statistic, Row, Col, Segmented, App, Tooltip, Spin } from 'antd';
import { PlusOutlined, ReloadOutlined, SearchOutlined, AppstoreOutlined, BarsOutlined, AlertOutlined, ClockCircleOutlined, FileTextOutlined, CheckCircleOutlined, RightOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { ecnApi, ECN, ECNStats } from '@/api/ecn';
import { projectApi } from '@/api/projects';
import { useIsMobile } from '@/hooks/useIsMobile';

const statusMap: Record<string, { color: string; label: string }> = {
  draft: { color: 'default', label: '草稿' },
  pending: { color: 'processing', label: '待审批' },
  approved: { color: 'success', label: '已批准' },
  rejected: { color: 'error', label: '已驳回' },
  executing: { color: 'warning', label: '执行中' },
  closed: { color: 'default', label: '已关闭' },
  implemented: { color: 'success', label: '已实施' },
  cancelled: { color: 'default', label: '已取消' },
};

const urgencyMap: Record<string, { color: string; label: string }> = {
  low: { color: 'blue', label: '常规' },
  medium: { color: 'blue', label: '常规' },
  high: { color: 'orange', label: '紧急' },
  critical: { color: 'red', label: '特急' },
};

const changeTypeMap: Record<string, string> = {
  design: '设计变更',
  material: '物料变更',
  process: '工艺变更',
  spec: '规格变更',
  document: '文档变更',
};

const ECNList = () => {
  const navigate = useNavigate();
  const { message } = App.useApp();
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [ecns, setEcns] = useState<ECN[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [stats, setStats] = useState<ECNStats>({ pending_approval: 0, executing: 0, month_created: 0, month_closed: 0 });
  const [_products, setProducts] = useState<{ id: string; name: string }[]>([]);
  const [filters, setFilters] = useState<{ status?: string; keyword?: string; product_id?: string; change_type?: string; urgency?: string }>({});
  const [viewMode, setViewMode] = useState<'card' | 'table'>('table');

  const fetchECNs = async () => {
    setLoading(true);
    try {
      const result = await ecnApi.list({ ...filters, page, page_size: 20 });
      setEcns(result.items || []);
      setTotal(result.total);
    } catch {
      message.error('加载ECN列表失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      const s = await ecnApi.getStats();
      setStats(s);
    } catch { /* ignore */ }
  };

  const fetchProducts = async () => {
    try {
      const data = await projectApi.list();
      const items = (data as any)?.items || [];
      setProducts(items.map((p: any) => ({ id: p.product_id || p.id, name: p.name })));
    } catch { /* ignore */ }
  };

  useEffect(() => { fetchProducts(); }, []);
  useEffect(() => { fetchECNs(); fetchStats(); }, [page, filters]);

  const columns = [
    {
      title: '编号',
      dataIndex: 'code',
      width: 150,
      render: (code: string, record: ECN) => (
        <a onClick={() => navigate(`/ecn/${record.id}`)}>{code}</a>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
    },
    {
      title: '产品',
      dataIndex: ['product', 'name'],
      width: 120,
    },
    {
      title: '变更类型',
      dataIndex: 'change_type',
      width: 100,
      render: (type: string) => changeTypeMap[type] || type,
    },
    {
      title: '紧急程度',
      dataIndex: 'urgency',
      width: 80,
      render: (u: string) => {
        const info = urgencyMap[u] || { color: 'default', label: u };
        return <Tag color={info.color}>{info.label}</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (s: string) => {
        const info = statusMap[s] || { color: 'default', label: s };
        return <Tag color={info.color}>{info.label}</Tag>;
      },
    },
    {
      title: '受影响项',
      width: 80,
      render: (_: any, record: ECN) => record.affected_items?.length || 0,
    },
    {
      title: '完成率',
      dataIndex: 'completion_rate',
      width: 80,
      render: (rate: number, record: ECN) =>
        ['executing', 'closed'].includes(record.status) ? `${rate || 0}%` : '-',
    },
    {
      title: '申请人',
      dataIndex: ['requester', 'name'],
      width: 80,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 110,
      render: (t: string) => t ? dayjs(t).format('MM-DD HH:mm') : '-',
    },
    {
      title: '操作',
      width: 80,
      render: (_: any, record: ECN) => (
        <Button type="link" size="small" onClick={() => navigate(`/ecn/${record.id}`)}>
          查看
        </Button>
      ),
    },
  ];

  const statusTabs = [
    { label: '全部', value: '' },
    { label: '草稿', value: 'draft' },
    { label: '待审批', value: 'pending' },
    { label: '执行中', value: 'executing' },
    { label: '已关闭', value: 'closed' },
  ];

  // ========== Mobile Layout ==========
  if (isMobile) {
    return (
      <div style={{ background: '#f5f5f5', minHeight: '100vh' }}>
        {/* Stats row */}
        <div style={{ display: 'flex', gap: 8, padding: '12px 12px 4px', overflowX: 'auto' }}>
          {[
            { label: '待审批', value: stats.pending_approval, color: stats.pending_approval > 0 ? '#ff4d4f' : undefined },
            { label: '执行中', value: stats.executing, color: '#fa8c16' },
            { label: '月新建', value: stats.month_created, color: '#1890ff' },
            { label: '月关闭', value: stats.month_closed, color: '#52c41a' },
          ].map(s => (
            <div key={s.label} style={{
              flex: 1, minWidth: 72, background: '#fff', borderRadius: 10, padding: '10px 8px',
              textAlign: 'center', boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
            }}>
              <div style={{ fontSize: 20, fontWeight: 700, color: s.color || '#333' }}>{s.value}</div>
              <div style={{ fontSize: 11, color: '#999' }}>{s.label}</div>
            </div>
          ))}
        </div>

        {/* Search */}
        <div style={{ padding: '8px 12px' }}>
          <Input
            placeholder="搜索编号/标题"
            prefix={<SearchOutlined style={{ color: '#bbb' }} />}
            allowClear
            style={{ borderRadius: 20 }}
            onPressEnter={(e) => { setFilters(f => ({ ...f, keyword: (e.target as HTMLInputElement).value })); setPage(1); }}
            onChange={(e) => { if (!e.target.value) { setFilters(f => ({ ...f, keyword: '' })); setPage(1); } }}
          />
        </div>

        {/* Status filter pills */}
        <div className="mobile-filter-pills" style={{ padding: '4px 12px 8px' }}>
          {statusTabs.map(tab => (
            <div
              key={tab.value}
              className={`mobile-filter-pill ${(filters.status || '') === tab.value ? 'active' : ''}`}
              onClick={() => { setFilters(f => ({ ...f, status: tab.value })); setPage(1); }}
            >{tab.label}</div>
          ))}
        </div>

        {/* Card list */}
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : (
          <div style={{ padding: '0 12px' }}>
            {ecns.map(ecn => {
              const statusInfo = statusMap[ecn.status] || { color: 'default', label: ecn.status };
              const urgencyInfo = urgencyMap[ecn.urgency] || { color: 'default', label: ecn.urgency };
              return (
                <div
                  key={ecn.id}
                  onClick={() => navigate(`/ecn/${ecn.id}`)}
                  style={{
                    background: '#fff', borderRadius: 10, padding: '12px 14px',
                    marginBottom: 8, boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 6 }}>
                    <span style={{ fontSize: 12, fontFamily: 'monospace', color: '#1677ff' }}>{ecn.code}</span>
                    <div style={{ marginLeft: 'auto', display: 'flex', gap: 4 }}>
                      {(ecn.urgency === 'high' || ecn.urgency === 'critical') && (
                        <Tag color={urgencyInfo.color} style={{ margin: 0, fontSize: 11 }}>{urgencyInfo.label}</Tag>
                      )}
                      <Tag color={statusInfo.color} style={{ margin: 0, fontSize: 11 }}>{statusInfo.label}</Tag>
                    </div>
                  </div>
                  <div style={{ fontSize: 15, fontWeight: 500, marginBottom: 6, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {ecn.title}
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', fontSize: 12, color: '#999', gap: 12 }}>
                    <span>{changeTypeMap[ecn.change_type] || ecn.change_type}</span>
                    <span>{ecn.product?.name || '-'}</span>
                    <span>{ecn.affected_items?.length || 0}项</span>
                    <span style={{ marginLeft: 'auto' }}>{ecn.created_at ? dayjs(ecn.created_at).format('M/D') : ''}</span>
                    <RightOutlined style={{ fontSize: 10, color: '#ccc' }} />
                  </div>
                </div>
              );
            })}
            {ecns.length === 0 && !loading && (
              <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>暂无ECN数据</div>
            )}
            {total > 20 && ecns.length >= 20 && (
              <div style={{ textAlign: 'center', padding: 12 }}>
                <Button size="small" onClick={() => setPage(p => p + 1)}>加载更多</Button>
              </div>
            )}
          </div>
        )}

        {/* FAB */}
        <div
          onClick={() => navigate('/ecn/new')}
          style={{
            position: 'fixed', bottom: 80, right: 20, width: 52, height: 52,
            borderRadius: 26, background: '#1677ff', color: '#fff',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: '0 4px 12px rgba(22,119,255,0.4)', zIndex: 100,
            fontSize: 22, cursor: 'pointer',
          }}
        >
          <PlusOutlined />
        </div>
      </div>
    );
  }

  // ========== Desktop Layout ==========
  return (
    <div style={{ padding: 0 }}>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ textAlign: 'center' }}>
            <Statistic
              title="待我审批"
              value={stats.pending_approval}
              prefix={<AlertOutlined style={{ color: '#ff4d4f' }} />}
              valueStyle={stats.pending_approval > 0 ? { color: '#ff4d4f' } : undefined}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ textAlign: 'center' }}>
            <Statistic
              title="进行中"
              value={stats.executing}
              prefix={<ClockCircleOutlined style={{ color: '#fa8c16' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ textAlign: 'center' }}>
            <Statistic
              title="本月新建"
              value={stats.month_created}
              prefix={<FileTextOutlined style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ textAlign: 'center' }}>
            <Statistic
              title="本月关闭"
              value={stats.month_closed}
              prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
      </Row>

      {/* 主要内容 */}
      <Card
        title="ECN工程变更"
        extra={
          <Space>
            <Segmented
              options={[
                { value: 'table', icon: <BarsOutlined /> },
                { value: 'card', icon: <AppstoreOutlined /> },
              ]}
              value={viewMode}
              onChange={(v) => setViewMode(v as 'card' | 'table')}
            />
            <Tooltip title="刷新">
              <Button icon={<ReloadOutlined />} onClick={() => { fetchECNs(); fetchStats(); }} />
            </Tooltip>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/ecn/new')}>
              新建变更请求
            </Button>
          </Space>
        }
      >
        {/* 筛选栏 */}
        <Space wrap style={{ marginBottom: 16, width: '100%' }}>
          <Segmented
            options={statusTabs}
            value={filters.status || ''}
            onChange={(v) => { setFilters(f => ({ ...f, status: v as string })); setPage(1); }}
          />
          <Input
            placeholder="搜索编号/标题"
            prefix={<SearchOutlined />}
            allowClear
            style={{ width: 200 }}
            onPressEnter={(e) => { setFilters(f => ({ ...f, keyword: (e.target as HTMLInputElement).value })); setPage(1); }}
            onChange={(e) => { if (!e.target.value) { setFilters(f => ({ ...f, keyword: '' })); setPage(1); } }}
          />
          <Select
            placeholder="变更类型"
            allowClear
            style={{ width: 120 }}
            options={Object.entries(changeTypeMap).map(([k, v]) => ({ label: v, value: k }))}
            onChange={(v) => { setFilters(f => ({ ...f, change_type: v })); setPage(1); }}
          />
          <Select
            placeholder="紧急程度"
            allowClear
            style={{ width: 110 }}
            options={[
              { label: '常规', value: 'medium' },
              { label: '紧急', value: 'high' },
              { label: '特急', value: 'critical' },
            ]}
            onChange={(v) => { setFilters(f => ({ ...f, urgency: v })); setPage(1); }}
          />
        </Space>

        {/* 列表 */}
        {viewMode === 'table' ? (
          <Table
            dataSource={ecns}
            columns={columns}
            rowKey="id"
            loading={loading}
            size="small"
            pagination={{
              current: page,
              total,
              pageSize: 20,
              showTotal: (t) => `共 ${t} 条`,
              onChange: (p) => setPage(p),
            }}
          />
        ) : (
          <>
            <Row gutter={[16, 16]}>
              {ecns.map(ecn => {
                const statusInfo = statusMap[ecn.status] || { color: 'default', label: ecn.status };
                const urgencyInfo = urgencyMap[ecn.urgency] || { color: 'default', label: ecn.urgency };
                return (
                  <Col key={ecn.id} xs={24} sm={24} md={12} lg={8}>
                    <Card
                      hoverable
                      size="small"
                      onClick={() => navigate(`/ecn/${ecn.id}`)}
                      style={{ cursor: 'pointer' }}
                    >
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                        <span style={{ fontWeight: 500, color: '#1890ff' }}>{ecn.code}</span>
                        <Space size={4}>
                          {(ecn.urgency === 'high' || ecn.urgency === 'critical') && (
                            <Tag color={urgencyInfo.color}>{urgencyInfo.label}</Tag>
                          )}
                          <Tag color={statusInfo.color}>{statusInfo.label}</Tag>
                        </Space>
                      </div>
                      <div style={{ fontSize: 14, fontWeight: 500, marginBottom: 8, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {ecn.title}
                      </div>
                      <div style={{ color: '#999', fontSize: 12 }}>
                        <Space split="·">
                          <span>产品：{ecn.product?.name || '-'}</span>
                          <span>{changeTypeMap[ecn.change_type] || ecn.change_type}</span>
                        </Space>
                      </div>
                      <div style={{ color: '#999', fontSize: 12, marginTop: 4 }}>
                        <Space split="·">
                          <span>受影响项：{ecn.affected_items?.length || 0}个</span>
                          <span>{ecn.requester?.name || '-'}</span>
                          <span>{ecn.created_at ? dayjs(ecn.created_at).format('M月D日') : '-'}</span>
                        </Space>
                      </div>
                    </Card>
                  </Col>
                );
              })}
            </Row>
            {total > 20 && (
              <div style={{ textAlign: 'center', marginTop: 16 }}>
                <Button onClick={() => setPage(p => p + 1)}>加载更多</Button>
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  );
};

export default ECNList;
