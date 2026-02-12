import React, { useState, useCallback, useMemo, useRef, useEffect } from 'react';
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
  Descriptions,
  Table,
  Alert,
  Card,
  Progress,
  Tooltip,
} from 'antd';
import {
  UploadOutlined,
  CheckCircleOutlined,
  UserOutlined,
  ArrowLeftOutlined,
  FolderOutlined,
  FileExcelOutlined,
} from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { projectApi, Task } from '@/api/projects';
import { taskFormApi, TaskFormField, TaskForm } from '@/api/taskForms';
import { userApi, User } from '@/api/users';
import { workflowApi } from '@/api/workflow';
import { taskRoleApi, TaskRole } from '@/constants/roles';
import BOMEditableTable, { type BOMItemRecord } from '@/components/BOMEditableTable';

const { Text, Title } = Typography;

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

// ========== UserTag Component (Feishu-style) ==========

const UserTag: React.FC<{ name?: string; avatarUrl?: string }> = ({ name, avatarUrl }) => {
  if (!name) return <Text type="secondary" style={{ fontSize: 11 }}>-</Text>;
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 4,
      background: '#f0f0f0', borderRadius: 12, padding: '2px 8px 2px 2px',
    }}>
      <Avatar size={20} src={avatarUrl} style={{ background: '#1677ff', fontSize: 11, flexShrink: 0 }}>
        {name[0]}
      </Avatar>
      <span style={{ fontSize: 12, lineHeight: '20px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: 60 }}>{name}</span>
    </span>
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

// ========== BOM Upload Field ==========

const BOMUploadField: React.FC<{
  value?: { filename: string; items: BOMItemRecord[]; item_count: number };
  onChange?: (value: { filename: string; items: BOMItemRecord[]; item_count: number } | undefined) => void;
  bomType?: 'EBOM' | 'SBOM';
  onSaveDraft?: () => void;
}> = ({ value, onChange, bomType = 'EBOM', onSaveDraft }) => {
  const [parsing, setParsing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const items: BOMItemRecord[] = value?.items || [];

  const emitChange = (newItems: BOMItemRecord[], filename?: string) => {
    onChange?.({
      filename: filename || value?.filename || '手动录入',
      items: newItems,
      item_count: newItems.length,
    });
    // Directly trigger draft save, bypassing unreliable Form.onFieldsChange for custom components
    onSaveDraft?.();
  };

  const handleImport = async (file: File) => {
    setError(null);
    setParsing(true);
    try {
      const result = await taskFormApi.parseBOM(file);
      const parsed = result.items || [];
      emitChange(parsed, file.name);
    } catch (err: any) {
      const errMsg = err?.response?.data?.error || err?.response?.data?.message || '解析BOM文件失败';
      setError(errMsg);
    } finally {
      setParsing(false);
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Space size={8}>
          <Upload
            accept=".rep,.xlsx,.xls"
            showUploadList={false}
            beforeUpload={(file) => { handleImport(file); return false; }}
            disabled={parsing}
          >
            <Button size="small" icon={<UploadOutlined />} loading={parsing}>导入模板</Button>
          </Upload>
          {value?.filename && value.filename !== '手动录入' && (
            <span style={{ fontSize: 12, color: '#999' }}>
              <FileExcelOutlined style={{ color: '#52c41a', marginRight: 4 }} />{value.filename}
            </span>
          )}
        </Space>
      </div>
      {error && <Alert message={error} type="error" showIcon style={{ marginBottom: 8 }} closable onClose={() => setError(null)} />}
      <BOMEditableTable
        bomType={bomType}
        items={items}
        onChange={(newItems) => emitChange(newItems)}
      />
    </div>
  );
};

// ========== BOM Data Display (read-only) ==========

const BOMDataDisplay: React.FC<{ data: { filename: string; items: BOMItemRecord[]; item_count: number }; bomType?: 'EBOM' | 'SBOM' }> = ({ data, bomType = 'EBOM' }) => {
  const [expanded, setExpanded] = useState(false);

  const categoryStats = useMemo(() => {
    if (!data?.items?.length) return [];
    const map: Record<string, number> = {};
    for (const item of data.items) {
      const cat = item.category || '未分类';
      map[cat] = (map[cat] || 0) + 1;
    }
    return Object.entries(map).sort((a, b) => b[1] - a[1]);
  }, [data?.items]);

  const ebomColumns = [
    { title: '序号', dataIndex: 'item_number', width: 50, align: 'center' as const },
    { title: '位号', dataIndex: 'reference', width: 80, ellipsis: true },
    { title: '名称', dataIndex: 'name', width: 100, ellipsis: true },
    { title: '规格', dataIndex: 'specification', width: 120, ellipsis: true },
    { title: '数量', dataIndex: 'quantity', width: 50, align: 'center' as const },
    { title: '单位', dataIndex: 'unit', width: 45, align: 'center' as const },
    { title: '类别', dataIndex: 'category', width: 70, ellipsis: true },
    { title: '制造商', dataIndex: 'manufacturer', width: 90, ellipsis: true },
  ];

  const sbomColumns = [
    { title: '序号', dataIndex: 'item_number', width: 50, align: 'center' as const },
    { title: '名称', dataIndex: 'name', width: 100, ellipsis: true },
    { title: '规格', dataIndex: 'specification', width: 120, ellipsis: true },
    { title: '数量', dataIndex: 'quantity', width: 50, align: 'center' as const },
    { title: '单位', dataIndex: 'unit', width: 45, align: 'center' as const },
    { title: '材质', dataIndex: 'material_type', width: 80, ellipsis: true },
    { title: '工艺类型', dataIndex: 'process_type', width: 80, ellipsis: true },
    { title: '图纸编号', dataIndex: 'drawing_no', width: 90, ellipsis: true },
    { title: '重量(g)', dataIndex: 'weight_grams', width: 70, align: 'right' as const },
    { title: '目标价', dataIndex: 'target_price', width: 80, align: 'right' as const,
      render: (v: number) => v != null ? `¥${Number(v).toFixed(2)}` : '-' },
  ];

  const columns = bomType === 'SBOM' ? sbomColumns : ebomColumns;

  return (
    <div>
      <Space size={8} style={{ marginBottom: 4 }}>
        <FileExcelOutlined style={{ color: '#52c41a' }} />
        <span style={{ fontSize: 13 }}>{data.filename}</span>
        <Tag color="blue">{data.item_count} 项物料</Tag>
        <Tag>{bomType === 'SBOM' ? '结构BOM' : '电子BOM'}</Tag>
        <Button type="link" size="small" onClick={() => setExpanded(!expanded)} style={{ padding: 0 }}>
          {expanded ? '收起' : '展开明细'}
        </Button>
      </Space>
      {expanded && (
        <div style={{ marginTop: 8 }}>
          {categoryStats.length > 0 && (
            <div style={{ marginBottom: 6, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
              {categoryStats.map(([cat, count]) => (
                <Tag key={cat} style={{ fontSize: 11 }}>{cat}: {count}</Tag>
              ))}
            </div>
          )}
          <Table
            columns={columns}
            dataSource={data.items}
            rowKey="item_number"
            size="small"
            pagination={data.items.length > 10 ? { pageSize: 10, size: 'small' } : false}
            scroll={{ x: bomType === 'SBOM' ? 800 : 600 }}
          />
        </div>
      )}
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
          // bom_upload: render as expandable BOM data
          if (field.type === 'bom_upload' && value && typeof value === 'object' && value.items) {
            return (
              <Descriptions.Item key={field.key} label={field.label}>
                <BOMDataDisplay data={value} bomType={field.bom_type || 'EBOM'} />
              </Descriptions.Item>
            );
          }
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

// ========== Task Detail Full Page ==========

const TaskDetailView: React.FC<{
  task: Task;
  onBack: () => void;
  onRefresh: () => void;
}> = ({ task, onBack, onRefresh }) => {
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

  const draftTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const draftSavingRef = useRef(false);

  useEffect(() => {
    setFormLoading(true);
    form.resetFields();
    setFileListMap({});
    Promise.all([
      taskFormApi.getForm(task.project_id, task.id),
      task.status === 'in_progress' ? taskFormApi.getDraft(task.id) : Promise.resolve(null),
    ]).then(([f, draft]) => {
      setTaskForm(f);
      if (draft?.data && f?.fields?.length) {
        form.setFieldsValue(draft.data);
      }
    }).catch(() => setTaskForm(null))
      .finally(() => setFormLoading(false));
    return () => {
      if (draftTimerRef.current) clearTimeout(draftTimerRef.current);
    };
  }, [task.project_id, task.id]);

  const saveDraftDebounced = useCallback(() => {
    console.log('[Draft] saveDraftDebounced called');
    if (draftTimerRef.current) clearTimeout(draftTimerRef.current);
    draftTimerRef.current = setTimeout(async () => {
      console.log('[Draft] setTimeout fired, draftSaving:', draftSavingRef.current);
      if (draftSavingRef.current) return;
      draftSavingRef.current = true;
      try {
        const values = form.getFieldsValue();
        console.log('[Draft] form values:', JSON.stringify(values).slice(0, 200));
        console.log('[Draft] task.id:', task.id);
        await taskFormApi.saveDraft(task.id, values);
        console.log('[Draft] save success');
      } catch (err: any) {
        const status = err?.response?.status;
        console.error('[Draft Save] failed:', status, err?.response?.data || err);
        if (status === 401) {
          message.warning('登录已过期，请刷新页面');
        } else {
          message.warning('草稿保存失败');
        }
      } finally { draftSavingRef.current = false; }
    }, 1000);
  }, [task.id, form]);

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
      case 'bom_upload': return <BOMUploadField bomType={field.bom_type || 'EBOM'} onSaveDraft={saveDraftDebounced} />;
      default: return <Input placeholder={field.placeholder} />;
    }
  };

  const cfg = statusConfig[task.status] || statusConfig.pending;
  const hasForm = !!(taskForm?.fields && taskForm.fields.length > 0);
  const isInProgress = task.status === 'in_progress';
  const showFormData = task.status === 'submitted' || task.status === 'completed' || task.status === 'reviewing';
  const projectName = (task as any).project?.name || (task as any).project_name || '-';
  const remainDays = task.due_date ? dayjs(task.due_date).diff(dayjs(), 'day') : null;

  return (
    <div style={{ padding: 24 }}>
      <Button type="link" icon={<ArrowLeftOutlined />} onClick={onBack} style={{ padding: 0, marginBottom: 16 }}>
        返回任务列表
      </Button>

      {/* Task header */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16 }}>
          <div>
            <Title level={4} style={{ margin: 0 }}>
              {task.title}
              <Tag color={cfg.tagColor} style={{ marginLeft: 8, verticalAlign: 'middle' }}>{cfg.text}</Tag>
            </Title>
            <Space size={8} style={{ marginTop: 4 }}>
              {(task.code || task.task_code) && <Text code style={{ fontSize: 12 }}>{task.code || task.task_code}</Text>}
              <Text type="secondary" style={{ fontSize: 12 }}><FolderOutlined style={{ marginRight: 4 }} />{projectName}</Text>
            </Space>
          </div>
          {/* Action buttons */}
          <Space>
            {task.status === 'pending' && (
              <Button type="primary" loading={starting} onClick={handleStart}
                style={{ background: '#52c41a', borderColor: '#52c41a' }}>
                开始任务
              </Button>
            )}
            {task.status === 'in_progress' && (
              <Button type="primary" loading={submitting} onClick={handleSubmit}
                icon={<CheckCircleOutlined />}
                style={{ background: '#52c41a', borderColor: '#52c41a' }}>
                {hasForm ? '提交表单' : '提交任务'}
              </Button>
            )}
            {task.status === 'rejected' && (
              <Button type="primary" loading={starting} onClick={handleStart}
                style={{ background: '#fa8c16', borderColor: '#fa8c16' }}>
                重新开始
              </Button>
            )}
          </Space>
        </div>

        <Descriptions column={3} size="small" bordered>
          <Descriptions.Item label="负责人">
            <UserTag name={task.assignee?.name || (task as any).assignee_name} avatarUrl={task.assignee?.avatar_url} />
          </Descriptions.Item>
          <Descriptions.Item label="创建人">
            <UserTag name={task.creator?.name} avatarUrl={task.creator?.avatar_url} />
          </Descriptions.Item>
          <Descriptions.Item label="剩余天数">
            <span style={{ fontWeight: 600, color: remainDays == null ? '#999' : remainDays < 0 ? '#ff4d4f' : remainDays <= 3 ? '#fa8c16' : '#52c41a' }}>
              {remainDays != null ? `${remainDays}天` : '-'}
            </span>
          </Descriptions.Item>
          <Descriptions.Item label="开始日期">
            {task.start_date ? dayjs(task.start_date).format('YYYY-MM-DD') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="截止日期">
            {task.due_date ? dayjs(task.due_date).format('YYYY-MM-DD') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="进度">
            <Progress percent={task.progress} size="small" style={{ width: 120 }} />
          </Descriptions.Item>
        </Descriptions>

        {task.description && (
          <div style={{ marginTop: 12 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>{task.description}</Text>
          </div>
        )}
      </Card>

      {/* Form area - full width */}
      <Card>
        {formLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : hasForm && isInProgress ? (
          <div>
            <Title level={5} style={{ marginBottom: 16 }}>任务表单</Title>
            {taskForm!.description && (
              <div style={{ marginBottom: 12, color: '#666', fontSize: 13 }}>{taskForm!.description}</div>
            )}
            <Form form={form} layout="vertical" preserve={false} onFieldsChange={saveDraftDebounced}>
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
        ) : (
          <Empty description="该任务暂无表单" />
        )}
      </Card>
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

  // Table columns for task list
  const taskColumns: ColumnsType<Task> = [
    {
      title: '任务标题', dataIndex: 'title', ellipsis: true,
      render: (title: string, task: Task) => {
        const cfg = statusConfig[task.status] || statusConfig.pending;
        return (
          <Space size={6}>
            <Tag color={cfg.tagColor} style={{ fontSize: 10, padding: '0 4px', margin: 0, lineHeight: '18px' }}>{cfg.text}</Tag>
            <Tooltip title={title}>
              <span style={{
                fontWeight: 500, fontSize: 13,
                color: task.status === 'completed' ? '#999' : '#333',
                textDecoration: task.status === 'completed' ? 'line-through' : 'none',
              }}>
                {title}
              </span>
            </Tooltip>
          </Space>
        );
      },
    },
    {
      title: '负责人', dataIndex: 'assignee', width: 120, align: 'center',
      render: (_: any, task: Task) => (
        <UserTag name={task.assignee?.name || (task as any).assignee_name} avatarUrl={task.assignee?.avatar_url} />
      ),
    },
    {
      title: '剩余天数', dataIndex: 'due_date', width: 85, align: 'center',
      render: (dueDate: string) => {
        if (!dueDate) return <Text type="secondary" style={{ fontSize: 12 }}>-</Text>;
        const days = dayjs(dueDate).diff(dayjs(), 'day');
        return (
          <span style={{ fontWeight: 500, fontSize: 12,
            color: days < 0 ? '#ff4d4f' : days <= 3 ? '#fa8c16' : '#52c41a' }}>
            {days}
          </span>
        );
      },
    },
    {
      title: '开始日期', dataIndex: 'start_date', width: 90, align: 'center',
      render: (d: string) => d ? <Text style={{ fontSize: 12 }}>{dayjs(d).format('MM-DD')}</Text> : <Text type="secondary" style={{ fontSize: 12 }}>-</Text>,
    },
    {
      title: '截止日期', dataIndex: 'due_date', width: 90, align: 'center',
      key: 'due_date_display',
      render: (d: string) => d ? <Text style={{ fontSize: 12 }}>{dayjs(d).format('MM-DD')}</Text> : <Text type="secondary" style={{ fontSize: 12 }}>-</Text>,
    },
    {
      title: '创建人', dataIndex: 'creator', width: 120, align: 'center',
      render: (_: any, task: Task) => (
        <UserTag name={task.creator?.name} avatarUrl={task.creator?.avatar_url} />
      ),
    },
  ];

  // If a task is selected, show full-page detail
  if (selectedTask) {
    return (
      <TaskDetailView
        task={selectedTask}
        onBack={() => setSelectedTaskId(null)}
        onRefresh={handleRefresh}
      />
    );
  }

  return (
    <div style={{ padding: 24 }}>
      {/* Header with filter tabs */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space size={8}>
          <Title level={4} style={{ margin: 0 }}>我的任务</Title>
          <Text type="secondary">{total} 条</Text>
        </Space>
        <Space size={4}>
          {filterItems.map((item) => (
            <Button
              key={item.key}
              type={activeFilter === item.key ? 'primary' : 'default'}
              size="small"
              onClick={() => { setActiveFilter(item.key); setPage(1); }}
            >
              {item.label}
            </Button>
          ))}
        </Space>
      </div>

      {/* Task Table */}
      <Table<Task>
        columns={taskColumns}
        dataSource={tasks}
        rowKey="id"
        size="small"
        loading={isLoading}
        locale={{ emptyText: <Empty description="暂无任务" /> }}
        pagination={total > pageSize ? {
          current: page,
          pageSize,
          total,
          onChange: (p) => setPage(p),
          size: 'small',
          showTotal: (t) => `共 ${t} 条`,
        } : false}
        onRow={(task) => ({
          onClick: () => setSelectedTaskId(task.id),
          style: { cursor: 'pointer' },
        })}
      />
    </div>
  );
};

export default MyTasks;
