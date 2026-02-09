import React, { useState, useCallback, useMemo } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useSSE } from '@/hooks/useSSE';
import {
  Tag,
  Button,
  Form,
  Input,
  InputNumber,
  Select,
  DatePicker,
  Checkbox,
  Upload,
  Typography,
  Space,
  message,
  Avatar,
  Empty,
  Spin,
  Divider,
  Descriptions,
} from 'antd';
import {
  UploadOutlined,
  CheckCircleOutlined,
  UserOutlined,
  ClockCircleOutlined,
  CloseOutlined,
  CalendarOutlined,
  FolderOutlined,
  CheckOutlined,
} from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload';
import dayjs from 'dayjs';
import { projectApi, Task } from '@/api/projects';
import { taskFormApi, TaskFormField, TaskForm } from '@/api/taskForms';
import { userApi, User } from '@/api/users';
import { workflowApi } from '@/api/workflow';
import { taskRoleApi, TaskRole } from '@/constants/roles';

const { Text } = Typography;

// ========== Status Configuration ==========

const statusConfig: Record<string, { color: string; text: string; tagColor: string }> = {
  pending:     { color: '#bfbfbf', text: '待开始', tagColor: 'default' },
  in_progress: { color: '#1677ff', text: '进行中', tagColor: 'blue' },
  submitted:   { color: '#fa8c16', text: '已提交', tagColor: 'orange' },
  completed:   { color: '#52c41a', text: '已完成', tagColor: 'green' },
  reviewing:   { color: '#faad14', text: '审批中', tagColor: 'gold' },
  rejected:    { color: '#ff4d4f', text: '已驳回', tagColor: 'red' },
};

const filterItems = [
  { key: 'all',         label: '全部任务' },
  { key: 'pending',     label: '待开始' },
  { key: 'in_progress', label: '进行中' },
  { key: 'submitted',   label: '已提交' },
  { key: 'completed',   label: '已完成' },
  { key: 'rejected',    label: '已驳回' },
];

// ========== Status Circle ==========

const StatusCircle: React.FC<{ status: string; size?: number }> = ({ status, size = 18 }) => {
  const cfg = statusConfig[status] || statusConfig.pending;
  if (status === 'completed') {
    return (
      <span style={{
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        width: size, height: size, borderRadius: '50%', background: cfg.color, flexShrink: 0,
      }}>
        <CheckOutlined style={{ color: '#fff', fontSize: size * 0.55 }} />
      </span>
    );
  }
  const isFilled = status === 'in_progress';
  return (
    <span style={{
      display: 'inline-block', width: size, height: size, borderRadius: '50%',
      border: `2px solid ${cfg.color}`,
      background: isFilled ? cfg.color : 'transparent',
      flexShrink: 0, boxSizing: 'border-box',
    }} />
  );
};

// ========== Role Assignment Field ==========

const RoleAssignmentField: React.FC<{
  value?: Record<string, string>;
  onChange?: (value: Record<string, string>) => void;
  allUsers: User[];
  projectId?: string;
}> = ({ value = {}, onChange, allUsers, projectId }) => {
  const { data: taskRoles = [] } = useQuery<TaskRole[]>({
    queryKey: ['task-roles'],
    queryFn: () => taskRoleApi.list(),
  });

  const { data: projectTasks = [] } = useQuery({
    queryKey: ['project-tasks', projectId],
    queryFn: () => projectApi.listTasks(projectId!),
    enabled: !!projectId,
  });

  const roleLabelMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const r of taskRoles) map[r.code] = r.name;
    return map;
  }, [taskRoles]);

  const displayRoles = useMemo(() => {
    if (projectId && projectTasks.length > 0) {
      const uniqueCodes = new Set<string>();
      for (const t of projectTasks) {
        const role = (t as any).default_assignee_role;
        if (role) uniqueCodes.add(role);
      }
      return Array.from(uniqueCodes).sort().map(code => ({ code, label: roleLabelMap[code] || code }));
    }
    return taskRoles.map(r => ({ code: r.code, label: r.name }));
  }, [projectId, projectTasks, taskRoles, roleLabelMap]);

  const handleChange = (roleCode: string, userId: string) => {
    onChange?.({ ...value, [roleCode]: userId });
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {displayRoles.map((role) => (
        <div key={role.code} style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <div style={{ width: 90, fontWeight: 500, fontSize: 13, flexShrink: 0 }}>{role.label}</div>
          <Select
            showSearch optionFilterProp="children" placeholder="选择负责人"
            value={value[role.code] || undefined}
            onChange={(val) => handleChange(role.code, val)}
            allowClear
            onClear={() => { const next = { ...value }; delete next[role.code]; onChange?.(next); }}
            style={{ flex: 1 }}
          >
            {allUsers.map((u) => (
              <Select.Option key={u.id} value={u.id}>
                <Space size={8}>
                  <Avatar size={20} src={u.avatar_url} icon={<UserOutlined />}>{u.name?.[0]}</Avatar>
                  <span>{u.name}</span>
                </Space>
              </Select.Option>
            ))}
          </Select>
        </div>
      ))}
    </div>
  );
};

