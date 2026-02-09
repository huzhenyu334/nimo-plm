import React, { useState, useEffect, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  Tabs,
  Tag,
  Typography,
  Space,
  Button,
  Modal,
  Input,
  InputNumber,
  Select,
  DatePicker,
  Form,
  message,
  Empty,
  Avatar,
  Timeline,
  Spin,
  Menu,
  Upload,
  Descriptions,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  SearchOutlined,
  UserOutlined,
  FileTextOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import {
  approvalDefinitionApi,
  approvalInstanceApi,
  ApprovalDefinition,
  ApprovalInstance,
  FormField,
} from '@/api/approvalDefinitions';
import { taskFormApi } from '@/api/taskForms';
import { userApi } from '@/api/users';
import { ROLE_CODES } from '@/constants/roles';
import UserSelect from '@/components/UserSelect';
import dayjs from 'dayjs';

const { Title, Text } = Typography;

const statusConfig: Record<string, { color: string; text: string; icon: React.ReactNode }> = {
  pending: { color: 'processing', text: '审批中', icon: <ClockCircleOutlined /> },
  approved: { color: 'success', text: '已通过', icon: <CheckCircleOutlined /> },
  rejected: { color: 'error', text: '已拒绝', icon: <CloseCircleOutlined /> },
  cancelled: { color: 'default', text: '已取消', icon: <CloseCircleOutlined /> },
};

const Approvals: React.FC = () => {
  const [activeTab, setActiveTab] = useState('submit');

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={3} style={{ margin: 0 }}>审批中心</Title>
      </div>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        size="large"
        items={[
          { key: 'submit', label: '发起申请' },
          { key: 'center', label: '审批中心' },
        ]}
      />

      {activeTab === 'submit' ? <SubmitTab /> : <ApprovalCenterTab />}
    </div>
  );
};

// ==================== Submit Tab ====================

const SubmitTab: React.FC = () => {
  const [searchText, setSearchText] = useState('');
  const [submitModalOpen, setSubmitModalOpen] = useState(false);
  const [selectedDef, setSelectedDef] = useState<ApprovalDefinition | null>(null);

  const { data: defData, isLoading } = useQuery({
    queryKey: ['approval-definitions'],
    queryFn: approvalDefinitionApi.list,
  });

  const allDefs = useMemo(() => {
    const groups = defData?.groups || [];
    const defs: ApprovalDefinition[] = [];
    groups.forEach((g: { definitions: ApprovalDefinition[] }) => {
      g.definitions.forEach((d: ApprovalDefinition) => {
        if (d.status === 'published') defs.push(d);
      });
    });
    return defs;
  }, [defData]);

  const filteredDefs = useMemo(() => {
    if (!searchText) return allDefs;
    return allDefs.filter((d) => d.name.toLowerCase().includes(searchText.toLowerCase()));
  }, [allDefs, searchText]);

  // Group by group_name
  const groupedDefs = useMemo(() => {
    const map: Record<string, ApprovalDefinition[]> = {};
    filteredDefs.forEach((d) => {
      const key = d.group_name || '未分组';
      if (!map[key]) map[key] = [];
      map[key].push(d);
    });
    return map;
  }, [filteredDefs]);

  const handleClickDef = (def: ApprovalDefinition) => {
    setSelectedDef(def);
    setSubmitModalOpen(true);
  };

  return (
    <div>
      <Input
        placeholder="搜索审批类型"
        prefix={<SearchOutlined />}
        value={searchText}
        onChange={(e) => setSearchText(e.target.value)}
        style={{ width: 300, marginBottom: 24 }}
        allowClear
      />

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
      ) : allDefs.length === 0 ? (
        <Empty description="暂无可用的审批类型" />
      ) : (
        Object.entries(groupedDefs).map(([groupName, defs]) => (
          <div key={groupName} style={{ marginBottom: 24 }}>
            <Text strong style={{ fontSize: 16, marginBottom: 12, display: 'block' }}>{groupName}</Text>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
              {defs.map((def) => (
                <Card
                  key={def.id}
                  hoverable
                  style={{ width: 160, textAlign: 'center' }}
                  bodyStyle={{ padding: 16 }}
                  onClick={() => handleClickDef(def)}
                >
                  <div
                    style={{
                      width: 48,
                      height: 48,
                      borderRadius: 12,
                      background: '#1890ff',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 24,
                      color: '#fff',
                      margin: '0 auto 8px',
                    }}
                  >
                    {def.icon || <FileTextOutlined />}
                  </div>
                  <Text strong>{def.name}</Text>
                </Card>
              ))}
            </div>
          </div>
        ))
      )}

      {/* Submit Modal */}
      {selectedDef && (
        <SubmitApprovalModal
          open={submitModalOpen}
          definition={selectedDef}
          onClose={() => { setSubmitModalOpen(false); setSelectedDef(null); }}
        />
      )}
    </div>
  );
};

