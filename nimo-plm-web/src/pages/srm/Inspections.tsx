import React, { useState } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Input,
  Select,
  message,
  Drawer,
  Descriptions,
  Form,
} from 'antd';
import { ReloadOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { srmApi, Inspection } from '@/api/srm';
import dayjs from 'dayjs';

const { Search } = Input;

const statusLabels: Record<string, string> = {
  pending: '待检验', in_progress: '检验中', completed: '已完成',
};
const statusColors: Record<string, string> = {
  pending: 'default', in_progress: 'processing', completed: 'success',
};

const resultLabels: Record<string, string> = {
  passed: '合格', failed: '不合格', conditional: '条件放行', '': '待检',
};
const resultColors: Record<string, string> = {
  passed: 'green', failed: 'red', conditional: 'orange',
};

const Inspections: React.FC = () => {
  const queryClient = useQueryClient();
  const [searchText, setSearchText] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>();
  const [filterResult, setFilterResult] = useState<string>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentInsp, setCurrentInsp] = useState<Inspection | null>(null);
  const [completeForm] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['srm-inspections', searchText, filterStatus, filterResult, page, pageSize],
    queryFn: () =>
      srmApi.listInspections({
        status: filterStatus,
        result: filterResult,
        page,
        page_size: pageSize,
      }),
  });

  const { data: inspDetail } = useQuery({
    queryKey: ['srm-inspection', currentInsp?.id],
    queryFn: () => srmApi.getInspection(currentInsp!.id),
    enabled: !!currentInsp?.id && drawerVisible,
  });

  const completeMutation = useMutation({
    mutationFn: (values: { result: string; notes?: string; inspection_items?: unknown }) =>
      srmApi.completeInspection(currentInsp!.id, values),
    onSuccess: () => {
      message.success('检验完成');
      completeForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ['srm-inspections'] });
      queryClient.invalidateQueries({ queryKey: ['srm-inspection', currentInsp?.id] });
    },
    onError: () => message.error('操作失败'),
  });

  const columns = [
    {
      title: '检验编码',
      dataIndex: 'inspection_code',
      key: 'inspection_code',
      width: 140,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    {
      title: 'PO编码',
      dataIndex: 'po_id',
      key: 'po_id',
      width: 100,
      render: (id: string) => id?.slice(0, 8) || '-',
    },
    { title: '物料', dataIndex: 'material_name', key: 'material_name', width: 160, ellipsis: true },
    {
      title: '供应商',
      dataIndex: 'supplier_id',
      key: 'supplier_id',
      width: 100,
      render: (id: string) => id?.slice(0, 8) || '-',
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      key: 'quantity',
      width: 80,
      render: (q?: number) => q ?? '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (s: string) => <Tag color={statusColors[s]}>{statusLabels[s] || s}</Tag>,
    },
    {
      title: '结果',
      dataIndex: 'result',
      key: 'result',
      width: 90,
      render: (r: string) => {
        if (!r) return <Tag>待检</Tag>;
        return <Tag color={resultColors[r]}>{resultLabels[r] || r}</Tag>;
      },
    },
    {
      title: '检验员',
      dataIndex: 'inspector_id',
      key: 'inspector_id',
      width: 100,
      render: (id: string) => id?.slice(0, 8) || '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: Inspection) => (
        <Button
          type="link"
          size="small"
          onClick={() => { setCurrentInsp(record); setDrawerVisible(true); }}
        >
          详情
        </Button>
      ),
    },
  ];

  const detail = inspDetail || currentInsp;
  const canComplete = detail?.status !== 'completed';

  return (
    <div>
      <Card
        title="来料检验"
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
            <Select
              placeholder="结果"
              allowClear
              style={{ width: 110 }}
              options={[
                { value: 'passed', label: '合格' },
                { value: 'failed', label: '不合格' },
                { value: 'conditional', label: '条件放行' },
              ]}
              value={filterResult}
              onChange={(v) => { setFilterResult(v); setPage(1); }}
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
              onClick={() => queryClient.invalidateQueries({ queryKey: ['srm-inspections'] })}
            >
              刷新
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={data?.items || []}
          rowKey="id"
          loading={isLoading}
          scroll={{ x: 950 }}
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
            onClick: () => { setCurrentInsp(record); setDrawerVisible(true); },
            style: { cursor: 'pointer' },
          })}
        />
      </Card>

      {/* 详情抽屉 */}
      <Drawer
        title={detail?.inspection_code || '检验详情'}
        open={drawerVisible}
        onClose={() => { setDrawerVisible(false); setCurrentInsp(null); completeForm.resetFields(); }}
        width={600}
      >
        {detail && (
          <>
            <Descriptions column={2} bordered size="small" style={{ marginBottom: 24 }}>
              <Descriptions.Item label="检验编码">{detail.inspection_code}</Descriptions.Item>
              <Descriptions.Item label="物料">{detail.material_name}</Descriptions.Item>
              <Descriptions.Item label="物料编码">{detail.material_code || '-'}</Descriptions.Item>
              <Descriptions.Item label="数量">{detail.quantity ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="抽样数">{detail.sample_qty ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={statusColors[detail.status]}>{statusLabels[detail.status] || detail.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="结果">
                {detail.result ? (
                  <Tag color={resultColors[detail.result]}>{resultLabels[detail.result] || detail.result}</Tag>
                ) : (
                  '待检'
                )}
              </Descriptions.Item>
              <Descriptions.Item label="检验时间">
                {detail.inspected_at ? dayjs(detail.inspected_at).format('YYYY-MM-DD HH:mm') : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="备注" span={2}>{detail.notes || '-'}</Descriptions.Item>
              <Descriptions.Item label="检验项目" span={2}>
                <pre style={{ margin: 0, fontSize: 12, maxHeight: 200, overflow: 'auto' }}>
                  {detail.inspection_items ? JSON.stringify(detail.inspection_items, null, 2) : '无'}
                </pre>
              </Descriptions.Item>
            </Descriptions>

            {canComplete && (
              <>
                <h4>完成检验</h4>
                <Form
                  form={completeForm}
                  layout="vertical"
                  onFinish={(values) => {
                    let items = undefined;
                    if (values.inspection_items_json) {
                      try {
                        items = JSON.parse(values.inspection_items_json);
                      } catch {
                        message.error('检验项目JSON格式错误');
                        return;
                      }
                    }
                    completeMutation.mutate({
                      result: values.result,
                      notes: values.notes,
                      inspection_items: items,
                    });
                  }}
                >
                  <Form.Item name="result" label="检验结果" rules={[{ required: true, message: '请选择结果' }]}>
                    <Select
                      placeholder="选择结果"
                      options={[
                        { value: 'passed', label: '合格' },
                        { value: 'failed', label: '不合格' },
                        { value: 'conditional', label: '条件放行' },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item name="notes" label="备注">
                    <Input.TextArea rows={2} placeholder="检验备注" />
                  </Form.Item>
                  <Form.Item name="inspection_items_json" label="检验项目 (JSON)">
                    <Input.TextArea rows={4} placeholder='[{"item": "外观", "result": "OK"}, ...]' />
                  </Form.Item>
                  <Button
                    type="primary"
                    htmlType="submit"
                    icon={<CheckCircleOutlined />}
                    loading={completeMutation.isPending}
                  >
                    完成检验
                  </Button>
                </Form>
              </>
            )}
          </>
        )}
      </Drawer>
    </div>
  );
};

export default Inspections;