// ========== Form Data Display (read-only for submitted/completed) ==========

const FormDataDisplay: React.FC<{ projectId: string; taskId: string }> = ({ projectId, taskId }) => {
  const [formDef, setFormDef] = useState<any>(null);
  const [submission, setSubmission] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [userMap, setUserMap] = useState<Record<string, string>>({});

  React.useEffect(() => {
    setLoading(true);
    Promise.all([
      taskFormApi.getForm(projectId, taskId),
      taskFormApi.getSubmission(projectId, taskId),
    ]).then(([form, sub]) => {
      setFormDef(form);
      setSubmission(sub);
      if (form?.fields?.some((f: any) => f.type === 'user' || f.type === 'role_assignment')) {
        userApi.list().then((users) => {
          const map: Record<string, string> = {};
          users.forEach((u) => { map[u.id] = u.name; });
          setUserMap(map);
        });
      }
    }).catch(() => {}).finally(() => setLoading(false));
  }, [projectId, taskId]);

  if (loading) return <div style={{ color: '#999', fontSize: 12, padding: 8 }}>加载表单数据...</div>;
  if (!formDef || !submission) return null;

  const fields = formDef.fields || [];
  const data = submission.data || {};

  return (
    <div>
      <Text strong style={{ fontSize: 14, marginBottom: 12, display: 'block' }}>已提交的表单</Text>
      <Descriptions size="small" column={1} bordered>
        {fields.map((field: any) => {
          let value = data[field.key];
          if (value === undefined || value === null) value = '-';
          else if (field.type === 'role_assignment' && typeof value === 'object' && !Array.isArray(value)) {
            value = Object.entries(value as Record<string, string>)
              .map(([code, uid]) => `${code}: ${userMap[uid] || uid}`)
              .join('; ') || '-';
          }
          else if (field.type === 'user') value = userMap[value] || value;
          else if (typeof value === 'boolean') value = value ? '是' : '否';
          else if (Array.isArray(value)) {
            if (value.length > 0 && typeof value[0] === 'object' && value[0].filename) {
              value = value.map((f: any) => f.filename).join(', ');
            } else {
              value = value.join(', ');
            }
          }
          return (
            <Descriptions.Item key={field.key} label={field.label}>
              {String(value)}
            </Descriptions.Item>
          );
        })}
      </Descriptions>
      {submission.submitted_at && (
        <Text type="secondary" style={{ fontSize: 11, marginTop: 8, display: 'block' }}>
          提交时间: {dayjs(submission.submitted_at).format('YYYY-MM-DD HH:mm')}
        </Text>
      )}
    </div>
  );
};

// ========== Task Detail Panel ==========

