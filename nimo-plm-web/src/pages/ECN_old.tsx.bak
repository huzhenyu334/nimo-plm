import React, { useState, useEffect } from 'react';
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
  message,
  Descriptions,
  Drawer,
  Timeline,
  Steps,
  Divider,
  Popconfirm,
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  EyeOutlined,
  EditOutlined,
  SendOutlined,
  CheckOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import { ecnApi, productApi, ECN, CreateECNRequest } from '@/api';
import { Product } from '@/types';
const { Option } = Select;
const { TextArea } = Input;

const ECNPage: React.FC = () => {
  const [ecns, setEcns] = useState<ECN[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedStatus, setSelectedStatus] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [approveModalVisible, setApproveModalVisible] = useState(false);
  const [currentECN, setCurrentECN] = useState<ECN | null>(null);
  const [form] = Form.useForm();
  const [approveForm] = Form.useForm();
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 });

  const fetchEcns = async (page = 1) => {
    setLoading(true);
    try {
      const res = await ecnApi.list({
        status: selectedStatus || undefined,
        page,
        page_size: pagination.pageSize,
      });
      setEcns(res.items || []);
      setPagination({ ...pagination, current: page, total: res.total });
    } catch (error) {
      console.error('获取ECN列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchProducts = async () => {
    try {
      const res = await productApi.list();
      setProducts(res.items || []);
    } catch (error) {
      console.error('获取产品列表失败:', error);
    }
  };

  useEffect(() => {
    fetchEcns();
    fetchProducts();
  }, [selectedStatus]);

  const handleCreate = () => {
    form.resetFields();
    setCurrentECN(null);
    setModalVisible(true);
  };

  const handleView = async (record: ECN) => {
    try {
      const detail = await ecnApi.get(record.id);
      setCurrentECN(detail);
      setDetailVisible(true);
    } catch (error) {
      message.error('获取详情失败');
    }
  };

  const handleEdit = (record: ECN) => {
    setCurrentECN(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (currentECN) {
        await ecnApi.update(currentECN.id, values);
        message.success('更新成功');
      } else {
        await ecnApi.create(values as CreateECNRequest);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchEcns();
    } catch (error) {
      console.error('保存失败:', error);
      message.error('保存失败');
    }
  };

  const handleSubmitForApproval = async (id: string) => {
    try {
      await ecnApi.submit(id);
      message.success('已提交审批');
      fetchEcns();
    } catch (error) {
      message.error('提交失败');
    }
  };

  const handleApprove = async () => {
    if (!currentECN) return;
    try {
      const values = await approveForm.validateFields();
      await ecnApi.approve(currentECN.id, { comment: values.comment });
      message.success('审批通过');
      setApproveModalVisible(false);
      setDetailVisible(false);
      fetchEcns();
    } catch (error) {
      message.error('审批失败');
    }
  };

  const handleReject = async () => {
    if (!currentECN) return;
    try {
      const values = await approveForm.validateFields();
      if (!values.comment) {
        message.warning('请填写驳回原因');
        return;
      }
      await ecnApi.reject(currentECN.id, { comment: values.comment });
      message.success('已驳回');
      setApproveModalVisible(false);
      setDetailVisible(false);
      fetchEcns();
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleImplement = async (id: string) => {
    try {
      await ecnApi.implement(id);
      message.success('已标记为实施完成');
      fetchEcns();
      if (detailVisible) {
        const detail = await ecnApi.get(id);
        setCurrentECN(detail);
      }
    } catch (error) {
      message.error('操作失败');
    }
  };

  const getStatusConfig = (status: string) => {
    const config: Record<string, { color: string; text: string; step: number }> = {
      draft: { color: 'default', text: '草稿', step: 0 },
      pending: { color: 'processing', text: '待审批', step: 1 },
      approved: { color: 'success', text: '已批准', step: 2 },
      rejected: { color: 'error', text: '已驳回', step: -1 },
      implementing: { color: 'warning', text: '实施中', step: 3 },
      completed: { color: 'green', text: '已完成', step: 4 },
      cancelled: { color: 'default', text: '已取消', step: -1 },
    };
    return config[status] || { color: 'default', text: status, step: 0 };
  };

  const getUrgencyTag = (urgency: string) => {
    const config: Record<string, { color: string; text: string }> = {
      low: { color: 'default', text: '低' },
      medium: { color: 'blue', text: '中' },
      high: { color: 'orange', text: '高' },
      urgent: { color: 'red', text: '紧急' },
    };
    const { color, text } = config[urgency] || { color: 'default', text: urgency };
    return <Tag color={color}>{text}</Tag>;
  };

  const getChangeTypeText = (type: string) => {
    const texts: Record<string, string> = {
      design: '设计变更',
      material: '物料变更',
      process: '工艺变更',
      spec: '规格变更',
      other: '其他',
    };
    return texts[type] || type;
  };

  const columns = [
    {
      title: 'ECN编号',
      dataIndex: 'code',
      key: 'code',
      width: 140,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '产品',
      dataIndex: 'product',
      key: 'product',
      width: 150,
      render: (product: Product) => product?.name || '-',
    },
    {
      title: '变更类型',
      dataIndex: 'change_type',
      key: 'change_type',
      width: 100,
      render: (type: string) => getChangeTypeText(type),
    },
    {
      title: '紧急程度',
      dataIndex: 'urgency',
      key: 'urgency',
      width: 80,
      render: (urgency: string) => getUrgencyTag(urgency),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (status: string) => {
        const { color, text } = getStatusConfig(status);
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: '申请人',
      dataIndex: 'requester',
      key: 'requester',
      width: 100,
      render: (user: any) => user?.name || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (t: string) => new Date(t).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_: any, record: ECN) => (
        <Space>
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => handleView(record)}>
            查看
          </Button>
          {record.status === 'draft' && (
            <>
              <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
                编辑
              </Button>
              <Button type="link" size="small" icon={<SendOutlined />} onClick={() => handleSubmitForApproval(record.id)}>
                提交
              </Button>
            </>
          )}
          {record.status === 'approved' && (
            <Popconfirm title="确认标记为实施完成？" onConfirm={() => handleImplement(record.id)}>
              <Button type="link" size="small" icon={<CheckOutlined />}>
                完成
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="ECN变更管理"
        extra={
          <Space>
            <Select
              placeholder="状态筛选"
              allowClear
              style={{ width: 120 }}
              value={selectedStatus || undefined}
              onChange={(v) => setSelectedStatus(v || '')}
            >
              <Option value="draft">草稿</Option>
              <Option value="pending">待审批</Option>
              <Option value="approved">已批准</Option>
              <Option value="rejected">已驳回</Option>
              <Option value="implementing">实施中</Option>
              <Option value="completed">已完成</Option>
            </Select>
            <Button icon={<ReloadOutlined />} onClick={() => fetchEcns()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              新建ECN
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={ecns}
          rowKey="id"
          loading={loading}
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => {
              setPagination({ ...pagination, current: page, pageSize });
              fetchEcns(page);
            },
          }}
        />
      </Card>

      {/* 创建/编辑弹窗 */}
      <Modal
        title={currentECN ? '编辑ECN' : '新建ECN'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]}>
            <Input placeholder="请输入ECN标题" />
          </Form.Item>
          <Form.Item name="product_id" label="关联产品" rules={[{ required: true, message: '请选择产品' }]}>
            <Select placeholder="选择产品">
              {products.map((p) => (
                <Option key={p.id} value={p.id}>
                  {p.code} - {p.name}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="change_type" label="变更类型" rules={[{ required: true, message: '请选择变更类型' }]}>
            <Select placeholder="选择变更类型">
              <Option value="design">设计变更</Option>
              <Option value="material">物料变更</Option>
              <Option value="process">工艺变更</Option>
              <Option value="spec">规格变更</Option>
              <Option value="other">其他</Option>
            </Select>
          </Form.Item>
          <Form.Item name="urgency" label="紧急程度">
            <Select placeholder="选择紧急程度" defaultValue="medium">
              <Option value="low">低</Option>
              <Option value="medium">中</Option>
              <Option value="high">高</Option>
              <Option value="urgent">紧急</Option>
            </Select>
          </Form.Item>
          <Form.Item name="reason" label="变更原因" rules={[{ required: true, message: '请输入变更原因' }]}>
            <TextArea rows={2} placeholder="请输入变更原因" />
          </Form.Item>
          <Form.Item name="description" label="详细描述">
            <TextArea rows={3} placeholder="请输入详细描述" />
          </Form.Item>
          <Form.Item name="impact_analysis" label="影响分析">
            <TextArea rows={2} placeholder="请输入影响分析" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title="ECN详情"
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
        width={700}
        extra={
          currentECN?.status === 'pending' && (
            <Space>
              <Button type="primary" onClick={() => setApproveModalVisible(true)}>
                审批
              </Button>
            </Space>
          )
        }
      >
        {currentECN && (
          <>
            {/* 流程进度 */}
            <Steps
              current={getStatusConfig(currentECN.status).step}
              status={currentECN.status === 'rejected' ? 'error' : undefined}
              size="small"
              style={{ marginBottom: 24 }}
              items={[
                { title: '草稿' },
                { title: '审批中' },
                { title: '已批准' },
                { title: '实施中' },
                { title: '已完成' },
              ]}
            />

            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="ECN编号">{currentECN.code}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={getStatusConfig(currentECN.status).color}>
                  {getStatusConfig(currentECN.status).text}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="标题" span={2}>{currentECN.title}</Descriptions.Item>
              <Descriptions.Item label="关联产品">{currentECN.product?.name || '-'}</Descriptions.Item>
              <Descriptions.Item label="变更类型">{getChangeTypeText(currentECN.change_type)}</Descriptions.Item>
              <Descriptions.Item label="紧急程度">{getUrgencyTag(currentECN.urgency)}</Descriptions.Item>
              <Descriptions.Item label="申请人">{currentECN.requester?.name || '-'}</Descriptions.Item>
              <Descriptions.Item label="变更原因" span={2}>{currentECN.reason}</Descriptions.Item>
              <Descriptions.Item label="详细描述" span={2}>{currentECN.description || '-'}</Descriptions.Item>
              <Descriptions.Item label="影响分析" span={2}>{currentECN.impact_analysis || '-'}</Descriptions.Item>
              {currentECN.rejection_reason && (
                <Descriptions.Item label="驳回原因" span={2}>
                  <span style={{ color: 'red' }}>{currentECN.rejection_reason}</span>
                </Descriptions.Item>
              )}
              <Descriptions.Item label="创建时间">
                {new Date(currentECN.created_at).toLocaleString()}
              </Descriptions.Item>
              <Descriptions.Item label="审批时间">
                {currentECN.approved_at ? new Date(currentECN.approved_at).toLocaleString() : '-'}
              </Descriptions.Item>
            </Descriptions>

            {/* 审批记录 */}
            {currentECN.approvals && currentECN.approvals.length > 0 && (
              <>
                <Divider orientation="left">审批记录</Divider>
                <Timeline
                  items={currentECN.approvals.map((a) => ({
                    color: a.status === 'approved' ? 'green' : a.status === 'rejected' ? 'red' : 'blue',
                    children: (
                      <div>
                        <div>
                          <strong>{a.approver?.name || '审批人'}</strong>
                          {a.status === 'approved' ? ' 批准' : a.status === 'rejected' ? ' 驳回' : ' 待审批'}
                        </div>
                        {a.comment && <div style={{ color: '#666' }}>{a.comment}</div>}
                        {a.approved_at && (
                          <div style={{ fontSize: 12, color: '#999' }}>
                            {new Date(a.approved_at).toLocaleString()}
                          </div>
                        )}
                      </div>
                    ),
                  }))}
                />
              </>
            )}
          </>
        )}
      </Drawer>

      {/* 审批弹窗 */}
      <Modal
        title="ECN审批"
        open={approveModalVisible}
        onCancel={() => setApproveModalVisible(false)}
        footer={[
          <Button key="cancel" onClick={() => setApproveModalVisible(false)}>
            取消
          </Button>,
          <Button key="reject" danger icon={<CloseOutlined />} onClick={handleReject}>
            驳回
          </Button>,
          <Button key="approve" type="primary" icon={<CheckOutlined />} onClick={handleApprove}>
            批准
          </Button>,
        ]}
      >
        <Form form={approveForm} layout="vertical">
          <Form.Item name="comment" label="审批意见">
            <TextArea rows={3} placeholder="请输入审批意见（驳回时必填）" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ECNPage;
