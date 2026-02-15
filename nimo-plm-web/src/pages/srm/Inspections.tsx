import React, { useState } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Input,
  Select,
  message,
  Drawer,
  Descriptions,
  Form,
  Spin,
} from 'antd';
import { ReloadOutlined, CheckCircleOutlined, SearchOutlined, RightOutlined } from '@ant-design/icons';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { srmApi, Inspection } from '@/api/srm';
import dayjs from 'dayjs';

const { Search } = Input;

const statusLabels: Record<string, string> = {
  pending: '待检验', in_progress: '检验中', completed: '已完成',
};
const statusColors: Record<string, string> = {
  pending: 'default', in_progress: 'processing', completed: 'success',
};

const resultLabels: Record<string, string> = {
  passed: '合格', failed: '不合格', conditional: '条件放行', '': '待检',
};
const resultColors: Record<string, string> = {
  passed: 'green', failed: 'red', conditional: 'orange',
};

const Inspections: React.FC = () => {
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();
  const [searchText, setSearchText] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>();
  const [filterResult, setFilterResult] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentInsp, setCurrentInsp] = useState<Inspection | null>(null);
  const [completeForm] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['srm-inspections', searchText, filterStatus, filterResult, page, pageSize],
    queryFn: () =>
      srmApi.listInspections({
        status: filterStatus,
        result: filterResult,
        page,
        page_size: pageSize,
      }),
  });

  const { data: inspDetail } = useQuery({
    queryKey: ['srm-inspection', currentInsp?.id],
    queryFn: () => srmApi.getInspection(currentInsp!.id),
    enabled: !!currentInsp?.id && drawerVisible,
  });

  const completeMutation = useMutation({
    mutationFn: (values: { result: string; notes?: string; inspection_items?: unknown }) =>
      srmApi.completeInspection(currentInsp!.id, values),
    onSuccess: () => {
      message.success('检验完成');
      completeForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ['srm-inspections'] });
      queryClient.invalidateQueries({ queryKey: ['srm-inspection', currentInsp?.id] });
    },
    onError: () => message.error('操作失败'),
  });

  const columns = [
    {
      title: '检验编码',
      dataIndex: 'inspection_code',
      key: 'inspection_code',
      width: 140,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    {
      title: 'PO编码',
      key: 'po_code',
      width: 130,
      render: (_: unknown, record: Inspection) => {
        const poCode = (record as any).po?.po_code;
        return poCode ? <span style={{ fontFamily: 'monospace' }}>{poCode}</span> : (record.po_id ? record.po_id.slice(0, 8) : '-');
      },
    },
    { title: '物料', dataIndex: 'material_name', key: 'material_name', width: 160, ellipsis: true },
    {
      title: '供应商',
      key: 'supplier_name',
      width: 120,
      render: (_: unknown, record: Inspection) => {
        const supplierName = (record as any).supplier?.name || (record as any).supplier?.short_name;
        return supplierName || (record.supplier_id ? record.supplier_id.slice(0, 8) : '-');
      },
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      key: 'quantity',
      width: 80,
      render: (q?: number) => q ?? '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (s: string) => <Tag color={statusColors[s]}>{statusLabels[s] || s}</Tag>,
    },
    {
      title: '结果',
      dataIndex: 'result',
      key: 'result',
      width: 90,
      render: (r: string) => {
        if (!r) return <Tag>待检</Tag>;
        return <Tag color={resultColors[r]}>{resultLabels[r] || r}</Tag>;
      },
    },
    {
      title: '检验员',
      dataIndex: 'inspector_id',
      key: 'inspector_id',
      width: 100,
      render: (id: string) => id?.slice(0, 8) || '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: Inspection) => (
        <Button
          type="link"
          size="small"
          onClick={() => { setCurrentInsp(record); setDrawerVisible(true); }}
        >
          详情
        </Button>
      ),
    },
  ];

  const detail = inspDetail || currentInsp;
  const canComplete = detail?.status !== 'completed';
  const inspections = data?.items || [];

  // Shared drawer component rendered in both mobile and desktop
  const drawerNode = (
    <Drawer
      title={detail?.inspection_code || '检验详情'}
      open={drawerVisible}
      onClose={() => { setDrawerVisible(false); setCurrentInsp(null); completeForm.resetFields(); }}
      width={isMobile ? '100%' : 600}
    >
      {detail && (
        <>
          <Descriptions column={isMobile ? 1 : 2} bordered size="small" style={{ marginBottom: 24 }}>
            <Descriptions.Item label="检验编码">{detail.inspection_code}</Descriptions.Item>
            <Descriptions.Item label="物料">{detail.material_name}</Descriptions.Item>
            <Descriptions.Item label="物料编码">{detail.material_code || '-'}</Descriptions.Item>
            <Descriptions.Item label="数量">{detail.quantity ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="抽样数">{detail.sample_qty ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusColors[detail.status]}>{statusLabels[detail.status] || detail.status}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="结果">
              {detail.result ? (
                <Tag color={resultColors[detail.result]}>{resultLabels[detail.result] || detail.result}</Tag>
              ) : (
                '待检'
              )}
            </Descriptions.Item>
            <Descriptions.Item label="检验时间">
              {detail.inspected_at ? dayjs(detail.inspected_at).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="备注" span={isMobile ? 1 : 2}>{detail.notes || '-'}</Descriptions.Item>
            <Descriptions.Item label="检验项目" span={isMobile ? 1 : 2}>
              <pre style={{ margin: 0, fontSize: 12, maxHeight: 200, overflow: 'auto' }}>
                {detail.inspection_items ? JSON.stringify(detail.inspection_items, null, 2) : '无'}
              </pre>
            </Descriptions.Item>
          </Descriptions>

          {canComplete && (
            <>
              <h4>完成检验</h4>
              <Form
                form={completeForm}
                layout="vertical"
                onFinish={(values) => {
                  let items = undefined;
                  if (values.inspection_items_json) {
                    try {
                      items = JSON.parse(values.inspection_items_json);
                    } catch {
                      message.error('检验项目JSON格式错误');
                      return;
                    }
                  }
                  completeMutation.mutate({
                    result: values.result,
                    notes: values.notes,
                    inspection_items: items,
                  });
                }}
              >
                <Form.Item name="result" label="检验结果" rules={[{ required: true, message: '请选择结果' }]}>
                  <Select
                    placeholder="选择结果"
                    options={[
                      { value: 'passed', label: '合格' },
                      { value: 'failed', label: '不合格' },
                      { value: 'conditional', label: '条件放行' },
                    ]}
                  />
                </Form.Item>
                <Form.Item name="notes" label="备注">
                  <Input.TextArea rows={2} placeholder="检验备注" />
                </Form.Item>
                <Form.Item name="inspection_items_json" label="检验项目 (JSON)">
                  <Input.TextArea rows={4} placeholder='[{"item": "外观", "result": "OK"}, ...]' />
                </Form.Item>
                <Button
                  type="primary"
                  htmlType="submit"
                  icon={<CheckCircleOutlined />}
                  loading={completeMutation.isPending}
                >
                  完成检验
                </Button>
              </Form>
            </>
          )}
        </>
      )}
    </Drawer>
  );

  // ========== Mobile Layout ==========
  if (isMobile) {
    const statusFilterOptions = [
      { label: '全部', value: undefined as string | undefined },
      ...Object.entries(statusLabels).map(([k, v]) => ({ label: v, value: k as string | undefined })),
    ];
    return (
      <div style={{ background: '#f5f5f5', minHeight: '100vh' }}>
        {/* Sticky search bar */}
        <div style={{ padding: '12px 16px', background: '#fff', position: 'sticky', top: 0, zIndex: 10 }}>
          <Input
            placeholder="搜索检验编码/物料"
            prefix={<SearchOutlined style={{ color: '#bbb' }} />}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            onPressEnter={() => setPage(1)}
            allowClear
            style={{ borderRadius: 20 }}
          />
        </div>

        {/* Status filter pills */}
        <div className="mobile-filter-pills" style={{ padding: '8px 12px' }}>
          {statusFilterOptions.map(opt => (
            <div
              key={opt.value || 'all'}
              className={`mobile-filter-pill ${filterStatus === opt.value ? 'active' : ''}`}
              onClick={() => { setFilterStatus(opt.value); setPage(1); }}
            >{opt.label}</div>
          ))}
        </div>

        {/* Card list */}
        <div style={{ padding: '0 12px' }}>
          {isLoading ? (
            <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载中..." /></div>
          ) : inspections.length === 0 ? (
            <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>暂无检验记录</div>
          ) : inspections.map((insp) => {
            const poCode = (insp as any).po?.po_code;
            const supplierName = (insp as any).supplier?.name || (insp as any).supplier?.short_name;
            return (
              <div
                key={insp.id}
                onClick={() => { setCurrentInsp(insp); setDrawerVisible(true); }}
                style={{
                  background: '#fff',
                  borderRadius: 10,
                  padding: '12px 14px',
                  marginBottom: 8,
                  boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
                  cursor: 'pointer',
                }}
              >
                {/* Row 1: inspection_code + status tag */}
                <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
                  <span style={{ fontFamily: 'monospace', color: '#1677ff', fontSize: 13, flex: 1 }}>
                    {insp.inspection_code}
                  </span>
                  <Tag color={statusColors[insp.status]} style={{ margin: 0 }}>
                    {statusLabels[insp.status] || insp.status}
                  </Tag>
                </div>

                {/* Row 2: material name (bold) */}
                <div style={{ fontWeight: 600, fontSize: 15, marginBottom: 6, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {insp.material_name || '-'}
                </div>

                {/* Row 3: po_code, supplier, quantity, result */}
                <div style={{ display: 'flex', alignItems: 'center', fontSize: 13, color: '#666', gap: 8, flexWrap: 'wrap' }}>
                  {poCode && (
                    <span style={{ fontFamily: 'monospace', fontSize: 12 }}>PO: {poCode}</span>
                  )}
                  {supplierName && (
                    <span>{supplierName}</span>
                  )}
                  {insp.quantity != null && (
                    <span>x{insp.quantity}</span>
                  )}
                  {insp.result ? (
                    <Tag color={resultColors[insp.result]} style={{ margin: 0, fontSize: 11 }}>
                      {resultLabels[insp.result] || insp.result}
                    </Tag>
                  ) : null}
                  <RightOutlined style={{ fontSize: 10, color: '#ccc', marginLeft: 'auto' }} />
                </div>
              </div>
            );
          })}
        </div>

        {drawerNode}
      </div>
    );
  }

  // ========== Desktop Layout ==========
  return (
    <div>
      <Card
        title="来料检验"
        extra={
          <Space wrap>
            <Select
              placeholder="状态"
              allowClear
              style={{ width: 110 }}
              options={Object.entries(statusLabels).map(([k, v]) => ({ value: k, label: v }))}
              value={filterStatus}
              onChange={(v) => { setFilterStatus(v); setPage(1); }}
            />
            <Select
              placeholder="结果"
              allowClear
              style={{ width: 110 }}
              options={[
                { value: 'passed', label: '合格' },
                { value: 'failed', label: '不合格' },
                { value: 'conditional', label: '条件放行' },
              ]}
              value={filterResult}
              onChange={(v) => { setFilterResult(v); setPage(1); }}
            />
            <Search
              placeholder="搜索"
              allowClear
              style={{ width: 180 }}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              onSearch={() => setPage(1)}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={() => queryClient.invalidateQueries({ queryKey: ['srm-inspections'] })}
            >
              刷新
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={inspections}
          rowKey="id"
          loading={isLoading}
          scroll={{ x: 950 }}
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
            onClick: () => { setCurrentInsp(record); setDrawerVisible(true); },
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      {drawerNode}
    </div>
  );
};

export default Inspections;
