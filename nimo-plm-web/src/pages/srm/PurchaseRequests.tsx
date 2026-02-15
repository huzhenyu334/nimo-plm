import React, { useState, useMemo } from 'react';
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
  Spin,
} from 'antd';
import { PlusOutlined, ReloadOutlined, ImportOutlined, ShoppingCartOutlined, SearchOutlined, RightOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { srmApi, PurchaseRequest, PRItem } from '@/api/srm';
import { projectApi } from '@/api/projects';
import { projectBomApi } from '@/api/projectBom';
import { useIsMobile } from '@/hooks/useIsMobile';
import dayjs from 'dayjs';

const { Search } = Input;

const priorityLabels: Record<string, string> = { low: '低', medium: '中', high: '高', urgent: '紧急' };
const priorityColors: Record<string, string> = { low: 'default', medium: 'blue', high: 'orange', urgent: 'red' };

const statusLabels: Record<string, string> = {
  draft: '草稿', pending: '待审批', approved: '已批准', rejected: '已拒绝',
  sourcing: '寻源中', in_progress: '进行中', completed: '已完成', cancelled: '已取消',
};
const statusColors: Record<string, string> = {
  draft: 'default', pending: 'processing', approved: 'success', rejected: 'error',
  sourcing: 'cyan', in_progress: 'processing', completed: 'success', cancelled: 'default',
};

const itemStatusLabels: Record<string, string> = {
  pending: '待处理', sourcing: '寻源中', ordered: '已下单', received: '已收货',
  inspecting: '检验中', passed: '已通过', failed: '未通过',
};
const itemStatusColors: Record<string, string> = {
  pending: 'default', sourcing: 'cyan', ordered: 'processing', received: 'blue',
  inspecting: 'orange', passed: 'success', failed: 'error',
};

const PurchaseRequests: React.FC = () => {
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();
  const [searchText, setSearchText] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [bomModalVisible, setBomModalVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentPR, setCurrentPR] = useState<PurchaseRequest | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<string>();
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [assigningItem, setAssigningItem] = useState<PRItem | null>(null);
  const [form] = Form.useForm();
  const [bomForm] = Form.useForm();
  const [assignForm] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['srm-prs', searchText, filterStatus, page, pageSize],
    queryFn: () =>
      srmApi.listPRs({
        search: searchText || undefined,
        status: filterStatus,
        page,
        page_size: pageSize,
      }),
  });

  const { data: prDetail } = useQuery({
    queryKey: ['srm-pr', currentPR?.id],
    queryFn: () => srmApi.getPR(currentPR!.id),
    enabled: !!currentPR?.id && drawerVisible,
  });

  const { data: projectData } = useQuery({
    queryKey: ['projects'],
    queryFn: () => projectApi.list({ page_size: 100 }),
  });

  const { data: bomData } = useQuery({
    queryKey: ['project-boms', selectedProjectId],
    queryFn: () => projectBomApi.list(selectedProjectId!),
    enabled: !!selectedProjectId,
  });

  const { data: supplierData } = useQuery({
    queryKey: ['srm-suppliers-select'],
    queryFn: () => srmApi.listSuppliers({ page_size: 200 }),
  });

  // 供应商ID→名称映射
  const supplierMap = useMemo(() => {
    const map: Record<string, string> = {};
    (supplierData?.items || []).forEach((s) => { map[s.id] = s.name; });
    return map;
  }, [supplierData]);

  const createMutation = useMutation({
    mutationFn: (values: any) => {
      const payload = {
        ...values,
        required_date: values.required_date?.toISOString(),
      };
      return srmApi.createPR(payload);
    },
    onSuccess: () => {
      message.success('采购需求创建成功');
      setModalVisible(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['srm-prs'] });
    },
    onError: () => message.error('创建失败'),
  });

  const createFromBOMMutation = useMutation({
    mutationFn: (values: { project_id: string; bom_id: string }) => srmApi.createPRFromBOM(values),
    onSuccess: () => {
      message.success('从BOM创建采购需求成功');
      setBomModalVisible(false);
      bomForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ['srm-prs'] });
    },
    onError: () => message.error('创建失败'),
  });

  const assignSupplierMutation = useMutation({
    mutationFn: (values: { supplier_id: string; unit_price?: number; expected_date?: string }) =>
      srmApi.assignSupplier(currentPR!.id, assigningItem!.id, values),
    onSuccess: () => {
      message.success('供应商分配成功');
      setAssignModalVisible(false);
      assignForm.resetFields();
      setAssigningItem(null);
      queryClient.invalidateQueries({ queryKey: ['srm-pr', currentPR?.id] });
    },
    onError: () => message.error('分配失败'),
  });

  const generatePOsMutation = useMutation({
    mutationFn: () => srmApi.generatePOs(currentPR!.id),
    onSuccess: (pos) => {
      message.success(`成功生成 ${pos.length} 张采购订单`);
      queryClient.invalidateQueries({ queryKey: ['srm-pr', currentPR?.id] });
      queryClient.invalidateQueries({ queryKey: ['srm-prs'] });
      queryClient.invalidateQueries({ queryKey: ['srm-pos'] });
    },
    onError: () => message.error('生成采购订单失败'),
  });

  // 统计已分配供应商的行项
  const assignedItems = useMemo(() => {
    const items = (prDetail as PurchaseRequest)?.items || [];
    return items.filter((item) => item.supplier_id);
  }, [prDetail]);

  const supplierCount = useMemo(() => {
    const ids = new Set(assignedItems.map((item) => item.supplier_id));
    return ids.size;
  }, [assignedItems]);

  const handleGeneratePOs = () => {
    Modal.confirm({
      title: '生成采购订单',
      content: `将为 ${supplierCount} 个供应商生成 ${supplierCount} 张采购订单，确认？`,
      okText: '确认生成',
      cancelText: '取消',
      onOk: () => generatePOsMutation.mutate(),
    });
  };

  const columns = [
    {
      title: 'PR编码',
      dataIndex: 'pr_code',
      key: 'pr_code',
      width: 140,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    { title: '标题', dataIndex: 'title', key: 'title', width: 200, ellipsis: true },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      render: (p: string) => <Tag color={priorityColors[p]}>{priorityLabels[p] || p}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (s: string) => <Tag color={statusColors[s]}>{statusLabels[s] || s}</Tag>,
    },
    {
      title: '关联项目',
      dataIndex: 'project_id',
      key: 'project_id',
      width: 120,
      render: (pid: string) => {
        if (!pid) return '-';
        const proj = projectData?.items?.find((p: any) => p.id === pid);
        return proj?.name || pid.slice(0, 8);
      },
    },
    { title: '阶段', dataIndex: 'phase', key: 'phase', width: 80, render: (p: string) => p?.toUpperCase() || '-' },
    {
      title: '需求日期',
      dataIndex: 'required_date',
      key: 'required_date',
      width: 110,
      render: (d: string) => (d ? dayjs(d).format('YYYY-MM-DD') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: PurchaseRequest) => (
        <Button
          type="link"
          size="small"
          onClick={() => {
            setCurrentPR(record);
            setDrawerVisible(true);
          }}
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
    { title: '名称', dataIndex: 'material_name', key: 'material_name', width: 140, ellipsis: true },
    { title: '规格', dataIndex: 'specification', key: 'specification', width: 100, ellipsis: true },
    { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 60 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (s: string) => <Tag color={itemStatusColors[s]}>{itemStatusLabels[s] || s}</Tag>,
    },
    {
      title: '供应商',
      dataIndex: 'supplier_id',
      key: 'supplier_id',
      width: 120,
      render: (id: string) => (id ? (supplierMap[id] || id.slice(0, 8)) : '-'),
    },
    {
      title: '单价',
      dataIndex: 'unit_price',
      key: 'unit_price',
      width: 80,
      render: (p?: number) => (p != null ? `¥${p.toFixed(2)}` : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: PRItem) => (
        <Button
          type="link"
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            setAssigningItem(record);
            assignForm.setFieldsValue({
              supplier_id: record.supplier_id || undefined,
              unit_price: record.unit_price,
              expected_date: record.expected_date ? dayjs(record.expected_date) : undefined,
            });
            setAssignModalVisible(true);
          }}
        >
          分配供应商
        </Button>
      ),
    },
  ];

  const detail = prDetail || currentPR;
  const prItems = data?.items || [];

  // ========== Mobile Layout ==========
  if (isMobile) {
    return (
      <div style={{ background: '#f5f5f5', minHeight: '100vh' }}>
        {/* Sticky search bar */}
        <div style={{ padding: '12px 16px', background: '#fff', position: 'sticky', top: 0, zIndex: 10 }}>
          <Input
            placeholder="搜索采购需求编码/标题"
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
          {[{ label: '全部', value: undefined as string | undefined }, ...Object.entries(statusLabels).map(([k, v]) => ({ value: k, label: v }))].map(opt => (
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
            <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
          ) : prItems.map(pr => {
            const proj = projectData?.items?.find((p: any) => p.id === pr.project_id);
            return (
              <div
                key={pr.id}
                onClick={() => { setCurrentPR(pr); setDrawerVisible(true); }}
                style={{ background: '#fff', borderRadius: 10, padding: '12px 14px', marginBottom: 8, boxShadow: '0 1px 3px rgba(0,0,0,0.04)', cursor: 'pointer' }}
              >
                <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
                  <span style={{ fontWeight: 600, fontSize: 15, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{pr.title}</span>
                  <Tag color={statusColors[pr.status]} style={{ margin: 0, fontSize: 11 }}>{statusLabels[pr.status] || pr.status}</Tag>
                </div>
                <div style={{ fontSize: 12, fontFamily: 'monospace', color: '#1677ff', marginBottom: 6 }}>{pr.pr_code}</div>
                <div style={{ display: 'flex', alignItems: 'center', fontSize: 13, color: '#666', gap: 8, flexWrap: 'wrap' }}>
                  <Tag color={priorityColors[pr.priority]} style={{ margin: 0, fontSize: 11 }}>{priorityLabels[pr.priority] || pr.priority}</Tag>
                  {proj && <span style={{ fontSize: 12 }}>{proj.name}</span>}
                  {pr.required_date && <span style={{ marginLeft: 'auto', fontSize: 12, color: '#999' }}>{dayjs(pr.required_date).format('YYYY-MM-DD')}</span>}
                  <RightOutlined style={{ fontSize: 10, color: '#ccc' }} />
                </div>
              </div>
            );
          })}
          {prItems.length === 0 && !isLoading && (
            <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>暂无采购需求</div>
          )}
        </div>

        {/* FAB for create */}
        <div
          onClick={() => setModalVisible(true)}
          style={{ position: 'fixed', bottom: 80, right: 20, width: 52, height: 52, borderRadius: 26, background: '#1677ff', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 4px 12px rgba(22,119,255,0.4)', zIndex: 100, fontSize: 22, cursor: 'pointer' }}
        >
          <PlusOutlined />
        </div>

        {/* Reuse existing Modal/Drawer/Form components */}
        <Modal
          title="新建采购需求"
          open={modalVisible}
          onOk={() => form.validateFields().then((values) => createMutation.mutate(values))}
          onCancel={() => { setModalVisible(false); form.resetFields(); }}
          confirmLoading={createMutation.isPending}
          width={520}
        >
          <Form form={form} layout="vertical" initialValues={{ priority: 'medium' }}>
            <Form.Item name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]}>
              <Input placeholder="采购需求标题" />
            </Form.Item>
            <Form.Item name="priority" label="优先级">
              <Select style={{ width: 140 }} options={Object.entries(priorityLabels).map(([k, v]) => ({ value: k, label: v }))} />
            </Form.Item>
            <Form.Item name="project_id" label="关联项目">
              <Select
                placeholder="选择项目（可选）"
                allowClear
                showSearch
                optionFilterProp="label"
                options={(projectData?.items || []).map((p: any) => ({ value: p.id, label: p.name }))}
              />
            </Form.Item>
            <Form.Item name="required_date" label="需求日期">
              <DatePicker style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item name="notes" label="备注">
              <Input.TextArea rows={2} placeholder="备注信息" />
            </Form.Item>
          </Form>
        </Modal>

        <Modal
          title="从BOM创建采购需求"
          open={bomModalVisible}
          onOk={() => bomForm.validateFields().then((values) => createFromBOMMutation.mutate(values))}
          onCancel={() => { setBomModalVisible(false); bomForm.resetFields(); setSelectedProjectId(undefined); }}
          confirmLoading={createFromBOMMutation.isPending}
          width={480}
        >
          <Form form={bomForm} layout="vertical">
            <Form.Item name="project_id" label="选择项目" rules={[{ required: true, message: '请选择项目' }]}>
              <Select
                placeholder="选择项目"
                showSearch
                optionFilterProp="label"
                options={(projectData?.items || []).map((p: any) => ({ value: p.id, label: p.name }))}
                onChange={(v) => { setSelectedProjectId(v); bomForm.setFieldValue('bom_id', undefined); }}
              />
            </Form.Item>
            <Form.Item name="bom_id" label="选择BOM" rules={[{ required: true, message: '请选择BOM' }]}>
              <Select
                placeholder={selectedProjectId ? '选择BOM' : '请先选择项目'}
                disabled={!selectedProjectId}
                options={(bomData || []).map((b: any) => ({ value: b.id, label: `${b.name || 'BOM'} - v${b.version || '1'}` }))}
              />
            </Form.Item>
          </Form>
        </Modal>

        <Modal
          title={`分配供应商 - ${assigningItem?.material_name || ''}`}
          open={assignModalVisible}
          onOk={() =>
            assignForm.validateFields().then((values) => {
              const payload = {
                ...values,
                expected_date: values.expected_date?.toISOString(),
              };
              assignSupplierMutation.mutate(payload);
            })
          }
          onCancel={() => { setAssignModalVisible(false); assignForm.resetFields(); setAssigningItem(null); }}
          confirmLoading={assignSupplierMutation.isPending}
          width={480}
        >
          <Form form={assignForm} layout="vertical">
            <Form.Item name="supplier_id" label="供应商" rules={[{ required: true, message: '请选择供应商' }]}>
              <Select
                placeholder="选择供应商"
                showSearch
                optionFilterProp="label"
                options={(supplierData?.items || []).map((s) => ({ value: s.id, label: `${s.name} (${s.code})` }))}
              />
            </Form.Item>
            <Form.Item name="unit_price" label="单价">
              <InputNumber placeholder="单价" min={0} precision={4} style={{ width: '100%' }} addonBefore="¥" />
            </Form.Item>
            <Form.Item name="expected_date" label="预计交期">
              <DatePicker style={{ width: '100%' }} />
            </Form.Item>
          </Form>
        </Modal>

        <Drawer
          title={
            <Space>
              <span>{detail?.pr_code || '采购需求详情'}</span>
              {detail && assignedItems.length > 0 && (
                <Button
                  type="primary"
                  icon={<ShoppingCartOutlined />}
                  size="small"
                  loading={generatePOsMutation.isPending}
                  onClick={handleGeneratePOs}
                >
                  生成采购订单
                </Button>
              )}
            </Space>
          }
          open={drawerVisible}
          onClose={() => { setDrawerVisible(false); setCurrentPR(null); }}
          width="100%"
        >
          {detail && (
            <>
              <Descriptions column={1} bordered size="small" style={{ marginBottom: 24 }}>
                <Descriptions.Item label="PR编码">{detail.pr_code}</Descriptions.Item>
                <Descriptions.Item label="标题">{detail.title}</Descriptions.Item>
                <Descriptions.Item label="优先级">
                  <Tag color={priorityColors[detail.priority]}>{priorityLabels[detail.priority] || detail.priority}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Tag color={statusColors[detail.status]}>{statusLabels[detail.status] || detail.status}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="阶段">{detail.phase?.toUpperCase() || '-'}</Descriptions.Item>
                <Descriptions.Item label="需求日期">
                  {detail.required_date ? dayjs(detail.required_date).format('YYYY-MM-DD') : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="创建时间">{dayjs(detail.created_at).format('YYYY-MM-DD')}</Descriptions.Item>
                <Descriptions.Item label="备注">{detail.notes || '-'}</Descriptions.Item>
              </Descriptions>
              <h4>需求明细</h4>
              <Table
                columns={itemColumns}
                dataSource={(prDetail as PurchaseRequest)?.items || []}
                rowKey="id"
                size="small"
                scroll={{ x: 800 }}
                pagination={false}
              />
            </>
          )}
        </Drawer>
      </div>
    );
  }

  // ========== Desktop Layout ==========
  return (
    <div>
      <Card
        title="采购需求"
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
              onClick={() => queryClient.invalidateQueries({ queryKey: ['srm-prs'] })}
            >
              刷新
            </Button>
            <Button icon={<ImportOutlined />} onClick={() => setBomModalVisible(true)}>
              从BOM创建
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
              新建需求
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
            onClick: () => { setCurrentPR(record); setDrawerVisible(true); },
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      {/* 新建需求 */}
      <Modal
        title="新建采购需求"
        open={modalVisible}
        onOk={() => form.validateFields().then((values) => createMutation.mutate(values))}
        onCancel={() => { setModalVisible(false); form.resetFields(); }}
        confirmLoading={createMutation.isPending}
        width={520}
      >
        <Form form={form} layout="vertical" initialValues={{ priority: 'medium' }}>
          <Form.Item name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]}>
            <Input placeholder="采购需求标题" />
          </Form.Item>
          <Form.Item name="priority" label="优先级">
            <Select style={{ width: 140 }} options={Object.entries(priorityLabels).map(([k, v]) => ({ value: k, label: v }))} />
          </Form.Item>
          <Form.Item name="project_id" label="关联项目">
            <Select
              placeholder="选择项目（可选）"
              allowClear
              showSearch
              optionFilterProp="label"
              options={(projectData?.items || []).map((p: any) => ({ value: p.id, label: p.name }))}
            />
          </Form.Item>
          <Form.Item name="required_date" label="需求日期">
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="notes" label="备注">
            <Input.TextArea rows={2} placeholder="备注信息" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 从BOM创建 */}
      <Modal
        title="从BOM创建采购需求"
        open={bomModalVisible}
        onOk={() => bomForm.validateFields().then((values) => createFromBOMMutation.mutate(values))}
        onCancel={() => { setBomModalVisible(false); bomForm.resetFields(); setSelectedProjectId(undefined); }}
        confirmLoading={createFromBOMMutation.isPending}
        width={480}
      >
        <Form form={bomForm} layout="vertical">
          <Form.Item name="project_id" label="选择项目" rules={[{ required: true, message: '请选择项目' }]}>
            <Select
              placeholder="选择项目"
              showSearch
              optionFilterProp="label"
              options={(projectData?.items || []).map((p: any) => ({ value: p.id, label: p.name }))}
              onChange={(v) => { setSelectedProjectId(v); bomForm.setFieldValue('bom_id', undefined); }}
            />
          </Form.Item>
          <Form.Item name="bom_id" label="选择BOM" rules={[{ required: true, message: '请选择BOM' }]}>
            <Select
              placeholder={selectedProjectId ? '选择BOM' : '请先选择项目'}
              disabled={!selectedProjectId}
              options={(bomData || []).map((b: any) => ({ value: b.id, label: `${b.name || 'BOM'} - v${b.version || '1'}` }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 分配供应商 */}
      <Modal
        title={`分配供应商 - ${assigningItem?.material_name || ''}`}
        open={assignModalVisible}
        onOk={() =>
          assignForm.validateFields().then((values) => {
            const payload = {
              ...values,
              expected_date: values.expected_date?.toISOString(),
            };
            assignSupplierMutation.mutate(payload);
          })
        }
        onCancel={() => { setAssignModalVisible(false); assignForm.resetFields(); setAssigningItem(null); }}
        confirmLoading={assignSupplierMutation.isPending}
        width={480}
      >
        <Form form={assignForm} layout="vertical">
          <Form.Item name="supplier_id" label="供应商" rules={[{ required: true, message: '请选择供应商' }]}>
            <Select
              placeholder="选择供应商"
              showSearch
              optionFilterProp="label"
              options={(supplierData?.items || []).map((s) => ({ value: s.id, label: `${s.name} (${s.code})` }))}
            />
          </Form.Item>
          <Form.Item name="unit_price" label="单价">
            <InputNumber placeholder="单价" min={0} precision={4} style={{ width: '100%' }} addonBefore="¥" />
          </Form.Item>
          <Form.Item name="expected_date" label="预计交期">
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title={
          <Space>
            <span>{detail?.pr_code || '采购需求详情'}</span>
            {detail && assignedItems.length > 0 && (
              <Button
                type="primary"
                icon={<ShoppingCartOutlined />}
                size="small"
                loading={generatePOsMutation.isPending}
                onClick={handleGeneratePOs}
              >
                生成采购订单
              </Button>
            )}
          </Space>
        }
        open={drawerVisible}
        onClose={() => { setDrawerVisible(false); setCurrentPR(null); }}
        width={820}
      >
        {detail && (
          <>
            <Descriptions column={2} bordered size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label="PR编码">{detail.pr_code}</Descriptions.Item>
              <Descriptions.Item label="标题">{detail.title}</Descriptions.Item>
              <Descriptions.Item label="优先级">
                <Tag color={priorityColors[detail.priority]}>{priorityLabels[detail.priority] || detail.priority}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={statusColors[detail.status]}>{statusLabels[detail.status] || detail.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="阶段">{detail.phase?.toUpperCase() || '-'}</Descriptions.Item>
              <Descriptions.Item label="需求日期">
                {detail.required_date ? dayjs(detail.required_date).format('YYYY-MM-DD') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">{dayjs(detail.created_at).format('YYYY-MM-DD')}</Descriptions.Item>
              <Descriptions.Item label="备注" span={2}>{detail.notes || '-'}</Descriptions.Item>
            </Descriptions>
            <h4>需求明细</h4>
            <Table
              columns={itemColumns}
              dataSource={(prDetail as PurchaseRequest)?.items || []}
              rowKey="id"
              size="small"
              scroll={{ x: 800 }}
              pagination={false}
            />
          </>
        )}
      </Drawer>
    </div>
  );
};

export default PurchaseRequests;
