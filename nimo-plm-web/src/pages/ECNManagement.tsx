import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Select,
  Modal,
  Input,
  message,
  Typography,
  Descriptions,
  Alert,
} from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, EyeOutlined } from '@ant-design/icons';
import { bomEcnApi } from '@/api/bomEcn';
import type { BOMECN } from '@/api/projectBom';
import type { ColumnsType } from 'antd/es/table';

const { Text, Title } = Typography;

const STATUS_CONFIG: Record<string, { color: string; text: string }> = {
  pending: { color: 'processing', text: '待审批' },
  approved: { color: 'success', text: '已批准' },
  rejected: { color: 'error', text: '已拒绝' },
};

const ECNManagement: React.FC = () => {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [selectedECN, setSelectedECN] = useState<BOMECN | null>(null);
  const [rejectModalOpen, setRejectModalOpen] = useState(false);
  const [rejectNote, setRejectNote] = useState('');

  // Fetch ECN list
  const { data, isLoading } = useQuery({
    queryKey: ['bom-ecns', statusFilter],
    queryFn: () => bomEcnApi.list({ status: statusFilter || undefined }),
  });

  const ecnList = data?.items || [];

  // Approve ECN
  const approveMutation = useMutation({
    mutationFn: (id: string) => bomEcnApi.approve(id),
    onSuccess: () => {
      message.success('ECN已批准');
      queryClient.invalidateQueries({ queryKey: ['bom-ecns'] });
      setDetailModalOpen(false);
      setSelectedECN(null);
    },
    onError: () => message.error('操作失败'),
  });

  // Reject ECN
  const rejectMutation = useMutation({
    mutationFn: ({ id, note }: { id: string; note: string }) => bomEcnApi.reject(id, { note }),
    onSuccess: () => {
      message.success('ECN已拒绝');
      queryClient.invalidateQueries({ queryKey: ['bom-ecns'] });
      setRejectModalOpen(false);
      setRejectNote('');
      setDetailModalOpen(false);
      setSelectedECN(null);
    },
    onError: () => message.error('操作失败'),
  });

  const handleViewDetail = (ecn: BOMECN) => {
    setSelectedECN(ecn);
    setDetailModalOpen(true);
  };

  const handleApprove = () => {
    if (selectedECN) {
      Modal.confirm({
        title: '确认批准此ECN？',
        content: 'BOM将应用此次变更，版本号自动升级',
        onOk: () => approveMutation.mutate(selectedECN.id),
      });
    }
  };

  const handleReject = () => {
    setRejectModalOpen(true);
  };

  const columns: ColumnsType<BOMECN> = [
    {
      title: 'ECN编号',
      dataIndex: 'ecn_number',
      key: 'ecn_number',
      width: 150,
      render: (text) => <Text code>{text}</Text>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '关联BOM',
      dataIndex: ['bom', 'name'],
      key: 'bom_name',
      render: (_, record) => record.bom?.name || record.bom_id,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={STATUS_CONFIG[status]?.color}>
          {STATUS_CONFIG[status]?.text || status}
        </Tag>
      ),
    },
    {
      title: '创建人',
      dataIndex: ['creator', 'name'],
      key: 'creator',
      width: 100,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (date) => new Date(date).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      fixed: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => handleViewDetail(record)}
        >
          查看
        </Button>
      ),
    },
  ];

  const renderChangeSummary = (summary: Record<string, any>) => {
    const { added = [], removed = [], modified = [] } = summary;

    return (
      <div style={{ marginTop: 16 }}>
        <Title level={5}>变更摘要</Title>

        {added.length > 0 && (
          <Alert
            message={`新增 ${added.length} 项`}
            type="success"
            showIcon
            style={{ marginBottom: 8 }}
          />
        )}

        {removed.length > 0 && (
          <Alert
            message={`删除 ${removed.length} 项`}
            type="error"
            showIcon
            style={{ marginBottom: 8 }}
          />
        )}

        {modified.length > 0 && (
          <Alert
            message={`修改 ${modified.length} 项`}
            type="warning"
            showIcon
            style={{ marginBottom: 8 }}
          />
        )}

        {modified.length > 0 && (
          <div style={{ marginTop: 12 }}>
            <Text strong>修改详情：</Text>
            {modified.slice(0, 5).map((item: any, idx: number) => (
              <div key={idx} style={{ marginTop: 8, paddingLeft: 16 }}>
                <Text type="secondary">• {item.changes?.join(', ')}</Text>
              </div>
            ))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div style={{ padding: 24 }}>
      <Card>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Title level={4} style={{ margin: 0 }}>ECN管理</Title>
          <Select
            placeholder="筛选状态"
            allowClear
            style={{ width: 150 }}
            value={statusFilter || undefined}
            onChange={(value) => setStatusFilter(value || '')}
            options={[
              { label: '待审批', value: 'pending' },
              { label: '已批准', value: 'approved' },
              { label: '已拒绝', value: 'rejected' },
            ]}
          />
        </div>

        <Table
          columns={columns}
          dataSource={ecnList}
          loading={isLoading}
          rowKey="id"
          pagination={{ pageSize: 20 }}
        />
      </Card>

      {/* ECN Detail Modal */}
      <Modal
        title={selectedECN?.ecn_number}
        open={detailModalOpen}
        onCancel={() => {
          setDetailModalOpen(false);
          setSelectedECN(null);
        }}
        width={800}
        footer={
          selectedECN?.status === 'pending' ? (
            <Space>
              <Button onClick={() => setDetailModalOpen(false)}>关闭</Button>
              <Button
                danger
                icon={<CloseCircleOutlined />}
                onClick={handleReject}
              >
                拒绝
              </Button>
              <Button
                type="primary"
                icon={<CheckCircleOutlined />}
                onClick={handleApprove}
                loading={approveMutation.isPending}
              >
                批准
              </Button>
            </Space>
          ) : (
            <Button onClick={() => setDetailModalOpen(false)}>关闭</Button>
          )
        }
      >
        {selectedECN && (
          <>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="ECN编号" span={2}>
                <Text code>{selectedECN.ecn_number}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="标题" span={2}>
                {selectedECN.title}
              </Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>
                {selectedECN.description || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={STATUS_CONFIG[selectedECN.status]?.color}>
                  {STATUS_CONFIG[selectedECN.status]?.text}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="创建人">
                {selectedECN.creator?.name || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">
                {new Date(selectedECN.created_at).toLocaleString('zh-CN')}
              </Descriptions.Item>
              {selectedECN.approved_at && (
                <>
                  <Descriptions.Item label="批准人">
                    {selectedECN.approver?.name || '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="批准时间">
                    {new Date(selectedECN.approved_at).toLocaleString('zh-CN')}
                  </Descriptions.Item>
                </>
              )}
              {selectedECN.rejected_at && (
                <>
                  <Descriptions.Item label="拒绝人">
                    {selectedECN.rejecter?.name || '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="拒绝时间">
                    {new Date(selectedECN.rejected_at).toLocaleString('zh-CN')}
                  </Descriptions.Item>
                  <Descriptions.Item label="拒绝原因" span={2}>
                    {selectedECN.rejection_note || '-'}
                  </Descriptions.Item>
                </>
              )}
            </Descriptions>

            {renderChangeSummary(selectedECN.change_summary)}
          </>
        )}
      </Modal>

      {/* Reject Modal */}
      <Modal
        title="拒绝ECN"
        open={rejectModalOpen}
        onCancel={() => {
          setRejectModalOpen(false);
          setRejectNote('');
        }}
        onOk={() => {
          if (selectedECN) {
            rejectMutation.mutate({ id: selectedECN.id, note: rejectNote });
          }
        }}
        confirmLoading={rejectMutation.isPending}
        okText="确认拒绝"
        okButtonProps={{ danger: true }}
      >
        <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
          请说明拒绝原因，以便提交者了解并重新修改。
        </Text>
        <Input.TextArea
          rows={4}
          placeholder="请输入拒绝原因..."
          value={rejectNote}
          onChange={(e) => setRejectNote(e.target.value)}
        />
      </Modal>
    </div>
  );
};

export default ECNManagement;