const TaskDetailPanel: React.FC<{
  task: Task;
  onClose: () => void;
  onRefresh: () => void;
}> = ({ task, onClose, onRefresh }) => {
  const [taskForm, setTaskForm] = useState<TaskForm | null>(null);
  const [formLoading, setFormLoading] = useState(true);
  const [fileListMap, setFileListMap] = useState<Record<string, UploadFile[]>>({});
  const [submitting, setSubmitting] = useState(false);
  const [starting, setStarting] = useState(false);
  const [form] = Form.useForm();

  const { data: allUsers = [] } = useQuery<User[]>({
    queryKey: ['users'],
    queryFn: () => userApi.list(),
  });

  React.useEffect(() => {
    setFormLoading(true);
    form.resetFields();
    setFileListMap({});
    taskFormApi.getForm(task.project_id, task.id)
      .then((f) => setTaskForm(f))
      .catch(() => setTaskForm(null))
      .finally(() => setFormLoading(false));
  }, [task.project_id, task.id]);

  const handleError = (err: any) => {
    const status = err?.response?.status;
    const errMsg = err?.response?.data?.error || err?.response?.data?.message || '操作失败';
    if (status === 400) {
      message.error(`前置任务未完成，${errMsg}`);
    } else {
      message.error(errMsg);
    }
  };

  const handleStart = async () => {
    setStarting(true);
    try {
      await workflowApi.startTask(task.project_id, task.id);
      message.success('任务已开始');
      onRefresh();
    } catch (err) {
      handleError(err);
    } finally {
      setStarting(false);
    }
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      let formData: Record<string, any> | undefined;
      if (taskForm?.fields?.length) {
        const values = await form.validateFields();
        for (const field of taskForm.fields) {
          if (field.type === 'file' && fileListMap[field.key]) {
            const uploaded: any[] = [];
            for (const f of fileListMap[field.key]) {
              if (f.response) uploaded.push(f.response);
              else if (f.originFileObj) {
                try {
                  const result = await taskFormApi.uploadFile(f.originFileObj);
                  uploaded.push(result);
                } catch {
                  message.error(`文件 ${f.name} 上传失败`);
                  setSubmitting(false);
                  return;
                }
              }
            }
            values[field.key] = uploaded;
          }
          if (field.type === 'date' && values[field.key]) {
            values[field.key] = dayjs(values[field.key]).format('YYYY-MM-DD');
          }
        }
        formData = values;
      }
      await taskFormApi.completeMyTask(task.id, formData);
      message.success('任务已提交');
      onRefresh();
    } catch (err: any) {
      if (err?.errorFields) return;
      handleError(err);
    } finally {
      setSubmitting(false);
    }
  };

  const renderFormField = (field: TaskFormField) => {
    switch (field.type) {
      case 'text': return <Input placeholder={field.placeholder} />;
      case 'textarea': return <Input.TextArea rows={3} placeholder={field.placeholder} />;
      case 'number': return <InputNumber style={{ width: '100%' }} placeholder={field.placeholder} />;
      case 'select': return (
        <Select placeholder={field.placeholder}>
          {(field.options || []).map((opt) => (
            <Select.Option key={opt} value={opt}>{opt}</Select.Option>
          ))}
        </Select>
      );
      case 'multiselect': return (
        <Select mode="multiple" placeholder={field.placeholder}>
          {(field.options || []).map((opt) => (
            <Select.Option key={opt} value={opt}>{opt}</Select.Option>
          ))}
        </Select>
      );
      case 'date': return <DatePicker style={{ width: '100%' }} />;
      case 'file': return (
        <Upload
          fileList={fileListMap[field.key] || []}
          onChange={({ fileList }) => setFileListMap((prev) => ({ ...prev, [field.key]: fileList }))}
          beforeUpload={() => false}
          multiple={field.multiple}
          accept={field.accept}
        >
          <Button icon={<UploadOutlined />}>选择文件</Button>
        </Upload>
      );
      case 'checkbox': return <Checkbox>{field.description || field.label}</Checkbox>;
      case 'user': return (
        <Select showSearch optionFilterProp="children" placeholder="请选择用户">
          {allUsers.map((u) => (
            <Select.Option key={u.id} value={u.id}>
              <Space size={8}>
                <Avatar size={20} src={u.avatar_url} icon={<UserOutlined />}>{u.name?.[0]}</Avatar>
                <span>{u.name}</span>
              </Space>
            </Select.Option>
          ))}
        </Select>
      );
      case 'role_assignment': return <RoleAssignmentField allUsers={allUsers} projectId={task.project_id} />;
      default: return <Input placeholder={field.placeholder} />;
    }
  };

  const cfg = statusConfig[task.status] || statusConfig.pending;
  const hasForm = !!(taskForm?.fields && taskForm.fields.length > 0);
  const isInProgress = task.status === 'in_progress';
  const showFormData = task.status === 'submitted' || task.status === 'completed' || task.status === 'reviewing';
  const projectName = (task as any).project?.name || (task as any).project_name || '-';
  const assigneeName = task.assignee?.name || (task as any).assignee_name;

  return (
    <div style={{
      width: 480, borderLeft: '1px solid #e8e8e8',
      display: 'flex', flexDirection: 'column', background: '#fff', flexShrink: 0,
    }}>
      {/* Header */}
      <div style={{
        padding: '16px 20px', borderBottom: '1px solid #f0f0f0',
        display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start',
      }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 18, fontWeight: 600, marginBottom: 8, wordBreak: 'break-word', lineHeight: 1.4 }}>
            {task.title}
          </div>
          <Space size={8}>
            <Tag color={cfg.tagColor}>{cfg.text}</Tag>
            {(task.code || task.task_code) && <Text code style={{ fontSize: 12 }}>{task.code || task.task_code}</Text>}
          </Space>
        </div>
        <Button type="text" icon={<CloseOutlined />} onClick={onClose} style={{ flexShrink: 0, marginLeft: 8 }} />
      </div>

      {/* Scrollable content */}
      <div style={{ flex: 1, overflow: 'auto', padding: '16px 20px' }}>
        {/* Task info fields */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14, marginBottom: 20 }}>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <Text type="secondary" style={{ width: 70, flexShrink: 0, fontSize: 13 }}>负责人</Text>
            {assigneeName ? (
              <Space size={8}>
                <Avatar size={24} src={task.assignee?.avatar_url} icon={<UserOutlined />}>{assigneeName[0]}</Avatar>
                <span style={{ fontSize: 13 }}>{assigneeName}</span>
              </Space>
            ) : <Text type="secondary" style={{ fontSize: 13 }}>未指派</Text>}
          </div>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <Text type="secondary" style={{ width: 70, flexShrink: 0, fontSize: 13 }}>截止日期</Text>
            {task.due_date ? (
              <Space size={6}>
                <CalendarOutlined style={{ color: dayjs(task.due_date).isBefore(dayjs(), 'day') ? '#ff4d4f' : '#999', fontSize: 13 }} />
                <Text type={dayjs(task.due_date).isBefore(dayjs(), 'day') ? 'danger' : undefined} style={{ fontSize: 13 }}>
                  {dayjs(task.due_date).format('YYYY-MM-DD')}
                </Text>
              </Space>
            ) : <Text type="secondary" style={{ fontSize: 13 }}>无</Text>}
          </div>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <Text type="secondary" style={{ width: 70, flexShrink: 0, fontSize: 13 }}>所属项目</Text>
            <Space size={6}>
              <FolderOutlined style={{ color: '#1677ff', fontSize: 13 }} />
              <span style={{ fontSize: 13 }}>{projectName}</span>
            </Space>
          </div>
        </div>

        <Divider style={{ margin: '0 0 16px 0' }} />

        {/* Form area for in_progress tasks */}
        {formLoading ? (
          <div style={{ textAlign: 'center', padding: 20 }}><Spin size="small" /></div>
        ) : hasForm && isInProgress ? (
          <div>
            <Text strong style={{ fontSize: 14, marginBottom: 12, display: 'block' }}>任务表单</Text>
            {taskForm!.description && (
              <div style={{ marginBottom: 12, color: '#666', fontSize: 12 }}>{taskForm!.description}</div>
            )}
            <Form form={form} layout="vertical" preserve={false} size="small">
              {taskForm!.fields.map((field) => (
                <Form.Item
                  key={field.key}
                  name={field.key}
                  label={field.label}
                  rules={field.type !== 'checkbox' && field.required ? [{ required: true, message: `请填写${field.label}` }] : undefined}
                  valuePropName={field.type === 'checkbox' ? 'checked' : 'value'}
                  extra={field.type !== 'checkbox' ? field.description : undefined}
                >
                  {renderFormField(field)}
                </Form.Item>
              ))}
            </Form>
          </div>
        ) : showFormData ? (
          <FormDataDisplay projectId={task.project_id} taskId={task.id} />
        ) : null}
      </div>

      {/* Footer actions */}
      <div style={{ padding: '12px 20px', borderTop: '1px solid #f0f0f0' }}>
        {task.status === 'pending' && (
          <Button type="primary" block size="large" loading={starting} onClick={handleStart}
            style={{ background: '#52c41a', borderColor: '#52c41a' }}>
            开始任务
          </Button>
        )}
        {task.status === 'in_progress' && (
          <Button type="primary" block size="large" loading={submitting} onClick={handleSubmit}
            icon={<CheckCircleOutlined />}
            style={{ background: '#52c41a', borderColor: '#52c41a' }}>
            {hasForm ? '提交表单' : '提交任务'}
          </Button>
        )}
        {task.status === 'submitted' && (
          <div style={{ textAlign: 'center', padding: 8, color: '#fa8c16', fontSize: 14 }}>
            <ClockCircleOutlined style={{ marginRight: 8 }} />已提交，等待审批/确认
          </div>
        )}
        {task.status === 'reviewing' && (
          <div style={{ textAlign: 'center', padding: 8, color: '#faad14', fontSize: 14 }}>
            <ClockCircleOutlined style={{ marginRight: 8 }} />审批中
          </div>
        )}
        {task.status === 'completed' && (
          <div style={{ textAlign: 'center', padding: 8, color: '#52c41a', fontSize: 14 }}>
            <CheckCircleOutlined style={{ marginRight: 8 }} />已完成
          </div>
        )}
        {task.status === 'rejected' && (
          <Button type="primary" block size="large" loading={starting} onClick={handleStart}
            style={{ background: '#fa8c16', borderColor: '#fa8c16' }}>
            重新开始
          </Button>
        )}
      </div>
    </div>
  );
};

