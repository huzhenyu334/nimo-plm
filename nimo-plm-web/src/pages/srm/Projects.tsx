import React, { useState } from 'react';
import { Table, Card, Tag, Progress, Select, Input, Button, Space } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { srmApi, SRMProject } from '@/api/srm';
import dayjs from 'dayjs';

const { Search } = Input;

const typeLabels: Record<string, string> = { sample: '打样', production: '量产' };
const typeColors: Record<string, string> = { sample: 'blue', production: 'green' };

const phaseLabels: Record<string, string> = { concept: 'Concept', evt: 'EVT', dvt: 'DVT', pvt: 'PVT', mp: 'MP' };
const phaseColors: Record<string, string> = { concept: 'purple', evt: 'blue', dvt: 'cyan', pvt: 'orange', mp: 'green' };

const statusLabels: Record<string, string> = { active: '进行中', completed: '已完成', cancelled: '已取消' };
const statusColors: Record<string, string> = { active: 'processing', completed: 'success', cancelled: 'default' };

const SRMProjects: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchText, setSearchText] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>();
  const [filterType, setFilterType] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading } = useQuery({
    queryKey: ['srm-projects', searchText, filterStatus, filterType, page, pageSize],
    queryFn: () =>
      srmApi.listProjects({
        search: searchText || undefined,
        status: filterStatus,
        type: filterType,
        page,
        page_size: pageSize,
      }),
  });

  const columns = [
    {
      title: '编码',
      dataIndex: 'code',
      key: 'code',
      width: 150,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 80,
      render: (t: string) => <Tag color={typeColors[t]}>{typeLabels[t] || t}</Tag>,
    },
    {
      title: '阶段',
      dataIndex: 'phase',
      key: 'phase',
      width: 80,
      render: (p: string) => p ? <Tag color={phaseColors[p]}>{phaseLabels[p] || p.toUpperCase()}</Tag> : '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (s: string) => <Tag color={statusColors[s]}>{statusLabels[s] || s}</Tag>,
    },
    {
      title: '进度',
      key: 'progress',
      width: 180,
      render: (_: unknown, record: SRMProject) => {
        const total = record.total_items || 0;
        const passed = record.passed_count || 0;
        const pct = total > 0 ? Math.round((passed / total) * 100) : 0;
        return (
          <div>
            <Progress percent={pct} size="small" style={{ marginBottom: 0 }} />
            <span style={{ fontSize: 12, color: '#999' }}>{passed}/{total}</span>
          </div>
        );
      },
    },
    {
      title: '预估交期',
      dataIndex: 'estimated_days',
      key: 'estimated_days',
      width: 90,
      render: (d?: number) => d != null ? `${d}天` : '-',
    },
    {
      title: '目标日期',
      dataIndex: 'target_date',
      key: 'target_date',
      width: 110,
      render: (d?: string) => d ? dayjs(d).format('YYYY-MM-DD') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: SRMProject) => (
        <Button
          type="link"
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            navigate(`/srm/kanban?project=${record.id}`);
          }}
        >
          看板
        </Button>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="采购项目"
        extra={
          <Space wrap>
            <Select
              placeholder="类型"
              allowClear
              style={{ width: 100 }}
              options={Object.entries(typeLabels).map(([k, v]) => ({ value: k, label: v }))}
              value={filterType}
              onChange={(v) => { setFilterType(v); setPage(1); }}
            />
            <Select
              placeholder="状态"
              allowClear
              style={{ width: 110 }}
              options={Object.entries(statusLabels).map(([k, v]) => ({ value: k, label: v }))}
              value={filterStatus}
              onChange={(v) => { setFilterStatus(v); setPage(1); }}
            />
            <Search
              placeholder="搜索项目"
              allowClear
              style={{ width: 180 }}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              onSearch={() => setPage(1)}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={() => queryClient.invalidateQueries({ queryKey: ['srm-projects'] })}
            >
              刷新
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data?.items || []}
          rowKey="id"
          loading={isLoading}
          scroll={{ x: 1000 }}
          pagination={{
            current: page,
            pageSize,
            total: data?.pagination?.total || 0,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps); },
          }}
          onRow={(record) => ({
            onClick: () => navigate(`/srm/kanban?project=${record.id}`),
            style: { cursor: 'pointer' },
          })}
        />
      </Card>
    </div>
  );
};

export default SRMProjects;
