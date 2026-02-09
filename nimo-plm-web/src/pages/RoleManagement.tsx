import React, { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Input,
  Button,
  Table,
  Tag,
  Modal,
  Form,
  Space,
  Dropdown,
  Avatar,
  Checkbox,
  Tree,
  message,
  Spin,
  Empty,
  Popconfirm,
} from 'antd';
import {
  SearchOutlined,
  PlusOutlined,
  MoreOutlined,
  TeamOutlined,
  UserOutlined,
  DeleteOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import {
  roleApi,
  departmentApi,
  Role,
  RoleMember,
  DepartmentTreeNode,
  DepartmentUser,
} from '@/api/roles';

const RoleManagement: React.FC = () => {
  const queryClient = useQueryClient();
  const [selectedRoleId, setSelectedRoleId] = useState<string>('');
  const [roleSearch, setRoleSearch] = useState('');
  const [memberSearch, setMemberSearch] = useState('');
  const [selectedMemberIds, setSelectedMemberIds] = useState<string[]>([]);
  const [addMemberOpen, setAddMemberOpen] = useState(false);
  const [roleModalOpen, setRoleModalOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [roleForm] = Form.useForm();

  // ========== Queries ==========
  const { data: roles = [], isLoading: rolesLoading } = useQuery({
    queryKey: ['roles'],
    queryFn: roleApi.list,
  });

  const selectedRole = roles.find((r) => r.id === selectedRoleId);

  // Auto-select first role
  React.useEffect(() => {
    if (roles.length > 0 && !selectedRoleId) {
      setSelectedRoleId(roles[0].id);
    }
  }, [roles, selectedRoleId]);

  const { data: members = [], isLoading: membersLoading } = useQuery({
    queryKey: ['role-members', selectedRoleId],
    queryFn: () => roleApi.listMembers(selectedRoleId),
    enabled: !!selectedRoleId,
  });

  // ========== Mutations ==========
  const createRoleMut = useMutation({
    mutationFn: (data: { code: string; name: string }) => roleApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['roles'] });
      message.success('角色创建成功');
      setRoleModalOpen(false);
      roleForm.resetFields();
    },
    onError: () => message.error('角色创建失败'),
  });

  const updateRoleMut = useMutation({
    mutationFn: ({ id, data }: { id: string; data: { code?: string; name?: string } }) =>
      roleApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['roles'] });
      message.success('角色更新成功');
      setRoleModalOpen(false);
      setEditingRole(null);
      roleForm.resetFields();
    },
    onError: () => message.error('角色更新失败'),
  });

  const deleteRoleMut = useMutation({
    mutationFn: (id: string) => roleApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['roles'] });
      message.success('角色删除成功');
      if (selectedRoleId === editingRole?.id) setSelectedRoleId('');
    },
    onError: () => message.error('角色删除失败'),
  });

  const addMembersMut = useMutation({
    mutationFn: (userIds: string[]) => roleApi.addMembers(selectedRoleId, userIds),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['role-members', selectedRoleId] });
      message.success('添加成员成功');
      setAddMemberOpen(false);
    },
    onError: () => message.error('添加成员失败'),
  });

  const removeMembersMut = useMutation({
    mutationFn: (userIds: string[]) => roleApi.removeMembers(selectedRoleId, userIds),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['role-members', selectedRoleId] });
      setSelectedMemberIds([]);
      message.success('移除成员成功');
    },
    onError: () => message.error('移除成员失败'),
  });

  // ========== Filtered data ==========
  const filteredRoles = useMemo(() => {
    if (!roleSearch) return roles;
    return roles.filter((r) => r.name.includes(roleSearch) || r.code.includes(roleSearch));
  }, [roles, roleSearch]);

  const filteredMembers = useMemo(() => {
    if (!memberSearch) return members;
    return members.filter((m) => m.name.includes(memberSearch));
  }, [members, memberSearch]);

  // ========== Handlers ==========
  const handleRoleSubmit = () => {
    roleForm.validateFields().then((values) => {
      if (editingRole) {
        updateRoleMut.mutate({ id: editingRole.id, data: values });
      } else {
        createRoleMut.mutate(values);
      }
    });
  };

  const openEditRole = (role: Role) => {
    setEditingRole(role);
    roleForm.setFieldsValue({ code: role.code, name: role.name });
    setRoleModalOpen(true);
  };

  const openCreateRole = () => {
    setEditingRole(null);
    roleForm.resetFields();
    setRoleModalOpen(true);
  };

  // ========== Member table columns ==========
  const memberColumns = [
    {
      title: '姓名',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: RoleMember) => (
        <Space>
          <Avatar size="small" src={record.avatar_url} icon={<UserOutlined />} />
          {name}
        </Space>
      ),
    },
    {
      title: '电子邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '所属部门',
      dataIndex: 'department_name',
      key: 'department_name',
      render: (v: string) => v || '-',
    },
  ];

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 56px)', background: '#f5f5f5' }}>
      {/* ===== 左侧栏 ===== */}
      <div
        style={{
          width: 280,
          minWidth: 280,
          background: '#fff',
          borderRight: '1px solid #f0f0f0',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {/* 搜索 */}
        <div style={{ padding: '16px 16px 8px' }}>
          <Input
            placeholder="搜索角色"
            prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
            value={roleSearch}
            onChange={(e) => setRoleSearch(e.target.value)}
            allowClear
          />
        </div>

        {/* 角色列表 */}
        <div style={{ flex: 1, overflow: 'auto', padding: '0 8px' }}>
          {rolesLoading ? (
            <div style={{ textAlign: 'center', paddingTop: 40 }}>
              <Spin />
            </div>
          ) : filteredRoles.length === 0 ? (
            <Empty description="暂无角色" style={{ marginTop: 40 }} />
          ) : (
            filteredRoles.map((role) => (
              <div
                key={role.id}
                onClick={() => setSelectedRoleId(role.id)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  padding: '10px 12px',
                  borderRadius: 6,
                  cursor: 'pointer',
                  marginBottom: 2,
                  background: selectedRoleId === role.id ? '#e6f7ff' : 'transparent',
                  transition: 'background 0.2s',
                }}
              >
                <TeamOutlined style={{ fontSize: 16, color: '#8c8c8c', marginRight: 10 }} />
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <span
                      style={{
                        fontSize: 14,
                        fontWeight: selectedRoleId === role.id ? 500 : 400,
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {role.name}
                    </span>
                    {role.is_system && (
                      <Tag color="blue" style={{ fontSize: 11, lineHeight: '18px', padding: '0 4px', margin: 0 }}>
                        系统
                      </Tag>
                    )}
                  </div>
                </div>
                <Dropdown
                  trigger={['click']}
                  menu={{
                    items: [
                      { key: 'edit', label: '编辑', onClick: () => openEditRole(role) },
                      {
                        key: 'delete',
                        label: '删除',
                        danger: true,
                        onClick: () => {
                          Modal.confirm({
                            title: '确认删除',
                            content: `确定要删除角色「${role.name}」吗？`,
                            onOk: () => deleteRoleMut.mutate(role.id),
                          });
                        },
                      },
                    ],
                  }}
                >
                  <MoreOutlined
                    onClick={(e) => e.stopPropagation()}
                    style={{ fontSize: 16, color: '#8c8c8c', padding: 4 }}
                  />
                </Dropdown>
              </div>
            ))
          )}
        </div>

        {/* 新增角色按钮 */}
        <div style={{ padding: 16, borderTop: '1px solid #f0f0f0' }}>
          <Button type="dashed" block icon={<PlusOutlined />} onClick={openCreateRole}>
            新增角色
          </Button>
        </div>
      </div>

      {/* ===== 右侧面板 ===== */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {selectedRole ? (
          <>
            {/* 头部 */}
            <div
              style={{
                padding: '20px 24px 16px',
                background: '#fff',
                borderBottom: '1px solid #f0f0f0',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <h2 style={{ margin: 0, fontSize: 20 }}>{selectedRole.name}</h2>
                  <span style={{ color: '#8c8c8c', fontSize: 13 }}>角色编码：{selectedRole.code}</span>
                </div>
                <Space>
                  <TeamOutlined style={{ fontSize: 16, color: '#8c8c8c' }} />
                  <span style={{ color: '#8c8c8c' }}>{members.length} 名成员</span>
                </Space>
              </div>
            </div>

            {/* 操作栏 */}
            <div
              style={{
                padding: '12px 24px',
                background: '#fff',
                display: 'flex',
                alignItems: 'center',
                gap: 12,
              }}
            >
              <Input
                placeholder="搜索成员姓名"
                prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
                style={{ width: 240 }}
                value={memberSearch}
                onChange={(e) => setMemberSearch(e.target.value)}
                allowClear
              />
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddMemberOpen(true)}>
                添加成员
              </Button>
              {selectedMemberIds.length > 0 && (
                <Popconfirm
                  title={`确定移除选中的 ${selectedMemberIds.length} 名成员？`}
                  onConfirm={() => removeMembersMut.mutate(selectedMemberIds)}
                >
                  <Button danger icon={<DeleteOutlined />}>
                    移除成员 ({selectedMemberIds.length})
                  </Button>
                </Popconfirm>
              )}
            </div>

            {/* 成员表格 */}
            <div style={{ flex: 1, padding: '0 24px 24px', overflow: 'auto' }}>
              <Table
                rowKey="user_id"
                columns={memberColumns}
                dataSource={filteredMembers}
                loading={membersLoading}
                pagination={false}
                size="middle"
                rowSelection={{
                  selectedRowKeys: selectedMemberIds,
                  onChange: (keys) => setSelectedMemberIds(keys as string[]),
                }}
                locale={{ emptyText: <Empty description="暂无成员" /> }}
              />
            </div>
          </>
        ) : (
          <div
            style={{
              flex: 1,
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              background: '#fff',
            }}
          >
            <Empty description="请在左侧选择一个角色" />
          </div>
        )}
      </div>

      {/* ===== 新增/编辑角色弹窗 ===== */}
      <Modal
        title={editingRole ? '编辑角色' : '新增角色'}
        open={roleModalOpen}
        onOk={handleRoleSubmit}
        onCancel={() => {
          setRoleModalOpen(false);
          setEditingRole(null);
          roleForm.resetFields();
        }}
        confirmLoading={createRoleMut.isPending || updateRoleMut.isPending}
      >
        <Form form={roleForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="code" label="角色编码" rules={[{ required: true, message: '请输入角色编码' }]}>
            <Input placeholder="如 project_manager" disabled={!!editingRole} />
          </Form.Item>
          <Form.Item name="name" label="角色名称" rules={[{ required: true, message: '请输入角色名称' }]}>
            <Input placeholder="如 项目经理" />
          </Form.Item>
        </Form>
      </Modal>

      {/* ===== 添加成员弹窗 ===== */}
      {addMemberOpen && (
        <AddMemberModal
          open={addMemberOpen}
          existingMemberIds={members.map((m) => m.user_id)}
          onOk={(userIds) => addMembersMut.mutate(userIds)}
          onCancel={() => setAddMemberOpen(false)}
          confirmLoading={addMembersMut.isPending}
        />
      )}
    </div>
  );
};

// ==========================================
// 添加成员弹窗组件
// ==========================================
interface AddMemberModalProps {
  open: boolean;
  existingMemberIds: string[];
  onOk: (userIds: string[]) => void;
  onCancel: () => void;
  confirmLoading: boolean;
}

const AddMemberModal: React.FC<AddMemberModalProps> = ({
  open,
  existingMemberIds,
  onOk,
  onCancel,
  confirmLoading,
}) => {
  const [search, setSearch] = useState('');
  const [selectedUsers, setSelectedUsers] = useState<Map<string, DepartmentUser>>(new Map());
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);

  const { data: deptTree = [], isLoading } = useQuery({
    queryKey: ['department-tree'],
    queryFn: departmentApi.tree,
    enabled: open,
  });

  // Auto-expand first level
  React.useEffect(() => {
    if (deptTree.length > 0 && expandedKeys.length === 0) {
      setExpandedKeys(deptTree.map((d) => d.id));
    }
  }, [deptTree, expandedKeys.length]);

  // Collect all users for search
  const allUsers = useMemo(() => {
    const users: (DepartmentUser & { deptName: string })[] = [];
    const traverse = (nodes: DepartmentTreeNode[]) => {
      for (const node of nodes) {
        for (const u of node.users) {
          users.push({ ...u, deptName: node.name });
        }
        traverse(node.children);
      }
    };
    traverse(deptTree);
    return users;
  }, [deptTree]);

  const filteredUsers = useMemo(() => {
    if (!search) return null; // null means show tree view
    return allUsers.filter(
      (u) => u.name.includes(search) || (u.email && u.email.includes(search))
    );
  }, [allUsers, search]);

  const toggleUser = (user: DepartmentUser) => {
    setSelectedUsers((prev) => {
      const next = new Map(prev);
      if (next.has(user.id)) {
        next.delete(user.id);
      } else {
        next.set(user.id, user);
      }
      return next;
    });
  };

  const removeUser = (userId: string) => {
    setSelectedUsers((prev) => {
      const next = new Map(prev);
      next.delete(userId);
      return next;
    });
  };

  const handleOk = () => {
    const userIds = Array.from(selectedUsers.keys());
    if (userIds.length === 0) {
      message.warning('请至少选择一名用户');
      return;
    }
    onOk(userIds);
  };

  // Build tree data for antd Tree
  const treeData = useMemo(() => {
    const buildTree = (nodes: DepartmentTreeNode[]): any[] => {
      return nodes.map((node) => ({
        title: node.name,
        key: `dept-${node.id}`,
        selectable: false,
        children: [
          ...node.users.map((u) => ({
            title: (
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '2px 0',
                  cursor: 'pointer',
                }}
                onClick={() => toggleUser(u)}
              >
                <Checkbox
                  checked={selectedUsers.has(u.id)}
                  disabled={existingMemberIds.includes(u.id)}
                />
                <Avatar size="small" src={u.avatar_url} icon={<UserOutlined />} />
                <span>{u.name}</span>
                {existingMemberIds.includes(u.id) && (
                  <Tag color="default" style={{ fontSize: 11, margin: 0 }}>
                    已添加
                  </Tag>
                )}
              </div>
            ),
            key: `user-${u.id}`,
            isLeaf: true,
            selectable: false,
          })),
          ...buildTree(node.children),
        ],
      }));
    };
    return buildTree(deptTree);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [deptTree, selectedUsers, existingMemberIds]);

  return (
    <Modal
      title="添加成员"
      open={open}
      onOk={handleOk}
      onCancel={onCancel}
      confirmLoading={confirmLoading}
      width={720}
      styles={{ body: { padding: 0 } }}
    >
      <div style={{ display: 'flex', height: 480, borderTop: '1px solid #f0f0f0' }}>
        {/* 左侧: 部门树 + 用户列表 */}
        <div
          style={{
            flex: 1,
            borderRight: '1px solid #f0f0f0',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          <div style={{ padding: '12px 16px 8px' }}>
            <Input
              placeholder="搜索用户"
              prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              allowClear
            />
          </div>
          <div style={{ flex: 1, overflow: 'auto', padding: '0 8px' }}>
            {isLoading ? (
              <div style={{ textAlign: 'center', paddingTop: 40 }}>
                <Spin />
              </div>
            ) : filteredUsers !== null ? (
              // 搜索模式: 平铺用户列表
              filteredUsers.length === 0 ? (
                <Empty description="未找到用户" style={{ marginTop: 40 }} />
              ) : (
                filteredUsers.map((u) => (
                  <div
                    key={u.id}
                    onClick={() => !existingMemberIds.includes(u.id) && toggleUser(u)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      padding: '8px 12px',
                      borderRadius: 4,
                      cursor: existingMemberIds.includes(u.id) ? 'not-allowed' : 'pointer',
                      opacity: existingMemberIds.includes(u.id) ? 0.5 : 1,
                    }}
                  >
                    <Checkbox
                      checked={selectedUsers.has(u.id)}
                      disabled={existingMemberIds.includes(u.id)}
                    />
                    <Avatar size="small" src={u.avatar_url} icon={<UserOutlined />} />
                    <div>
                      <div>{u.name}</div>
                      <div style={{ fontSize: 12, color: '#8c8c8c' }}>{u.deptName}</div>
                    </div>
                  </div>
                ))
              )
            ) : (
              // 树模式
              <Tree
                treeData={treeData}
                expandedKeys={expandedKeys}
                onExpand={(keys) => setExpandedKeys(keys)}
                selectable={false}
                blockNode
              />
            )}
          </div>
        </div>

        {/* 右侧: 已选用户列表 */}
        <div
          style={{
            width: 240,
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              padding: '12px 16px',
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              borderBottom: '1px solid #f0f0f0',
            }}
          >
            <span style={{ fontWeight: 500 }}>已选：{selectedUsers.size} 名用户</span>
            {selectedUsers.size > 0 && (
              <Button type="link" size="small" onClick={() => setSelectedUsers(new Map())}>
                清空
              </Button>
            )}
          </div>
          <div style={{ flex: 1, overflow: 'auto', padding: 8 }}>
            {selectedUsers.size === 0 ? (
              <Empty description="暂未选择" style={{ marginTop: 40 }} />
            ) : (
              Array.from(selectedUsers.values()).map((u) => (
                <div
                  key={u.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '6px 8px',
                    borderRadius: 4,
                  }}
                >
                  <Space size={8}>
                    <Avatar size="small" src={u.avatar_url} icon={<UserOutlined />} />
                    <span>{u.name}</span>
                  </Space>
                  <CloseOutlined
                    onClick={() => removeUser(u.id)}
                    style={{ fontSize: 12, color: '#8c8c8c', cursor: 'pointer' }}
                  />
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </Modal>
  );
};

export default RoleManagement;
