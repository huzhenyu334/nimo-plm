import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  Steps,
  Button,
  Form,
  Input,
  Select,
  Radio,
  Switch,
  Space,
  Typography,
  message,
  Spin,
  Tag,
  Empty,
  Popconfirm,
} from 'antd';
import {
  ArrowLeftOutlined,
  PlusOutlined,
  DeleteOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  SaveOutlined,
  SendOutlined,
  FontSizeOutlined,
  AlignLeftOutlined,
  NumberOutlined,
  DollarOutlined,
  CheckSquareOutlined,
  AppstoreOutlined,
  CalendarOutlined,
  SwapOutlined,
  UserOutlined,
  PaperClipOutlined,
  TableOutlined,
  InfoCircleOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import {
  approvalDefinitionApi,
  approvalGroupApi,
  FormField,
  FlowNode,
  ApprovalDefinition,
} from '@/api/approvalDefinitions';
import UserSelect from '@/components/UserSelect';

const { Title, Text } = Typography;

// Field type config
const FIELD_TYPES: { category: string; items: { type: FormField['type']; label: string; icon: React.ReactNode }[] }[] = [
  {
    category: '文本',
    items: [
      { type: 'text', label: '单行文本', icon: <FontSizeOutlined /> },
      { type: 'textarea', label: '多行文本', icon: <AlignLeftOutlined /> },
      { type: 'description', label: '说明', icon: <InfoCircleOutlined /> },
    ],
  },
  {
    category: '数值',
    items: [
      { type: 'number', label: '数字', icon: <NumberOutlined /> },
      { type: 'money', label: '金额', icon: <DollarOutlined /> },
    ],
  },
  {
    category: '选项',
    items: [
      { type: 'select', label: '单选', icon: <CheckSquareOutlined /> },
      { type: 'multiselect', label: '多选', icon: <AppstoreOutlined /> },
    ],
  },
  {
    category: '日期',
    items: [
      { type: 'date', label: '日期', icon: <CalendarOutlined /> },
      { type: 'daterange', label: '日期区间', icon: <SwapOutlined /> },
    ],
  },
  {
    category: '其他',
    items: [
      { type: 'user', label: '人员选择', icon: <UserOutlined /> },
      { type: 'attachment', label: '附件', icon: <PaperClipOutlined /> },
      { type: 'table', label: '明细表格', icon: <TableOutlined /> },
    ],
  },
];

const FIELD_TYPE_LABEL: Record<string, string> = {};
const FIELD_TYPE_ICON: Record<string, React.ReactNode> = {};
FIELD_TYPES.forEach((cat) =>
  cat.items.forEach((item) => {
    FIELD_TYPE_LABEL[item.type] = item.label;
    FIELD_TYPE_ICON[item.type] = item.icon;
  })
);

const DEFAULT_FLOW_NODES: FlowNode[] = [
  { type: 'submit', name: '提交', config: { submitter: 'all' } },
  { type: 'approve', name: '审批', config: { approver_type: 'designated', multi_approve: 'all' } },
  { type: 'end', name: '结束', config: {} },
];

const ApprovalEditor: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isNew = id === 'new';

  const [currentStep, setCurrentStep] = useState(0);
  const [basicForm] = Form.useForm();

  // Form designer state
  const [formFields, setFormFields] = useState<FormField[]>([]);
  const [selectedFieldIndex, setSelectedFieldIndex] = useState<number | null>(null);

  // Flow designer state
  const [flowNodes, setFlowNodes] = useState<FlowNode[]>(DEFAULT_FLOW_NODES);
  const [selectedNodeIndex, setSelectedNodeIndex] = useState<number | null>(null);

  // More settings
  const [allowRevoke, setAllowRevoke] = useState(true);
  const [allowRecall, setAllowRecall] = useState(false);

  // Saved definition id (for new → created)
  const [savedId, setSavedId] = useState<string | null>(isNew ? null : id || null);

  // Load existing definition
  const { data: definition, isLoading } = useQuery({
    queryKey: ['approval-definition', id],
    queryFn: () => approvalDefinitionApi.get(id!),
    enabled: !isNew && !!id,
  });

  const { data: groups = [] } = useQuery({
    queryKey: ['approval-groups'],
    queryFn: approvalGroupApi.list,
  });

  // Populate form when definition loads
  useEffect(() => {
    if (definition) {
      basicForm.setFieldsValue({
        name: definition.name,
        description: definition.description,
        group_name: definition.group_name,
        visibility: definition.visibility || 'all',
      });
      setFormFields(definition.form_schema || []);
      setFlowNodes(
        definition.flow_schema?.nodes?.length > 0
          ? definition.flow_schema.nodes
          : DEFAULT_FLOW_NODES
      );
      setSavedId(definition.id);
    }
  }, [definition, basicForm]);

  // Save mutations
  const createMutation = useMutation({
    mutationFn: (data: Partial<ApprovalDefinition>) => approvalDefinitionApi.create(data),
    onSuccess: (result) => {
      message.success('创建成功');
      setSavedId(result.id);
      queryClient.invalidateQueries({ queryKey: ['approval-definitions'] });
      // Update URL without full reload
      window.history.replaceState(null, '', `/approval-editor/${result.id}`);
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '创建失败');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ defId, data }: { defId: string; data: Partial<ApprovalDefinition> }) =>
      approvalDefinitionApi.update(defId, data),
    onSuccess: () => {
      message.success('保存成功');
      queryClient.invalidateQueries({ queryKey: ['approval-definitions'] });
      queryClient.invalidateQueries({ queryKey: ['approval-definition', savedId] });
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '保存失败');
    },
  });

  const publishMutation = useMutation({
    mutationFn: (defId: string) => approvalDefinitionApi.publish(defId),
    onSuccess: () => {
      message.success('发布成功');
      queryClient.invalidateQueries({ queryKey: ['approval-definitions'] });
      navigate('/approval-admin');
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '发布失败');
    },
  });

  const buildPayload = useCallback(() => {
    const values = basicForm.getFieldsValue();
    return {
      name: values.name,
      description: values.description,
      group_name: values.group_name,
      icon: '📋',
      visibility: values.visibility || 'all',
      form_schema: formFields,
      flow_schema: { nodes: flowNodes },
    };
  }, [basicForm, formFields, flowNodes]);

  const handleSave = useCallback(async () => {
    const payload = buildPayload();
    if (!payload.name) {
      message.warning('请填写审批名称');
      setCurrentStep(0);
      return;
    }
    if (savedId) {
      updateMutation.mutate({ defId: savedId, data: payload });
    } else {
      createMutation.mutate(payload);
    }
  }, [buildPayload, savedId, updateMutation, createMutation]);

  const handlePublish = useCallback(async () => {
    const payload = buildPayload();
    if (!payload.name) {
      message.warning('请填写审批名称');
      setCurrentStep(0);
      return;
    }
    if (savedId) {
      await updateMutation.mutateAsync({ defId: savedId, data: payload });
      publishMutation.mutate(savedId);
    } else {
      const result = await createMutation.mutateAsync(payload);
      publishMutation.mutate(result.id);
    }
  }, [buildPayload, savedId, updateMutation, createMutation, publishMutation]);

  // Form Designer helpers
  const addField = (type: FormField['type']) => {
    const key = Date.now().toString(36) + Math.random().toString(36).substr(2, 4);
    const label = FIELD_TYPE_LABEL[type] || type;
    const newField: FormField = { key, type, label, required: false };
    if (type === 'select' || type === 'multiselect') {
      newField.options = ['选项1', '选项2', '选项3'];
    }
    if (type === 'table') {
      newField.columns = [
        { key: 'col1', label: '列1', type: 'text' },
        { key: 'col2', label: '列2', type: 'text' },
      ];
    }
    if (type === 'money') {
      newField.prefix = '¥';
    }
    setFormFields([...formFields, newField]);
    setSelectedFieldIndex(formFields.length);
  };

  const updateField = (index: number, updates: Partial<FormField>) => {
    const newFields = [...formFields];
    newFields[index] = { ...newFields[index], ...updates };
    setFormFields(newFields);
  };

  const removeField = (index: number) => {
    const newFields = formFields.filter((_, i) => i !== index);
    setFormFields(newFields);
    if (selectedFieldIndex === index) setSelectedFieldIndex(null);
    else if (selectedFieldIndex !== null && selectedFieldIndex > index) {
      setSelectedFieldIndex(selectedFieldIndex - 1);
    }
  };

  const moveField = (index: number, direction: 'up' | 'down') => {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= formFields.length) return;
    const newFields = [...formFields];
    [newFields[index], newFields[newIndex]] = [newFields[newIndex], newFields[index]];
    setFormFields(newFields);
    setSelectedFieldIndex(newIndex);
  };

  // Flow Designer helpers
  const addNodeAfter = (index: number) => {
    const newNode: FlowNode = {
      type: 'approve',
      name: '审批',
      config: { approver_type: 'designated', multi_approve: 'all' },
    };
    const newNodes = [...flowNodes];
    newNodes.splice(index + 1, 0, newNode);
    setFlowNodes(newNodes);
  };

  const updateNode = (index: number, updates: Partial<FlowNode>) => {
    const newNodes = [...flowNodes];
    newNodes[index] = { ...newNodes[index], ...updates };
    setFlowNodes(newNodes);
  };

  const updateNodeConfig = (index: number, configUpdates: Partial<FlowNode['config']>) => {
    const newNodes = [...flowNodes];
    newNodes[index] = {
      ...newNodes[index],
      config: { ...newNodes[index].config, ...configUpdates },
    };
    setFlowNodes(newNodes);
  };

  const removeNode = (index: number) => {
    const approveNodes = flowNodes.filter((n) => n.type === 'approve');
    if (approveNodes.length <= 1) {
      message.warning('至少需要一个审批节点');
      return;
    }
    const newNodes = flowNodes.filter((_, i) => i !== index);
    setFlowNodes(newNodes);
    if (selectedNodeIndex === index) setSelectedNodeIndex(null);
  };

  if (!isNew && isLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

  return (
    <div style={{ padding: 24 }}>
      {/* Header */}
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/approval-admin')} />
          <Title level={3} style={{ margin: 0 }}>
            {isNew ? '创建审批' : `编辑审批 - ${definition?.name || ''}`}
          </Title>
          {definition?.status && (
            <Tag color={definition.status === 'published' ? 'green' : 'default'}>
              {definition.status === 'published' ? '已发布' : '草稿'}
            </Tag>
          )}
        </Space>
        <Space>
          <Button icon={<SaveOutlined />} onClick={handleSave} loading={isSaving}>
            保存
          </Button>
          <Button
            type="primary"
            icon={<SendOutlined />}
            onClick={handlePublish}
            loading={publishMutation.isPending}
          >
            发布
          </Button>
        </Space>
      </div>

      {/* Steps */}
      <Card style={{ marginBottom: 24 }}>
        <Steps
          current={currentStep}
          onChange={setCurrentStep}
          items={[
            { title: '基础信息' },
            { title: '表单设计' },
            { title: '流程设计' },
            { title: '更多设置' },
          ]}
        />
      </Card>

      {/* Step Content */}
      {currentStep === 0 && (
        <Card title="基础信息">
          <Form
            form={basicForm}
            layout="vertical"
            style={{ maxWidth: 600 }}
            initialValues={{ visibility: 'all' }}
          >
            <Form.Item name="name" label="审批名称" rules={[{ required: true, message: '请输入审批名称' }]}>
              <Input placeholder="请输入审批名称" />
            </Form.Item>
            <Form.Item name="description" label="说明">
              <Input.TextArea placeholder="请输入审批说明" rows={3} />
            </Form.Item>
            <Form.Item name="group_name" label="所属分组">
              <Select placeholder="请选择分组" allowClear>
                {groups.map((g: any) => (
                  <Select.Option key={g.id || g.name} value={g.name}>
                    {g.name}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="visibility" label="谁可以提交">
              <Radio.Group>
                <Radio value="all">全员</Radio>
                <Radio value="specified">指定人员</Radio>
              </Radio.Group>
            </Form.Item>
          </Form>
        </Card>
      )}

      {currentStep === 1 && <FormDesigner
        fields={formFields}
        selectedIndex={selectedFieldIndex}
        onAdd={addField}
        onSelect={setSelectedFieldIndex}
        onUpdate={updateField}
        onRemove={removeField}
        onMove={moveField}
      />}

      {currentStep === 2 && <FlowDesigner
        nodes={flowNodes}
        selectedIndex={selectedNodeIndex}
        onSelect={setSelectedNodeIndex}
        onAddAfter={addNodeAfter}
        onUpdateNode={updateNode}
        onUpdateConfig={updateNodeConfig}
        onRemoveNode={removeNode}
      />}

      {currentStep === 3 && (
        <Card title="更多设置">
          <div style={{ maxWidth: 500 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px 0', borderBottom: '1px solid #f0f0f0' }}>
              <div>
                <Text strong>允许撤销审批中的申请</Text>
                <br />
                <Text type="secondary" style={{ fontSize: 12 }}>提交人可以撤销正在审批中的申请</Text>
              </div>
              <Switch checked={allowRevoke} onChange={setAllowRevoke} />
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px 0' }}>
              <div>
                <Text strong>允许审批人撤回</Text>
                <br />
                <Text type="secondary" style={{ fontSize: 12 }}>审批人可以撤回已通过的审批</Text>
              </div>
              <Switch checked={allowRecall} onChange={setAllowRecall} />
            </div>
          </div>
        </Card>
      )}

      {/* Bottom navigation */}
      <div style={{ marginTop: 24, textAlign: 'center' }}>
        <Space>
          {currentStep > 0 && (
            <Button onClick={() => setCurrentStep(currentStep - 1)}>上一步</Button>
          )}
          {currentStep < 3 && (
            <Button type="primary" onClick={() => setCurrentStep(currentStep + 1)}>下一步</Button>
          )}
        </Space>
      </div>
    </div>
  );
};

// ==================== Form Designer ====================

interface FormDesignerProps {
  fields: FormField[];
  selectedIndex: number | null;
  onAdd: (type: FormField['type']) => void;
  onSelect: (index: number | null) => void;
  onUpdate: (index: number, updates: Partial<FormField>) => void;
  onRemove: (index: number) => void;
  onMove: (index: number, direction: 'up' | 'down') => void;
}

const FormDesigner: React.FC<FormDesignerProps> = ({
  fields,
  selectedIndex,
  onAdd,
  onSelect,
  onUpdate,
  onRemove,
  onMove,
}) => {
  const selectedField = selectedIndex !== null ? fields[selectedIndex] : null;

  return (
    <div style={{ display: 'flex', gap: 16, minHeight: 500 }}>
      {/* Left: Widget Palette */}
      <Card
        title="控件"
        size="small"
        style={{ width: 200, flexShrink: 0 }}
        bodyStyle={{ padding: '8px 12px' }}
      >
        {FIELD_TYPES.map((cat) => (
          <div key={cat.category} style={{ marginBottom: 12 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>{cat.category}</Text>
            <div style={{ marginTop: 4 }}>
              {cat.items.map((item) => (
                <Button
                  key={item.type}
                  block
                  size="small"
                  style={{ textAlign: 'left', marginBottom: 4 }}
                  icon={item.icon}
                  onClick={() => onAdd(item.type)}
                >
                  {item.label}
                </Button>
              ))}
            </div>
          </div>
        ))}
      </Card>

      {/* Middle: Form Preview */}
      <Card
        title={`表单字段 (${fields.length})`}
        size="small"
        style={{ flex: 1, minWidth: 300 }}
      >
        {fields.length === 0 ? (
          <Empty description="点击左侧控件添加字段" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <div>
            {fields.map((field, index) => (
              <div
                key={field.key}
                onClick={() => onSelect(index)}
                style={{
                  padding: '10px 12px',
                  marginBottom: 8,
                  border: `2px solid ${selectedIndex === index ? '#1890ff' : '#f0f0f0'}`,
                  borderRadius: 6,
                  cursor: 'pointer',
                  background: selectedIndex === index ? '#e6f7ff' : '#fff',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  transition: 'all 0.2s',
                }}
              >
                <Space>
                  <span style={{ color: '#999' }}>{FIELD_TYPE_ICON[field.type]}</span>
                  <Text strong>{field.label}</Text>
                  <Tag>{FIELD_TYPE_LABEL[field.type]}</Tag>
                  {field.required && <Tag color="red">必填</Tag>}
                </Space>
                <Space size={2}>
                  <Button
                    type="text"
                    size="small"
                    icon={<ArrowUpOutlined />}
                    disabled={index === 0}
                    onClick={(e) => { e.stopPropagation(); onMove(index, 'up'); }}
                  />
                  <Button
                    type="text"
                    size="small"
                    icon={<ArrowDownOutlined />}
                    disabled={index === fields.length - 1}
                    onClick={(e) => { e.stopPropagation(); onMove(index, 'down'); }}
                  />
                  <Button
                    type="text"
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={(e) => { e.stopPropagation(); onRemove(index); }}
                  />
                </Space>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Right: Field Config */}
      <Card
        title="字段设置"
        size="small"
        style={{ width: 300, flexShrink: 0 }}
        extra={selectedField && (
          <Button
            type="text"
            size="small"
            icon={<CloseOutlined />}
            onClick={() => onSelect(null)}
          />
        )}
      >
        {selectedField && selectedIndex !== null ? (
          <FieldConfigPanel
            field={selectedField}
            onChange={(updates) => onUpdate(selectedIndex, updates)}
          />
        ) : (
          <Empty description="选择字段以编辑属性" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
      </Card>
    </div>
  );
};

// ==================== Field Config Panel ====================

interface FieldConfigPanelProps {
  field: FormField;
  onChange: (updates: Partial<FormField>) => void;
}

const FieldConfigPanel: React.FC<FieldConfigPanelProps> = ({ field, onChange }) => {
  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>标签名称</Text>
        <Input
          value={field.label}
          onChange={(e) => onChange({ label: e.target.value })}
          style={{ marginTop: 4 }}
        />
      </div>

      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text type="secondary" style={{ fontSize: 12 }}>是否必填</Text>
        <Switch
          checked={field.required}
          onChange={(checked) => onChange({ required: checked })}
          size="small"
        />
      </div>

      {field.type !== 'description' && (
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>提示文字</Text>
          <Input
            value={field.placeholder}
            onChange={(e) => onChange({ placeholder: e.target.value })}
            placeholder="请输入提示文字"
            style={{ marginTop: 4 }}
          />
        </div>
      )}

      {field.type === 'money' && (
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>货币符号</Text>
          <Input
            value={field.prefix || '¥'}
            onChange={(e) => onChange({ prefix: e.target.value })}
            style={{ marginTop: 4, width: 80 }}
          />
        </div>
      )}

      {(field.type === 'select' || field.type === 'multiselect') && (
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>选项列表</Text>
          <div style={{ marginTop: 4 }}>
            {(field.options || []).map((opt, idx) => (
              <div key={idx} style={{ display: 'flex', gap: 4, marginBottom: 4 }}>
                <Input
                  size="small"
                  value={opt}
                  onChange={(e) => {
                    const newOpts = [...(field.options || [])];
                    newOpts[idx] = e.target.value;
                    onChange({ options: newOpts });
                  }}
                />
                <Button
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => {
                    const newOpts = (field.options || []).filter((_, i) => i !== idx);
                    onChange({ options: newOpts });
                  }}
                />
              </div>
            ))}
            <Button
              size="small"
              type="dashed"
              block
              icon={<PlusOutlined />}
              onClick={() => {
                const newOpts = [...(field.options || []), `选项${(field.options?.length || 0) + 1}`];
                onChange({ options: newOpts });
              }}
            >
              添加选项
            </Button>
          </div>
        </div>
      )}

      {field.type === 'table' && (
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>列定义</Text>
          <div style={{ marginTop: 4 }}>
            {(field.columns || []).map((col, idx) => (
              <div key={idx} style={{ display: 'flex', gap: 4, marginBottom: 4 }}>
                <Input
                  size="small"
                  placeholder="列名"
                  value={col.label}
                  onChange={(e) => {
                    const newCols = [...(field.columns || [])];
                    newCols[idx] = { ...newCols[idx], label: e.target.value };
                    onChange({ columns: newCols });
                  }}
                  style={{ flex: 1 }}
                />
                <Select
                  size="small"
                  value={col.type}
                  onChange={(val) => {
                    const newCols = [...(field.columns || [])];
                    newCols[idx] = { ...newCols[idx], type: val };
                    onChange({ columns: newCols });
                  }}
                  style={{ width: 80 }}
                  options={[
                    { label: '文本', value: 'text' },
                    { label: '数字', value: 'number' },
                    { label: '日期', value: 'date' },
                  ]}
                />
                <Button
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => {
                    const newCols = (field.columns || []).filter((_, i) => i !== idx);
                    onChange({ columns: newCols });
                  }}
                />
              </div>
            ))}
            <Button
              size="small"
              type="dashed"
              block
              icon={<PlusOutlined />}
              onClick={() => {
                const key = 'col' + Date.now().toString(36);
                const newCols = [...(field.columns || []), { key, label: '新列', type: 'text' }];
                onChange({ columns: newCols });
              }}
            >
              添加列
            </Button>
          </div>
        </div>
      )}

      {field.type === 'description' && (
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>说明内容会在表单中显示为提示文字</Text>
        </div>
      )}
    </div>
  );
};

// ==================== Flow Designer ====================

interface FlowDesignerProps {
  nodes: FlowNode[];
  selectedIndex: number | null;
  onSelect: (index: number | null) => void;
  onAddAfter: (index: number) => void;
  onUpdateNode: (index: number, updates: Partial<FlowNode>) => void;
  onUpdateConfig: (index: number, configUpdates: Partial<FlowNode['config']>) => void;
  onRemoveNode: (index: number) => void;
}

const APPROVER_TYPE_LABEL: Record<string, string> = {
  designated: '指定成员',
  self_select: '提交人自选',
  supervisor: '直属上级',
  dept_leader: '部门负责人',
  submitter: '提交人本人',
  role: '角色',
};

const MULTI_APPROVE_LABEL: Record<string, string> = {
  all: '会签（需所有人同意）',
  any: '或签（一人同意即可）',
  sequential: '依次审批',
};

const FlowDesigner: React.FC<FlowDesignerProps> = ({
  nodes,
  selectedIndex,
  onSelect,
  onAddAfter,
  onUpdateNode,
  onUpdateConfig,
  onRemoveNode,
}) => {
  const selectedNode = selectedIndex !== null ? nodes[selectedIndex] : null;

  const getNodeDescription = (node: FlowNode): string => {
    if (node.type === 'submit') return '发起人提交';
    if (node.type === 'end') return '流程结束';
    const typeLabel = APPROVER_TYPE_LABEL[node.config.approver_type || ''] || '';
    return typeLabel;
  };

  return (
    <div style={{ display: 'flex', gap: 16, minHeight: 500 }}>
      {/* Flow Diagram */}
      <div style={{ flex: 1, display: 'flex', justifyContent: 'center' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
          {nodes.map((node, index) => (
            <React.Fragment key={index}>
              {/* Node */}
              <div
                onClick={() => onSelect(index)}
                style={{
                  width: 300,
                  padding: '12px 16px',
                  border: `2px solid ${
                    selectedIndex === index
                      ? '#1890ff'
                      : node.type === 'approve'
                      ? '#fa8c16'
                      : '#d9d9d9'
                  }`,
                  borderRadius: 8,
                  background:
                    selectedIndex === index
                      ? '#e6f7ff'
                      : node.type === 'approve'
                      ? '#fff7e6'
                      : '#fafafa',
                  cursor: 'pointer',
                  transition: 'all 0.2s',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Space>
                    <Tag color={node.type === 'approve' ? 'orange' : 'default'}>
                      {node.name}
                    </Tag>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {getNodeDescription(node)}
                    </Text>
                  </Space>
                  {node.type === 'approve' && (
                    <Popconfirm
                      title="确定删除该节点？"
                      onConfirm={(e) => { e?.stopPropagation(); onRemoveNode(index); }}
                      onCancel={(e) => e?.stopPropagation()}
                    >
                      <Button
                        type="text"
                        size="small"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </Popconfirm>
                  )}
                </div>
              </div>

              {/* Connector */}
              {index < nodes.length - 1 && (
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', padding: '4px 0' }}>
                  <div style={{ width: 2, height: 16, background: '#d9d9d9' }} />
                  <Button
                    icon={<PlusOutlined />}
                    shape="circle"
                    size="small"
                    onClick={() => onAddAfter(index)}
                  />
                  <div style={{ width: 2, height: 16, background: '#d9d9d9' }} />
                </div>
              )}
            </React.Fragment>
          ))}
        </div>
      </div>

      {/* Right: Node Config */}
      <Card
        title="节点设置"
        size="small"
        style={{ width: 320, flexShrink: 0 }}
        extra={selectedNode && (
          <Button
            type="text"
            size="small"
            icon={<CloseOutlined />}
            onClick={() => onSelect(null)}
          />
        )}
      >
        {selectedNode && selectedIndex !== null ? (
          <NodeConfigPanel
            node={selectedNode}
            index={selectedIndex}
            onUpdateNode={onUpdateNode}
            onUpdateConfig={onUpdateConfig}
          />
        ) : (
          <Empty description="选择节点以编辑配置" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
      </Card>
    </div>
  );
};

// ==================== Node Config Panel ====================

interface NodeConfigPanelProps {
  node: FlowNode;
  index: number;
  onUpdateNode: (index: number, updates: Partial<FlowNode>) => void;
  onUpdateConfig: (index: number, configUpdates: Partial<FlowNode['config']>) => void;
}

const NodeConfigPanel: React.FC<NodeConfigPanelProps> = ({
  node,
  index,
  onUpdateNode,
  onUpdateConfig,
}) => {
  if (node.type === 'submit') {
    return (
      <div>
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>节点名称</Text>
          <Input
            value={node.name}
            onChange={(e) => onUpdateNode(index, { name: e.target.value })}
            style={{ marginTop: 4 }}
          />
        </div>
        <Text type="secondary" style={{ fontSize: 12 }}>
          提交人根据审批定义的可见范围确定
        </Text>
      </div>
    );
  }

  if (node.type === 'end') {
    return (
      <div>
        <Text type="secondary">流程结束节点，无需配置</Text>
      </div>
    );
  }

  // Approve node
  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>节点名称</Text>
        <Input
          value={node.name}
          onChange={(e) => onUpdateNode(index, { name: e.target.value })}
          style={{ marginTop: 4 }}
        />
      </div>

      <div style={{ marginBottom: 16 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>审批人类型</Text>
        <Select
          value={node.config.approver_type || 'designated'}
          onChange={(val) => onUpdateConfig(index, { approver_type: val })}
          style={{ width: '100%', marginTop: 4 }}
          options={Object.entries(APPROVER_TYPE_LABEL).map(([value, label]) => ({ value, label }))}
        />
      </div>

      {node.config.approver_type === 'designated' && (
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ fontSize: 12 }}>指定审批人</Text>
          <div style={{ marginTop: 4 }}>
            <UserSelect
              mode="multiple"
              value={node.config.approver_ids || []}
              onChange={(val) => onUpdateConfig(index, { approver_ids: val as string[] })}
              placeholder="选择审批人"
              style={{ width: '100%' }}
            />
          </div>
        </div>
      )}

      <div style={{ marginBottom: 16 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>多人审批方式</Text>
        <Select
          value={node.config.multi_approve || 'all'}
          onChange={(val) => onUpdateConfig(index, { multi_approve: val })}
          style={{ width: '100%', marginTop: 4 }}
          options={Object.entries(MULTI_APPROVE_LABEL).map(([value, label]) => ({ value, label }))}
        />
      </div>
    </div>
  );
};

export default ApprovalEditor;
