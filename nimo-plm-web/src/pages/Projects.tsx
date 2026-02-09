import React from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  Table,
  Button,
  Tag,
  Progress,
  Typography,
  Badge,
  Space,
  App,
} from 'antd';
import {
  PlusOutlined,
  EyeOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { projectApi, Project } from '@/api/projects';
import type { ColumnsType } from 'antd/es/table';

const { Title, Text } = Typography;

const phaseColors: Record<string, string> = {
  concept: 'purple',
  evt: 'blue',
  dvt: 'cyan',
  pvt: 'orange',
  mp: 'green',
  CONCEPT: 'purple',
  EVT: 'blue',
  DVT: 'cyan',
  PVT: 'orange',
  MP: 'green',
};

const statusColors: Record<string, string> = {
  planning: 'default',
  active: 'processing',
  on_hold: 'warning',
  completed: 'success',
  cancelled: 'error',
};

const Projects: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { modal, message } = App.useApp();

  const { data, isLoading } = useQuery({
    queryKey: ['projects'],
    queryFn: () => projectApi.list(),
  });

  const handleDelete = (record: Project) => {
    modal.confirm({
      title: '删除项目',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除项目「${record.name}」吗？此操作不可恢复。`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await projectApi.delete(record.id);
          message.success('项目已删除');
          queryClient.invalidateQueries({ queryKey: ['projects'] });
        } catch (error) {
          message.error('删除失败');
        }
      },
    });
  };

  const columns: ColumnsType<Project> = [
    {
      title: '项目编码',
      dataIndex: 'code',
      key: 'code',
      width: 150,
      render: (text: string) => <Text code>{text}</Text>,
    },
    {
      title: '项目名称',
      dataIndex: 'name',
      key: 'name',
      width: 250,
    },
    {
      title: '阶段',
      dataIndex: 'phase',
      key: 'phase',
      width: 100,
      render: (phase: string) => (
        <Tag color={phaseColors[phase]}>{phase?.toUpperCase()}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Badge status={statusColors[status] as any} text={
          status === 'planning' ? '规划中' :
          status === 'active' ? '进行中' :
          status === 'completed' ? '已完成' :
          status === 'on_hold' ? '暂停' : status
        } />
      ),
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      width: 150,
      render: (progress: number) => (
        <Progress percent={progress} size="small" />
      ),
    },
    {
      title: '项目经理',
      dataIndex: 'manager_name',
      key: 'manager_name',
      width: 100,
      render: (name: string) => name || '-',
    },
    {
      title: '开始日期',
      dataIndex: 'start_date',
      key: 'start_date',
      width: 120,
      render: (date: string) => date ? new Date(date).toLocaleDateString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => navigate(`/projects/${record.id}`)}
          >
            详情
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Card>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
          <Title level={4} style={{ margin: 0 }}>研发项目</Title>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate('/templates')}
          >
            从研发流程创建项目
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={data?.items || []}
          rowKey="id"
          loading={isLoading}
          pagination={{
            total: data?.pagination?.total || 0,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </Card>
    </div>
  );
};

export default Projects;
