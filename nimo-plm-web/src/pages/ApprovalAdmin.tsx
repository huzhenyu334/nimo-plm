import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  Input,
  Button,
  Space,
  Tag,
  Modal,
  message,
  Typography,
  Empty,
  Popconfirm,
  Spin,
  List,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  SendOutlined,
  StopOutlined,
  FolderAddOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import {
  approvalDefinitionApi,
  approvalGroupApi,
  ApprovalDefinition,
} from '@/api/approvalDefinitions';

const { Title, Text } = Typography;

const ApprovalAdmin: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchText, setSearchText] = useState('');
  const [groupModalOpen, setGroupModalOpen] = useState(false);
  const [newGroupName, setNewGroupName] = useState('');

  const { data: defData, isLoading } = useQuery({
    queryKey: ['approval-definitions'],
    queryFn: approvalDefinitionApi.list,
  });

  const createGroupMutation = useMutation({
    mutationFn: (name: string) => approvalGroupApi.create(name),
    onSuccess: () => {
      message.success('分组创建成功');
      setGroupModalOpen(false);
      setNewGroupName('');
      queryClient.invalidateQueries({ queryKey: ['approval-definitions'] });
      queryClient.invalidateQueries({ queryKey: ['approval-groups'] });
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '创建失败');
    },
  });

  const deleteGroupMutation = useMutation({
    mutationFn: (id: string) => approvalGroupApi.delete(id),
    onSuccess: () => {
      message.success('分组已删除');
      queryClient.invalidateQueries({ queryKey: ['approval-definitions'] });
      queryClient.invalidateQueries({ queryKey: ['approval-groups'] });
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '删除失败');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => approvalDefinitionApi.delete(id),
    onSuccess: () => {
      message.success('审批已删除');
      queryClient.invalidateQueries({ queryKey: ['approval-definitions'] });
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '删除失败');
    },
  });

  const publishMutation = useMutation({
    mutationFn: (id: string) => approvalDefinitionApi.publish(id),
    onSuccess: () => {
      message.success('已发布');
      queryClient.invalidateQueries({ queryKey: ['approval-definitions'] });
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '发布失败');
    },
  });

  const unpublishMutation = useMutation({
    mutationFn: (id: string) => approvalDefinitionApi.unpublish(id),
    onSuccess: () => {
      message.success('已取消发布');
      queryClient.invalidateQueries({ queryKey: ['approval-definitions'] });
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '操作失败');
    },
  });

  const groups = defData?.groups || [];

  // Filter by search text
  const filteredGroups = groups
    .map((g: { name: string; definitions: ApprovalDefinition[] }) => ({
      ...g,
      definitions: g.definitions.filter((d: ApprovalDefinition) =>
        d.name.toLowerCase().includes(searchText.toLowerCase())
      ),
    }))
    .filter((g: { name: string; definitions: ApprovalDefinition[] }) =>
      searchText ? g.definitions.length > 0 : true
    );

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Title level={3} style={{ margin: 0 }}>审批管理后台</Title>
        <Space>
          <Input
            placeholder="搜索审批"
            prefix={<SearchOutlined />}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            style={{ width: 200 }}
            allowClear
          />
          <Button icon={<FolderAddOutlined />} onClick={() => setGroupModalOpen(true)}>
            新建分组
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/approval-editor/new')}>
            创建审批
          </Button>
        </Space>
      </div>

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : filteredGroups.length === 0 ? (
        <Card>
          <Empty description="暂无审批定义">
            <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/approval-editor/new')}>
              创建第一个审批
            </Button>
          </Empty>
        </Card>
      ) : (
        filteredGroups.map((group: { name: string; id?: string; definitions: ApprovalDefinition[] }) => (
          <Card
            key={group.name}
            title={
              <Space>
                <Text strong style={{ fontSize: 16 }}>{group.name || '未分组'}</Text>
                <Tag>{group.definitions.length}</Tag>
              </Space>
            }
            extra={
              group.name && (group as any).id ? (
                <Popconfirm
                  title="确定删除该分组？"
                  description="分组下的审批将移至未分组"
                  onConfirm={() => deleteGroupMutation.mutate((group as any).id)}
                >
                  <Button type="text" danger size="small" icon={<DeleteOutlined />}>
                    删除分组
                  </Button>
                </Popconfirm>
              ) : null
            }
            style={{ marginBottom: 16 }}
          >
            <List
              dataSource={group.definitions}
              locale={{ emptyText: <Empty description="该分组下暂无审批" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
              renderItem={(def: ApprovalDefinition) => (
                <List.Item
                  actions={[
                    <Tooltip title="编辑" key="edit">
                      <Button
                        type="text"
                        icon={<EditOutlined />}
                        onClick={() => navigate(`/approval-editor/${def.id}`)}
                      />
                    </Tooltip>,
                    def.status === 'published' ? (
                      <Tooltip title="取消发布" key="unpublish">
                        <Button
                          type="text"
                          icon={<StopOutlined />}
                          onClick={() => unpublishMutation.mutate(def.id)}
                        />
                      </Tooltip>
                    ) : (
                      <Tooltip title="发布" key="publish">
                        <Button
                          type="text"
                          style={{ color: '#52c41a' }}
                          icon={<SendOutlined />}
                          onClick={() => publishMutation.mutate(def.id)}
                        />
                      </Tooltip>
                    ),
                    <Popconfirm
                      key="delete"
                      title="确定删除该审批？"
                      onConfirm={() => deleteMutation.mutate(def.id)}
                    >
                      <Button type="text" danger icon={<DeleteOutlined />} />
                    </Popconfirm>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={
                      <div
                        style={{
                          width: 40,
                          height: 40,
                          borderRadius: 8,
                          background: '#1890ff',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: 20,
                          color: '#fff',
                        }}
                      >
                        {def.icon || <FileTextOutlined />}
                      </div>
                    }
                    title={
                      <Space>
                        <Text strong>{def.name}</Text>
                        <Tag color={def.status === 'published' ? 'green' : 'default'}>
                          {def.status === 'published' ? '已发布' : '草稿'}
                        </Tag>
                      </Space>
                    }
                    description={
                      <Space>
                        {def.description && <Text type="secondary">{def.description}</Text>}
                        <Text type="secondary">可见范围: {def.visibility === 'all' ? '全员' : '指定人员'}</Text>
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          </Card>
        ))
      )}

      {/* Create Group Modal */}
      <Modal
        title="新建分组"
        open={groupModalOpen}
        onCancel={() => { setGroupModalOpen(false); setNewGroupName(''); }}
        onOk={() => {
          if (!newGroupName.trim()) {
            message.warning('请输入分组名称');
            return;
          }
          createGroupMutation.mutate(newGroupName.trim());
        }}
        confirmLoading={createGroupMutation.isPending}
        okText="创建"
        cancelText="取消"
      >
        <Input
          placeholder="请输入分组名称"
          value={newGroupName}
          onChange={(e) => setNewGroupName(e.target.value)}
          onPressEnter={() => {
            if (newGroupName.trim()) createGroupMutation.mutate(newGroupName.trim());
          }}
        />
      </Modal>
    </div>
  );
};

export default ApprovalAdmin;