// ==================== Submit Approval Modal ====================

interface SubmitApprovalModalProps {
  open: boolean;
  definition: ApprovalDefinition;
  onClose: () => void;
}

const SubmitApprovalModal: React.FC<SubmitApprovalModalProps> = ({ open, definition, onClose }) => {
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const [approverIds, setApproverIds] = useState<string[]>([]);

  const hasSelfSelect = definition.flow_schema?.nodes?.some(
    (n) => n.type === 'approve' && n.config.approver_type === 'self_select'
  );

  const submitMutation = useMutation({
    mutationFn: (formData: Record<string, any>) =>
      approvalDefinitionApi.submit(definition.id, {
        form_data: formData,
        approver_ids: hasSelfSelect ? approverIds : undefined,
      }),
    onSuccess: () => {
      message.success('提交成功');
      queryClient.invalidateQueries({ queryKey: ['approvals'] });
      form.resetFields();
      setApproverIds([]);
      onClose();
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '提交失败');
    },
  });

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      // Process date fields
      const formData: Record<string, any> = {};
      (definition.form_schema || []).forEach((field) => {
        let val = values[field.key];
        if (field.type === 'date' && val) {
          val = val.format('YYYY-MM-DD');
        } else if (field.type === 'daterange' && val) {
          val = [val[0]?.format('YYYY-MM-DD'), val[1]?.format('YYYY-MM-DD')];
        }
        formData[field.key] = val;
      });
      submitMutation.mutate(formData);
    } catch {
      // form validation failed
    }
  };

  return (
    <Modal
      title={`提交审批 - ${definition.name}`}
      open={open}
      onCancel={onClose}
      onOk={handleSubmit}
      confirmLoading={submitMutation.isPending}
      okText="提交"
      cancelText="取消"
      width={600}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        {(definition.form_schema || []).map((field) => (
          <DynamicFormField key={field.key} field={field} />
        ))}

        {hasSelfSelect && (
          <Form.Item label="选择审批人" required>
            <UserSelect
              mode="multiple"
              value={approverIds}
              onChange={(val) => setApproverIds(val as string[])}
              placeholder="请选择审批人"
              style={{ width: '100%' }}
            />
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
};

// ==================== Dynamic Form Field ====================

const DynamicFormField: React.FC<{ field: FormField }> = ({ field }) => {
  if (field.type === 'description') {
    return (
      <div style={{ marginBottom: 16 }}>
        <Text type="secondary">{field.label}</Text>
      </div>
    );
  }

  const rules = field.required ? [{ required: true, message: `请填写${field.label}` }] : [];

  switch (field.type) {
    case 'text':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <Input placeholder={field.placeholder || `请输入${field.label}`} />
        </Form.Item>
      );
    case 'textarea':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <Input.TextArea placeholder={field.placeholder || `请输入${field.label}`} rows={3} />
        </Form.Item>
      );
    case 'number':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <InputNumber
            placeholder={field.placeholder || `请输入${field.label}`}
            style={{ width: '100%' }}
          />
        </Form.Item>
      );
    case 'money':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <InputNumber
            prefix={field.prefix || '¥'}
            placeholder={field.placeholder || '0.00'}
            style={{ width: '100%' }}
            precision={2}
          />
        </Form.Item>
      );
    case 'select':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <Select
            placeholder={field.placeholder || `请选择${field.label}`}
            options={(field.options || []).map((o) => ({ label: o, value: o }))}
          />
        </Form.Item>
      );
    case 'multiselect':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <Select
            mode="multiple"
            placeholder={field.placeholder || `请选择${field.label}`}
            options={(field.options || []).map((o) => ({ label: o, value: o }))}
          />
        </Form.Item>
      );
    case 'date':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <DatePicker placeholder={field.placeholder || '选择日期'} style={{ width: '100%' }} />
        </Form.Item>
      );
    case 'daterange':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <DatePicker.RangePicker style={{ width: '100%' }} />
        </Form.Item>
      );
    case 'user':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <UserSelect
            mode={field.multiple ? 'multiple' : 'single'}
            placeholder={field.placeholder || '选择人员'}
            style={{ width: '100%' }}
          />
        </Form.Item>
      );
    case 'attachment':
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <Upload.Dragger>
            <p className="ant-upload-drag-icon"><InboxOutlined /></p>
            <p className="ant-upload-text">点击或拖拽上传</p>
          </Upload.Dragger>
        </Form.Item>
      );
    case 'table':
      return (
        <Form.Item name={field.key} label={field.label}>
          <Text type="secondary">明细表格（暂未实现动态编辑）</Text>
        </Form.Item>
      );
    default:
      return (
        <Form.Item name={field.key} label={field.label} rules={rules}>
          <Input placeholder={field.placeholder} />
        </Form.Item>
      );
  }
};

