import React, { useState, useEffect } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Input,
  TreeSelect,
  Modal,
  Form,
  InputNumber,
  message,
  Descriptions,
  Drawer,
  Spin,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  EyeOutlined,
  EditOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { materialsApi, Material, MaterialCategory } from '@/api';
import { useIsMobile } from '@/hooks/useIsMobile';

const { Search } = Input;

const Materials: React.FC = () => {
  const isMobile = useIsMobile();
  const [materials, setMaterials] = useState<Material[]>([]);
  const [categories, setCategories] = useState<MaterialCategory[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [currentMaterial, setCurrentMaterial] = useState<Material | null>(null);
  const [form] = Form.useForm();

  const fetchMaterials = async () => {
    setLoading(true);
    try {
      const res = await materialsApi.list({
        category: selectedCategory || undefined,
        search: searchText || undefined,
      });
      setMaterials(res.materials || []);
    } catch (error) {
      console.error('获取物料列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchCategories = async () => {
    try {
      const res = await materialsApi.listCategories();
      setCategories(res.categories || []);
    } catch (error) {
      console.error('获取分类失败:', error);
    }
  };

  useEffect(() => {
    fetchMaterials();
    fetchCategories();
  }, [selectedCategory]);

  const handleSearch = () => {
    fetchMaterials();
  };

  const handleCreate = () => {
    form.resetFields();
    setCurrentMaterial(null);
    setModalVisible(true);
  };

  const handleEdit = (record: Material) => {
    setCurrentMaterial(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleView = (record: Material) => {
    setCurrentMaterial(record);
    setDetailVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (currentMaterial) {
        await materialsApi.update(currentMaterial.id, values);
        message.success('更新成功');
      } else {
        await materialsApi.create(values);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchMaterials();
    } catch (error) {
      console.error('保存失败:', error);
      message.error('保存失败');
    }
  };

  // 将分类树转为 TreeSelect 的 treeData 格式
  const categoryTreeData = categories.map((cat) => ({
    title: cat.name,
    value: cat.id,
    key: cat.id,
    children: cat.children?.map((child) => ({
      title: child.name,
      value: child.id,
      key: child.id,
    })),
  }));

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      active: 'green',
      inactive: 'default',
      deprecated: 'red',
    };
    return colors[status] || 'default';
  };

  const getStatusLabel = (status: string) => {
    return status === 'active' ? '有效' : status === 'inactive' ? '停用' : '废弃';
  };

  const columns = [
    {
      title: '物料编码',
      dataIndex: 'code',
      key: 'code',
      width: 150,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    {
      title: '物料名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 150,
      render: (category: MaterialCategory) => {
        if (!category) return '-';
        return category.code ? `${category.name} (${category.code})` : category.name;
      },
    },
    {
      title: '规格',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '单位',
      dataIndex: 'unit',
      key: 'unit',
      width: 80,
    },
    {
      title: '标准成本',
      dataIndex: 'standard_cost',
      key: 'standard_cost',
      width: 100,
      render: (cost: number) => `¥${cost?.toFixed(2) || '0.00'}`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>
          {getStatusLabel(status)}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: any, record: Material) => (
        <Space>
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => handleView(record)}>
            查看
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
        </Space>
      ),
    },
  ];

  // ========== Mobile Layout ==========
  if (isMobile) {
    return (
      <div style={{ background: '#f5f5f5', minHeight: '100vh' }}>
        {/* Search bar */}
        <div style={{ padding: '12px 16px', background: '#fff', position: 'sticky', top: 0, zIndex: 10 }}>
          <Input
            placeholder="搜索物料编码/名称"
            prefix={<SearchOutlined style={{ color: '#bbb' }} />}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            onPressEnter={handleSearch}
            allowClear
            style={{ borderRadius: 20 }}
          />
        </div>

        {/* Category filter pills */}
        {categories.length > 0 && (
          <div style={{ display: 'flex', gap: 8, padding: '8px 16px', overflowX: 'auto', WebkitOverflowScrolling: 'touch' }} className="mobile-filter-pills">
            <div
              className={`mobile-filter-pill ${!selectedCategory ? 'active' : ''}`}
              onClick={() => setSelectedCategory('')}
            >全部</div>
            {categories.map(cat => (
              <div
                key={cat.id}
                className={`mobile-filter-pill ${selectedCategory === cat.id ? 'active' : ''}`}
                onClick={() => setSelectedCategory(cat.id)}
              >{cat.name}</div>
            ))}
          </div>
        )}

        {/* Count */}
        <div style={{ padding: '4px 16px 8px', fontSize: 12, color: '#999' }}>
          共 {materials.length} 条物料
        </div>

        {/* Card list */}
        {loading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : (
          <div style={{ padding: '0 12px' }}>
            {materials.map(m => (
              <div
                key={m.id}
                onClick={() => handleView(m)}
                style={{
                  background: '#fff',
                  borderRadius: 10,
                  padding: '12px 14px',
                  marginBottom: 8,
                  boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                  <span style={{ fontWeight: 600, fontSize: 15, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {m.name}
                  </span>
                  <Tag color={getStatusColor(m.status)} style={{ marginLeft: 8, flexShrink: 0 }}>
                    {getStatusLabel(m.status)}
                  </Tag>
                </div>
                <div style={{ fontSize: 12, fontFamily: 'monospace', color: '#1677ff', marginBottom: 6 }}>
                  {m.code}
                </div>
                <div style={{ display: 'flex', gap: 16, fontSize: 13, color: '#666' }}>
                  {m.category && <span>{m.category.name}</span>}
                  <span>{m.unit}</span>
                  <span style={{ marginLeft: 'auto', color: '#333', fontWeight: 500 }}>
                    ¥{m.standard_cost?.toFixed(2) || '0.00'}
                  </span>
                </div>
                {m.description && (
                  <div style={{ fontSize: 12, color: '#999', marginTop: 4, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {m.description}
                  </div>
                )}
              </div>
            ))}
            {materials.length === 0 && (
              <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>暂无物料数据</div>
            )}
          </div>
        )}

        {/* FAB */}
        <div
          onClick={handleCreate}
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

        {/* Modals/Drawer stay the same */}
        <Modal
          title={currentMaterial ? '编辑物料' : '新建物料'}
          open={modalVisible}
          onOk={handleSubmit}
          onCancel={() => setModalVisible(false)}
          width={600}
        >
          <Form form={form} layout="vertical">
            <Form.Item name="name" label="物料名称" rules={[{ required: true, message: '请输入物料名称' }]}>
              <Input placeholder="请输入物料名称" />
            </Form.Item>
            <Form.Item name="category_id" label="分类">
              <TreeSelect placeholder="选择分类" allowClear treeDefaultExpandAll treeData={categoryTreeData} />
            </Form.Item>
            <Form.Item name="unit" label="单位" rules={[{ required: true, message: '请输入单位' }]}>
              <Input placeholder="如: pcs, kg, m" />
            </Form.Item>
            <Form.Item name="description" label="规格描述">
              <Input.TextArea rows={2} placeholder="请输入规格描述" />
            </Form.Item>
            <Form.Item name="standard_cost" label="标准成本">
              <InputNumber style={{ width: '100%' }} min={0} precision={2} placeholder="0.00" />
            </Form.Item>
            <Form.Item name="lead_time_days" label="采购周期(天)">
              <InputNumber style={{ width: '100%' }} min={0} placeholder="0" />
            </Form.Item>
            <Form.Item name="safety_stock" label="安全库存">
              <InputNumber style={{ width: '100%' }} min={0} placeholder="0" />
            </Form.Item>
          </Form>
        </Modal>

        <Drawer
          title="物料详情"
          open={detailVisible}
          onClose={() => setDetailVisible(false)}
          width="100%"
        >
          {currentMaterial && (
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="物料编码">{currentMaterial.code}</Descriptions.Item>
              <Descriptions.Item label="物料名称">{currentMaterial.name}</Descriptions.Item>
              <Descriptions.Item label="分类">{currentMaterial.category?.name || '-'}</Descriptions.Item>
              <Descriptions.Item label="单位">{currentMaterial.unit}</Descriptions.Item>
              <Descriptions.Item label="规格描述">{currentMaterial.description || '-'}</Descriptions.Item>
              <Descriptions.Item label="标准成本">¥{currentMaterial.standard_cost?.toFixed(2) || '0.00'}</Descriptions.Item>
              <Descriptions.Item label="最近成本">¥{currentMaterial.last_cost?.toFixed(2) || '0.00'}</Descriptions.Item>
              <Descriptions.Item label="采购周期">{currentMaterial.lead_time_days || 0} 天</Descriptions.Item>
              <Descriptions.Item label="最小订购量">{currentMaterial.min_order_qty || 0}</Descriptions.Item>
              <Descriptions.Item label="安全库存">{currentMaterial.safety_stock || 0}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={getStatusColor(currentMaterial.status)}>
                  {currentMaterial.status === 'active' ? '有效' : '停用'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="规格参数">
                <pre style={{ margin: 0, fontSize: 12 }}>
                  {JSON.stringify(currentMaterial.specs, null, 2)}
                </pre>
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {new Date(currentMaterial.created_at).toLocaleString()}
              </Descriptions.Item>
            </Descriptions>
          )}
        </Drawer>
      </div>
    );
  }

  // ========== Desktop Layout ==========
  return (
    <div>
      <Card
        title="物料管理"
        extra={
          <Space>
            <TreeSelect
              placeholder="选择分类"
              allowClear
              treeDefaultExpandAll
              style={{ width: 180 }}
              value={selectedCategory || undefined}
              onChange={(v) => setSelectedCategory(v || '')}
              treeData={categoryTreeData}
            />
            <Search
              placeholder="搜索物料"
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              onSearch={handleSearch}
              style={{ width: 200 }}
            />
            <Button icon={<ReloadOutlined />} onClick={fetchMaterials}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              新建物料
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={materials}
          rowKey="id"
          loading={loading}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </Card>

      {/* 创建/编辑弹窗 */}
      <Modal
        title={currentMaterial ? '编辑物料' : '新建物料'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="物料名称" rules={[{ required: true, message: '请输入物料名称' }]}>
            <Input placeholder="请输入物料名称" />
          </Form.Item>
          <Form.Item name="category_id" label="分类">
            <TreeSelect
              placeholder="选择分类"
              allowClear
              treeDefaultExpandAll
              treeData={categoryTreeData}
            />
          </Form.Item>
          <Form.Item name="unit" label="单位" rules={[{ required: true, message: '请输入单位' }]}>
            <Input placeholder="如: pcs, kg, m" />
          </Form.Item>
          <Form.Item name="description" label="规格描述">
            <Input.TextArea rows={2} placeholder="请输入规格描述" />
          </Form.Item>
          <Form.Item name="standard_cost" label="标准成本">
            <InputNumber style={{ width: '100%' }} min={0} precision={2} placeholder="0.00" />
          </Form.Item>
          <Form.Item name="lead_time_days" label="采购周期(天)">
            <InputNumber style={{ width: '100%' }} min={0} placeholder="0" />
          </Form.Item>
          <Form.Item name="safety_stock" label="安全库存">
            <InputNumber style={{ width: '100%' }} min={0} placeholder="0" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title="物料详情"
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
        width={500}
      >
        {currentMaterial && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="物料编码">{currentMaterial.code}</Descriptions.Item>
            <Descriptions.Item label="物料名称">{currentMaterial.name}</Descriptions.Item>
            <Descriptions.Item label="分类">{currentMaterial.category?.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="单位">{currentMaterial.unit}</Descriptions.Item>
            <Descriptions.Item label="规格描述">{currentMaterial.description || '-'}</Descriptions.Item>
            <Descriptions.Item label="标准成本">¥{currentMaterial.standard_cost?.toFixed(2) || '0.00'}</Descriptions.Item>
            <Descriptions.Item label="最近成本">¥{currentMaterial.last_cost?.toFixed(2) || '0.00'}</Descriptions.Item>
            <Descriptions.Item label="采购周期">{currentMaterial.lead_time_days || 0} 天</Descriptions.Item>
            <Descriptions.Item label="最小订购量">{currentMaterial.min_order_qty || 0}</Descriptions.Item>
            <Descriptions.Item label="安全库存">{currentMaterial.safety_stock || 0}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={getStatusColor(currentMaterial.status)}>
                {currentMaterial.status === 'active' ? '有效' : '停用'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="规格参数">
              <pre style={{ margin: 0, fontSize: 12 }}>
                {JSON.stringify(currentMaterial.specs, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {new Date(currentMaterial.created_at).toLocaleString()}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
};

export default Materials;
