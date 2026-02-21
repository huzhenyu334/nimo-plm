import React, { useState } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Badge,
  Input,
  Select,
  Modal,
  Form,
  message,
  Drawer,
  Descriptions,
  Tabs,
  Popconfirm,
} from 'antd';
import { PlusOutlined, ReloadOutlined, DeleteOutlined, SearchOutlined, RightOutlined, EditOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { srmApi, Supplier, SupplierContact } from '@/api/srm';
import { useIsMobile } from '@/hooks/useIsMobile';
import dayjs from 'dayjs';

const { Search } = Input;

const categoryOptions = [
  { value: 'structural', label: '结构件' },
  { value: 'electronic', label: '电子元器件' },
  { value: 'optical', label: '光学组件' },
  { value: 'packaging', label: '包装件' },
  { value: 'other', label: '其他' },
];

const categoryLabels: Record<string, string> = {
  structural: '结构件',
  electronic: '电子元器件',
  optical: '光学组件',
  packaging: '包装件',
  manufacturer: '制造商',
  other: '其他',
};

const categoryColors: Record<string, string> = {
  structural: 'blue',
  electronic: 'green',
  optical: 'purple',
  packaging: 'orange',
  manufacturer: 'cyan',
  other: 'default',
};

const levelOptions = [
  { value: 'potential', label: '潜在' },
  { value: 'qualified', label: '合格' },
  { value: 'preferred', label: '优选' },
  { value: 'strategic', label: '战略' },
];

const levelColors: Record<string, string> = {
  potential: 'default',
  qualified: 'blue',
  preferred: 'green',
  strategic: 'gold',
};

const levelLabels: Record<string, string> = {
  potential: '潜在',
  qualified: '合格',
  preferred: '优选',
  strategic: '战略',
};

const statusOptions = [
  { value: 'pending', label: '待审核' },
  { value: 'active', label: '已激活' },
  { value: 'suspended', label: '已暂停' },
  { value: 'blacklisted', label: '已拉黑' },
];

const statusMap: Record<string, { text: string; status: 'warning' | 'success' | 'error' | 'default' }> = {
  pending: { text: '待审核', status: 'warning' },
  active: { text: '已激活', status: 'success' },
  approved: { text: '已认证', status: 'success' },
  suspended: { text: '已暂停', status: 'error' },
  blacklisted: { text: '已拉黑', status: 'default' },
};

// Shared form fields for create/edit
const SupplierFormFields: React.FC<{ compact?: boolean }> = ({ compact }) => {
  if (compact) {
    return (
      <>
        <Form.Item name="name" label="供应商名称" rules={[{ required: true, message: '请输入供应商名称' }]}><Input placeholder="请输入供应商名称" /></Form.Item>
        <Form.Item name="short_name" label="简称"><Input placeholder="请输入简称" /></Form.Item>
        <Form.Item name="category" label="分类" rules={[{ required: true, message: '请选择分类' }]}><Select placeholder="请选择分类" options={categoryOptions} /></Form.Item>
        <Form.Item name="level" label="等级"><Select placeholder="请选择等级" options={levelOptions} allowClear /></Form.Item>
        <Form.Item name="city" label="城市"><Input placeholder="深圳" /></Form.Item>
        <Form.Item name="business_scope" label="业务范围"><Input.TextArea rows={2} placeholder="主营业务描述" /></Form.Item>
      </>
    );
  }
  return (
    <>
      <Form.Item name="name" label="供应商名称" rules={[{ required: true, message: '请输入供应商名称' }]}>
        <Input placeholder="请输入供应商名称" />
      </Form.Item>
      <Form.Item name="short_name" label="简称">
        <Input placeholder="请输入简称" />
      </Form.Item>
      <Form.Item name="category" label="分类" rules={[{ required: true, message: '请选择分类' }]}>
        <Select placeholder="请选择分类" options={categoryOptions} />
      </Form.Item>
      <Form.Item name="level" label="等级">
        <Select placeholder="请选择等级" options={levelOptions} allowClear />
      </Form.Item>
      <Space style={{ width: '100%' }} size="middle">
        <Form.Item name="country" label="国家" style={{ width: 180 }}>
          <Input placeholder="中国" />
        </Form.Item>
        <Form.Item name="city" label="城市" style={{ width: 180 }}>
          <Input placeholder="深圳" />
        </Form.Item>
      </Space>
      <Form.Item name="address" label="地址">
        <Input placeholder="详细地址" />
      </Form.Item>
      <Form.Item name="business_scope" label="业务范围">
        <Input.TextArea rows={2} placeholder="主营业务描述" />
      </Form.Item>
      <Form.Item name="payment_terms" label="付款条件">
        <Input placeholder="如: 月结30天" />
      </Form.Item>
    </>
  );
};

const Suppliers: React.FC = () => {
  const isMobile = useIsMobile();
  const queryClient = useQueryClient();
  const [searchText, setSearchText] = useState('');
  const [filterCategory, setFilterCategory] = useState<string>();
  const [filterLevel, setFilterLevel] = useState<string>();
  const [filterStatus, setFilterStatus] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentSupplier, setCurrentSupplier] = useState<Supplier | null>(null);
  const [editingSupplier, setEditingSupplier] = useState<Supplier | null>(null);
  const [contactForm] = Form.useForm();
  const [form] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['srm-suppliers', searchText, filterCategory, filterLevel, filterStatus, page, pageSize],
    queryFn: () =>
      srmApi.listSuppliers({
        search: searchText || undefined,
        category: filterCategory,
        level: filterLevel,
        status: filterStatus,
        page,
        page_size: pageSize,
      }),
  });

  const { data: supplierDetail } = useQuery({
    queryKey: ['srm-supplier', currentSupplier?.id],
    queryFn: () => srmApi.getSupplier(currentSupplier!.id),
    enabled: !!currentSupplier?.id && drawerVisible,
  });

  const { data: contacts, refetch: refetchContacts } = useQuery({
    queryKey: ['srm-supplier-contacts', currentSupplier?.id],
    queryFn: () => srmApi.listContacts(currentSupplier!.id),
    enabled: !!currentSupplier?.id && drawerVisible,
  });

  const createMutation = useMutation({
    mutationFn: (values: Partial<Supplier>) => srmApi.createSupplier(values),
    onSuccess: () => {
      message.success('供应商创建成功');
      closeModal();
      queryClient.invalidateQueries({ queryKey: ['srm-suppliers'] });
    },
    onError: () => message.error('创建失败'),
  });

  const updateMutation = useMutation({
    mutationFn: (values: Partial<Supplier>) => srmApi.updateSupplier(editingSupplier!.id, values),
    onSuccess: () => {
      message.success('供应商更新成功');
      closeModal();
      queryClient.invalidateQueries({ queryKey: ['srm-suppliers'] });
      queryClient.invalidateQueries({ queryKey: ['srm-supplier', editingSupplier!.id] });
    },
    onError: () => message.error('更新失败'),
  });

  const createContactMutation = useMutation({
    mutationFn: (values: Partial<SupplierContact>) => srmApi.createContact(currentSupplier!.id, values),
    onSuccess: () => {
      message.success('联系人添加成功');
      contactForm.resetFields();
      refetchContacts();
    },
    onError: () => message.error('添加失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => srmApi.deleteSupplier(id),
    onSuccess: () => {
      message.success('供应商已删除');
      setDrawerVisible(false);
      setCurrentSupplier(null);
      queryClient.invalidateQueries({ queryKey: ['srm-suppliers'] });
    },
    onError: () => message.error('删除失败'),
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => srmApi.updateSupplier(id, { status }),
    onSuccess: () => {
      message.success('状态更新成功');
      queryClient.invalidateQueries({ queryKey: ['srm-suppliers'] });
      if (currentSupplier) {
        queryClient.invalidateQueries({ queryKey: ['srm-supplier', currentSupplier.id] });
      }
    },
    onError: () => message.error('状态更新失败'),
  });

  const deleteContactMutation = useMutation({
    mutationFn: (contactId: string) => srmApi.deleteContact(currentSupplier!.id, contactId),
    onSuccess: () => {
      message.success('联系人已删除');
      refetchContacts();
    },
    onError: () => message.error('删除失败'),
  });

  const closeModal = () => {
    setModalVisible(false);
    setEditingSupplier(null);
    form.resetFields();
  };

  const handleEdit = (supplier: Supplier) => {
    setEditingSupplier(supplier);
    form.setFieldsValue(supplier);
    setDrawerVisible(false);
    setModalVisible(true);
  };

  const handleModalOk = () => {
    form.validateFields().then((values) => {
      if (editingSupplier) {
        updateMutation.mutate(values);
      } else {
        createMutation.mutate(values);
      }
    });
  };

  const columns = [
    {
      title: '编码',
      dataIndex: 'code',
      key: 'code',
      width: 130,
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
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 110,
      render: (cat: string) => (
        <Tag color={categoryColors[cat] || 'default'}>{categoryLabels[cat] || cat}</Tag>
      ),
    },
    {
      title: '等级',
      dataIndex: 'level',
      key: 'level',
      width: 80,
      render: (level: string) => (
        <Tag color={levelColors[level] || 'default'}>{levelLabels[level] || level}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: string) => {
        const s = statusMap[status] || { text: status, status: 'default' as const };
        return <Badge status={s.status} text={s.text} />;
      },
    },
    {
      title: '综合评分',
      dataIndex: 'overall_score',
      key: 'overall_score',
      width: 90,
      render: (score?: number) => (score != null ? score.toFixed(1) : '-'),
    },
    {
      title: '城市',
      dataIndex: 'city',
      key: 'city',
      width: 100,
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: Supplier) => (
        <Button
          type="link"
          size="small"
          onClick={() => {
            setCurrentSupplier(record);
            setDrawerVisible(true);
          }}
        >
          详情
        </Button>
      ),
    },
  ];

  const contactColumns = [
    { title: '姓名', dataIndex: 'name', key: 'name', width: 100 },
    { title: '职务', dataIndex: 'title', key: 'title', width: 100 },
    { title: '电话', dataIndex: 'phone', key: 'phone', width: 130 },
    { title: '邮箱', dataIndex: 'email', key: 'email', width: 180 },
    {
      title: '操作',
      key: 'action',
      width: 60,
      render: (_: unknown, record: SupplierContact) => (
        <Popconfirm title="确认删除该联系人？" onConfirm={() => deleteContactMutation.mutate(record.id)}>
          <Button type="link" size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  const detail = supplierDetail || currentSupplier;
  const suppliers = data?.items || [];

  // ========== Mobile Layout ==========
  if (isMobile) {
    return (
      <div style={{ background: '#f5f5f5', minHeight: '100vh' }}>
        <div style={{ padding: '12px 16px', background: '#fff', position: 'sticky', top: 0, zIndex: 10 }}>
          <Input
            placeholder="搜索供应商名称/编码"
            prefix={<SearchOutlined style={{ color: '#bbb' }} />}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            onPressEnter={() => setPage(1)}
            allowClear
            style={{ borderRadius: 20 }}
          />
        </div>
        <div className="mobile-filter-pills" style={{ padding: '8px 12px' }}>
          {[{ label: '全部', value: undefined }, ...categoryOptions].map(opt => (
            <div
              key={opt.value || 'all'}
              className={`mobile-filter-pill ${filterCategory === opt.value ? 'active' : ''}`}
              onClick={() => { setFilterCategory(opt.value); setPage(1); }}
            >{opt.label}</div>
          ))}
        </div>
        <div style={{ padding: '0 12px' }}>
          {isLoading ? (
            <div style={{ textAlign: 'center', padding: 40 }}><Badge status="processing" text="加载中..." /></div>
          ) : suppliers.map(s => {
            const st = statusMap[s.status] || { text: s.status, status: 'default' as const };
            return (
              <div key={s.id} onClick={() => { setCurrentSupplier(s); setDrawerVisible(true); }}
                style={{ background: '#fff', borderRadius: 10, padding: '12px 14px', marginBottom: 8, boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
                <div style={{ display: 'flex', alignItems: 'center', marginBottom: 4 }}>
                  <span style={{ fontWeight: 600, fontSize: 15, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{s.name}</span>
                  <Badge status={st.status} text={st.text} />
                </div>
                <div style={{ fontSize: 12, fontFamily: 'monospace', color: '#1677ff', marginBottom: 6 }}>{s.code}</div>
                <div style={{ display: 'flex', alignItems: 'center', fontSize: 13, color: '#666', gap: 8 }}>
                  <Tag color={categoryColors[s.category]} style={{ margin: 0, fontSize: 11 }}>{categoryLabels[s.category] || s.category}</Tag>
                  <Tag color={levelColors[s.level]} style={{ margin: 0, fontSize: 11 }}>{levelLabels[s.level] || s.level}</Tag>
                  {s.overall_score != null && <span style={{ marginLeft: 'auto', fontWeight: 500 }}>{s.overall_score.toFixed(1)}分</span>}
                  <RightOutlined style={{ fontSize: 10, color: '#ccc' }} />
                </div>
              </div>
            );
          })}
          {suppliers.length === 0 && !isLoading && (
            <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>暂无供应商</div>
          )}
        </div>
        <div onClick={() => setModalVisible(true)}
          style={{ position: 'fixed', bottom: 80, right: 20, width: 52, height: 52, borderRadius: 26, background: '#1677ff', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 4px 12px rgba(22,119,255,0.4)', zIndex: 100, fontSize: 22, cursor: 'pointer' }}>
          <PlusOutlined />
        </div>

        <Modal title={editingSupplier ? '编辑供应商' : '新建供应商'} open={modalVisible} onOk={handleModalOk} onCancel={closeModal} confirmLoading={editingSupplier ? updateMutation.isPending : createMutation.isPending} width={600}>
          <Form form={form} layout="vertical">
            <SupplierFormFields compact />
          </Form>
        </Modal>

        <Drawer
          title={detail?.name || '供应商详情'}
          open={drawerVisible}
          onClose={() => { setDrawerVisible(false); setCurrentSupplier(null); }}
          width="100%"
          extra={detail && (
            <Space>
              <Button type="primary" icon={<EditOutlined />} onClick={() => handleEdit(detail)}>编辑</Button>
              <Popconfirm title="确认删除该供应商？" onConfirm={() => deleteMutation.mutate(detail.id)}>
                <Button danger icon={<DeleteOutlined />} loading={deleteMutation.isPending} />
              </Popconfirm>
            </Space>
          )}
        >
          {detail && (
            <>
            <div style={{ marginBottom: 12 }}>
              <span style={{ marginRight: 8 }}>状态：</span>
              <Select
                value={detail.status}
                style={{ width: 110 }}
                loading={statusMutation.isPending}
                options={statusOptions}
                onChange={(v) => statusMutation.mutate({ id: detail.id, status: v })}
              />
            </div>
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="编码">{detail.code}</Descriptions.Item>
              <Descriptions.Item label="名称">{detail.name}</Descriptions.Item>
              <Descriptions.Item label="分类"><Tag color={categoryColors[detail.category]}>{categoryLabels[detail.category]}</Tag></Descriptions.Item>
              <Descriptions.Item label="等级"><Tag color={levelColors[detail.level]}>{levelLabels[detail.level]}</Tag></Descriptions.Item>
              <Descriptions.Item label="状态"><Badge status={statusMap[detail.status]?.status || 'default'} text={statusMap[detail.status]?.text || detail.status} /></Descriptions.Item>
              <Descriptions.Item label="综合评分">{detail.overall_score?.toFixed(1) || '-'}</Descriptions.Item>
              <Descriptions.Item label="城市">{detail.city || '-'}</Descriptions.Item>
              <Descriptions.Item label="业务范围">{detail.business_scope || '-'}</Descriptions.Item>
              <Descriptions.Item label="付款条件">{detail.payment_terms || '-'}</Descriptions.Item>
            </Descriptions>
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
        title="供应商管理"
        extra={
          <Space wrap>
            <Select
              placeholder="分类"
              allowClear
              style={{ width: 120 }}
              options={categoryOptions}
              value={filterCategory}
              onChange={(v) => { setFilterCategory(v); setPage(1); }}
            />
            <Select
              placeholder="等级"
              allowClear
              style={{ width: 100 }}
              options={levelOptions}
              value={filterLevel}
              onChange={(v) => { setFilterLevel(v); setPage(1); }}
            />
            <Select
              placeholder="状态"
              allowClear
              style={{ width: 100 }}
              options={statusOptions}
              value={filterStatus}
              onChange={(v) => { setFilterStatus(v); setPage(1); }}
            />
            <Search
              placeholder="搜索名称/编码"
              allowClear
              style={{ width: 200 }}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              onSearch={() => setPage(1)}
            />
            <Button
              icon={<ReloadOutlined />}
              onClick={() => queryClient.invalidateQueries({ queryKey: ['srm-suppliers'] })}
            >
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
              新建供应商
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data?.items || []}
          rowKey="id"
          loading={isLoading}
          scroll={{ x: 900 }}
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
            onClick: () => {
              setCurrentSupplier(record);
              setDrawerVisible(true);
            },
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      {/* 创建/编辑弹窗 */}
      <Modal
        title={editingSupplier ? '编辑供应商' : '新建供应商'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={closeModal}
        confirmLoading={editingSupplier ? updateMutation.isPending : createMutation.isPending}
        width={600}
      >
        <Form form={form} layout="vertical">
          <SupplierFormFields />
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title={detail?.name || '供应商详情'}
        open={drawerVisible}
        onClose={() => { setDrawerVisible(false); setCurrentSupplier(null); }}
        width={640}
        extra={detail && (
          <Space>
            <Select
              value={detail.status}
              style={{ width: 110 }}
              loading={statusMutation.isPending}
              options={statusOptions}
              onChange={(v) => statusMutation.mutate({ id: detail.id, status: v })}
            />
            <Button type="primary" icon={<EditOutlined />} onClick={() => handleEdit(detail)}>编辑</Button>
            <Popconfirm title="确认删除该供应商？" onConfirm={() => deleteMutation.mutate(detail.id)}>
              <Button danger icon={<DeleteOutlined />} loading={deleteMutation.isPending}>删除</Button>
            </Popconfirm>
          </Space>
        )}
      >
        {detail && (
          <Tabs
            items={[
              {
                key: 'info',
                label: '基本信息',
                children: (
                  <Descriptions column={2} bordered size="small">
                    <Descriptions.Item label="编码">{detail.code}</Descriptions.Item>
                    <Descriptions.Item label="简称">{detail.short_name || '-'}</Descriptions.Item>
                    <Descriptions.Item label="分类">
                      <Tag color={categoryColors[detail.category]}>{categoryLabels[detail.category] || detail.category}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="等级">
                      <Tag color={levelColors[detail.level]}>{levelLabels[detail.level] || detail.level}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="状态">
                      <Badge status={statusMap[detail.status]?.status || 'default'} text={statusMap[detail.status]?.text || detail.status} />
                    </Descriptions.Item>
                    <Descriptions.Item label="综合评分">{detail.overall_score?.toFixed(1) || '-'}</Descriptions.Item>
                    <Descriptions.Item label="国家">{detail.country || '-'}</Descriptions.Item>
                    <Descriptions.Item label="城市">{detail.city || '-'}</Descriptions.Item>
                    <Descriptions.Item label="地址" span={2}>{detail.address || '-'}</Descriptions.Item>
                    <Descriptions.Item label="业务范围" span={2}>{detail.business_scope || '-'}</Descriptions.Item>
                    <Descriptions.Item label="付款条件">{detail.payment_terms || '-'}</Descriptions.Item>
                    <Descriptions.Item label="创建时间">{dayjs(detail.created_at).format('YYYY-MM-DD')}</Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: 'contacts',
                label: '联系人',
                children: (
                  <div>
                    <Form form={contactForm} layout="inline" style={{ marginBottom: 16 }}>
                      <Form.Item name="name" rules={[{ required: true, message: '请输入姓名' }]}>
                        <Input placeholder="姓名" style={{ width: 100 }} />
                      </Form.Item>
                      <Form.Item name="title">
                        <Input placeholder="职务" style={{ width: 100 }} />
                      </Form.Item>
                      <Form.Item name="phone">
                        <Input placeholder="电话" style={{ width: 130 }} />
                      </Form.Item>
                      <Form.Item name="email">
                        <Input placeholder="邮箱" style={{ width: 160 }} />
                      </Form.Item>
                      <Form.Item>
                        <Button
                          type="primary"
                          icon={<PlusOutlined />}
                          loading={createContactMutation.isPending}
                          onClick={() =>
                            contactForm.validateFields().then((values) => createContactMutation.mutate(values))
                          }
                        >
                          添加
                        </Button>
                      </Form.Item>
                    </Form>
                    <Table
                      columns={contactColumns}
                      dataSource={contacts || []}
                      rowKey="id"
                      size="small"
                      pagination={false}
                    />
                  </div>
                ),
              },
            ]}
          />
        )}
      </Drawer>
    </div>
  );
};

export default Suppliers;
