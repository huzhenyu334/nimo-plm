import React, { useState } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Input,
  Select,
  Modal,
  Form,
  DatePicker,
  InputNumber,
  message,
  Drawer,
  Descriptions,
  Popconfirm,
  Spin,
} from 'antd';
import { PlusOutlined, ReloadOutlined, MinusCircleOutlined, SearchOutlined, RightOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { srmApi, PurchaseOrder, POItem } from '@/api/srm';
import dayjs from 'dayjs';
import { useIsMobile } from '@/hooks/useIsMobile';

const { Search } = Input;

const typeLabels: Record<string, string> = { sample: '打样', production: '量产' };
const typeColors: Record<string, string> = { sample: 'blue', production: 'green' };

const statusLabels: Record<string, string> = {
  draft: '草稿', pending: '待审批', approved: '已批准', rejected: '已拒绝',
  in_progress: '进行中', shipped: '已发货', received: '已收货',
  completed: '已完成', cancelled: '已取消',
};
const statusColors: Record<string, string> = {
  draft: 'default', pending: 'processing', approved: 'success', rejected: 'error',
  in_progress: 'processing', shipped: 'cyan', received: 'blue',
  completed: 'success', cancelled: 'default',
};

const PurchaseOrders: React.FC = () => {
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();
  const [searchText, setSearchText] = useState('');
  const [filterType, setFilterType] = useState<string>();
  const [filterStatus, setFilterStatus] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentPO, setCurrentPO] = useState<PurchaseOrder | null>(null);
  const [receiveQty, setReceiveQty] = useState<number>(0);
  const [form] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['srm-pos', searchText, filterType, filterStatus, page, pageSize],
    queryFn: () =>
      srmApi.listPOs({
        search: searchText || undefined,
        type: filterType,
        status: filterStatus,
        page,
        page_size: pageSize,
      }),
  });

  const { data: poDetail } = useQuery({
    queryKey: ['srm-po', currentPO?.id],
    queryFn: () => srmApi.getPO(currentPO!.id),
    enabled: !!currentPO?.id && drawerVisible,
  });

  const { data: supplierData } = useQuery({
    queryKey: ['srm-suppliers-select'],
    queryFn: () => srmApi.listSuppliers({ page_size: 100 }),
  });

  const createMutation = useMutation({
    mutationFn: (values: any) => {
      const payload = {
        ...values,
        expected_date: values.expected_date?.toISOString(),
      };
      return srmApi.createPO(payload);
    },
    onSuccess: () => {
      message.success('采购订单创建成功');
      setModalVisible(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['srm-pos'] });
    },
    onError: () => message.error('创建失败'),
  });

  const receiveMutation = useMutation({
    mutationFn: ({ itemId, qty }: { itemId: string; qty: number }) =>
      srmApi.receivePOItem(currentPO!.id, itemId, qty),
    onSuccess: () => {
      message.success('收货成功');
      queryClient.invalidateQueries({ queryKey: ['srm-po', currentPO?.id] });
      queryClient.invalidateQueries({ queryKey: ['srm-pos'] });
    },
    onError: () => message.error('收货失败'),
  });

  const columns = [
    {
      title: 'PO编码',
      dataIndex: 'po_code',
      key: 'po_code',
      width: 140,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    {
      title: '供应商',
      dataIndex: 'supplier',
      key: 'supplier',
      width: 160,
      ellipsis: true,
      render: (s: any) => s?.name || '-',
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 80,
      render: (t: string) => <Tag color={typeColors[t]}>{typeLabels[t] || t}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (s: string) => <Tag color={statusColors[s]}>{statusLabels[s] || s}</Tag>,
    },
    {
      title: '总金额',
      dataIndex: 'total_amount',
      key: 'total_amount',
      width: 110,
      render: (a?: number) => (a != null ? `¥${a.toFixed(2)}` : '-'),
    },
    {
      title: '预计交期',
      dataIndex: 'expected_date',
      key: 'expected_date',
      width: 110,
      render: (d: string) => (d ? dayjs(d).format('YYYY-MM-DD') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: PurchaseOrder) => (
        <Button
          type="link"
          size="small"
          onClick={() => { setCurrentPO(record); setDrawerVisible(true); }}
        >
          详情
        </Button>
      ),
    },
  ];

  const itemColumns = [
    {
      title: '物料编码',
      dataIndex: 'material_code',
      key: 'material_code',
      width: 120,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text || '-'}</span>,
    },
    { title: '名称', dataIndex: 'material_name', key: 'material_name', width: 150, ellipsis: true },
    { title: '规格', dataIndex: 'specification', key: 'specification', width: 120, ellipsis: true },
    { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 70 },
    {
      title: '单价',
      dataIndex: 'unit_price',
      key: 'unit_price',
      width: 90,
      render: (p?: number) => (p != null ? `¥${p.toFixed(2)}` : '-'),
    },
    {
      title: '已收货',
      dataIndex: 'received_qty',
      key: 'received_qty',
      width: 70,
      render: (qty: number) => qty || 0,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (s: string) => <Tag color={statusColors[s] || 'default'}>{s}</Tag>,
    },
    {
      title: '收货',
      key: 'receive',
      width: 140,
      render: (_: unknown, record: POItem) => {
        const remaining = record.quantity - (record.received_qty || 0);
        if (remaining <= 0) return <Tag color="success">已完成</Tag>;
        return (
          <Popconfirm
            title="确认收货"
            description={
              <InputNumber
                min={0.01}
                max={remaining}
                defaultValue={remaining}
                style={{ width: '100%' }}
                onChange={(v) => setReceiveQty(v || 0)}
                addonAfter={record.unit}
              />
            }
            onConfirm={() => receiveMutation.mutate({ itemId: record.id, qty: receiveQty || remaining })}
          >
            <Button type="link" size="small">
              收货 (余{remaining})
            </Button>
          </Popconfirm>
        );
      },
    },
  ];

  const detail = poDetail || currentPO;
  const orders = data?.items || [];

  // ========== Shared Modals/Drawers ==========
  const modalsAndDrawers = (
    <>
      {/* 新建订单 */}
      <Modal
        title="新建采购订单"
        open={modalVisible}
        onOk={() => form.validateFields().then((values) => createMutation.mutate(values))}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        confirmLoading={createMutation.isPending}
        width={680}
      >
        <Form form={form} layout="vertical" initialValues={{ type: 'sample', items: [{}] }}>
          <Form.Item name="supplier_id" label="供应商" rules={[{ required: true, message: '请选择供应商' }]}>
            <Select
              placeholder="选择供应商"
              showSearch
              optionFilterProp="label"
              options={(supplierData?.items || []).map((s) => ({ value: s.id, label: `${s.name} (${s.code})` }))}
            />
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="type" label="类型" rules={[{ required: true }]}>
              <Select style={{ width: 140 }} options={[{ value: 'sample', label: '打样' }, { value: 'production', label: '量产' }]} />
            </Form.Item>
            <Form.Item name="expected_date" label="预计交期">
              <DatePicker />
            </Form.Item>
          </Space>
          <Form.Item name="notes" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
          <h4>订单明细</h4>
          <Form.List name="items">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...rest }) => (
                  <Space key={key} align="baseline" style={{ display: 'flex', marginBottom: 8 }}>
                    <Form.Item {...rest} name={[name, 'material_name']} rules={[{ required: true, message: '物料名称' }]}>
                      <Input placeholder="物料名称" style={{ width: 140 }} />
                    </Form.Item>
                    <Form.Item {...rest} name={[name, 'specification']}>
                      <Input placeholder="规格" style={{ width: 120 }} />
                    </Form.Item>
                    <Form.Item {...rest} name={[name, 'quantity']} rules={[{ required: true, message: '数量' }]}>
                      <InputNumber placeholder="数量" min={0.01} style={{ width: 90 }} />
                    </Form.Item>
                    <Form.Item {...rest} name={[name, 'unit_price']}>
                      <InputNumber placeholder="单价" min={0} precision={2} style={{ width: 100 }} />
                    </Form.Item>
                    {fields.length > 1 && (
                      <MinusCircleOutlined onClick={() => remove(name)} style={{ color: '#ff4d4f' }} />
                    )}
                  </Space>
                ))}
                <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                  添加明细
                </Button>
              </>
            )}
          </Form.List>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title={detail?.po_code || '采购订单详情'}
        open={drawerVisible}
        onClose={() => { setDrawerVisible(false); setCurrentPO(null); }}
        width={isMobile ? '100%' : 760}
      >
        {detail && (
          <>
            <Descriptions column={isMobile ? 1 : 2} bordered size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label="PO编码">{detail.po_code}</Descriptions.Item>
              <Descriptions.Item label="供应商">{detail.supplier?.name || '-'}</Descriptions.Item>
              <Descriptions.Item label="类型">
                <Tag color={typeColors[detail.type]}>{typeLabels[detail.type] || detail.type}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={statusColors[detail.status]}>{statusLabels[detail.status] || detail.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="总金额">
                {detail.total_amount != null ? `¥${detail.total_amount.toFixed(2)}` : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="预计交期">
                {detail.expected_date ? dayjs(detail.expected_date).format('YYYY-MM-DD') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="付款条件">{detail.payment_terms || '-'}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{dayjs(detail.created_at).format('YYYY-MM-DD')}</Descriptions.Item>
              <Descriptions.Item label="备注" span={isMobile ? 1 : 2}>{detail.notes || '-'}</Descriptions.Item>
            </Descriptions>
            <h4>订单明细</h4>
            <Table
              columns={itemColumns}
              dataSource={(poDetail as PurchaseOrder)?.items || []}
              rowKey="id"
              size="small"
              scroll={{ x: 800 }}
              pagination={false}
            />
          </>
        )}
      </Drawer>
    </>
  );

  // ========== Mobile Layout ==========
  if (isMobile) {
    return (
      <div style={{ background: '#f5f5f5', minHeight: '100vh' }}>
        <div style={{ padding: '12px 16px', background: '#fff', position: 'sticky', top: 0, zIndex: 10 }}>
          <Input
            placeholder="搜索PO编码/供应商"
            prefix={<SearchOutlined style={{ color: '#bbb' }} />}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            onPressEnter={() => setPage(1)}
            allowClear
            style={{ borderRadius: 20 }}
          />
        </div>
        <div className="mobile-filter-pills" style={{ padding: '8px 12px' }}>
          {[{ label: '全部', value: undefined as string | undefined }, ...Object.entries(statusLabels).map(([k, v]) => ({ label: v, value: k as string | undefined }))].map(opt => (
            <div
              key={opt.value || 'all'}
              className={`mobile-filter-pill ${filterStatus === opt.value ? 'active' : ''}`}
              onClick={() => { setFilterStatus(opt.value); setPage(1); }}
            >{opt.label}</div>
          ))}
        </div>
        <div style={{ padding: '0 12px' }}>
          {isLoading ? (
            <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载中..." /></div>
          ) : orders.map(record => (
            <div
              key={record.id}
              onClick={() => { setCurrentPO(record); setDrawerVisible(true); }}
              style={{ background: '#fff', borderRadius: 10, padding: '12px 14px', marginBottom: 8, boxShadow: '0 1px 3px rgba(0,0,0,0.04)', cursor: 'pointer' }}
            >
              <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
                <span style={{ fontFamily: 'monospace', color: '#1677ff', fontSize: 14, flex: 1 }}>{record.po_code}</span>
                <Tag color={statusColors[record.status]} style={{ margin: 0 }}>{statusLabels[record.status] || record.status}</Tag>
              </div>
              <div style={{ fontWeight: 600, fontSize: 15, marginBottom: 6, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {record.supplier?.name || '-'}
              </div>
              <div style={{ display: 'flex', alignItems: 'center', fontSize: 13, color: '#666', gap: 8 }}>
                <Tag color={typeColors[record.type]} style={{ margin: 0, fontSize: 11 }}>{typeLabels[record.type] || record.type}</Tag>
                {record.expected_date && (
                  <span>{dayjs(record.expected_date).format('YYYY-MM-DD')}</span>
                )}
                <span style={{ marginLeft: 'auto', fontWeight: 700, color: '#cf1322', fontSize: 14 }}>
                  {record.total_amount != null ? `¥${record.total_amount.toFixed(2)}` : ''}
                </span>
                <RightOutlined style={{ fontSize: 10, color: '#ccc' }} />
              </div>
            </div>
          ))}
          {orders.length === 0 && !isLoading && (
            <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>暂无采购订单</div>
          )}
        </div>
        <div
          onClick={() => setModalVisible(true)}
          style={{ position: 'fixed', bottom: 80, right: 20, width: 52, height: 52, borderRadius: 26, background: '#1677ff', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 4px 12px rgba(22,119,255,0.4)', zIndex: 100, fontSize: 22, cursor: 'pointer' }}
        >
          <PlusOutlined />
        </div>
        {modalsAndDrawers}
      </div>
    );
  }

  // ========== Desktop Layout ==========
  return (
    <div>
      <Card
        title="采购订单"
        extra={
          <Space wrap>
            <Select
              placeholder="类型"
              allowClear
              style={{ width: 100 }}
              options={[{ value: 'sample', label: '打样' }, { value: 'production', label: '量产' }]}
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
              placeholder="搜索"
              allowClear
              style={{ width: 180 }}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              onSearch={() => setPage(1)}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={() => queryClient.invalidateQueries({ queryKey: ['srm-pos'] })}
            >
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
              新建订单
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={orders}
          rowKey="id"
          loading={isLoading}
          scroll={{ x: 800 }}
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
            onClick: () => { setCurrentPO(record); setDrawerVisible(true); },
            style: { cursor: 'pointer' },
          })}
        />
      </Card>
      {modalsAndDrawers}
    </div>
  );
};

export default PurchaseOrders;
