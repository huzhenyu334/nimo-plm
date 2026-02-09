import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  Tag,
  Modal,
  Form,
  message,
  Popconfirm,
  Typography,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { productApi, CreateProductInput, UpdateProductInput } from '@/api/products';
import { Product } from '@/types';
import type { ColumnsType } from 'antd/es/table';

const { Title } = Typography;

const statusMap: Record<string, { text: string; color: string }> = {
  draft: { text: '草稿', color: 'default' },
  active: { text: '启用', color: 'green' },
  deprecated: { text: '已弃用', color: 'orange' },
  obsolete: { text: '已废弃', color: 'red' },
};

const Products: React.FC = () => {
  const queryClient = useQueryClient();
  const [searchText, setSearchText] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);
  const [form] = Form.useForm();

  // 获取产品列表
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['products', { page, pageSize, status: statusFilter, search: searchText }],
    queryFn: () => productApi.list({ page, page_size: pageSize, status: statusFilter, search: searchText }),
  });

  // 创建产品
  const createMutation = useMutation({
    mutationFn: (data: CreateProductInput) => productApi.create(data),
    onSuccess: () => {
      message.success('产品创建成功');
      setModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
    onError: () => {
      message.error('产品创建失败');
    },
  });

  // 更新产品
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateProductInput }) => productApi.update(id, data),
    onSuccess: () => {
      message.success('产品更新成功');
      setModalOpen(false);
      setEditingProduct(null);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
    onError: () => {
      message.error('产品更新失败');
    },
  });

  // 删除产品
  const deleteMutation = useMutation({
    mutationFn: (id: string) => productApi.delete(id),
    onSuccess: () => {
      message.success('产品删除成功');
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
    onError: () => {
      message.error('产品删除失败');
    },
  });

  const columns: ColumnsType<Product> = [
    {
      title: '产品编码',
      dataIndex: 'code',
      key: 'code',
      width: 120,
      render: (text: string) => <a>{text}</a>,
    },
    {
      title: '产品名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '分类',
      dataIndex: 'category_name',
      key: 'category_name',
      width: 120,
    },
    {
      title: '版本',
      dataIndex: 'current_version',
      key: 'current_version',
      width: 80,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const { text, color } = statusMap[status] || { text: status, color: 'default' };
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: '创建人',
      dataIndex: 'created_by_name',
      key: 'created_by_name',
      width: 100,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (text: string) => new Date(text).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />}>
            查看
          </Button>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除此产品吗？"
            onConfirm={() => deleteMutation.mutate(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const handleEdit = (product: Product) => {
    setEditingProduct(product);
    form.setFieldsValue({
      code: product.code,
      name: product.name,
      description: product.description,
      category_id: product.category_id,
    });
    setModalOpen(true);
  };

  const handleModalOk = () => {
    form.validateFields().then((values) => {
      if (editingProduct) {
        updateMutation.mutate({ id: editingProduct.id, data: values });
      } else {
        createMutation.mutate(values);
      }
    });
  };

  const handleModalCancel = () => {
    setModalOpen(false);
    setEditingProduct(null);
    form.resetFields();
  };

  return (
    <div style={{ padding: 24 }}>
      <Card>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
          <Title level={4} style={{ margin: 0 }}>产品管理</Title>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
            新建产品
          </Button>
        </div>

        {/* 搜索栏 */}
        <Space style={{ marginBottom: 16 }}>
          <Input
            placeholder="搜索产品编码/名称"
            prefix={<SearchOutlined />}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            onPressEnter={() => refetch()}
            style={{ width: 250 }}
            allowClear
          />
          <Select
            placeholder="状态筛选"
            value={statusFilter}
            onChange={setStatusFilter}
            style={{ width: 120 }}
            allowClear
            options={[
              { value: 'draft', label: '草稿' },
              { value: 'active', label: '启用' },
              { value: 'deprecated', label: '已弃用' },
              { value: 'obsolete', label: '已废弃' },
            ]}
          />
          <Button onClick={() => refetch()}>搜索</Button>
        </Space>

        {/* 产品表格 */}
        <Table
          columns={columns}
          dataSource={data?.items || []}
          rowKey="id"
          loading={isLoading}
          pagination={{
            current: page,
            pageSize: pageSize,
            total: data?.pagination?.total || 0,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      {/* 新建/编辑弹窗 */}
      <Modal
        title={editingProduct ? '编辑产品' : '新建产品'}
        open={modalOpen}
        onOk={handleModalOk}
        onCancel={handleModalCancel}
        confirmLoading={createMutation.isPending || updateMutation.isPending}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="code"
            label="产品编码"
            rules={[{ required: true, message: '请输入产品编码' }]}
          >
            <Input placeholder="如：NIMO-AIR-001" disabled={!!editingProduct} />
          </Form.Item>
          <Form.Item
            name="name"
            label="产品名称"
            rules={[{ required: true, message: '请输入产品名称' }]}
          >
            <Input placeholder="如：nimo Air 2" />
          </Form.Item>
          <Form.Item name="description" label="产品描述">
            <Input.TextArea rows={3} placeholder="产品描述信息" />
          </Form.Item>
          <Form.Item name="category_id" label="产品分类">
            <Select placeholder="选择分类" allowClear>
              <Select.Option value="glasses">智能眼镜</Select.Option>
              <Select.Option value="accessory">配件</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Products;