// ========== Main Component ==========

const MyTasks: React.FC = () => {
  const queryClient = useQueryClient();
  const [activeFilter, setActiveFilter] = useState('all');
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const pageSize = 50;

  const statusParam = activeFilter === 'all' ? undefined : activeFilter;
  const { data, isLoading } = useQuery({
    queryKey: ['my-tasks', statusParam, page, pageSize],
    queryFn: () => projectApi.myTasks({ status: statusParam, page, page_size: pageSize }),
  });

  const tasks = data?.items || [];
  const total = data?.pagination?.total || 0;

  const selectedTask = useMemo(
    () => tasks.find((t) => t.id === selectedTaskId) || null,
    [tasks, selectedTaskId],
  );

  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['my-tasks'] });
  }, [queryClient]);

  useSSE({
    onTaskUpdate: useCallback(() => {
      queryClient.invalidateQueries({ queryKey: ['my-tasks'] });
    }, [queryClient]),
    onMyTaskUpdate: useCallback(() => {
      queryClient.invalidateQueries({ queryKey: ['my-tasks'] });
    }, [queryClient]),
  });

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 64px)', background: '#fff' }}>
      {/* ===== Left Sidebar ===== */}
      <div style={{
        width: 200, borderRight: '1px solid #f0f0f0', flexShrink: 0,
        background: '#fafafa', display: 'flex', flexDirection: 'column',
      }}>
        <div style={{ padding: '20px 16px 12px', fontWeight: 600, fontSize: 16, color: '#333' }}>
          我的任务
        </div>
        <div style={{ flex: 1 }}>
          {filterItems.map((item) => {
            const isActive = activeFilter === item.key;
            return (
              <div
                key={item.key}
                onClick={() => { setActiveFilter(item.key); setPage(1); setSelectedTaskId(null); }}
                style={{
                  padding: '10px 16px', cursor: 'pointer',
                  background: isActive ? '#e6f4ff' : 'transparent',
                  color: isActive ? '#1677ff' : '#555',
                  fontWeight: isActive ? 500 : 400,
                  borderRight: isActive ? '3px solid #1677ff' : '3px solid transparent',
                  fontSize: 13, transition: 'all 0.15s',
                }}
              >
                {item.label}
              </div>
            );
          })}
        </div>
      </div>

      {/* ===== Middle: Task List ===== */}
      <div style={{ flex: 1, minWidth: 280, display: 'flex', flexDirection: 'column' }}>
        {/* List header */}
        <div style={{
          padding: '16px 20px', borderBottom: '1px solid #f0f0f0',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          flexShrink: 0,
        }}>
          <Space size={8}>
            <Text strong style={{ fontSize: 15 }}>
              {filterItems.find((f) => f.key === activeFilter)?.label || '全部任务'}
            </Text>
            <Text type="secondary" style={{ fontSize: 13 }}>{total} 条</Text>
          </Space>
        </div>

        {/* Column headers */}
        <div style={{
          padding: '8px 20px', borderBottom: '1px solid #f0f0f0',
          display: 'flex', alignItems: 'center', fontSize: 12, color: '#999', flexShrink: 0,
        }}>
          <span style={{ width: 30 }} />
          <span style={{ flex: 1 }}>任务标题</span>
          <span style={{ width: 120, textAlign: 'right' }}>负责人</span>
        </div>

        {/* Task list */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          {isLoading ? (
            <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
          ) : tasks.length === 0 ? (
            <Empty description="暂无任务" style={{ padding: 60 }} />
          ) : (
            <>
              {tasks.map((task) => {
                const isSelected = task.id === selectedTaskId;
                const assigneeName = task.assignee?.name || (task as any).assignee_name;
                const projectName = (task as any).project?.name || (task as any).project_name;
                const isOverdue = task.due_date && dayjs(task.due_date).isBefore(dayjs(), 'day') && task.status !== 'completed';

                return (
                  <div
                    key={task.id}
                    onClick={() => setSelectedTaskId(task.id)}
                    style={{
                      padding: '10px 20px', cursor: 'pointer',
                      background: isSelected ? '#e6f4ff' : 'transparent',
                      borderBottom: '1px solid #f5f5f5',
                      borderLeft: isSelected ? '3px solid #1677ff' : '3px solid transparent',
                      display: 'flex', alignItems: 'center', gap: 10,
                      transition: 'background 0.15s',
                    }}
                    onMouseEnter={(e) => { if (!isSelected) e.currentTarget.style.background = '#f8f9fa'; }}
                    onMouseLeave={(e) => { if (!isSelected) e.currentTarget.style.background = 'transparent'; }}
                  >
                    <StatusCircle status={task.status} />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{
                        fontSize: 13, fontWeight: 500,
                        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                        color: task.status === 'completed' ? '#999' : '#333',
                        textDecoration: task.status === 'completed' ? 'line-through' : 'none',
                      }}>
                        {task.title}
                      </div>
                      <div style={{
                        fontSize: 12, color: '#999', marginTop: 2,
                        display: 'flex', alignItems: 'center', gap: 8,
                        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                      }}>
                        {projectName && <span>{projectName}</span>}
                        {task.due_date && (
                          <span style={{ color: isOverdue ? '#ff4d4f' : '#bbb' }}>
                            {dayjs(task.due_date).format('MM-DD')}
                          </span>
                        )}
                      </div>
                    </div>
                    {assigneeName && (
                      <Avatar size={24} src={task.assignee?.avatar_url} icon={<UserOutlined />}
                        style={{ flexShrink: 0, fontSize: 11 }}>
                        {assigneeName[0]}
                      </Avatar>
                    )}
                  </div>
                );
              })}
              {/* Pagination */}
              {total > pageSize && (
                <div style={{
                  padding: '12px 20px', display: 'flex', justifyContent: 'center',
                  alignItems: 'center', gap: 12, borderTop: '1px solid #f0f0f0',
                }}>
                  <Button size="small" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>上一页</Button>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {page} / {Math.ceil(total / pageSize)}
                  </Text>
                  <Button size="small" disabled={page >= Math.ceil(total / pageSize)} onClick={() => setPage((p) => p + 1)}>下一页</Button>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* ===== Right: Task Detail Panel ===== */}
      {selectedTask && (
        <TaskDetailPanel
          task={selectedTask}
          onClose={() => setSelectedTaskId(null)}
          onRefresh={handleRefresh}
        />
      )}
    </div>
  );
};

export default MyTasks;
