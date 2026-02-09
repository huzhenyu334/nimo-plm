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
  message,
  Drawer,
  Descriptions,
} from 'antd';
import { PlusOutlined, ReloadOutlined, ImportOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { srmApi, PurchaseRequest } from '@/api/srm';
import { projectApi } from '@/api/projects';
import { projectBomApi } from '@/api/projectBom';
import dayjs from 'dayjs';

const { Search } = Input;

const typeLabels: Record<string, string> = { sample: '打样', production: '量产' };
const typeColors: Record<string, string> = { sample: 'blue', production: 'green' };

const priorityLabels: Record<string, string> = { low: '低', medium: '中', high: '高', urgent: '紧急' };
const priorityColors: Record<string, string> = { low: 'default', medium: 'blue', high: 'orange', urgent: 'red' };

const statusLabels: Record<string, string> = {
  draft: '草稿', pending: '待审批', approved: '已批准', rejected: '已拒绝',
  in_progress: '进行中', completed: '已完成', cancelled: '已取消',
};
const statusColors: Record<string, string> = {
  draft: 'default', pending: 'processing', approved: 'success', rejected: 'error',
  in_progress: 'processing', completed: 'success', cancelled: 'default',
};

const itemStatusLabels: Record<string, string> = {
  pending: '待处理', ordered: '已下单', received: '已收货',
  inspecting: '检验中', passed: '已通过', failed: '未通过',
};
const itemStatusColors: Record<string, string> = {
  pending: 'default', ordered: 'processing', received: 'blue',
  inspecting: 'orange', passed: 'success', failed: 'error',
};

const PurchaseRequests: React.FC = () => {
  const queryClient = useQueryClient();
  const [searchText, setSearchText] = useState('');
  const [filterType, setFilterType] = useState<string>();
  const [filterStatus, setFilterStatus] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [bomModalVisible, setBomModalVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentPR, setCurrentPR] = useState<PurchaseRequest | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<string>();
  const [form] = Form.useForm();
  const [bomForm] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['srm-prs', searchText, filterType, filterStatus, page, pageSize],
    queryFn: () =>
      srmApi.listPRs({
        search: searchText || undefined,
        type: filterType,
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
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 80,
      render: (t: string) => <Tag color={typeColors[t]}>{typeLabels[t] || t}</Tag>,
    },
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
    { title: '名称', dataIndex: 'material_name', key: 'material_name', width: 160, ellipsis: true },
    { title: '规格', dataIndex: 'specification', key: 'specification', width: 120, ellipsis: true },
    { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 80 },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (s: string) => <Tag color={itemStatusColors[s]}>{itemStatusLabels[s] || s}</Tag>,
    },
    {
      title: '供应商',
      dataIndex: 'supplier_id',
      key: 'supplier_id',
      width: 100,
      render: (id: string) => id?.slice(0, 8) || '-',
    },
    {
      title: '单价',
      dataIndex: 'unit_price',
      key: 'unit_price',
      width: 90,
      render: (p?: number) => (p != null ? `¥${p.toFixed(2)}` : '-'),
    },
  ];

  const detail = prDetail || currentPR;

  return (
    <div>
      <Card
        title="采购需求"
        extra={
          <Space wrap>
            <Select
              placeholder="类型"
              allowClear
              style={{ width: 100 }}
              options={[
                { value: 'sample', label: '打样' },
                { value: 'production', label: '量产' },
              ]}
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
        <Form form={form} layout="vertical" initialValues={{ type: 'sample', priority: 'medium' }}>
          <Form.Item name="title" label="标题" rules={[{ required: true, message: '请输入标题' }]}>
            <Input placeholder="采购需求标题" />
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="type" label="类型" rules={[{ required: true }]}>
              <Select style={{ width: 140 }} options={[{ value: 'sample', label: '打样' }, { value: 'production', label: '量产' }]} />
            </Form.Item>
            <Form.Item name="priority" label="优先级">
              <Select style={{ width: 140 }} options={Object.entries(priorityLabels).map(([k, v]) => ({ value: k, label: v }))} />
            </Form.Item>
          </Space>
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

      {/* 详情抽屉 */}
      <Drawer
        title={detail?.pr_code || '采购需求详情'}
        open={drawerVisible}
        onClose={() => { setDrawerVisible(false); setCurrentPR(null); }}
        width={720}
      >
        {detail && (
          <>
            <Descriptions column={2} bordered size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label="PR编码">{detail.pr_code}</Descriptions.Item>
              <Descriptions.Item label="标题">{detail.title}</Descriptions.Item>
              <Descriptions.Item label="类型">
                <Tag color={typeColors[detail.type]}>{typeLabels[detail.type] || detail.type}</Tag>
              </Descriptions.Item>
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
              scroll={{ x: 700 }}
              pagination={false}
            />
          </>
        )}
      </Drawer>
    </div>
  );
};

export default PurchaseRequests;