// ==================== Approval Center Tab ====================

const ApprovalCenterTab: React.FC = () => {
  const queryClient = useQueryClient();
  const [listType, setListType] = useState('my_pending');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [rejectModalOpen, setRejectModalOpen] = useState(false);
  const [rejectComment, setRejectComment] = useState('');

  const { data: approvals = [], isLoading } = useQuery({
    queryKey: ['approvals', listType],
    queryFn: () => approvalInstanceApi.list({ type: listType }),
    refetchInterval: 10000,
  });

  const { data: selectedApproval, isLoading: detailLoading } = useQuery({
    queryKey: ['approval-detail', selectedId],
    queryFn: () => approvalInstanceApi.get(selectedId!),
    enabled: !!selectedId,
  });

  const approveMutation = useMutation({
    mutationFn: ({ id, comment }: { id: string; comment?: string }) =>
      approvalInstanceApi.approve(id, comment),
    onSuccess: () => {
      message.success('已同意');
      queryClient.invalidateQueries({ queryKey: ['approvals'] });
      queryClient.invalidateQueries({ queryKey: ['approval-detail'] });
      // 审批通过会改变任务状态，刷新所有项目和任务缓存
      queryClient.invalidateQueries({ queryKey: ['project-tasks'] });
      queryClient.invalidateQueries({ queryKey: ['project'] });
      queryClient.invalidateQueries({ queryKey: ['my-tasks'] });
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '操作失败');
    },
  });

  const rejectMutation = useMutation({
    mutationFn: ({ id, comment }: { id: string; comment: string }) =>
      approvalInstanceApi.reject(id, comment),
    onSuccess: () => {
      message.success('已拒绝');
      setRejectModalOpen(false);
      setRejectComment('');
      queryClient.invalidateQueries({ queryKey: ['approvals'] });
      queryClient.invalidateQueries({ queryKey: ['approval-detail'] });
      // 驳回也会改变任务状态
      queryClient.invalidateQueries({ queryKey: ['project-tasks'] });
      queryClient.invalidateQueries({ queryKey: ['project'] });
      queryClient.invalidateQueries({ queryKey: ['my-tasks'] });
    },
    onError: (err: any) => {
      message.error(err?.response?.data?.error || '操作失败');
    },
  });

  const handleApprove = (id: string) => {
    approveMutation.mutate({ id });
  };

  const approvalList = Array.isArray(approvals) ? approvals : [];

  return (
    <div style={{ display: 'flex', gap: 16, minHeight: 500 }}>
      {/* Left sidebar */}
      <div style={{ width: 160, flexShrink: 0 }}>
        <Menu
          mode="inline"
          selectedKeys={[listType]}
          onClick={(e) => { setListType(e.key); setSelectedId(null); }}
          items={[
            { key: 'my_pending', label: '待我处理', icon: <ClockCircleOutlined /> },
            { key: 'my_approved', label: '我已处理', icon: <CheckCircleOutlined /> },
            { key: 'my_submitted', label: '我发起的', icon: <UserOutlined /> },
          ]}
          style={{ border: 'none' }}
        />
      </div>

      {/* Approval list */}
      <div style={{ width: 320, flexShrink: 0, overflowY: 'auto', borderRight: '1px solid #f0f0f0', paddingRight: 16 }}>
        {isLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : approvalList.length === 0 ? (
          <Empty description="暂无审批" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          approvalList.map((item: ApprovalInstance) => (
            <Card
              key={item.id}
              size="small"
              hoverable
              style={{
                marginBottom: 8,
                border: selectedId === item.id ? '2px solid #1890ff' : '1px solid #f0f0f0',
              }}
              bodyStyle={{ padding: 12 }}
              onClick={() => setSelectedId(item.id)}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div>
                  <Text strong style={{ fontSize: 14 }}>
                    {item.definition_name || '审批'}
                  </Text>
                  <br />
                  <Space size={4} style={{ marginTop: 4 }}>
                    <Avatar size="small" icon={<UserOutlined />} src={item.requester?.avatar_url}>
                      {item.requester?.name?.[0]}
                    </Avatar>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {item.requester?.name || '未知'}
                    </Text>
                  </Space>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <Tag color={statusConfig[item.status]?.color || 'default'}>
                    {statusConfig[item.status]?.text || item.status}
                  </Tag>
                  <br />
                  <Text type="secondary" style={{ fontSize: 11 }}>
                    {item.created_at ? dayjs(item.created_at).format('MM-DD HH:mm') : ''}
                  </Text>
                </div>
              </div>
            </Card>
          ))
        )}
      </div>

      {/* Approval detail */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {!selectedId ? (
          <div style={{ textAlign: 'center', padding: 80 }}>
            <Empty description="选择一个审批查看详情" />
          </div>
        ) : detailLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : selectedApproval ? (
          <ApprovalDetail
            approval={selectedApproval}
            onApprove={handleApprove}
            onReject={(id: string) => { setSelectedId(id); setRejectModalOpen(true); }}
            approvePending={approveMutation.isPending}
          />
        ) : (
          <Empty description="无法加载详情" />
        )}
      </div>

      {/* Reject Modal */}
      <Modal
        title="拒绝审批"
        open={rejectModalOpen}
        onCancel={() => { setRejectModalOpen(false); setRejectComment(''); }}
        onOk={() => {
          if (!rejectComment.trim()) {
            message.warning('请输入拒绝原因');
            return;
          }
          if (selectedId) rejectMutation.mutate({ id: selectedId, comment: rejectComment.trim() });
        }}
        confirmLoading={rejectMutation.isPending}
        okText="确认拒绝"
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <Input.TextArea
          rows={4}
          placeholder="请输入拒绝原因..."
          value={rejectComment}
          onChange={(e) => setRejectComment(e.target.value)}
        />
      </Modal>
    </div>
  );
};

