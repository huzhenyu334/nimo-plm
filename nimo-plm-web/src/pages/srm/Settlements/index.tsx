import { useState } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Badge,
  Select,
  Modal,
  Form,
  Input,
  DatePicker,
  Drawer,
  Descriptions,
  Popconfirm,
  InputNumber,
  App,
} from 'antd';
import { PlusOutlined, ReloadOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { srmApi, Settlement, SettlementDispute } from '@/api/srm';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;
const { TextArea } = Input;

const statusMap: Record<string, { text: string; color: string; status: 'default' | 'processing' | 'success' | 'warning' | 'error' }> = {
  draft: { text: '草稿', color: 'default', status: 'default' },
  confirmed: { text: '已确认', color: 'blue', status: 'processing' },
  invoiced: { text: '已开票', color: 'orange', status: 'warning' },
  paid: { text: '已付款', color: 'green', status: 'success' },
};

const disputeTypeLabels: Record<string, string> = {
  price_diff: '价格差异',
  quantity_diff: '数量差异',
  quality_deduction: '质量扣款',
  other: '其他',
};

const disputeTypeOptions = [
  { value: 'price_diff', label: '价格差异' },
  { value: 'quantity_diff', label: '数量差异' },
  { value: 'quality_deduction', label: '质量扣款' },
  { value: 'other', label: '其他' },
];

export default function Settlements() {
  const { message: messageApi, modal } = App.useApp();
  const queryClient = useQueryClient();
  const [filterSupplier, setFilterSupplier] = useState<string>();
  const [filterStatus, setFilterStatus] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [generateVisible, setGenerateVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentSettlement, setCurrentSettlement] = useState<Settlement | null>(null);
  const [disputeVisible, setDisputeVisible] = useState(false);
  const [form] = Form.useForm();
  const [genForm] = Form.useForm();
  const [disputeForm] = Form.useForm();

  // 供应商列表（用于选择）
  const { data: suppliers } = useQuery({
    queryKey: ['srm-suppliers-all'],
    queryFn: () => srmApi.listSuppliers({ page_size: 100 }),
    staleTime: 60_000,
  });

  // 对账单列表
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['srm-settlements', filterSupplier, filterStatus, page, pageSize],
    queryFn: () =>
      srmApi.listSettlements({
        supplier_id: filterSupplier,
        status: filterStatus,
        page,
        page_size: pageSize,
      }),
    staleTime: 30_000,
  });

  // 对账单详情
  const { data: detail, refetch: refetchDetail } = useQuery({
    queryKey: ['srm-settlement', currentSettlement?.id],
    queryFn: () => srmApi.getSettlement(currentSettlement!.id),
    enabled: !!currentSettlement?.id && drawerVisible,
    staleTime: 60_000,
  });

  const createMutation = useMutation({
    mutationFn: (values: { supplier_id: string; period_start?: string; period_end?: string; notes?: string }) =>
      srmApi.createSettlement(values),
    onSuccess: () => {
      messageApi.success('对账单创建成功');
      setCreateVisible(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['srm-settlements'] });
    },
    onError: () => messageApi.error('创建失败'),
  });

  const generateMutation = useMutation({
    mutationFn: (values: { supplier_id: string; period_start: string; period_end: string }) =>
      srmApi.generateSettlement(values),
    onSuccess: () => {
      messageApi.success('对账单生成成功');
      setGenerateVisible(false);
      genForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ['srm-settlements'] });
    },
    onError: () => messageApi.error('生成失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => srmApi.deleteSettlement(id),
    onSuccess: () => {
      messageApi.success('已删除');
      queryClient.invalidateQueries({ queryKey: ['srm-settlements'] });
    },
    onError: () => messageApi.error('删除失败'),
  });

  const confirmBuyerMutation = useMutation({
    mutationFn: (id: string) => srmApi.confirmSettlementBuyer(id),
    onSuccess: () => {
      messageApi.success('采购方已确认');
      refetchDetail();
      queryClient.invalidateQueries({ queryKey: ['srm-settlements'] });
    },
    onError: () => messageApi.error('确认失败'),
  });

  const confirmSupplierMutation = useMutation({
    mutationFn: (id: string) => srmApi.confirmSettlementSupplier(id),
    onSuccess: () => {
      messageApi.success('供应商已确认');
      refetchDetail();
      queryClient.invalidateQueries({ queryKey: ['srm-settlements'] });
    },
    onError: () => messageApi.error('确认失败'),
  });

  const addDisputeMutation = useMutation({
    mutationFn: (values: { dispute_type: string; description?: string; amount_diff?: number }) =>
      srmApi.addSettlementDispute(currentSettlement!.id, values),
    onSuccess: () => {
      messageApi.success('差异记录已添加');
      setDisputeVisible(false);
      disputeForm.resetFields();
      refetchDetail();
    },
    onError: () => messageApi.error('添加失败'),
  });

  const resolveDisputeMutation = useMutation({
    mutationFn: ({ disputeId, resolution }: { disputeId: string; resolution: string }) =>
      srmApi.updateSettlementDispute(currentSettlement!.id, disputeId, { status: 'resolved', resolution }),
    onSuccess: () => {
      messageApi.success('差异已解决');
      refetchDetail();
    },
    onError: () => messageApi.error('操作失败'),
  });

  const supplierOptions = (suppliers?.items || []).map((s: { id: string; name: string }) => ({
    value: s.id,
    label: s.name,
  }));

  const formatAmount = (amount?: number) => {
    if (amount == null) return '-';
    return `¥${amount.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`;
  };

  const columns = [
    {
      title: '对账单号',
      dataIndex: 'settlement_code',
      key: 'settlement_code',
      width: 180,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    {
      title: '供应商',
      key: 'supplier',
      width: 160,
      ellipsis: true,
      render: (_: unknown, record: Settlement) => record.supplier?.name || '-',
    },
    {
      title: '对账周期',
      key: 'period',
      width: 200,
      render: (_: unknown, record: Settlement) => {
        const start = record.period_start ? dayjs(record.period_start).format('YYYY-MM-DD') : '';
        const end = record.period_end ? dayjs(record.period_end).format('YYYY-MM-DD') : '';
        return start && end ? `${start} ~ ${end}` : '-';
      },
    },
    {
      title: 'PO金额',
      dataIndex: 'po_amount',
      key: 'po_amount',
      width: 120,
      align: 'right' as const,
      render: (v?: number) => formatAmount(v),
    },
    {
      title: '扣款',
      dataIndex: 'deduction',
      key: 'deduction',
      width: 100,
      align: 'right' as const,
      render: (v?: number) => (v ? <span style={{ color: '#f5222d' }}>{formatAmount(v)}</span> : '-'),
    },
    {
      title: '结算金额',
      dataIndex: 'final_amount',
      key: 'final_amount',
      width: 120,
      align: 'right' as const,
      render: (v?: number) => <strong>{formatAmount(v)}</strong>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const s = statusMap[status] || { text: status, status: 'default' as const };
        return <Badge status={s.status} text={s.text} />;
      },
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 110,
      render: (t: string) => dayjs(t).format('YYYY-MM-DD'),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: Settlement) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            onClick={() => {
              setCurrentSettlement(record);
              setDrawerVisible(true);
            }}
          >
            详情
          </Button>
          {record.status === 'draft' && (
            <Popconfirm title="确认删除该对账单？" onConfirm={() => deleteMutation.mutate(record.id)}>
              <Button type="link" size="small" danger>
                删除
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const disputeColumns = [
    {
      title: '类型',
      dataIndex: 'dispute_type',
      key: 'dispute_type',
      width: 100,
      render: (t: string) => <Tag>{disputeTypeLabels[t] || t}</Tag>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '差异金额',
      dataIndex: 'amount_diff',
      key: 'amount_diff',
      width: 120,
      render: (v?: number) => (v != null ? formatAmount(v) : '-'),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (s: string) => (
        <Tag color={s === 'resolved' ? 'green' : 'orange'}>{s === 'resolved' ? '已解决' : '待处理'}</Tag>
      ),
    },
    {
      title: '解决方案',
      dataIndex: 'resolution',
      key: 'resolution',
      width: 160,
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: SettlementDispute) => {
        if (record.status === 'resolved') return null;
        return (
          <Button
            type="link"
            size="small"
            onClick={() => {
              modal.confirm({
                title: '解决差异',
                content: (
                  <Input.TextArea
                    id="resolve-input"
                    placeholder="请输入解决方案"
                    rows={3}
                  />
                ),
                onOk: () => {
                  const el = document.getElementById('resolve-input') as HTMLTextAreaElement;
                  resolveDisputeMutation.mutate({
                    disputeId: record.id,
                    resolution: el?.value || '',
                  });
                },
              });
            }}
          >
            解决
          </Button>
        );
      },
    },
  ];

  const detailData = detail || currentSettlement;

  return (
    <div>
      <Card
        title="对账结算"
        extra={
          <Space wrap>
            <Select
              placeholder="供应商"
              allowClear
              showSearch
              optionFilterProp="label"
              style={{ width: 160 }}
              options={supplierOptions}
              value={filterSupplier}
              onChange={(v) => { setFilterSupplier(v); setPage(1); }}
            />
            <Select
              placeholder="状态"
              allowClear
              style={{ width: 120 }}
              options={[
                { value: 'draft', label: '草稿' },
                { value: 'confirmed', label: '已确认' },
                { value: 'invoiced', label: '已开票' },
                { value: 'paid', label: '已付款' },
              ]}
              value={filterStatus}
              onChange={(v) => { setFilterStatus(v); setPage(1); }}
            />
            <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
              刷新
            </Button>
            <Button icon={<ThunderboltOutlined />} onClick={() => setGenerateVisible(true)}>
              自动生成
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
              新建对账单
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={data?.items || []}
          loading={isLoading}
          pagination={{
            current: page,
            pageSize,
            total: data?.pagination?.total || 0,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (p, ps) => { setPage(p); setPageSize(ps); },
          }}
          scroll={{ x: 1200 }}
          size="middle"
        />
      </Card>

      {/* 创建对账单 */}
      <Modal
        title="新建对账单"
        open={createVisible}
        onCancel={() => { setCreateVisible(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            const payload: { supplier_id: string; period_start?: string; period_end?: string; notes?: string } = {
              supplier_id: values.supplier_id,
              notes: values.notes,
            };
            if (values.period?.[0]) payload.period_start = values.period[0].format('YYYY-MM-DD');
            if (values.period?.[1]) payload.period_end = values.period[1].format('YYYY-MM-DD');
            createMutation.mutate(payload);
          }}
        >
          <Form.Item name="supplier_id" label="供应商" rules={[{ required: true, message: '请选择供应商' }]}>
            <Select placeholder="选择供应商" showSearch optionFilterProp="label" options={supplierOptions} />
          </Form.Item>
          <Form.Item name="period" label="对账周期">
            <RangePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="notes" label="备注">
            <TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 自动生成对账单 */}
      <Modal
        title="自动生成对账单"
        open={generateVisible}
        onCancel={() => { setGenerateVisible(false); genForm.resetFields(); }}
        onOk={() => genForm.submit()}
        confirmLoading={generateMutation.isPending}
      >
        <Form
          form={genForm}
          layout="vertical"
          onFinish={(values) => {
            generateMutation.mutate({
              supplier_id: values.supplier_id,
              period_start: values.period[0].format('YYYY-MM-DD'),
              period_end: values.period[1].format('YYYY-MM-DD'),
            });
          }}
        >
          <Form.Item name="supplier_id" label="供应商" rules={[{ required: true, message: '请选择供应商' }]}>
            <Select placeholder="选择供应商" showSearch optionFilterProp="label" options={supplierOptions} />
          </Form.Item>
          <Form.Item name="period" label="对账周期" rules={[{ required: true, message: '请选择对账周期' }]}>
            <RangePicker style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 对账单详情 */}
      <Drawer
        title={`对账单详情 - ${detailData?.settlement_code || ''}`}
        open={drawerVisible}
        onClose={() => { setDrawerVisible(false); setCurrentSettlement(null); }}
        width={720}
        extra={
          detailData?.status === 'draft' && (
            <Space>
              <Button onClick={() => confirmBuyerMutation.mutate(detailData!.id)}>
                采购方确认
              </Button>
              <Button onClick={() => confirmSupplierMutation.mutate(detailData!.id)}>
                供应商确认
              </Button>
            </Space>
          )
        }
      >
        {detailData && (
          <>
            <Descriptions column={2} bordered size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label="对账单号">{detailData.settlement_code}</Descriptions.Item>
              <Descriptions.Item label="供应商">{detailData.supplier?.name || '-'}</Descriptions.Item>
              <Descriptions.Item label="对账周期">
                {detailData.period_start && detailData.period_end
                  ? `${dayjs(detailData.period_start).format('YYYY-MM-DD')} ~ ${dayjs(detailData.period_end).format('YYYY-MM-DD')}`
                  : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Badge
                  status={statusMap[detailData.status]?.status || 'default'}
                  text={statusMap[detailData.status]?.text || detailData.status}
                />
              </Descriptions.Item>
              <Descriptions.Item label="PO金额">{formatAmount(detailData.po_amount)}</Descriptions.Item>
              <Descriptions.Item label="到货金额">{formatAmount(detailData.received_amount)}</Descriptions.Item>
              <Descriptions.Item label="扣款">{formatAmount(detailData.deduction)}</Descriptions.Item>
              <Descriptions.Item label="结算金额">
                <strong>{formatAmount(detailData.final_amount)}</strong>
              </Descriptions.Item>
              <Descriptions.Item label="发票号">{detailData.invoice_no || '-'}</Descriptions.Item>
              <Descriptions.Item label="发票金额">{formatAmount(detailData.invoice_amount)}</Descriptions.Item>
              <Descriptions.Item label="采购方确认">
                <Tag color={detailData.confirmed_by_buyer ? 'green' : 'default'}>
                  {detailData.confirmed_by_buyer ? '已确认' : '未确认'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="供应商确认">
                <Tag color={detailData.confirmed_by_supplier ? 'green' : 'default'}>
                  {detailData.confirmed_by_supplier ? '已确认' : '未确认'}
                </Tag>
              </Descriptions.Item>
              {detailData.notes && (
                <Descriptions.Item label="备注" span={2}>{detailData.notes}</Descriptions.Item>
              )}
            </Descriptions>

            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
              <h4 style={{ margin: 0 }}>差异记录</h4>
              <Button
                type="primary"
                size="small"
                icon={<PlusOutlined />}
                onClick={() => setDisputeVisible(true)}
              >
                添加差异
              </Button>
            </div>
            <Table
              rowKey="id"
              columns={disputeColumns}
              dataSource={detailData.disputes || []}
              pagination={false}
              size="small"
            />
          </>
        )}
      </Drawer>

      {/* 添加差异记录 */}
      <Modal
        title="添加差异记录"
        open={disputeVisible}
        onCancel={() => { setDisputeVisible(false); disputeForm.resetFields(); }}
        onOk={() => disputeForm.submit()}
        confirmLoading={addDisputeMutation.isPending}
      >
        <Form
          form={disputeForm}
          layout="vertical"
          onFinish={(values) => addDisputeMutation.mutate(values)}
        >
          <Form.Item name="dispute_type" label="差异类型" rules={[{ required: true, message: '请选择差异类型' }]}>
            <Select options={disputeTypeOptions} placeholder="选择差异类型" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={3} placeholder="请描述差异详情" />
          </Form.Item>
          <Form.Item name="amount_diff" label="差异金额">
            <InputNumber style={{ width: '100%' }} precision={2} placeholder="差异金额（元）" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
