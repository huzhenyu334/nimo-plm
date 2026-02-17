import React, { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Input, Table, Tag, Typography, Select, Card, Space,
} from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import { projectBomApi } from '@/api/projectBom';
import { projectApi } from '@/api/projects';
import { srmApi } from '@/api/srm';
import { CATEGORY_LABELS, SUB_CATEGORY_LABELS } from '@/components/BOM/bomConstants';
import { useIsMobile } from '@/hooks/useIsMobile';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

const CATEGORY_OPTIONS = [
  { label: '全部分类', value: '' },
  { label: '电子', value: 'electronic' },
  { label: '结构', value: 'structural' },
  { label: '光学', value: 'optical' },
  { label: '包装', value: 'packaging' },
  { label: '治工具', value: 'tooling' },
  { label: '辅料', value: 'consumable' },
];

const CATEGORY_COLORS: Record<string, string> = {
  electronic: 'blue',
  structural: 'green',
  optical: 'orange',
  packaging: 'gold',
  tooling: 'purple',
  consumable: 'cyan',
};

const formatCurrency = (v: number | null | undefined): string => {
  if (v == null || isNaN(v)) return '-';
  return `\u00a5${v.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
};

const MaterialSearch: React.FC = () => {
  const navigate = useNavigate();
  const isMobile = useIsMobile();
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('');
  const [projectId, setProjectId] = useState('');
  const [supplierId, setSupplierId] = useState('');
  const [manufacturerId, setManufacturerId] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Fetch projects for filter dropdown
  const { data: projectsData } = useQuery({
    queryKey: ['projects-filter'],
    queryFn: () => projectApi.list({ page_size: 200 }),
    staleTime: 60_000,
  });
  const projectOptions = useMemo(() => {
    const items = projectsData?.items || [];
    return [{ label: '全部项目', value: '' }, ...items.map((p: any) => ({ label: p.name, value: p.id }))];
  }, [projectsData]);

  // Fetch suppliers for filter dropdown
  const { data: suppliersData } = useQuery({
    queryKey: ['suppliers-filter'],
    queryFn: () => srmApi.listSuppliers({ page_size: 200 }),
    staleTime: 60_000,
  });
  const supplierOptions = useMemo(() => {
    const items = suppliersData?.items || [];
    return [{ label: '全部供应商', value: '' }, ...items.map((s: any) => ({ label: s.short_name || s.name, value: s.id }))];
  }, [suppliersData]);

  // Manufacturer options: filter suppliers that are manufacturer type
  const manufacturerOptions = useMemo(() => {
    const items = suppliersData?.items || [];
    const mfrs = items.filter((s: any) => s.category === 'manufacturer' || s.category === 'both');
    return [{ label: '全部制造商', value: '' }, ...mfrs.map((s: any) => ({ label: s.short_name || s.name, value: s.id }))];
  }, [suppliersData]);

  const { data, isLoading } = useQuery({
    queryKey: ['material-global-search', search, category, projectId, supplierId, manufacturerId, page, pageSize],
    queryFn: () => projectBomApi.globalSearch({
      q: search || undefined,
      category: category || undefined,
      project_id: projectId || undefined,
      supplier_id: supplierId || undefined,
      manufacturer_id: manufacturerId || undefined,
      page,
      page_size: pageSize,
    }),
    staleTime: 10_000,
  });

  const items = data?.data || [];
  const total = data?.total || 0;

  // Cost summary for current results
  const costSummary = useMemo(() => {
    let totalCost = 0;
    let unpriced = 0;
    for (const item of items) {
      const price = item.unit_price;
      if (price == null || price === 0) {
        unpriced++;
      } else {
        totalCost += (item.quantity || 0) * price;
      }
    }
    return { totalCost, unpriced, count: items.length };
  }, [items]);

  const columns: ColumnsType<any> = [
    {
      title: '名称', dataIndex: 'name', key: 'name', width: 160, ellipsis: true,
      render: (v: string) => <Text strong style={{ fontSize: 13 }}>{v || '-'}</Text>,
    },
    {
      title: '分类', dataIndex: 'category', key: 'category', width: 80,
      render: (v: string) => v ? (
        <Tag color={CATEGORY_COLORS[v] || 'default'} style={{ fontSize: 11 }}>
          {CATEGORY_LABELS[v] || v}
        </Tag>
      ) : '-',
    },
    {
      title: '子分类', dataIndex: 'sub_category', key: 'sub_category', width: 90,
      render: (v: string) => <Text style={{ fontSize: 12 }}>{SUB_CATEGORY_LABELS[v] || v || '-'}</Text>,
    },
    {
      title: 'MPN', dataIndex: 'mpn', key: 'mpn', width: 130, ellipsis: true,
      render: (v: string) => v ? (
        <Text code style={{ fontSize: 11 }}>{v}</Text>
      ) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>,
    },
    {
      title: '制造商', key: 'manufacturer_name', width: 100, ellipsis: true,
      render: (_: any, record: any) => {
        const name = record.manufacturer_name || record.extended_attrs?.manufacturer || '';
        return name ? <Text style={{ fontSize: 12 }}>{name}</Text> : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>;
      },
    },
    {
      title: '供应商', key: 'supplier_display', width: 100, ellipsis: true,
      render: (_: any, record: any) => {
        const name = record.supplier_name || record.supplier || '';
        return name ? <Text style={{ fontSize: 12 }}>{name}</Text> : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>;
      },
    },
    {
      title: '数量', dataIndex: 'quantity', key: 'quantity', width: 70, align: 'right',
      render: (v: number, record: any) => (
        <Text style={{ fontSize: 12 }}>{v} {record.unit || ''}</Text>
      ),
    },
    {
      title: '单价', dataIndex: 'unit_price', key: 'unit_price', width: 90, align: 'right',
      render: (v: number) => v ? (
        <Text style={{ fontSize: 12 }}>{formatCurrency(v)}</Text>
      ) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>,
    },
    {
      title: 'BOM', key: 'bom_info', width: 110, ellipsis: true,
      render: (_: any, record: any) => (
        <span style={{ fontSize: 12 }}>
          {record.bom_type && <Tag style={{ fontSize: 10, marginRight: 4 }}>{record.bom_type}</Tag>}
          {record.bom_name || '-'}
        </span>
      ),
    },
    {
      title: '所属项目', dataIndex: 'project_name', key: 'project_name', width: 130, ellipsis: true,
      render: (v: string, record: any) => v ? (
        <a onClick={() => record.project_id && navigate(`/bom-management/${record.project_id}`)} style={{ fontSize: 12 }}>
          {v}
        </a>
      ) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>,
    },
  ];

  const resetFilters = () => {
    setSearch('');
    setCategory('');
    setProjectId('');
    setSupplierId('');
    setManufacturerId('');
    setPage(1);
  };

  // Mobile: simplified card view
  if (isMobile) {
    return (
      <div style={{ padding: 12 }}>
        <Input
          prefix={<SearchOutlined />}
          placeholder="搜索物料名称、MPN、供应商..."
          value={search}
          onChange={e => { setSearch(e.target.value); setPage(1); }}
          allowClear
          style={{ marginBottom: 8 }}
        />
        <div style={{ display: 'flex', gap: 8, marginBottom: 8, flexWrap: 'wrap' }}>
          <Select
            value={category}
            onChange={v => { setCategory(v); setPage(1); }}
            options={CATEGORY_OPTIONS}
            style={{ flex: 1, minWidth: 100 }}
            size="small"
          />
          <Select
            value={projectId}
            onChange={v => { setProjectId(v); setPage(1); }}
            options={projectOptions}
            style={{ flex: 1, minWidth: 100 }}
            size="small"
            showSearch
            optionFilterProp="label"
          />
        </div>
        {items.length > 0 && (
          <div style={{
            padding: '8px 12px', background: '#f6ffed', borderRadius: 6,
            marginBottom: 12, fontSize: 12,
          }}>
            <Text>共 {total} 项物料</Text>
            {costSummary.totalCost > 0 && (
              <Text style={{ marginLeft: 12 }}>
                总成本: <Text strong style={{ color: '#1677ff' }}>{formatCurrency(costSummary.totalCost)}</Text>
              </Text>
            )}
            {costSummary.unpriced > 0 && (
              <Text type="warning" style={{ marginLeft: 8 }}>{costSummary.unpriced}项未定价</Text>
            )}
          </div>
        )}
        {items.map((item: any) => (
          <Card
            key={item.id}
            size="small"
            style={{ marginBottom: 8 }}
            onClick={() => item.project_id && navigate(`/bom-management/${item.project_id}`)}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Text strong style={{ fontSize: 13 }}>{item.name}</Text>
              <Tag color={CATEGORY_COLORS[item.category] || 'default'} style={{ fontSize: 10 }}>
                {CATEGORY_LABELS[item.category] || item.category}
              </Tag>
            </div>
            <div style={{ marginTop: 4, fontSize: 12, color: '#666' }}>
              {item.mpn && <Text code style={{ fontSize: 11, marginRight: 8 }}>{item.mpn}</Text>}
              {(item.supplier_name || item.supplier) && (
                <Text style={{ marginRight: 8 }}>供应商: {item.supplier_name || item.supplier}</Text>
              )}
              <Text>数量: {item.quantity} {item.unit}</Text>
              {item.unit_price > 0 && <Text style={{ marginLeft: 8 }}>单价: {formatCurrency(item.unit_price)}</Text>}
            </div>
            {item.project_name && (
              <div style={{ marginTop: 4 }}>
                <Tag style={{ fontSize: 10 }}>{item.bom_type}</Tag>
                <Text type="secondary" style={{ fontSize: 11 }}>{item.project_name}</Text>
              </div>
            )}
          </Card>
        ))}
        {!isLoading && items.length === 0 && (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>
            {search ? '未找到匹配物料' : '输入关键字开始搜索'}
          </div>
        )}
      </div>
    );
  }

  return (
    <div>
      {/* Search & Filter Bar */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap size="middle">
          <Input
            prefix={<SearchOutlined />}
            placeholder="搜索物料名称、MPN、供应商..."
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(1); }}
            allowClear
            style={{ width: 320 }}
          />
          <Select
            value={category}
            onChange={v => { setCategory(v); setPage(1); }}
            options={CATEGORY_OPTIONS}
            style={{ width: 130 }}
          />
          <Select
            value={projectId}
            onChange={v => { setProjectId(v); setPage(1); }}
            options={projectOptions}
            style={{ width: 180 }}
            showSearch
            optionFilterProp="label"
            placeholder="筛选项目"
          />
          <Select
            value={supplierId}
            onChange={v => { setSupplierId(v); setPage(1); }}
            options={supplierOptions}
            style={{ width: 160 }}
            showSearch
            optionFilterProp="label"
            placeholder="筛选供应商"
          />
          <Select
            value={manufacturerId}
            onChange={v => { setManufacturerId(v); setPage(1); }}
            options={manufacturerOptions}
            style={{ width: 160 }}
            showSearch
            optionFilterProp="label"
            placeholder="筛选制造商"
          />
          {(search || category || projectId || supplierId || manufacturerId) && (
            <a onClick={resetFilters} style={{ fontSize: 12 }}>
              <ReloadOutlined /> 重置
            </a>
          )}
        </Space>
      </Card>

      {/* Cost Summary Bar */}
      {items.length > 0 && (
        <div style={{
          display: 'flex', gap: 24, padding: '8px 16px', background: '#fafafa',
          borderRadius: 6, border: '1px solid #f0f0f0', marginBottom: 12, alignItems: 'center',
        }}>
          <Text style={{ fontSize: 13 }}>
            共 <Text strong>{total}</Text> 项物料
          </Text>
          {costSummary.totalCost > 0 && (
            <Text style={{ fontSize: 13 }}>
              当前页总成本: <Text strong style={{ color: '#1677ff' }}>{formatCurrency(costSummary.totalCost)}</Text>
            </Text>
          )}
          {costSummary.unpriced > 0 && (
            <Text type="warning" style={{ fontSize: 12 }}>
              {costSummary.unpriced}项未定价
            </Text>
          )}
        </div>
      )}

      {/* Table */}
      <Table
        dataSource={items}
        columns={columns}
        rowKey="id"
        loading={isLoading}
        size="small"
        scroll={{ x: 1300 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (t) => `共 ${t} 项`,
          onChange: (p, ps) => { setPage(p); setPageSize(ps); },
        }}
      />
    </div>
  );
};

export default MaterialSearch;