// ==================== Task Form Section ====================

const TaskFormSection: React.FC<{ projectId: string; taskId: string }> = ({ projectId, taskId }) => {
  const [formDef, setFormDef] = useState<any>(null);
  const [submission, setSubmission] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [userMap, setUserMap] = useState<Record<string, string>>({});

  useEffect(() => {
    Promise.all([
      taskFormApi.getForm(projectId, taskId),
      taskFormApi.getSubmission(projectId, taskId),
    ]).then(([form, sub]) => {
      setFormDef(form);
      setSubmission(sub);
      if (form?.fields?.some((f: any) => f.type === 'user' || f.type === 'role_assignment')) {
        userApi.list().then((users) => {
          const map: Record<string, string> = {};
          users.forEach((u: any) => { map[u.id] = u.name; });
          setUserMap(map);
        });
      }
    }).finally(() => setLoading(false));
  }, [projectId, taskId]);

  if (loading) return <div style={{ color: '#999', fontSize: 12, padding: '8px 0' }}>加载任务表单数据...</div>;
  if (!formDef || !submission) return null;

  const fields = formDef.fields || [];
  const data = submission.data || {};

  return (
    <div style={{ marginTop: 16, padding: 12, background: '#f6ffed', borderRadius: 6, border: '1px solid #b7eb8f' }}>
      <Text strong style={{ fontSize: 13, marginBottom: 8, display: 'block' }}>关联任务表单数据</Text>
      <Descriptions size="small" column={2} bordered>
        {fields.map((field: any) => {
          let value = data[field.key];
          if (value === undefined || value === null) value = '-';
          else if (field.type === 'role_assignment' && typeof value === 'object' && !Array.isArray(value)) {
            const roleMap = Object.fromEntries(ROLE_CODES.map((r: any) => [r.code, r.label]));
            const lines = Object.entries(value as Record<string, string>)
              .map(([code, uid]) => `${roleMap[code] || code}: ${userMap[uid] || uid}`)
              .join('; ');
            value = lines || '-';
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
      <Text type="secondary" style={{ fontSize: 11, marginTop: 4, display: 'block' }}>
        提交时间: {submission.submitted_at ? dayjs(submission.submitted_at).format('YYYY-MM-DD HH:mm') : '-'}
      </Text>
    </div>
  );
};

// ==================== Approval Detail ====================

interface ApprovalDetailProps {
  approval: ApprovalInstance;
  onApprove: (id: string) => void;
  onReject: (id: string) => void;
  approvePending: boolean;
}

const ApprovalDetail: React.FC<ApprovalDetailProps> = ({
  approval,
  onApprove,
  onReject,
  approvePending,
}) => {
  const [detailTab, setDetailTab] = useState('detail');

  return (
    <Card
      title={
        <Space>
          <Text strong style={{ fontSize: 16 }}>{approval.definition_name || '审批详情'}</Text>
          <Tag color={statusConfig[approval.status]?.color || 'default'}>
            {statusConfig[approval.status]?.text || approval.status}
          </Tag>
        </Space>
      }
      extra={
        <Text type="secondary" style={{ fontSize: 12 }}>
          {approval.created_at ? dayjs(approval.created_at).format('YYYY-MM-DD HH:mm') : ''}
        </Text>
      }
    >
      <Tabs
        activeKey={detailTab}
        onChange={setDetailTab}
        items={[
          { key: 'detail', label: '审批详情' },
          { key: 'timeline', label: '审批记录' },
        ]}
      />

      {detailTab === 'detail' ? (
        <div>
          {/* Submitter info */}
          <div style={{ marginBottom: 16, padding: 12, background: '#fafafa', borderRadius: 6 }}>
            <Space>
              <Avatar icon={<UserOutlined />} src={approval.requester?.avatar_url}>
                {approval.requester?.name?.[0]}
              </Avatar>
              <div>
                <Text strong>{approval.requester?.name || '未知'}</Text>
                <br />
                <Text type="secondary" style={{ fontSize: 12 }}>提交人</Text>
              </div>
            </Space>
          </div>

          {/* Form data */}
          {approval.form_data && Object.keys(approval.form_data).length > 0 ? (
            <div>
              {Object.entries(approval.form_data).map(([key, value]) => {
                // Try to find field label from definition
                const fieldDef = approval.definition?.form_schema?.find((f) => f.key === key);
                const label = fieldDef?.label || key;
                return (
                  <div key={key} style={{ marginBottom: 12, padding: '8px 0', borderBottom: '1px solid #f5f5f5' }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>{label}</Text>
                    <div style={{ marginTop: 2 }}>
                      <Text>{formatValue(value)}</Text>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <Empty description="无表单数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}

          {/* Task form data */}
          {(approval as any).project_id && (approval as any).task_id && (
            <TaskFormSection
              projectId={(approval as any).project_id}
              taskId={(approval as any).task_id}
            />
          )}
        </div>
      ) : (
        <div>
          {/* Timeline */}
          <Timeline
            items={[
              {
                color: 'green',
                children: (
                  <div>
                    <Text strong>{approval.requester?.name || '提交人'}</Text>
                    <Text type="secondary"> 提交了审批</Text>
                    <div style={{ fontSize: 12, color: '#999', marginTop: 2 }}>
                      {approval.created_at ? dayjs(approval.created_at).format('YYYY-MM-DD HH:mm:ss') : ''}
                    </div>
                  </div>
                ),
              },
              ...(approval.steps || []).flatMap((step) =>
                (step.approvers || []).map((approver) => ({
                  color:
                    approver.status === 'approved'
                      ? 'green'
                      : approver.status === 'rejected'
                      ? 'red'
                      : 'gray',
                  children: (
                    <div>
                      <Space>
                        <Avatar size="small" icon={<UserOutlined />} src={approver.user?.avatar_url}>
                          {approver.user?.name?.[0]}
                        </Avatar>
                        <Text strong>{approver.user?.name || approver.user_id}</Text>
                        <Tag color={statusConfig[approver.status]?.color || 'default'}>
                          {statusConfig[approver.status]?.text || approver.status}
                        </Tag>
                      </Space>
                      {approver.comment && (
                        <div style={{ marginTop: 4, color: '#666', fontSize: 12 }}>
                          {approver.comment}
                        </div>
                      )}
                      {approver.decided_at && (
                        <div style={{ fontSize: 12, color: '#999', marginTop: 2 }}>
                          {dayjs(approver.decided_at).format('YYYY-MM-DD HH:mm:ss')}
                        </div>
                      )}
                    </div>
                  ),
                }))
              ),
            ]}
          />
        </div>
      )}

      {/* Action buttons */}
      {approval.status === 'pending' && (
        <div style={{ marginTop: 24, padding: '16px 0', borderTop: '1px solid #f0f0f0', textAlign: 'right' }}>
          <Space>
            <Button
              type="primary"
              style={{ background: '#52c41a', borderColor: '#52c41a' }}
              onClick={() => onApprove(approval.id)}
              loading={approvePending}
            >
              同意
            </Button>
            <Button danger onClick={() => onReject(approval.id)}>
              拒绝
            </Button>
          </Space>
        </div>
      )}
    </Card>
  );
};

function formatValue(value: any): string {
  if (value === null || value === undefined) return '-';
  if (Array.isArray(value)) return value.join(', ');
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

export default Approvals;
