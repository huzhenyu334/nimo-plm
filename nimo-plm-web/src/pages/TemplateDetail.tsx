import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  Button,
  Space,
  Tag,
  Typography,
  App,
  Badge,
  Input,
  Popconfirm,
  Switch,
  Select,
  InputNumber,
  Modal,
  Form,
  DatePicker,
  Spin,
  Tooltip,
  Alert,
  Timeline,
  Drawer,
} from 'antd';
import {
  ArrowLeftOutlined,
  SaveOutlined,
  RocketOutlined,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  CheckOutlined,
  CloseOutlined,
  UpOutlined,
  DownOutlined,
  SendOutlined,
  RollbackOutlined,
  HistoryOutlined,
  FormOutlined,
  LockOutlined,
} from '@ant-design/icons';
import { templateApi, TemplateTask, TemplateVersion } from '@/api/templates';
import { codenameApi, Codename } from '@/api/codenames';
import { approvalDefinitionApi } from '@/api/approvalDefinitions';
import { taskFormApi, TaskFormField } from '@/api/taskForms';
import UserSelect from '@/components/UserSelect';
import { taskRoleApi, TaskRole } from '@/constants/roles';
import { useAuth } from '@/contexts/AuthContext';

const { Title, Text } = Typography;

// ============ Constants ============

const PHASES = ['CONCEPT', 'EVT', 'DVT', 'PVT', 'MP'] as const;

const phaseColors: Record<string, string> = {
  CONCEPT: 'purple',
  EVT: 'blue',
  DVT: 'cyan',
  PVT: 'orange',
  MP: 'green',
};

const phaseLabels: Record<string, string> = {
  CONCEPT: 'Concept',
  EVT: 'EVT',
  DVT: 'DVT',
  PVT: 'PVT',
  MP: 'MP',
};

const TASK_TYPES = [
  { value: 'MILESTONE', label: '里程碑' },
  { value: 'TASK', label: '任务' },
  { value: 'SUBTASK', label: '子任务' },
];

// ROLES are now fetched from API (see rolesData query below)

// ============ Editable Cell Components ============

const EditableText: React.FC<{
  value: string;
  onChange: (val: string) => void;
  placeholder?: string;
  style?: React.CSSProperties;
  readOnly?: boolean;
}> = ({ value, onChange, placeholder, style, readOnly }) => {
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(value);
  const inputRef = useRef<any>(null);

  useEffect(() => {
    setText(value);
  }, [value]);

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus();
    }
  }, [editing]);

  if (readOnly) {
    return (
      <div
        style={{
          padding: '1px 7px',
          minHeight: 24,
          lineHeight: '24px',
          ...style,
        }}
      >
        {value || <span style={{ color: '#bfbfbf' }}>-</span>}
      </div>
    );
  }

  if (editing) {
    return (
      <Input
        ref={inputRef}
        size="small"
        value={text}
        placeholder={placeholder}
        style={{ ...style, minWidth: 80 }}
        onChange={(e) => setText(e.target.value)}
        onBlur={() => {
          setEditing(false);
          if (text !== value) onChange(text);
        }}
        onPressEnter={() => {
          setEditing(false);
          if (text !== value) onChange(text);
        }}
      />
    );
  }

  return (
    <div
      onClick={() => setEditing(true)}
      style={{
        cursor: 'pointer',
        padding: '1px 7px',
        borderRadius: 4,
        minHeight: 24,
        lineHeight: '24px',
        border: '1px solid transparent',
        ...style,
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = '#d9d9d9';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = 'transparent';
      }}
    >
      {value || <span style={{ color: '#bfbfbf' }}>{placeholder || '点击输入'}</span>}
    </div>
  );
};

const EditableSelect: React.FC<{
  value: string;
  onChange: (val: string) => void;
  options: { value: string; label: string }[];
  placeholder?: string;
  allowClear?: boolean;
  style?: React.CSSProperties;
  readOnly?: boolean;
}> = ({ value, onChange, options, placeholder, allowClear, style, readOnly }) => {
  const [editing, setEditing] = useState(false);
  const selectRef = useRef<any>(null);

  useEffect(() => {
    if (editing) {
      setTimeout(() => selectRef.current?.focus(), 50);
    }
  }, [editing]);

  if (readOnly) {
    const label = options.find((o) => o.value === value)?.label || value;
    return (
      <div
        style={{
          padding: '1px 7px',
          minHeight: 24,
          lineHeight: '24px',
          ...style,
        }}
      >
        {value ? label : <span style={{ color: '#bfbfbf' }}>-</span>}
      </div>
    );
  }

  if (editing) {
    return (
      <Select
        ref={selectRef}
        size="small"
        value={value || undefined}
        placeholder={placeholder}
        options={options}
        allowClear={allowClear}
        style={{ minWidth: 100, ...style }}
        onChange={(val) => {
          onChange(val || '');
          setEditing(false);
        }}
        onBlur={() => setEditing(false)}
        open={true}
      />
    );
  }

  const label = options.find((o) => o.value === value)?.label || value;
  return (
    <div
      onClick={() => setEditing(true)}
      style={{
        cursor: 'pointer',
        padding: '1px 7px',
        borderRadius: 4,
        minHeight: 24,
        lineHeight: '24px',
        border: '1px solid transparent',
        ...style,
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = '#d9d9d9';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = 'transparent';
      }}
    >
      {value ? label : <span style={{ color: '#bfbfbf' }}>{placeholder || '点击选择'}</span>}
    </div>
  );
};

const EditableNumber: React.FC<{
  value: number;
  onChange: (val: number) => void;
  min?: number;
  style?: React.CSSProperties;
  readOnly?: boolean;
}> = ({ value, onChange, min = 0, style, readOnly }) => {
  const [editing, setEditing] = useState(false);
  const inputRef = useRef<any>(null);

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus();
    }
  }, [editing]);

  if (readOnly) {
    return (
      <div
        style={{
          padding: '1px 7px',
          minHeight: 24,
          lineHeight: '24px',
          textAlign: 'center',
          ...style,
        }}
      >
        {value}
      </div>
    );
  }

  if (editing) {
    return (
      <InputNumber
        ref={inputRef}
        size="small"
        value={value}
        min={min}
        style={{ width: 70, ...style }}
        onChange={(val) => {
          if (val !== null && val !== value) onChange(val);
        }}
        onBlur={() => setEditing(false)}
        onPressEnter={() => setEditing(false)}
      />
    );
  }

  return (
    <div
      onClick={() => setEditing(true)}
      style={{
        cursor: 'pointer',
        padding: '1px 7px',
        borderRadius: 4,
        minHeight: 24,
        lineHeight: '24px',
        border: '1px solid transparent',
        textAlign: 'center',
        ...style,
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = '#d9d9d9';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = 'transparent';
      }}
    >
      {value}
    </div>
  );
};

// ============ Tree ordering helper ============

function buildTreeOrder(taskList: TaskRow[]): TaskRow[] {
  const childrenMap = new Map<string, TaskRow[]>();
  const topLevel: TaskRow[] = [];

  taskList.forEach((t) => {
    if (t.parent_task_code) {
      if (!childrenMap.has(t.parent_task_code)) childrenMap.set(t.parent_task_code, []);
      childrenMap.get(t.parent_task_code)!.push(t);
    } else {
      topLevel.push(t);
    }
  });

  const result: TaskRow[] = [];
  const addWithChildren = (task: TaskRow, depth: number) => {
    (task as any)._depth = depth;
    result.push(task);
    const children = childrenMap.get(task.task_code) || [];
    children.sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0));
    children.forEach((child) => addWithChildren(child, depth + 1));
  };

  topLevel.forEach((t) => addWithChildren(t, 0));
  return result;
}

