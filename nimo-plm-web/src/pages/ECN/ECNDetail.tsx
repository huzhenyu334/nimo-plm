import { useState, useEffect } from 'react';
import { Card, Tag, Space, Button, Tabs, Descriptions, Timeline, Progress, Modal, Input, App, Breadcrumb, Checkbox, Steps, Empty } from 'antd';
import { ArrowLeftOutlined, EditOutlined, SendOutlined, CheckOutlined, CloseOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { ecnApi, ECN, ECNHistory, ECNTask, ECNApproval } from '@/api/ecn';

const statusMap: Record<string, { color: string; label: string }> = {
  draft: { color: 'default', label: '草稿' },
  pending: { color: 'processing', label: '待审批' },
  approved: { color: 'success', label: '已批准' },
  rejected: { color: 'error', label: '已驳回' },
  executing: { color: 'warning', label: '执行中' },
  closed: { color: 'default', label: '已关闭' },
  implemented: { color: 'success', label: '已实施' },
};

const urgencyMap: Record<string, { color: string; label: string }> = {
  low: { color: 'blue', label: '常规' },
  medium: { color: 'blue', label: '常规' },
  high: { color: 'orange', label: '紧急' },
  critical: { color: 'red', label: '特急' },
};

const changeTypeMap: Record<string, string> = {
  design: '设计变更',
  material: '物料变更',
  process: '工艺变更',
  spec: '规格变更',
  document: '文档变更',
};

const taskTypeMap: Record<string, string> = {
  bom_update: 'BOM更新',
  drawing_update: '图纸更新',
  supplier_notify: '供应商通知',
  inventory_handle: '库存处理',
  doc_update: '文档更新',
  sop_update: 'SOP更新',
};

const historyActionMap: Record<string, string> = {
  created: '创建ECN',
  updated: '编辑ECN',
  submitted: '提交审批',
  approved: '审批通过',
  rejected: '审批驳回',
  executing: '开始执行',
  task_completed: '完成任务',
  closed: 'ECN关闭',
  bom_applied: '应用BOM变更',
};

const ECNDetail = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { message, modal } = App.useApp();
  const [ecn, setEcn] = useState<ECN | null>(null);
  const [loading, setLoading] = useState(true);
  const [history, setHistory] = useState<ECNHistory[]>([]);
  const [tasks, setTasks] = useState<ECNTask[]>([]);
  const [approvalComment, setApprovalComment] = useState('');
  const [rejectReason, setRejectReason] = useState('');
  const [approveModalOpen, setApproveModalOpen] = useState(false);
  const [rejectModalOpen, setRejectModalOpen] = useState(false);

  const fetchECN = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const data = await ecnApi.get(id);
      setEcn(data);
    } catch {
      message.error('加载ECN详情失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchHistory = async () => {
    if (!id) return;
    try {
      const data = await ecnApi.listHistory(id);
      setHistory(data.items || []);
    } catch { /* ignore */ }
  };

  const fetchTasks = async () => {
    if (!id) return;
    try {
      const data = await ecnApi.listTasks(id);
      setTasks(data.items || []);
    } catch { /* ignore */ }
  };

  useEffect(() => {
    fetchECN();
    fetchHistory();
    fetchTasks();
  }, [id]);

  const handleSubmit = async () => {
    if (!id) return;
    modal.confirm({
      title: '确认提交审批？',
      content: '提交后将进入审批流程，ECN将不可编辑。',
      onOk: async () => {
        try {
          await ecnApi.submit(id);
          message.success('已提交审批');
          fetchECN();
          fetchHistory();
        } catch {
          message.error('提交失败');
        }
      },
    });
  };

  const handleApprove = async () => {
    if (!id) return;
    try {
      await ecnApi.approve(id, { comment: approvalComment });
      message.success('已批准');
      setApproveModalOpen(false);
      setApprovalComment('');
      fetchECN();
      fetchHistory();
      fetchTasks();
    } catch {
      message.error('操作失败');
    }
  };

  const handleReject = async () => {
    if (!id || !rejectReason) return;
    try {
      await ecnApi.reject(id, { reason: rejectReason });
      message.success('已驳回');
      setRejectModalOpen(false);
      setRejectReason('');
      fetchECN();
      fetchHistory();
    } catch {
      message.error('操作失败');
    }
  };

  const handleTaskComplete = async (taskId: string) => {
    if (!id) return;
    try {
      await ecnApi.updateTask(id, taskId, { status: 'completed' });
      message.success('任务已完成');
      fetchTasks();
      fetchECN();
      fetchHistory();
    } catch {
      message.error('操作失败');
    }
  };

  const handleApplyBOM = async () => {
    if (!id) return;
    modal.confirm({
      title: '确认应用BOM变更？',
      content: '将根据变更内容自动更新相关BOM。',
      onOk: async () => {
        try {
          await ecnApi.applyBOMChanges(id);
          message.success('BOM变更已应用');
          fetchTasks();
          fetchECN();
          fetchHistory();
        } catch {
          message.error('操作失败');
        }
      },
    });
  };

  if (loading || !ecn) {
    return <Card loading={loading} />;
  }

  const statusInfo = statusMap[ecn.status] || { color: 'default', label: ecn.status };
  const urgencyInfo = urgencyMap[ecn.urgency] || { color: 'default', label: ecn.urgency };

  const renderActions = () => {
    const buttons: React.ReactNode[] = [];
    if (ecn.status === 'draft' || ecn.status === 'rejected') {
      buttons.push(
        <Button key="edit" icon={<EditOutlined />} onClick={() => navigate(`/ecn/${id}/edit`)}>编辑</Button>,
        <Button key="submit" type="primary" icon={<SendOutlined />} onClick={handleSubmit}>提交审批</Button>,
      );
    }
    if (ecn.status === 'pending') {
      buttons.push(
        <Button key="approve" type="primary" icon={<CheckOutlined />} onClick={() => setApproveModalOpen(true)}>批准</Button>,
        <Button key="reject" danger icon={<CloseOutlined />} onClick={() => setRejectModalOpen(true)}>驳回</Button>,
      );
    }
    if (ecn.status === 'executing') {
      buttons.push(
        <Button key="bom" type="primary" icon={<ThunderboltOutlined />} onClick={handleApplyBOM}>一键应用BOM变更</Button>,
      );
    }
    return buttons;
  };

  // Tab 1: 变更概要
  const renderOverview = () => (
    <div>
      <Descriptions bordered column={{ xs: 1, sm: 2 }} size="small">
        <Descriptions.Item label="ECN编号">{ecn.code}</Descriptions.Item>
        <Descriptions.Item label="状态">
          <Tag color={statusInfo.color}>{statusInfo.label}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="关联产品">{ecn.product?.name || '-'}</Descriptions.Item>
        <Descriptions.Item label="变更类型">{changeTypeMap[ecn.change_type] || ecn.change_type}</Descriptions.Item>
        <Descriptions.Item label="紧急程度">
          <Tag color={urgencyInfo.color}>{urgencyInfo.label}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="审批模式">{ecn.approval_mode === 'parallel' ? '并行会签' : '串行审批'}</Descriptions.Item>
        <Descriptions.Item label="申请人">{ecn.requester?.name || '-'}</Descriptions.Item>
        <Descriptions.Item label="创建时间">{ecn.created_at ? dayjs(ecn.created_at).format('YYYY-MM-DD HH:mm') : '-'}</Descriptions.Item>
        {ecn.planned_date && (
          <Descriptions.Item label="计划实施日期">{dayjs(ecn.planned_date).format('YYYY-MM-DD')}</Descriptions.Item>
        )}
        {ecn.completion_rate > 0 && (
          <Descriptions.Item label="执行进度">
            <Progress percent={ecn.completion_rate} size="small" style={{ width: 200 }} />
          </Descriptions.Item>
        )}
        <Descriptions.Item label="变更原因" span={2}>{ecn.reason || '-'}</Descriptions.Item>
        <Descriptions.Item label="变更描述" span={2}>{ecn.description || '-'}</Descriptions.Item>
        {ecn.technical_plan && (
          <Descriptions.Item label="技术方案" span={2}>{ecn.technical_plan}</Descriptions.Item>
        )}
        {ecn.impact_analysis && (
          <Descriptions.Item label="影响分析" span={2}>{ecn.impact_analysis}</Descriptions.Item>
        )}
        {ecn.rejection_reason && (
          <Descriptions.Item label="驳回原因" span={2}>
            <span style={{ color: '#ff4d4f' }}>{ecn.rejection_reason}</span>
          </Descriptions.Item>
        )}
      </Descriptions>
    </div>
  );

  // Tab 2: 受影响项 & 变更对比
  const renderAffectedItems = () => {
    const items = ecn.affected_items || [];
    if (items.length === 0) {
      return <Empty description="暂无受影响项" />;
    }
    return (
      <div>
        {items.map((item) => (
          <Card key={item.id} size="small" style={{ marginBottom: 12 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
              <Space>
                <Tag>{item.item_type === 'material' ? '物料' : item.item_type === 'bom_item' ? 'BOM项' : item.item_type}</Tag>
                <span style={{ fontWeight: 500 }}>{item.material_name || item.item_id}</span>
                {item.material_code && <span style={{ color: '#999' }}>({item.material_code})</span>}
              </Space>
            </div>
            {item.change_description && (
              <div style={{ color: '#666', marginBottom: 8 }}>{item.change_description}</div>
            )}
            {(item.before_value || item.after_value) && (
              <div style={{ background: '#fafafa', borderRadius: 4, padding: 12 }}>
                {Object.keys({ ...item.before_value, ...item.after_value }).filter(k => k !== 'ids').map(key => {
                  const before = item.before_value?.[key];
                  const after = item.after_value?.[key];
                  const changed = JSON.stringify(before) !== JSON.stringify(after);
                  return (
                    <div key={key} style={{ display: 'flex', gap: 16, marginBottom: 4 }}>
                      <span style={{ width: 100, color: '#999' }}>{key}</span>
                      <span style={{ color: changed ? '#ff4d4f' : '#333', textDecoration: changed ? 'line-through' : undefined }}>
                        {before !== undefined ? String(before) : '-'}
                      </span>
                      {changed && (
                        <>
                          <span style={{ color: '#999' }}>→</span>
                          <span style={{ color: '#52c41a', fontWeight: 500 }}>{after !== undefined ? String(after) : '-'}</span>
                        </>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
        ))}
      </div>
    );
  };

  // Tab 3: 审批流程
  const renderApprovalFlow = () => {
    const approvals = ecn.approvals || [];
    if (approvals.length === 0) {
      return <Empty description="暂无审批人" />;
    }
    const currentIdx = approvals.findIndex(a => a.status === 'pending');
    return (
      <Steps
        direction="vertical"
        current={currentIdx >= 0 ? currentIdx : approvals.length}
        items={approvals.map((a: ECNApproval) => ({
          title: (
            <Space>
              <span>{a.approver?.name || a.approver_id}</span>
              {a.status === 'approved' && <Tag color="success">已通过</Tag>}
              {a.status === 'rejected' && <Tag color="error">已驳回</Tag>}
              {a.status === 'pending' && <Tag color="processing">待审批</Tag>}
            </Space>
          ),
          description: (
            <div>
              {a.comment && <div style={{ color: '#666' }}>意见：{a.comment}</div>}
              {a.decided_at && <div style={{ color: '#999', fontSize: 12 }}>{dayjs(a.decided_at).format('YYYY-MM-DD HH:mm')}</div>}
            </div>
          ),
          status: a.status === 'approved' ? 'finish' as const : a.status === 'rejected' ? 'error' as const : a.status === 'pending' ? 'process' as const : 'wait' as const,
        }))}
      />
    );
  };

  // Tab 4: 执行任务
  const renderTasks = () => {
    if (tasks.length === 0) {
      return <Empty description={ecn.status === 'executing' ? '暂无执行任务' : '审批通过后将自动生成执行任务'} />;
    }
    const completed = tasks.filter(t => t.status === 'completed').length;
    const total = tasks.filter(t => t.status !== 'skipped').length;
    return (
      <div>
        <div style={{ marginBottom: 16 }}>
          <Progress percent={total > 0 ? Math.round(completed * 100 / total) : 0} />
        </div>
        {tasks.map((task) => (
          <Card key={task.id} size="small" style={{ marginBottom: 8 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Space>
                <Checkbox
                  checked={task.status === 'completed'}
                  disabled={task.status === 'completed' || task.status === 'skipped'}
                  onChange={() => handleTaskComplete(task.id)}
                />
                <Tag>{taskTypeMap[task.type] || task.type}</Tag>
                <span style={{ textDecoration: task.status === 'completed' ? 'line-through' : undefined, color: task.status === 'completed' ? '#999' : '#333' }}>
                  {task.title}
                </span>
              </Space>
              <Space>
                {task.assignee?.name && <span style={{ color: '#999', fontSize: 12 }}>{task.assignee.name}</span>}
                {task.due_date && <span style={{ color: '#999', fontSize: 12 }}>截止：{dayjs(task.due_date).format('MM-DD')}</span>}
                {task.status === 'completed' && <Tag color="success">已完成</Tag>}
                {task.status === 'pending' && <Tag>待处理</Tag>}
                {task.status === 'in_progress' && <Tag color="processing">进行中</Tag>}
              </Space>
            </div>
          </Card>
        ))}
      </div>
    );
  };

  // Tab 5: 变更历史
  const renderHistory = () => {
    if (history.length === 0) {
      return <Empty description="暂无操作历史" />;
    }
    return (
      <Timeline
        items={history.map((h) => ({
          color: h.action === 'rejected' ? 'red' : h.action === 'approved' ? 'green' : 'blue',
          children: (
            <div>
              <div>
                <span style={{ fontWeight: 500 }}>{historyActionMap[h.action] || h.action}</span>
                {h.user?.name && <span style={{ color: '#999', marginLeft: 8 }}>- {h.user.name}</span>}
              </div>
              {h.detail && Object.keys(h.detail).length > 0 && (
                <div style={{ color: '#666', fontSize: 12 }}>
                  {h.detail.comment && <span>意见：{h.detail.comment}</span>}
                  {h.detail.reason && <span>原因：{h.detail.reason}</span>}
                  {h.detail.task_title && <span>任务：{h.detail.task_title}</span>}
                </div>
              )}
              <div style={{ color: '#999', fontSize: 12 }}>{dayjs(h.created_at).format('YYYY-MM-DD HH:mm:ss')}</div>
            </div>
          ),
        }))}
      />
    );
  };

  const tabItems = [
    { key: 'overview', label: '变更概要', children: renderOverview() },
    { key: 'affected', label: `受影响项 (${ecn.affected_items?.length || 0})`, children: renderAffectedItems() },
    { key: 'approval', label: '审批流程', children: renderApprovalFlow() },
    { key: 'tasks', label: `执行任务 (${tasks.length})`, children: renderTasks() },
    { key: 'history', label: '变更历史', children: renderHistory() },
  ];

  return (
    <div>
      {/* 面包屑 */}
      <Breadcrumb items={[
        { title: <a onClick={() => navigate('/ecn')}>ECN管理</a> },
        { title: ecn.code },
      ]} style={{ marginBottom: 16 }} />

      {/* 顶部信息 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <Space align="center" style={{ marginBottom: 8 }}>
              <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/ecn')} />
              <h2 style={{ margin: 0 }}>{ecn.title}</h2>
              <Tag color={statusInfo.color}>{statusInfo.label}</Tag>
              {(ecn.urgency === 'high' || ecn.urgency === 'critical') && (
                <Tag color={urgencyInfo.color}>{urgencyInfo.label}</Tag>
              )}
            </Space>
            <div style={{ color: '#999', paddingLeft: 40 }}>
              {ecn.code} · {ecn.requester?.name} · {dayjs(ecn.created_at).format('YYYY-MM-DD HH:mm')}
            </div>
          </div>
          <Space>{renderActions()}</Space>
        </div>
      </Card>

      {/* Tab内容 */}
      <Card>
        <Tabs items={tabItems} />
      </Card>

      {/* 批准弹窗 */}
      <Modal
        title="批准ECN"
        open={approveModalOpen}
        onOk={handleApprove}
        onCancel={() => setApproveModalOpen(false)}
        okText="确认批准"
      >
        <Input.TextArea
          placeholder="审批意见（可选）"
          value={approvalComment}
          onChange={(e) => setApprovalComment(e.target.value)}
          rows={3}
        />
      </Modal>

      {/* 驳回弹窗 */}
      <Modal
        title="驳回ECN"
        open={rejectModalOpen}
        onOk={handleReject}
        onCancel={() => setRejectModalOpen(false)}
        okText="确认驳回"
        okButtonProps={{ danger: true, disabled: !rejectReason }}
      >
        <Input.TextArea
          placeholder="请输入驳回原因（必填）"
          value={rejectReason}
          onChange={(e) => setRejectReason(e.target.value)}
          rows={3}
        />
      </Modal>
    </div>
  );
};

export default ECNDetail;
