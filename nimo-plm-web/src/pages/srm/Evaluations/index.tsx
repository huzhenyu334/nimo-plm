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
  Drawer,
  Descriptions,
  Rate,
  Progress,
  message,
  InputNumber,
  Spin,
} from 'antd';
import { PlusOutlined, ReloadOutlined, SearchOutlined, RightOutlined } from '@ant-design/icons';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { srmApi, SupplierEvaluation } from '@/api/srm';
import dayjs from 'dayjs';

const { TextArea } = Input;

const statusMap: Record<string, { text: string; status: 'default' | 'processing' | 'success' | 'warning' }> = {
  draft: { text: '草稿', status: 'default' },
  submitted: { text: '已提交', status: 'processing' },
  approved: { text: '已审批', status: 'success' },
};

const evalTypeLabels: Record<string, string> = {
  monthly: '月度评估',
  quarterly: '季度评估',
  annual: '年度评估',
};

const gradeColors: Record<string, string> = {
  A: 'green',
  B: 'blue',
  C: 'orange',
  D: 'red',
};

const scoreToStars = (score?: number) => {
  if (score == null) return 0;
  return Math.round(score / 20);
};

export default function Evaluations() {
  const isMobile = useIsMobile();
  const queryClient = useQueryClient();
  const [filterSupplier, setFilterSupplier] = useState<string>();
  const [filterStatus, setFilterStatus] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createVisible, setCreateVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentEval, setCurrentEval] = useState<SupplierEvaluation | null>(null);
  const [historyVisible, setHistoryVisible] = useState(false);
  const [historySupplierId, setHistorySupplierId] = useState<string>('');
  const [form] = Form.useForm();

  // 供应商列表
  const { data: suppliers } = useQuery({
    queryKey: ['srm-suppliers-all'],
    queryFn: () => srmApi.listSuppliers({ page_size: 100 }),
  });

  // 评估列表
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['srm-evaluations', filterSupplier, filterStatus, page, pageSize],
    queryFn: () =>
      srmApi.listEvaluations({
        supplier_id: filterSupplier,
        status: filterStatus,
        page,
        page_size: pageSize,
      }),
  });

  // 评估详情
  const { data: detail } = useQuery({
    queryKey: ['srm-evaluation', currentEval?.id],
    queryFn: () => srmApi.getEvaluation(currentEval!.id),
    enabled: !!currentEval?.id && drawerVisible,
  });

  // 供应商评估历史
  const { data: history } = useQuery({
    queryKey: ['srm-evaluation-history', historySupplierId],
    queryFn: () => srmApi.getSupplierEvaluationHistory(historySupplierId),
    enabled: !!historySupplierId && historyVisible,
  });

  const createMutation = useMutation({
    mutationFn: (values: {
      supplier_id: string;
      period: string;
      eval_type?: string;
      quality_score?: number;
      delivery_score?: number;
      price_score?: number;
      service_score?: number;
      remarks?: string;
    }) => srmApi.createEvaluation(values),
    onSuccess: () => {
      message.success('评估创建成功');
      setCreateVisible(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['srm-evaluations'] });
    },
    onError: () => message.error('创建失败'),
  });

  const submitMutation = useMutation({
    mutationFn: (id: string) => srmApi.submitEvaluation(id),
    onSuccess: () => {
      message.success('评估已提交');
      queryClient.invalidateQueries({ queryKey: ['srm-evaluations'] });
      queryClient.invalidateQueries({ queryKey: ['srm-evaluation'] });
    },
    onError: () => message.error('提交失败'),
  });

  const approveMutation = useMutation({
    mutationFn: (id: string) => srmApi.approveEvaluation(id),
    onSuccess: () => {
      message.success('评估已审批');
      queryClient.invalidateQueries({ queryKey: ['srm-evaluations'] });
      queryClient.invalidateQueries({ queryKey: ['srm-evaluation'] });
    },
    onError: () => message.error('审批失败'),
  });

  const supplierOptions = (suppliers?.items || []).map((s: { id: string; name: string }) => ({
    value: s.id,
    label: s.name,
  }));

  const columns = [
    {
      title: '供应商',
      key: 'supplier',
      width: 160,
      ellipsis: true,
      render: (_: unknown, record: SupplierEvaluation) => (
        <Button
          type="link"
          size="small"
          style={{ padding: 0 }}
          onClick={() => {
            setHistorySupplierId(record.supplier_id);
            setHistoryVisible(true);
          }}
        >
          {record.supplier?.name || '-'}
        </Button>
      ),
    },
    {
      title: '评估周期',
      dataIndex: 'period',
      key: 'period',
      width: 110,
    },
    {
      title: '类型',
      dataIndex: 'eval_type',
      key: 'eval_type',
      width: 100,
      render: (t: string) => evalTypeLabels[t] || t,
    },
    {
      title: '质量',
      dataIndex: 'quality_score',
      key: 'quality_score',
      width: 80,
      align: 'center' as const,
      render: (v?: number) => (v != null ? v.toFixed(0) : '-'),
    },
    {
      title: '交期',
      dataIndex: 'delivery_score',
      key: 'delivery_score',
      width: 80,
      align: 'center' as const,
      render: (v?: number) => (v != null ? v.toFixed(0) : '-'),
    },
    {
      title: '价格',
      dataIndex: 'price_score',
      key: 'price_score',
      width: 80,
      align: 'center' as const,
      render: (v?: number) => (v != null ? v.toFixed(0) : '-'),
    },
    {
      title: '服务',
      dataIndex: 'service_score',
      key: 'service_score',
      width: 80,
      align: 'center' as const,
      render: (v?: number) => (v != null ? v.toFixed(0) : '-'),
    },
    {
      title: '综合评分',
      dataIndex: 'total_score',
      key: 'total_score',
      width: 100,
      align: 'center' as const,
      render: (v?: number) => (v != null ? <strong>{v.toFixed(1)}</strong> : '-'),
    },
    {
      title: '等级',
      dataIndex: 'grade',
      key: 'grade',
      width: 60,
      align: 'center' as const,
      render: (g: string) => g ? <Tag color={gradeColors[g] || 'default'}>{g}</Tag> : '-',
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
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: SupplierEvaluation) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            onClick={() => {
              setCurrentEval(record);
              setDrawerVisible(true);
            }}
          >
            详情
          </Button>
          {record.status === 'draft' && (
            <Button type="link" size="small" onClick={() => submitMutation.mutate(record.id)}>
              提交
            </Button>
          )}
          {record.status === 'submitted' && (
            <Button type="link" size="small" onClick={() => approveMutation.mutate(record.id)}>
              审批
            </Button>
          )}
        </Space>
      ),
    },
  ];

  const detailData = detail || currentEval;

  return (
    <div>
      <Card
        title="供应商评价"
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
                { value: 'submitted', label: '已提交' },
                { value: 'approved', label: '已审批' },
              ]}
              value={filterStatus}
              onChange={(v) => { setFilterStatus(v); setPage(1); }}
            />
            <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
              新建评价
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

      {/* 创建评估 */}
      <Modal
        title="新建供应商评价"
        open={createVisible}
        onCancel={() => { setCreateVisible(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
        width={560}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => createMutation.mutate(values)}
        >
          <Form.Item name="supplier_id" label="供应商" rules={[{ required: true, message: '请选择供应商' }]}>
            <Select placeholder="选择供应商" showSearch optionFilterProp="label" options={supplierOptions} />
          </Form.Item>
          <Form.Item name="period" label="评价周期" rules={[{ required: true, message: '请输入评价周期' }]}>
            <Input placeholder="如 2026-Q1 或 2026-01" />
          </Form.Item>
          <Form.Item name="eval_type" label="评价类型" initialValue="quarterly">
            <Select
              options={[
                { value: 'monthly', label: '月度评估' },
                { value: 'quarterly', label: '季度评估' },
                { value: 'annual', label: '年度评估' },
              ]}
            />
          </Form.Item>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 16px' }}>
            <Form.Item name="quality_score" label="质量评分 (0-100)">
              <InputNumber min={0} max={100} style={{ width: '100%' }} placeholder="0-100" />
            </Form.Item>
            <Form.Item name="delivery_score" label="交期评分 (0-100)">
              <InputNumber min={0} max={100} style={{ width: '100%' }} placeholder="0-100" />
            </Form.Item>
            <Form.Item name="price_score" label="价格竞争力 (0-100)">
              <InputNumber min={0} max={100} style={{ width: '100%' }} placeholder="0-100" />
            </Form.Item>
            <Form.Item name="service_score" label="服务响应 (0-100)">
              <InputNumber min={0} max={100} style={{ width: '100%' }} placeholder="0-100" />
            </Form.Item>
          </div>
          <Form.Item name="remarks" label="评价备注">
            <TextArea rows={3} placeholder="请输入评价备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 评估详情 */}
      <Drawer
        title={`评价详情 - ${detailData?.supplier?.name || ''}`}
        open={drawerVisible}
        onClose={() => { setDrawerVisible(false); setCurrentEval(null); }}
        width={600}
        extra={
          <Space>
            {detailData?.status === 'draft' && (
              <Button type="primary" onClick={() => submitMutation.mutate(detailData!.id)}>
                提交评价
              </Button>
            )}
            {detailData?.status === 'submitted' && (
              <Button type="primary" onClick={() => approveMutation.mutate(detailData!.id)}>
                审批通过
              </Button>
            )}
          </Space>
        }
      >
        {detailData && (
          <>
            <Descriptions column={2} bordered size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label="供应商">{detailData.supplier?.name || '-'}</Descriptions.Item>
              <Descriptions.Item label="评估周期">{detailData.period}</Descriptions.Item>
              <Descriptions.Item label="评估类型">{evalTypeLabels[detailData.eval_type] || detailData.eval_type}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Badge
                  status={statusMap[detailData.status]?.status || 'default'}
                  text={statusMap[detailData.status]?.text || detailData.status}
                />
              </Descriptions.Item>
              <Descriptions.Item label="综合评分" span={2}>
                <Space>
                  <strong style={{ fontSize: 20 }}>{detailData.total_score?.toFixed(1) || '-'}</strong>
                  {detailData.grade && <Tag color={gradeColors[detailData.grade]}>{detailData.grade}级</Tag>}
                  {detailData.total_score != null && (
                    <Rate disabled value={scoreToStars(detailData.total_score)} />
                  )}
                </Space>
              </Descriptions.Item>
            </Descriptions>

            <h4 style={{ marginBottom: 16 }}>评分明细</h4>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                  <span>质量 (权重 {((detailData.quality_weight || 0.3) * 100).toFixed(0)}%)</span>
                  <span>{detailData.quality_score?.toFixed(0) || '-'}</span>
                </div>
                <Progress percent={detailData.quality_score || 0} showInfo={false} strokeColor="#52c41a" />
              </div>
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                  <span>交期 (权重 {((detailData.delivery_weight || 0.25) * 100).toFixed(0)}%)</span>
                  <span>{detailData.delivery_score?.toFixed(0) || '-'}</span>
                </div>
                <Progress percent={detailData.delivery_score || 0} showInfo={false} strokeColor="#1890ff" />
              </div>
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                  <span>价格竞争力 (权重 {((detailData.price_weight || 0.25) * 100).toFixed(0)}%)</span>
                  <span>{detailData.price_score?.toFixed(0) || '-'}</span>
                </div>
                <Progress percent={detailData.price_score || 0} showInfo={false} strokeColor="#faad14" />
              </div>
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                  <span>服务响应 (权重 {((detailData.service_weight || 0.2) * 100).toFixed(0)}%)</span>
                  <span>{detailData.service_score?.toFixed(0) || '-'}</span>
                </div>
                <Progress percent={detailData.service_score || 0} showInfo={false} strokeColor="#722ed1" />
              </div>
            </div>

            {detailData.remarks && (
              <div style={{ marginTop: 24 }}>
                <h4>评价备注</h4>
                <p style={{ color: '#666' }}>{detailData.remarks}</p>
              </div>
            )}

            <div style={{ marginTop: 24, color: '#999', fontSize: 12 }}>
              创建时间：{dayjs(detailData.created_at).format('YYYY-MM-DD HH:mm')}
            </div>
          </>
        )}
      </Drawer>

      {/* 评估历史 */}
      <Modal
        title="评价历史趋势"
        open={historyVisible}
        onCancel={() => { setHistoryVisible(false); setHistorySupplierId(''); }}
        footer={null}
        width={700}
      >
        {history && history.length > 0 ? (
          <Table
            rowKey="id"
            dataSource={history}
            pagination={false}
            size="small"
            columns={[
              { title: '周期', dataIndex: 'period', key: 'period', width: 100 },
              { title: '类型', dataIndex: 'eval_type', key: 'eval_type', width: 90, render: (t: string) => evalTypeLabels[t] || t },
              { title: '质量', dataIndex: 'quality_score', key: 'quality_score', width: 70, align: 'center' as const, render: (v?: number) => v?.toFixed(0) || '-' },
              { title: '交期', dataIndex: 'delivery_score', key: 'delivery_score', width: 70, align: 'center' as const, render: (v?: number) => v?.toFixed(0) || '-' },
              { title: '价格', dataIndex: 'price_score', key: 'price_score', width: 70, align: 'center' as const, render: (v?: number) => v?.toFixed(0) || '-' },
              { title: '服务', dataIndex: 'service_score', key: 'service_score', width: 70, align: 'center' as const, render: (v?: number) => v?.toFixed(0) || '-' },
              { title: '综合', dataIndex: 'total_score', key: 'total_score', width: 80, align: 'center' as const, render: (v?: number) => v != null ? <strong>{v.toFixed(1)}</strong> : '-' },
              { title: '等级', dataIndex: 'grade', key: 'grade', width: 60, align: 'center' as const, render: (g: string) => g ? <Tag color={gradeColors[g]}>{g}</Tag> : '-' },
              { title: '状态', dataIndex: 'status', key: 'status', width: 80, render: (s: string) => <Badge status={statusMap[s]?.status || 'default'} text={statusMap[s]?.text || s} /> },
            ]}
          />
        ) : (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>暂无评价记录</div>
        )}
      </Modal>
    </div>
  );
}