// ============ Row type for local state ============

interface TaskRow extends Omit<TemplateTask, 'id' | 'template_id'> {
  _key: string; // local key for React
  _depth?: number; // tree depth level (set by buildTreeOrder)
  id?: string;
  template_id?: string;
  _depends_on?: string[]; // 前置任务 task_code 列表（本地状态）
}

// ============ Main Component ============

const TemplateDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const { message: msg, modal } = App.useApp();

  // Data state
  const [tasks, setTasks] = useState<TaskRow[]>([]);
  const [templateName, setTemplateName] = useState('');
  const [editingName, setEditingName] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);
  const [saving, setSaving] = useState(false);
  const [collapsedPhases, setCollapsedPhases] = useState<Record<string, boolean>>({});

  // Publish / Save-as-version state
  const [publishing, setPublishing] = useState(false);
  const [versionModalOpen, setVersionModalOpen] = useState(false);
  const [newVersionInput, setNewVersionInput] = useState('');
  const [versionSaving, setVersionSaving] = useState(false);

  // Version history drawer
  const [versionDrawerOpen, setVersionDrawerOpen] = useState(false);

  // Create project modal
  const [createProjectModalOpen, setCreateProjectModalOpen] = useState(false);
  const [codenames, setCodenames] = useState<Codename[]>([]);
  const [codenamesLoading, setCodenamesLoading] = useState(false);
  const [form] = Form.useForm();

  // Form config modal state
  const [formConfigModalOpen, setFormConfigModalOpen] = useState(false);
  const [formConfigTask, setFormConfigTask] = useState<TaskRow | null>(null);
  const [formFields, setFormFields] = useState<TaskFormField[]>([]);
  const [formSaving, setFormSaving] = useState(false);
  const [templateForms, setTemplateForms] = useState<Record<string, TaskFormField[]>>({});

  const nameInputRef = useRef<any>(null);

  // Fetch template detail
  const {
    data: template,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ['template', id],
    queryFn: () => templateApi.get(id!),
    enabled: !!id,
  });

  // Fetch version history
  const { data: versions = [] } = useQuery({
    queryKey: ['template-versions', id],
    queryFn: () => templateApi.listVersions(id!),
    enabled: !!id,
  });

  // Fetch approval definitions
  const { data: approvalDefs } = useQuery({
    queryKey: ['approval-definitions'],
    queryFn: () => approvalDefinitionApi.list(),
  });

  // Fetch task roles (用于模板任务分配的角色)
  const { data: taskRolesData = [] } = useQuery<TaskRole[]>({
    queryKey: ['task-roles'],
    queryFn: () => taskRoleApi.list(),
  });

  // Build role options from task roles
  const roleOptions = useMemo(() => {
    return taskRolesData.map(r => ({ value: r.code, label: r.name }));
  }, [taskRolesData]);

  // Fetch template task forms
  useEffect(() => {
    if (id) {
      taskFormApi.getTemplateTaskForms(id).then((forms) => {
        const map: Record<string, TaskFormField[]> = {};
        for (const f of forms) {
          try {
            map[f.task_code] = typeof f.fields === 'string' ? JSON.parse(f.fields) : f.fields;
          } catch {
            map[f.task_code] = [];
          }
        }
        setTemplateForms(map);
      });
    }
  }, [id]);

  const publishedApprovalDefs = useMemo(() => {
    if (!approvalDefs?.groups) return [];
    return approvalDefs.groups.flatMap(g =>
      g.definitions.filter(d => d.status === 'published').map(d => ({ ...d, group_name: g.name }))
    );
  }, [approvalDefs]);

  // Derived state
  const isPublished = template?.status === 'published';
  const isDraft = !isPublished; // treat undefined/draft as draft

  // 判断是否为最新版本（版本历史中版本号最高的）
  const isLatestVersion = useMemo(() => {
    if (!template || versions.length === 0) return true;
    const maxVersion = versions.reduce((max, v) => {
      const cv = parseFloat(String(v.version)) || 0;
      return cv > max ? cv : max;
    }, 0);
    const currentVersion = parseFloat(String(template.version)) || 0;
    return currentVersion >= maxVersion;
  }, [template, versions]);

  // 历史已发布版本（非最新）设为只读
  const isReadOnly = isPublished && !isLatestVersion;

  // 是否有可回退的上一个已发布版本（草稿且有 parent_template_id）
  const canRevert = isDraft && !!template?.parent_template_id;

  // Initialize local state from fetched data
  useEffect(() => {
    if (template) {
      setTemplateName(template.name);
      const rows: TaskRow[] = (template.tasks || []).map((t, idx) => ({
        ...t,
        phase: (t.phase || '').toUpperCase(),
        _key: t.id || `existing-${idx}`,
        _depends_on: (t.dependencies || []).map((d: any) => d.depends_on_task_code),
      }));
      setTasks(rows);
      setHasChanges(false);
    }
  }, [template]);

  useEffect(() => {
    if (editingName) {
      nameInputRef.current?.focus();
    }
  }, [editingName]);

  // ============ Task manipulation helpers ============

  const markChanged = useCallback(() => {
    setHasChanges(true);
  }, []);

  const updateTask = useCallback(
    (key: string, field: keyof TaskRow, value: any) => {
      if (isReadOnly) return;
      setTasks((prev) =>
        prev.map((t) => {
          if (t._key !== key) return t;
          const updated = { ...t, [field]: value };
          // Auto-set type when parent_task_code changes
          if (field === 'parent_task_code') {
            if (value) {
              updated.task_type = 'SUBTASK';
            } else if (t.task_type === 'SUBTASK') {
              updated.task_type = 'TASK';
            }
          }
          return updated;
        })
      );
      markChanged();
    },
    [markChanged, isReadOnly]
  );

  const deleteTask = useCallback(
    (key: string) => {
      if (isReadOnly) return;
      setTasks((prev) => {
        const taskToDelete = prev.find(t => t._key === key);
        if (!taskToDelete) return prev.filter((t) => t._key !== key);

        // Also remove any SRM procurement task linked to this task
        const srmTaskCode = `SRM-${taskToDelete.task_code}`;
        return prev.filter((t) => t._key !== key && !(t.task_type === 'srm_procurement' && t.task_code === srmTaskCode));
      });
      markChanged();
    },
    [markChanged, isReadOnly]
  );

  const addTask = useCallback(
    (phase: string) => {
      if (isReadOnly) return;
      const phaseTasks = tasks.filter((t) => t.phase === phase);
      const maxOrder = phaseTasks.length > 0
        ? Math.max(...phaseTasks.map((t) => t.sort_order))
        : 0;

      const prefix = phase === 'CONCEPT' ? 'CON' : phase;
      const existingCodes = tasks
        .filter((t) => t.phase === phase && t.task_type !== 'SUBTASK')
        .map((t) => {
          const match = t.task_code?.match(/-(\d+)$/);
          return match ? parseInt(match[1], 10) : 0;
        });
      const nextNum = existingCodes.length > 0 ? Math.max(...existingCodes) + 1 : 1;
      const code = `${prefix}-${String(nextNum).padStart(3, '0')}`;

      const newTask: TaskRow = {
        _key: `new-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        task_code: code,
        name: '',
        phase,
        task_type: 'TASK',
        parent_task_code: '',
        default_assignee_role: '',
        estimated_days: 1,
        is_critical: false,
        requires_approval: false,
        auto_create_feishu_task: false,
        feishu_approval_code: '',
        sort_order: maxOrder + 1,
      };
      setTasks((prev) => [...prev, newTask]);
      markChanged();
    },
    [tasks, markChanged, isReadOnly]
  );

  const addSubtask = useCallback(
    (parentTask: TaskRow) => {
      if (isReadOnly) return;
      // Find existing children of this parent
      const existingChildren = tasks.filter(
        (t) => t.parent_task_code === parentTask.task_code
      );
      // Generate next subtask number
      const existingNums = existingChildren.map((t) => {
        const match = t.task_code?.match(/-(\d+)$/);
        return match ? parseInt(match[1], 10) : 0;
      });
      const nextNum = existingNums.length > 0 ? Math.max(...existingNums) + 1 : 1;
      const code = `${parentTask.task_code}-${String(nextNum).padStart(2, '0')}`;

      // sort_order: place after parent and all its children
      const phaseTasks = tasks.filter((t) => t.phase === parentTask.phase);
      const maxOrder = phaseTasks.length > 0
        ? Math.max(...phaseTasks.map((t) => t.sort_order))
        : 0;

      const newKey = `new-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const newTask: TaskRow = {
        _key: newKey,
        task_code: code,
        name: '',
        phase: parentTask.phase,
        task_type: 'SUBTASK',
        parent_task_code: parentTask.task_code,
        default_assignee_role: parentTask.default_assignee_role || '',
        estimated_days: 1,
        is_critical: false,
        requires_approval: false,
        auto_create_feishu_task: false,
        feishu_approval_code: '',
        sort_order: maxOrder + 1,
      };
      setTasks((prev) => [...prev, newTask]);
      markChanged();

      // Auto-focus the new row's name after render
      setTimeout(() => {
        const el = document.querySelector(`[data-task-key="${newKey}"] .editable-name-trigger`) as HTMLElement;
        if (el) el.click();
      }, 100);
    },
    [tasks, markChanged, isReadOnly]
  );

  const moveTask = useCallback(
    (key: string, direction: 'up' | 'down') => {
      if (isReadOnly) return;
      setTasks((prev) => {
        const idx = prev.findIndex((t) => t._key === key);
        if (idx < 0) return prev;

        const task = prev[idx];
        const phase = task.phase;

        const phaseIndices = prev
          .map((t, i) => (t.phase === phase ? i : -1))
          .filter((i) => i >= 0);
        const posInPhase = phaseIndices.indexOf(idx);

        let swapIdx: number;
        if (direction === 'up') {
          if (posInPhase <= 0) return prev;
          swapIdx = phaseIndices[posInPhase - 1];
        } else {
          if (posInPhase >= phaseIndices.length - 1) return prev;
          swapIdx = phaseIndices[posInPhase + 1];
        }

        const newTasks = [...prev];
        const tempOrder = newTasks[idx].sort_order;
        newTasks[idx] = { ...newTasks[idx], sort_order: newTasks[swapIdx].sort_order };
        newTasks[swapIdx] = { ...newTasks[swapIdx], sort_order: tempOrder };
        [newTasks[idx], newTasks[swapIdx]] = [newTasks[swapIdx], newTasks[idx]];
        return newTasks;
      });
      markChanged();
    },
    [markChanged, isReadOnly]
  );

  // ============ Parent task options (per phase) ============

  const parentTaskOptionsByPhase = useMemo(() => {
    const map: Record<string, { value: string; label: string }[]> = {};
    for (const phase of PHASES) {
      map[phase] = tasks
        .filter(
          (t) =>
            t.phase === phase &&
            (t.task_type === 'MILESTONE' || t.task_type === 'TASK')
        )
        .map((t) => ({
          value: t.task_code,
          label: `${t.task_code} - ${t.name || '(未命名)'}`,
        }));
    }
    return map;
  }, [tasks]);

  // ============ Dependency task options (per phase) ============

  const dependencyOptionsByPhase = useMemo(() => {
    const map: Record<string, { value: string; label: string }[]> = {};
    for (const phase of PHASES) {
      map[phase] = tasks
        .filter((t) => t.phase === phase)
        .map((t) => ({
          value: t.task_code,
          label: `${t.task_code} - ${t.name || '(未命名)'}`,
        }));
    }
    return map;
  }, [tasks]);

  // ============ Group tasks by phase (tree ordered) ============

  const tasksByPhase = useMemo(() => {
    const grouped: Record<string, TaskRow[]> = {};
    for (const phase of PHASES) {
      const phaseTasks = tasks
        .filter((t) => t.phase === phase)
        .sort((a, b) => a.sort_order - b.sort_order);
      grouped[phase] = buildTreeOrder(phaseTasks);
    }
    return grouped;
  }, [tasks]);

  // ============ Save ============

  const buildSavePayload = () => {
    return tasks.map((t, idx) => ({
      task_code: t.task_code,
      name: t.name,
      phase: t.phase,
      task_type: t.task_type,
      parent_task_code: t.parent_task_code || '',
      default_assignee_role: t.default_assignee_role || '',
      estimated_days: t.estimated_days,
      is_critical: t.is_critical,
      requires_approval: t.requires_approval,
      auto_create_feishu_task: t.auto_create_feishu_task || false,
      feishu_approval_code: t.feishu_approval_code || '',
      sort_order: idx,
      description: (t as any).description || '',
      depends_on: (t as any)._depends_on || [],
      is_locked: t.is_locked || false,
    }));
  };

  const handleSave = async () => {
    if (!id || isReadOnly) return;

    // 已发布模板（最新版本）：弹出版本号对话框，创建新草稿版本
    if (isPublished && isLatestVersion) {
      setNewVersionInput('');
      setVersionModalOpen(true);
      return;
    }

    // 草稿模板：直接保存
    setSaving(true);
    try {
      const payload = buildSavePayload();
      await templateApi.batchSaveTasks(id, payload);
      msg.success('保存成功');
      setHasChanges(false);
      queryClient.invalidateQueries({ queryKey: ['template', id] });
      queryClient.invalidateQueries({ queryKey: ['templates'] });
    } catch (error: any) {
      msg.error(error?.response?.data?.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  // 确认版本号后创建新草稿版本并保存修改
  const handleVersionConfirm = async () => {
    if (!id || !template || !newVersionInput.trim()) return;

    const currentVer = parseFloat(String(template.version)) || 0;
    const newVer = parseFloat(newVersionInput.trim());
    if (isNaN(newVer) || newVer <= currentVer) {
      msg.error(`新版本号必须大于当前版本 ${template.version}`);
      return;
    }

    setVersionSaving(true);
    try {
      // 创建新草稿版本（传入用户指定的版本号，后端会复制任务+依赖+表单）
      const newTemplate = await templateApi.upgrade(id, newVersionInput.trim());

      // 保存当前编辑的任务到新草稿（不传版本号，草稿状态下不会递增版本）
      const payload = buildSavePayload();
      await templateApi.batchSaveTasks(newTemplate.id, payload);

      // 保存用户本地修改的表单配置到新草稿
      for (const [taskCode, fields] of Object.entries(templateForms)) {
        if (fields && fields.length > 0) {
          await taskFormApi.saveTemplateTaskForm(newTemplate.id, {
            task_code: taskCode,
            name: '完成表单',
            fields,
          });
        }
      }

      // 如果模板名称有改动，也更新
      if (templateName !== template.name) {
        await templateApi.update(newTemplate.id, { name: templateName.trim() });
      }

      msg.success(`已创建新版本 v${newVersionInput.trim()}`);
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      setVersionModalOpen(false);
      navigate(`/templates/${newTemplate.id}`);
    } catch (error: any) {
      msg.error(error?.response?.data?.message || '创建新版本失败');
    } finally {
      setVersionSaving(false);
    }
  };

  // 撤销草稿，回退到上一个已发布版本
  const handleRevert = () => {
    if (!id) return;
    modal.confirm({
      title: '撤销草稿',
      content: '确定要撤销当前草稿并回退到上一个已发布版本吗？此操作不可恢复。',
      okText: '确定撤销',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          const prevVersion = await templateApi.revert(id);
          msg.success('已回退到上一个版本');
          queryClient.invalidateQueries({ queryKey: ['templates'] });
          navigate(`/templates/${prevVersion.id}`);
        } catch (error: any) {
          msg.error(error?.response?.data?.message || '撤销失败');
        }
      },
    });
  };

  // ============ Save template name ============

  const handleSaveName = async () => {
    if (!id || !templateName.trim()) return;
    try {
      await templateApi.update(id, { name: templateName.trim() });
      msg.success('名称已更新');
      queryClient.invalidateQueries({ queryKey: ['template', id] });
      queryClient.invalidateQueries({ queryKey: ['templates'] });
    } catch {
      msg.error('名称更新失败');
    }
    setEditingName(false);
  };

  // ============ Publish ============

  const handlePublish = () => {
    if (!id || !template) return;
    modal.confirm({
      title: '发布流程',
      content: `发布后流程将被锁定，无法直接修改。如需修改请升级版本创建新的草稿。确定要发布 v${template.version || 1} 吗？`,
      okText: '确定发布',
      okType: 'primary',
      cancelText: '取消',
      onOk: async () => {
        setPublishing(true);
        try {
          await templateApi.publish(id);
          msg.success('发布成功');
          queryClient.invalidateQueries({ queryKey: ['template', id] });
          queryClient.invalidateQueries({ queryKey: ['template-versions', id] });
          queryClient.invalidateQueries({ queryKey: ['templates'] });
        } catch (error: any) {
          msg.error(error?.response?.data?.message || '发布失败');
        } finally {
          setPublishing(false);
        }
      },
    });
  };

  // ============ Create project logic ============

  const pinyinMap: Record<string, string> = {
    '微光': 'WG', '晨曦': 'CX', '朝霞': 'ZX', '旭日': 'XR', '明辉': 'MH',
    '皓月': 'HY', '星河': 'XH', '天枢': 'TS', '瑶光': 'YG', '紫微': 'ZW',
    '青龙': 'QL', '朱雀': 'ZQ', '玄武': 'XW', '白虎': 'BH', '麒麟': 'QiL',
    '凤凰': 'FH', '鲲鹏': 'KP', '九天': 'JT', '太极': 'TJ', '鸿蒙': 'HM',
  };

  const getCodenameType = (): string => {
    if (!template) return 'platform';
    const pt = template.product_type?.toLowerCase() || '';
    if (pt === 'platform') return 'platform';
    return 'product';
  };

  const getCodeAbbr = (codename: string, type: string) => {
    if (type === 'platform') return pinyinMap[codename] || codename;
    return codename;
  };

  const formatCodenameLabel = (c: Codename): string => {
    const codenameType = getCodenameType();
    if (codenameType === 'platform') {
      return `Gen ${c.generation}: ${c.codename} — ${c.description || c.theme}`;
    }
    return `${c.codename} — ${c.theme}`;
  };

  const handleCodenameChange = (codenameId: string) => {
    const selected = codenames.find((c) => c.id === codenameId);
    if (selected) {
      const codenameType = getCodenameType();
      const year = new Date().getFullYear();
      const abbr = getCodeAbbr(selected.codename, codenameType);
      const prefix = codenameType === 'platform' ? 'PLT' : 'PRD';
      form.setFieldsValue({
        project_name: selected.codename,
        project_code: `${prefix}-${abbr}-${year}`,
      });
    }
  };

  useEffect(() => {
    if (createProjectModalOpen && template) {
      const codenameType = getCodenameType();
      setCodenamesLoading(true);
      codenameApi
        .list(codenameType, true)
        .then((data) => {
          const sorted = [...data].sort((a, b) => (a.generation || 0) - (b.generation || 0));
          setCodenames(sorted);
          if (sorted.length > 0) {
            const first = sorted[0];
            const year = new Date().getFullYear();
            const abbr = getCodeAbbr(first.codename, codenameType);
            const prefix = codenameType === 'platform' ? 'PLT' : 'PRD';
            form.setFieldsValue({
              codename_id: first.id,
              project_name: first.codename,
              project_code: `${prefix}-${abbr}-${year}`,
            });
          }
        })
        .catch(() => {
          msg.warning('获取代号列表失败');
          setCodenames([]);
        })
        .finally(() => setCodenamesLoading(false));
    }
  }, [createProjectModalOpen, template]);

  const handleCreateProject = async () => {
    try {
      const values = await form.validateFields();
      const pmUserId = values.pm_user_id || user!.id;
      const project = await templateApi.createProjectFromTemplate({
        template_id: id!,
        project_name: values.project_name,
        project_code: values.project_code,
        start_date: values.start_date.format('YYYY-MM-DD'),
        pm_user_id: pmUserId,
        skip_weekends: values.skip_weekends,
        codename_id: values.codename_id,
        role_assignments: {
          pm: pmUserId,
        },
      }) as { id: string };
      msg.success('项目创建成功！');
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      setCreateProjectModalOpen(false);
      form.resetFields();
      navigate(`/projects/${project.id}`);
    } catch {
      msg.error('项目创建失败');
    }
  };

  // ============ Form Config helpers ============

  const FIELD_TYPES: { value: TaskFormField['type']; label: string }[] = [
    { value: 'text', label: '单行文本' },
    { value: 'textarea', label: '多行文本' },
    { value: 'number', label: '数字' },
    { value: 'select', label: '下拉选择' },
    { value: 'date', label: '日期' },
    { value: 'file', label: '文件上传' },
    { value: 'checkbox', label: '复选框' },
    { value: 'user', label: '用户选择' },
    { value: 'role_assignment', label: '角色分配' },
    { value: 'bom_upload', label: 'BOM上传' },
  ];

  const openFormConfig = (task: TaskRow) => {
    setFormConfigTask(task);
    const existing = templateForms[task.task_code] || [];
    setFormFields(existing.length > 0 ? existing.map(f => ({ ...f })) : []);
    setFormConfigModalOpen(true);
  };

  const addFormField = () => {
    const nextIdx = formFields.length + 1;
    setFormFields([...formFields, {
      key: `field_${nextIdx}`,
      label: '',
      type: 'text',
      required: false,
    }]);
  };

  const updateFormField = (idx: number, field: Partial<TaskFormField>) => {
    setFormFields(prev => prev.map((f, i) => i === idx ? { ...f, ...field } : f));
  };

  const removeFormField = (idx: number) => {
    setFormFields(prev => prev.filter((_, i) => i !== idx));
  };

  const moveFormField = (idx: number, direction: 'up' | 'down') => {
    if ((direction === 'up' && idx === 0) || (direction === 'down' && idx === formFields.length - 1)) return;
    const newFields = [...formFields];
    const swapIdx = direction === 'up' ? idx - 1 : idx + 1;
    [newFields[idx], newFields[swapIdx]] = [newFields[swapIdx], newFields[idx]];
    setFormFields(newFields);
  };

  const handleSaveFormConfig = async () => {
    if (!id || !formConfigTask) return;
    setFormSaving(true);
    try {
      await taskFormApi.saveTemplateTaskForm(id, {
        task_code: formConfigTask.task_code,
        name: '完成表单',
        fields: formFields,
      });

      // Check for bom_upload field changes → auto-create/remove SRM procurement task
      const oldFields = templateForms[formConfigTask.task_code] || [];
      const hadBomUpload = oldFields.some(f => f.type === 'bom_upload');
      const hasBomUpload = formFields.some(f => f.type === 'bom_upload');

      if (!hadBomUpload && hasBomUpload) {
        // bom_upload added → create SRM procurement task
        const srmTaskCode = `SRM-${formConfigTask.task_code}`;
        const existingSrmTask = tasks.find(t => t.task_type === 'srm_procurement' && t.task_code === srmTaskCode);

        if (!existingSrmTask) {
          const phaseTasks = tasks.filter(t => t.phase === formConfigTask.phase);
          const maxOrder = phaseTasks.length > 0 ? Math.max(...phaseTasks.map(t => t.sort_order)) : 0;

          const newKey = `srm-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
          const newTask: TaskRow = {
            _key: newKey,
            task_code: srmTaskCode,
            name: '打样采购',
            phase: formConfigTask.phase,
            task_type: 'srm_procurement',
            parent_task_code: '',
            default_assignee_role: '采购',
            estimated_days: 7,
            is_critical: false,
            requires_approval: false,
            auto_create_feishu_task: false,
            feishu_approval_code: '',
            sort_order: maxOrder + 1,
            is_locked: true,
            _depends_on: [formConfigTask.task_code],
          };
          setTasks(prev => [...prev, newTask]);
          markChanged();
          msg.info('已自动创建「打样采购」任务，依赖于当前BOM任务');
        }
      } else if (hadBomUpload && !hasBomUpload) {
        // bom_upload removed → remove associated SRM procurement task
        const srmTaskCode = `SRM-${formConfigTask.task_code}`;
        const srmTask = tasks.find(t => t.task_type === 'srm_procurement' && t.task_code === srmTaskCode);
        if (srmTask) {
          setTasks(prev => prev.filter(t => t.task_code !== srmTaskCode));
          markChanged();
          msg.info('已自动移除关联的「打样采购」任务');
        }
      }

      setTemplateForms(prev => ({ ...prev, [formConfigTask.task_code]: formFields }));
      msg.success('表单配置已保存');
      setFormConfigModalOpen(false);
    } catch {
      msg.error('保存表单配置失败');
    } finally {
      setFormSaving(false);
    }
  };

  // ============ Version History helpers ============

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-';
    const d = new Date(dateStr);
    return d.toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // ============ Render ============

  if (isLoading) {
    return (
      <div style={{ padding: 24, textAlign: 'center', marginTop: 100 }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  if (isError || !template) {
    return (
      <div style={{ padding: 24, textAlign: 'center', marginTop: 100 }}>
        <Text type="danger">加载失败，模板不存在或已删除</Text>
        <br />
        <Button type="link" onClick={() => navigate('/templates')}>
          返回列表
        </Button>
      </div>
    );
  }

  const isPlatformType = getCodenameType() === 'platform';

  return (
    <div style={{ padding: 24 }}>
      {/* Draft Alert Banner */}
      {isDraft && (
        <Alert
          message="该流程为草稿状态，发布后才能用于创建项目"
          type="warning"
          showIcon
          banner
          style={{ marginBottom: 16 }}
        />
      )}

      {/* Historical version read-only Banner */}
      {isReadOnly && (
        <Alert
          message="当前查看的是历史版本，内容为只读。请切换到最新版本进行编辑。"
          type="info"
          showIcon
          banner
          style={{ marginBottom: 16 }}
        />
      )}

      {/* Page Header */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Space size="middle" align="center">
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate('/templates')}
              type="text"
            />
            <div>
              {editingName && !isReadOnly ? (
                <Space>
                  <Input
                    ref={nameInputRef}
                    value={templateName}
                    onChange={(e) => setTemplateName(e.target.value)}
                    onPressEnter={handleSaveName}
                    style={{ fontSize: 20, fontWeight: 600, width: 300 }}
                  />
                  <Button
                    icon={<CheckOutlined />}
                    size="small"
                    type="primary"
                    onClick={handleSaveName}
                  />
                  <Button
                    icon={<CloseOutlined />}
                    size="small"
                    onClick={() => {
                      setEditingName(false);
                      setTemplateName(template.name);
                    }}
                  />
                </Space>
              ) : (
                <Space align="center">
                  <Title
                    level={4}
                    style={{ margin: 0, cursor: isReadOnly ? 'default' : 'pointer' }}
                    onClick={() => !isReadOnly && setEditingName(true)}
                  >
                    {templateName}
                    {!isReadOnly && (
                      <EditOutlined
                        style={{ fontSize: 14, marginLeft: 8, color: '#999' }}
                      />
                    )}
                  </Title>
                </Space>
              )}
              <Space size="small" style={{ marginTop: 4 }}>
                <Tag color="blue">v{template.version || 1}</Tag>
                {isDraft ? (
                  <Tag color="orange">草稿</Tag>
                ) : (
                  <Tag color="green">已发布</Tag>
                )}
                {template.product_type && (
                  <Tag>{template.product_type}</Tag>
                )}
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {template.tasks?.length || tasks.length} 个任务 · 预估 {template.estimated_days} 天
                </Text>
                {isPublished && template.published_at && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    · 发布于 {formatDate(template.published_at)}
                  </Text>
                )}
              </Space>
            </div>
          </Space>

          <Space>
            <Button
              icon={<HistoryOutlined />}
              onClick={() => setVersionDrawerOpen(true)}
            >
              版本历史
            </Button>
            {isDraft && (
              <Button
                type="primary"
                icon={<SendOutlined />}
                loading={publishing}
                onClick={handlePublish}
                style={{ backgroundColor: '#52c41a', borderColor: '#52c41a' }}
              >
                发布
              </Button>
            )}
            {isPublished && (
              <Button
                icon={<RocketOutlined />}
                onClick={() => setCreateProjectModalOpen(true)}
              >
                创建项目
              </Button>
            )}
            {canRevert && (
              <Button
                icon={<RollbackOutlined />}
                danger
                onClick={handleRevert}
              >
                撤销草稿
              </Button>
            )}
            {!isReadOnly && (
              <Badge dot={hasChanges} offset={[-4, 4]}>
                <Button
                  type={hasChanges ? 'primary' : 'default'}
                  icon={<SaveOutlined />}
                  loading={saving}
                  onClick={handleSave}
                >
                  {isPublished ? '保存为新版本' : '保存修改'}
                </Button>
              </Badge>
            )}
          </Space>
        </div>
      </Card>

      {/* Task Table by Phase */}
      <Card
        title="任务列表"
        bodyStyle={{ padding: 0 }}
        style={isReadOnly ? { background: '#fafafa' } : undefined}
      >
        {PHASES.filter((phase) => {
          return template.phases?.includes(phase) || (tasksByPhase[phase]?.length || 0) > 0;
        }).map((phase) => {
          const phaseTasks = tasksByPhase[phase] || [];
          const collapsed = collapsedPhases[phase] || false;

          return (
            <div key={phase} style={{ borderBottom: '1px solid #f0f0f0' }}>
              {/* Phase Header */}
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  padding: '10px 16px',
                  background: '#fafafa',
                  cursor: 'pointer',
                  userSelect: 'none',
                }}
                onClick={() =>
                  setCollapsedPhases((prev) => ({
                    ...prev,
                    [phase]: !prev[phase],
                  }))
                }
              >
                <Space>
                  {collapsed ? <DownOutlined style={{ fontSize: 10 }} /> : <UpOutlined style={{ fontSize: 10 }} />}
                  <Tag color={phaseColors[phase]} style={{ margin: 0 }}>
                    {phaseLabels[phase]}
                  </Tag>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {phaseTasks.length} 个任务
                  </Text>
                </Space>
                {!isReadOnly && (
                  <Button
                    type="link"
                    size="small"
                    icon={<PlusOutlined />}
                    onClick={(e) => {
                      e.stopPropagation();
                      addTask(phase);
                      setCollapsedPhases((prev) => ({ ...prev, [phase]: false }));
                    }}
                  >
                    添加任务
                  </Button>
                )}
              </div>

              {/* Tasks */}
              {!collapsed && (
                <div>
                  {/* Column Headers */}
                  {phaseTasks.length > 0 && (
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        padding: '6px 16px',
                        background: '#f5f5f5',
                        fontSize: 12,
                        color: '#8c8c8c',
                        fontWeight: 500,
                        borderBottom: '1px solid #f0f0f0',
                      }}
                    >
                      {!isReadOnly && <div style={{ width: 40 }}/>}
                      <div style={{ width: 120, flexShrink: 0 }}>任务编码</div>
                      <div style={{ flex: 1, minWidth: 150 }}>任务名称</div>
                      <div style={{ width: 100, flexShrink: 0 }}>任务类型</div>
                      <div style={{ width: 150, flexShrink: 0 }}>父任务</div>
                      <div style={{ width: 150, flexShrink: 0 }}>前置任务</div>
                      <div style={{ width: 120, flexShrink: 0 }}>负责角色</div>
                      <div style={{ width: 70, flexShrink: 0, textAlign: 'center' }}>工期(天)</div>
                      <div style={{ width: 60, flexShrink: 0, textAlign: 'center' }}>关键</div>
                      <div style={{ width: 200, flexShrink: 0, textAlign: 'center' }}>审批模板</div>
                      <div style={{ width: 90, flexShrink: 0, textAlign: 'center' }}>{isReadOnly ? '' : '操作'}</div>
                    </div>
                  )}

                  {phaseTasks.map((task) => {
                    const isMilestone = task.task_type === 'MILESTONE';
                    const isSrmTask = task.task_type === 'srm_procurement';
                    const isTaskLocked = !!task.is_locked;
                    const isSubtask = !!task.parent_task_code;
                    const depth = (task as any)._depth || 0;
                    const canHaveChildren =
                      !isSrmTask && (task.task_type === 'MILESTONE' || task.task_type === 'TASK');

                    // Task type options: subtasks cannot become MILESTONE
                    const typeOptions = isSubtask
                      ? TASK_TYPES.filter((t) => t.value !== 'MILESTONE')
                      : TASK_TYPES;

                    // Parent task options for this task's phase
                    const phaseParentOptions = parentTaskOptionsByPhase[task.phase] || [];

                    return (
                      <div
                        key={task._key}
                        data-task-key={task._key}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          padding: '4px 16px',
                          borderBottom: '1px solid #f0f0f0',
                          background: isReadOnly
                            ? '#f9f9f9'
                            : isSrmTask
                            ? '#f0f5ff'
                            : depth > 0
                            ? '#fcfcfc'
                            : isMilestone
                            ? '#f6f8fa'
                            : 'white',
                          fontWeight: isMilestone ? 600 : 400,
                          fontSize: 13,
                          minHeight: 40,
                          paddingLeft: 16 + depth * 24,
                        }}
                      >
                        {/* Drag Handle / Sort */}
                        {!isReadOnly && (
                          <div style={{ width: 40, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 0 }}>
                            <Tooltip title="上移">
                              <Button
                                type="text"
                                size="small"
                                icon={<UpOutlined style={{ fontSize: 10 }} />}
                                style={{ height: 16, width: 24, padding: 0 }}
                                onClick={() => moveTask(task._key, 'up')}
                              />
                            </Tooltip>
                            <Tooltip title="下移">
                              <Button
                                type="text"
                                size="small"
                                icon={<DownOutlined style={{ fontSize: 10 }} />}
                                style={{ height: 16, width: 24, padding: 0 }}
                                onClick={() => moveTask(task._key, 'down')}
                              />
                            </Tooltip>
                          </div>
                        )}

                        {/* Task Code */}
                        <div style={{ width: 120, flexShrink: 0 }}>
                          {depth > 0 && (
                            <span style={{ color: '#bfbfbf', marginRight: 4, fontSize: 12, fontFamily: 'monospace' }}>
                              {'│ '.repeat(depth - 1)}└
                            </span>
                          )}
                          <EditableText
                            value={task.task_code}
                            onChange={(val) => updateTask(task._key, 'task_code', val)}
                            placeholder="编码"
                            style={{ fontFamily: 'monospace', fontSize: 12, display: 'inline-block' }}
                            readOnly={isReadOnly || isTaskLocked}
                          />
                        </div>

                        {/* Task Name */}
                        <div className="editable-name-trigger" style={{ flex: 1, minWidth: 150, display: 'flex', alignItems: 'center', gap: 4 }}>
                          {isSrmTask && (
                            <>
                              <Tag color="blue" style={{ margin: 0, fontSize: 11, lineHeight: '18px', flexShrink: 0 }}>采购任务</Tag>
                              <Tooltip title="此任务由系统自动管理，不可编辑">
                                <LockOutlined style={{ color: '#999', fontSize: 12, flexShrink: 0 }} />
                              </Tooltip>
                            </>
                          )}
                          <EditableText
                            value={task.name}
                            onChange={(val) => updateTask(task._key, 'name', val)}
                            placeholder="输入任务名称"
                            style={isMilestone ? { fontWeight: 600 } : undefined}
                            readOnly={isReadOnly || isTaskLocked}
                          />
                        </div>

                        {/* Task Type */}
                        <div style={{ width: 100, flexShrink: 0 }}>
                          <EditableSelect
                            value={isSrmTask ? '采购' : task.task_type}
                            onChange={(val) =>
                              updateTask(task._key, 'task_type', val as TaskRow['task_type'])
                            }
                            options={typeOptions}
                            readOnly={isReadOnly || isSubtask || isTaskLocked}
                          />
                        </div>

                        {/* Parent Task */}
                        <div style={{ width: 150, flexShrink: 0 }}>
                          <EditableSelect
                            value={task.parent_task_code || ''}
                            onChange={(val) => updateTask(task._key, 'parent_task_code', val)}
                            options={[
                              { value: '', label: '无' },
                              ...phaseParentOptions.filter((o) => o.value !== task.task_code),
                            ]}
                            placeholder="无"
                            allowClear
                            readOnly={isReadOnly || isTaskLocked}
                          />
                        </div>

                        {/* 前置任务 (Dependencies) */}
                        <div style={{ width: 150, flexShrink: 0 }}>
                          {(isReadOnly || isTaskLocked) ? (
                            <div style={{ padding: '1px 7px', minHeight: 24, lineHeight: '24px', fontSize: 12 }}>
                              {((task as any)._depends_on && (task as any)._depends_on.length > 0)
                                ? (task as any)._depends_on.join(', ')
                                : <span style={{ color: '#bfbfbf' }}>-</span>}
                            </div>
                          ) : (
                            <Select
                              mode="multiple"
                              size="small"
                              style={{ width: '100%' }}
                              placeholder="选择前置任务"
                              value={(task as any)._depends_on || []}
                              onChange={(val) => updateTask(task._key, '_depends_on' as any, val)}
                              maxTagCount={1}
                              options={(dependencyOptionsByPhase[task.phase] || []).filter(
                                (o) => o.value !== task.task_code && !(tasks.find(t => t.task_code === o.value && t.parent_task_code === task.task_code))
                              )}
                            />
                          )}
                        </div>

                        {/* Role */}
                        <div style={{ width: 120, flexShrink: 0 }}>
                          <EditableSelect
                            value={task.default_assignee_role || ''}
                            onChange={(val) => updateTask(task._key, 'default_assignee_role', val)}
                            options={roleOptions}
                            placeholder="选择角色"
                            allowClear
                            readOnly={isReadOnly || isTaskLocked}
                          />
                        </div>

                        {/* Estimated Days */}
                        <div style={{ width: 70, flexShrink: 0, textAlign: 'center' }}>
                          <EditableNumber
                            value={task.estimated_days}
                            onChange={(val) => updateTask(task._key, 'estimated_days', val)}
                            min={1}
                            readOnly={isReadOnly || isTaskLocked}
                          />
                        </div>

                        {/* Is Critical */}
                        <div style={{ width: 60, flexShrink: 0, textAlign: 'center' }}>
                          <Switch
                            size="small"
                            checked={task.is_critical}
                            disabled={isReadOnly || isTaskLocked}
                            onChange={(val) => updateTask(task._key, 'is_critical', val)}
                          />
                        </div>

                        {/* Approval Template */}
                        <div style={{ width: 200, flexShrink: 0, display: 'flex', alignItems: 'center', gap: 4 }}>
                          <Switch
                            size="small"
                            checked={task.requires_approval}
                            disabled={isReadOnly || isTaskLocked}
                            onChange={(val) => updateTask(task._key, 'requires_approval', val)}
                          />
                          {task.requires_approval && (
                            <Select
                              size="small"
                              style={{ flex: 1, fontSize: 12 }}
                              value={task.feishu_approval_code || undefined}
                              onChange={(val) => updateTask(task._key, 'feishu_approval_code', val)}
                              placeholder="选择审批模板"
                              allowClear
                              disabled={isReadOnly || isTaskLocked}
                              options={publishedApprovalDefs.map(d => ({
                                value: d.id,
                                label: `${d.name}`
                              }))}
                            />
                          )}
                        </div>

                        {/* Actions */}
                        {!isReadOnly ? (
                          isTaskLocked ? (
                            <div style={{ width: 90, flexShrink: 0, textAlign: 'center' }}>
                              <Tooltip title="此任务由系统自动管理">
                                <LockOutlined style={{ color: '#999' }} />
                              </Tooltip>
                            </div>
                          ) : (
                          <div style={{ width: 90, flexShrink: 0, textAlign: 'center' }}>
                            <Space size={0}>
                              <Tooltip title={templateForms[task.task_code]?.length ? `表单(${templateForms[task.task_code].length}字段)` : '配置表单'}>
                                <Button
                                  type="text"
                                  size="small"
                                  icon={<FormOutlined />}
                                  style={{ color: templateForms[task.task_code]?.length ? '#52c41a' : '#999' }}
                                  onClick={() => openFormConfig(task)}
                                />
                              </Tooltip>
                              {canHaveChildren && (
                                <Tooltip title="添加子任务">
                                  <Button
                                    type="text"
                                    size="small"
                                    icon={<PlusOutlined />}
                                    style={{ color: '#1890ff' }}
                                    onClick={() => addSubtask(task)}
                                  />
                                </Tooltip>
                              )}
                              <Popconfirm
                                title="确定删除此任务？"
                                onConfirm={() => deleteTask(task._key)}
                                okText="删除"
                                cancelText="取消"
                              >
                                <Button
                                  type="text"
                                  size="small"
                                  danger
                                  icon={<DeleteOutlined />}
                                />
                              </Popconfirm>
                            </Space>
                          </div>
                          )
                        ) : (
                          templateForms[task.task_code]?.length ? (
                            <div style={{ width: 90, flexShrink: 0, textAlign: 'center' }}>
                              <Tooltip title={`查看表单(${templateForms[task.task_code].length}字段)`}>
                                <Button
                                  type="text"
                                  size="small"
                                  icon={<FormOutlined />}
                                  style={{ color: '#52c41a' }}
                                  onClick={() => openFormConfig(task)}
                                />
                              </Tooltip>
                            </div>
                          ) : null
                        )}
                      </div>
                    );
                  })}

                  {phaseTasks.length === 0 && (
                    <div
                      style={{
                        padding: '16px',
                        textAlign: 'center',
                        color: '#bfbfbf',
                        fontSize: 13,
                      }}
                    >
                      {isReadOnly ? '暂无任务' : '暂无任务，点击上方"添加任务"开始'}
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </Card>

      {/* Version History Drawer */}
      <Drawer
        title="版本历史"
        placement="right"
        width={380}
        open={versionDrawerOpen}
        onClose={() => setVersionDrawerOpen(false)}
      >
        {versions.length === 0 ? (
          <Text type="secondary">暂无版本历史</Text>
        ) : (
          <Timeline
            items={versions.map((v: TemplateVersion) => {
              const isCurrent = v.id === id;
              const isVersionPublished = v.status === 'published';
              return {
                color: isVersionPublished ? 'green' : 'gray',
                children: (
                  <div
                    style={{
                      padding: '8px 12px',
                      borderRadius: 6,
                      background: isCurrent ? '#e6f7ff' : '#fafafa',
                      border: isCurrent ? '1px solid #91d5ff' : '1px solid #f0f0f0',
                      cursor: isCurrent ? 'default' : 'pointer',
                      marginBottom: 4,
                    }}
                    onClick={() => {
                      if (!isCurrent) {
                        navigate(`/templates/${v.id}`);
                        setVersionDrawerOpen(false);
                      }
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                      <Space size={4}>
                        <Text strong>v{v.version}</Text>
                        {isVersionPublished ? (
                          <Tag color="green" style={{ margin: 0 }}>已发布</Tag>
                        ) : (
                          <Tag color="orange" style={{ margin: 0 }}>草稿</Tag>
                        )}
                      </Space>
                      {isCurrent && (
                        <Tag color="blue" style={{ margin: 0 }}>当前</Tag>
                      )}
                    </div>
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {v.task_count} 个任务
                      </Text>
                      {v.published_at && (
                        <Text type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>
                          发布于 {formatDate(v.published_at)}
                        </Text>
                      )}
                      {!v.published_at && v.created_at && (
                        <Text type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>
                          创建于 {formatDate(v.created_at)}
                        </Text>
                      )}
                    </div>
                  </div>
                ),
              };
            })}
          />
        )}
      </Drawer>

      {/* Form Config Modal */}
      <Modal
        title={`${isReadOnly ? '查看' : '配置'}任务表单 - ${formConfigTask?.name || formConfigTask?.task_code || ''}`}
        open={formConfigModalOpen}
        onCancel={() => { setFormConfigModalOpen(false); setFormConfigTask(null); setFormFields([]); }}
        onOk={isReadOnly ? undefined : handleSaveFormConfig}
        confirmLoading={formSaving}
        okText="保存"
        cancelText={isReadOnly ? '关闭' : '取消'}
        footer={isReadOnly ? <Button onClick={() => { setFormConfigModalOpen(false); setFormConfigTask(null); setFormFields([]); }}>关闭</Button> : undefined}
        width={720}
        destroyOnClose
      >
        <div style={{ marginBottom: 12 }}>
          <Text type="secondary">配置工程师完成任务时需要填写的表单字段</Text>
        </div>

        {formFields.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '24px 0', color: '#999' }}>
            暂无表单字段，点击下方按钮添加
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {formFields.map((field, idx) => (
              <Card key={idx} size="small" bodyStyle={{ padding: '8px 12px' }}>
                <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start', flexWrap: 'wrap' }}>
                  <div style={{ flex: 1, minWidth: 120 }}>
                    <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 2 }}>标签名</Text>
                    <Input
                      size="small"
                      placeholder="字段标签"
                      value={field.label}
                      onChange={(e) => updateFormField(idx, { label: e.target.value })}
                    />
                  </div>
                  <div style={{ width: 120 }}>
                    <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 2 }}>字段Key</Text>
                    <Input
                      size="small"
                      placeholder="field_key"
                      value={field.key}
                      onChange={(e) => updateFormField(idx, { key: e.target.value })}
                    />
                  </div>
                  <div style={{ width: 110 }}>
                    <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 2 }}>类型</Text>
                    <Select
                      size="small"
                      style={{ width: '100%' }}
                      value={field.type}
                      onChange={(val) => updateFormField(idx, { type: val })}
                      options={FIELD_TYPES}
                    />
                  </div>
                  <div style={{ width: 60, textAlign: 'center' }}>
                    <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 2 }}>必填</Text>
                    <Switch
                      size="small"
                      checked={field.required}
                      onChange={(val) => updateFormField(idx, { required: val })}
                    />
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 2, paddingTop: 18 }}>
                    <Button type="text" size="small" icon={<UpOutlined style={{ fontSize: 10 }} />} onClick={() => moveFormField(idx, 'up')} disabled={idx === 0} />
                    <Button type="text" size="small" icon={<DownOutlined style={{ fontSize: 10 }} />} onClick={() => moveFormField(idx, 'down')} disabled={idx === formFields.length - 1} />
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => removeFormField(idx)} />
                  </div>
                </div>
                {field.type === 'select' && (
                  <div style={{ marginTop: 8 }}>
                    <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 2 }}>选项（逗号分隔）</Text>
                    <Input
                      size="small"
                      placeholder="选项1,选项2,选项3"
                      value={(field.options || []).join(',')}
                      onChange={(e) => updateFormField(idx, { options: e.target.value.split(',').map(s => s.trim()).filter(Boolean) })}
                    />
                  </div>
                )}
                {field.type === 'file' && (
                  <div style={{ marginTop: 8 }}>
                    <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 2 }}>文件类型限制（如 .pdf,.doc）</Text>
                    <Input
                      size="small"
                      placeholder=".pdf,.doc,.xlsx"
                      value={field.accept || ''}
                      onChange={(e) => updateFormField(idx, { accept: e.target.value })}
                    />
                  </div>
                )}
                {field.type === 'bom_upload' && (
                  <div style={{ marginTop: 8, padding: '4px 8px', background: '#e6f7ff', borderRadius: 4 }}>
                    <Text type="secondary" style={{ fontSize: 11 }}>支持 .rep (PADS) 和 .xlsx/.xls (Excel) 格式，上传后自动解析预览BOM物料清单</Text>
                  </div>
                )}
              </Card>
            ))}
          </div>
        )}

        <Button
          type="dashed"
          block
          icon={<PlusOutlined />}
          style={{ marginTop: 12 }}
          onClick={addFormField}
        >
          添加字段
        </Button>
      </Modal>

      {/* Create Project Modal */}
      <Modal
        title={`基于研发流程创建项目 - ${template.name}${template.version ? ` v${template.version}` : ''}`}
        open={createProjectModalOpen}
        onCancel={() => {
          setCreateProjectModalOpen(false);
          form.resetFields();
          setCodenames([]);
        }}
        onOk={handleCreateProject}
        okText="创建项目"
        width={600}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="codename_id"
            label="项目代号"
            rules={[{ required: true, message: '请选择项目代号' }]}
          >
            <Select
              placeholder="选择项目代号"
              loading={codenamesLoading}
              onChange={handleCodenameChange}
              notFoundContent={codenamesLoading ? <Spin size="small" /> : '暂无可用代号'}
              options={codenames.map((c) => ({
                value: c.id,
                label: formatCodenameLabel(c),
              }))}
            />
          </Form.Item>
          <Form.Item
            name="project_code"
            label="项目编码"
            rules={[{ required: true, message: '请输入项目编码' }]}
          >
            <Input placeholder={isPlatformType ? '如：PLT-WG-2026' : '如：PRD-Nova-2026'} disabled />
          </Form.Item>
          <Form.Item name="project_name" label="项目名称（自动生成）">
            <Input disabled placeholder="选择代号后自动生成" />
          </Form.Item>
          <Form.Item
            name="start_date"
            label="计划开始日期"
            rules={[{ required: true, message: '请选择开始日期' }]}
          >
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            name="pm_user_id"
            label="项目经理"
            rules={[{ required: true, message: '请选择项目经理' }]}
            initialValue={user?.id}
          >
            <UserSelect
              mode="single"
              placeholder="选择项目经理"
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item name="skip_weekends" label="跳过周末" initialValue={true}>
            <Select
              options={[
                { value: true, label: '是 - 工期计算跳过周末' },
                { value: false, label: '否 - 包含周末' },
              ]}
            />
          </Form.Item>
        </Form>
        <div style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
          <Text type="secondary">
            将从研发流程「{template.name}
            {template.version ? ` v${template.version}` : ''}」复制{' '}
            {template.estimated_days} 天的任务计划
          </Text>
        </div>
      </Modal>

      {/* Version Number Modal */}
      <Modal
        title="保存为新版本"
        open={versionModalOpen}
        onCancel={() => setVersionModalOpen(false)}
        onOk={handleVersionConfirm}
        confirmLoading={versionSaving}
        okText="确认创建"
        cancelText="取消"
        width={400}
      >
        <div style={{ marginBottom: 16 }}>
          <Text>当前版本：<Tag color="blue">v{template.version}</Tag></Text>
        </div>
        <div>
          <Text style={{ display: 'block', marginBottom: 8 }}>新版本号（必须大于当前版本）：</Text>
          <Input
            placeholder={`如 ${nextVersionSuggestion(String(template.version))}`}
            value={newVersionInput}
            onChange={(e) => setNewVersionInput(e.target.value)}
            onPressEnter={handleVersionConfirm}
          />
        </div>
      </Modal>
    </div>
  );
};

// 建议下一个版本号
function nextVersionSuggestion(current: string): string {
  const v = current.replace(/^[vV]/, '');
  const parts = v.split('.');
  const major = parseInt(parts[0], 10) || 1;
  let minor = parseInt(parts[1], 10) || 0;
  minor++;
  if (minor >= 10) {
    return `${major + 1}.0`;
  }
  return `${major}.${minor}`;
}

export default TemplateDetail;
