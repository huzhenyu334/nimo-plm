import React, { useState, useEffect } from 'react';
import {
  Card,
  Select,
  Table,
  Tag,
  Space,
  Button,
  Statistic,
  Row,
  Col,
  Empty,
  Spin,
  Modal,
  Form,
  InputNumber,
  Input,
  message,
  Drawer,
} from 'antd';
import {
  ReloadOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  HistoryOutlined,
} from '@ant-design/icons';
import { productApi, bomApi, materialsApi, BOMHeader, BOMItem, Material } from '@/api';
import { Product } from '@/types';

const { Option } = Select;

const BOM: React.FC = () => {
  const [products, setProducts] = useState<Product[]>([]);
  const [materials, setMaterials] = useState<Material[]>([]);
  const [selectedProductId, setSelectedProductId] = useState<string>('');
  const [bom, setBom] = useState<BOMHeader | null>(null);
  const [loading, setLoading] = useState(false);
  const [addItemModalVisible, setAddItemModalVisible] = useState(false);
  const [releaseModalVisible, setReleaseModalVisible] = useState(false);
  const [versionsDrawerVisible, setVersionsDrawerVisible] = useState(false);
  const [versions, setVersions] = useState<BOMHeader[]>([]);
  const [form] = Form.useForm();
  const [releaseForm] = Form.useForm();

  const fetchProducts = async () => {
    try {
      const res = await productApi.list();
      setProducts(res.items || []);
      if (res.items?.length > 0 && !selectedProductId) {
        setSelectedProductId(res.items[0].id);
      }
    } catch (error) {
      console.error('获取产品列表失败:', error);
    }
  };

  const fetchMaterials = async () => {
    try {
      const res = await materialsApi.list();
      setMaterials(res.materials || []);
    } catch (error) {
      console.error('获取物料列表失败:', error);
    }
  };

  const fetchBOM = async () => {
    if (!selectedProductId) return;
    setLoading(true);
    try {
      const res = await bomApi.get(selectedProductId);
      setBom(res);
    } catch (error: any) {
      if (error?.response?.status === 404) {
        setBom(null);
      } else {
        console.error('获取BOM失败:', error);
      }
    } finally {
      setLoading(false);
    }
  };

  const fetchVersions = async () => {
    if (!selectedProductId) return;
    try {
      const res = await bomApi.listVersions(selectedProductId);
      setVersions(res.versions || []);
    } catch (error) {
      console.error('获取版本列表失败:', error);
    }
  };

  useEffect(() => {
    fetchProducts();
    fetchMaterials();
  }, []);

  useEffect(() => {
    if (selectedProductId) {
      fetchBOM();
    }
  }, [selectedProductId]);

  const handleAddItem = async () => {
    try {
      const values = await form.validateFields();
      await bomApi.addItem(selectedProductId, {
        material_id: values.material_id,
        quantity: values.quantity,
        unit: values.unit,
        position: values.position,
        notes: values.notes,
      });
      message.success('添加成功');
      setAddItemModalVisible(false);
      form.resetFields();
      fetchBOM();
    } catch (error) {
      console.error('添加失败:', error);
      message.error('添加失败');
    }
  };

  const handleDeleteItem = async (itemId: string) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个BOM项吗？',
      onOk: async () => {
        try {
          await bomApi.deleteItem(selectedProductId, itemId);
          message.success('删除成功');
          fetchBOM();
        } catch (error) {
          message.error('删除失败');
        }
      },
    });
  };

  const handleRelease = async () => {
    try {
      const values = await releaseForm.validateFields();
      await bomApi.release(selectedProductId, {
        version: values.version,
        release_notes: values.release_notes,
      });
      message.success('发布成功');
      setReleaseModalVisible(false);
      releaseForm.resetFields();
      fetchBOM();
    } catch (error) {
      console.error('发布失败:', error);
      message.error('发布失败');
    }
  };

  const handleViewVersions = () => {
    fetchVersions();
    setVersionsDrawerVisible(true);
  };

  const getStatusTag = (status: string) => {
    const config: Record<string, { color: string; text: string }> = {
      draft: { color: 'default', text: '草稿' },
      released: { color: 'green', text: '已发布' },
      obsolete: { color: 'red', text: '已废弃' },
    };
    const { color, text } = config[status] || { color: 'default', text: status };
    return <Tag color={color}>{text}</Tag>;
  };

  const columns = [
    {
      title: '序号',
      dataIndex: 'sequence',
      key: 'sequence',
      width: 60,
    },
    {
      title: '位号',
      dataIndex: 'position',
      key: 'position',
      width: 80,
    },
    {
      title: '物料编码',
      dataIndex: ['material', 'code'],
      key: 'material_code',
      width: 130,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    {
      title: '物料名称',
      dataIndex: ['material', 'name'],
      key: 'material_name',
      width: 200,
    },
    {
      title: '规格',
      dataIndex: ['material', 'description'],
      key: 'spec',
      ellipsis: true,
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      key: 'quantity',
      width: 80,
      align: 'right' as const,
    },
    {
      title: '单位',
      dataIndex: 'unit',
      key: 'unit',
      width: 60,
    },
    {
      title: '单价',
      dataIndex: 'unit_cost',
      key: 'unit_cost',
      width: 90,
      align: 'right' as const,
      render: (cost: number) => `¥${cost?.toFixed(2) || '0.00'}`,
    },
    {
      title: '金额',
      dataIndex: 'extended_cost',
      key: 'extended_cost',
      width: 90,
      align: 'right' as const,
      render: (cost: number) => `¥${cost?.toFixed(2) || '0.00'}`,
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: any, record: BOMItem) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} disabled={bom?.status === 'released'}>
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDeleteItem(record.id)}
            disabled={bom?.status === 'released'}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="BOM管理"
        extra={
          <Space>
            <span>选择产品：</span>
            <Select
              placeholder="选择产品"
              style={{ width: 250 }}
              value={selectedProductId || undefined}
              onChange={(v) => setSelectedProductId(v)}
            >
              {products.map((p) => (
                <Option key={p.id} value={p.id}>
                  {p.code} - {p.name}
                </Option>
              ))}
            </Select>
            <Button icon={<ReloadOutlined />} onClick={fetchBOM}>
              刷新
            </Button>
          </Space>
        }
      >
        {loading ? (
          <div style={{ textAlign: 'center', padding: 50 }}>
            <Spin size="large" />
          </div>
        ) : !selectedProductId ? (
          <Empty description="请选择产品" />
        ) : !bom ? (
          <Empty description="该产品暂无BOM数据" />
        ) : (
          <>
            {/* BOM 概览 */}
            <Row gutter={16} style={{ marginBottom: 24 }}>
              <Col span={4}>
                <Statistic title="BOM版本" value={bom.version} />
              </Col>
              <Col span={4}>
                <Statistic title="状态" valueRender={() => getStatusTag(bom.status)} />
              </Col>
              <Col span={4}>
                <Statistic title="物料数量" value={bom.total_items} suffix="项" />
              </Col>
              <Col span={4}>
                <Statistic title="总成本" value={bom.total_cost?.toFixed(2)} prefix="¥" />
              </Col>
              <Col span={8}>
                <Space>
                  <Button icon={<PlusOutlined />} onClick={() => setAddItemModalVisible(true)} disabled={bom.status === 'released'}>
                    添加物料
                  </Button>
                  <Button icon={<CheckCircleOutlined />} type="primary" onClick={() => setReleaseModalVisible(true)} disabled={bom.status === 'released'}>
                    发布BOM
                  </Button>
                  <Button icon={<HistoryOutlined />} onClick={handleViewVersions}>
                    版本历史
                  </Button>
                </Space>
              </Col>
            </Row>

            {/* BOM 明细表 */}
            <Table
              columns={columns}
              dataSource={bom.items || []}
              rowKey="id"
              pagination={false}
              size="middle"
              summary={(data) => {
                const totalQty = data.reduce((sum, item) => sum + item.quantity, 0);
                const totalCost = data.reduce((sum, item) => sum + (item.extended_cost || 0), 0);
                return (
                  <Table.Summary.Row>
                    <Table.Summary.Cell index={0} colSpan={5}>
                      <strong>合计</strong>
                    </Table.Summary.Cell>
                    <Table.Summary.Cell index={1} align="right">
                      <strong>{totalQty}</strong>
                    </Table.Summary.Cell>
                    <Table.Summary.Cell index={2} />
                    <Table.Summary.Cell index={3} />
                    <Table.Summary.Cell index={4} align="right">
                      <strong>¥{totalCost.toFixed(2)}</strong>
                    </Table.Summary.Cell>
                    <Table.Summary.Cell index={5} />
                  </Table.Summary.Row>
                );
              }}
            />
          </>
        )}
      </Card>

      {/* 添加物料弹窗 */}
      <Modal
        title="添加物料到BOM"
        open={addItemModalVisible}
        onOk={handleAddItem}
        onCancel={() => setAddItemModalVisible(false)}
        width={500}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="material_id" label="选择物料" rules={[{ required: true, message: '请选择物料' }]}>
            <Select
              placeholder="搜索并选择物料"
              showSearch
              filterOption={(input, option) =>
                (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
              }
            >
              {materials.map((m) => (
                <Option key={m.id} value={m.id}>
                  {m.code} - {m.name}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="quantity" label="数量" rules={[{ required: true, message: '请输入数量' }]}>
            <InputNumber min={0.01} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="unit" label="单位" rules={[{ required: true, message: '请输入单位' }]}>
            <Input placeholder="如: pcs, kg" />
          </Form.Item>
          <Form.Item name="position" label="位号">
            <Input placeholder="如: E1, M2" />
          </Form.Item>
          <Form.Item name="notes" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 发布弹窗 */}
      <Modal
        title="发布BOM"
        open={releaseModalVisible}
        onOk={handleRelease}
        onCancel={() => setReleaseModalVisible(false)}
        width={400}
      >
        <Form form={releaseForm} layout="vertical">
          <Form.Item name="version" label="版本号" rules={[{ required: true, message: '请输入版本号' }]}>
            <Input placeholder="如: 1.0, 2.0" />
          </Form.Item>
          <Form.Item name="release_notes" label="发布说明">
            <Input.TextArea rows={3} placeholder="请输入发布说明" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 版本历史抽屉 */}
      <Drawer
        title="BOM版本历史"
        open={versionsDrawerVisible}
        onClose={() => setVersionsDrawerVisible(false)}
        width={600}
      >
        <Table
          columns={[
            { title: '版本', dataIndex: 'version', key: 'version', width: 80 },
            {
              title: '状态',
              dataIndex: 'status',
              key: 'status',
              width: 80,
              render: (status: string) => getStatusTag(status),
            },
            { title: '物料数', dataIndex: 'total_items', key: 'total_items', width: 80 },
            {
              title: '总成本',
              dataIndex: 'total_cost',
              key: 'total_cost',
              width: 100,
              render: (cost: number) => `¥${cost?.toFixed(2)}`,
            },
            {
              title: '发布时间',
              dataIndex: 'released_at',
              key: 'released_at',
              render: (t: string) => (t ? new Date(t).toLocaleString() : '-'),
            },
          ]}
          dataSource={versions}
          rowKey="id"
          pagination={false}
          size="small"
        />
      </Drawer>
    </div>
  );
};

export default BOM;
