import React, { useState, useMemo, useRef, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useSSE, SSETaskEvent } from '@/hooks/useSSE';
import {
  Card,
  Tabs,
  Tag,
  Typography,
  Space,
  Button,
  Spin,
  Descriptions,
  Progress,
  Table,
  Modal,
  Form,
  Input,
  Select,
  Badge,
  message,
  Tooltip,
  Empty,
  Alert,
  Drawer,
  Timeline,
  Avatar,
  Popconfirm,
  InputNumber,
  Checkbox,
  Upload,
  Divider,
  AutoComplete,
} from 'antd';
import {
  ArrowLeftOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  RightOutlined,
  DownOutlined,
  PlusOutlined,
  EyeOutlined,
  UploadOutlined,
  DownloadOutlined,
  FileExcelOutlined,
  SwapOutlined,
  WarningOutlined,
  UserAddOutlined,
  UserOutlined,
  AuditOutlined,
  CloseCircleOutlined,
  HistoryOutlined,
  DeleteOutlined,
  SendOutlined,
  LockOutlined,
  SearchOutlined,
  ShoppingCartOutlined,
} from '@ant-design/icons';
import { projectApi, Project, Task } from '@/api/projects';
import { projectBomApi, ProjectBOMItem, CreateProjectBOMRequest, BOMItemRequest } from '@/api/projectBom';
import { materialsApi, Material } from '@/api/materials';
import { deliverablesApi } from '@/api/deliverables';
import { ecnApi, ECN } from '@/api/ecn';
import { documentsApi, Document } from '@/api/documents';
import { workflowApi, TaskActionLog } from '@/api/workflow';
import { approvalApi } from '@/api/approval';
import { taskFormApi, ParsedBOMItem } from '@/api/taskForms';
import { userApi } from '@/api/users';
import { srmApi } from '@/api/srm';
import { skuApi, ProductSKU, SKUCMFConfig, SKUBOMOverride } from '@/api/sku';
import { partDrawingApi, PartDrawing } from '@/api/partDrawing';
import UserSelect from '@/components/UserSelect';
import CMFPanel from '@/components/CMFPanel';
import { ROLE_CODES, taskRoleApi, TaskRole } from '@/constants/roles';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';

const { Title, Text, Paragraph } = Typography;

// ============ Constants ============

const PHASES = ['concept', 'evt', 'dvt', 'pvt', 'mp'];

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

const phaseLabels: Record<string, string> = {
  concept: '概念阶段',
  evt: 'EVT 工程验证',
  dvt: 'DVT 设计验证',
  pvt: 'PVT 生产验证',
  mp: 'MP 量产',
};

const statusColors: Record<string, string> = {
  planning: 'default',
  active: 'processing',
  on_hold: 'warning',
  completed: 'success',
  cancelled: 'error',
};

const taskStatusConfig: Record<string, { color: string; text: string; icon: React.ReactNode; barColor: string }> = {
  unassigned: { color: 'default', text: '待指派', icon: <UserAddOutlined />, barColor: '#d9d9d9' },
  pending: { color: 'default', text: '待开始', icon: <ClockCircleOutlined />, barColor: '#bfbfbf' },
  in_progress: { color: 'processing', text: '进行中', icon: <ClockCircleOutlined />, barColor: '#1677ff' },
  submitted: { color: 'warning', text: '已提交', icon: <CheckCircleOutlined />, barColor: '#faad14' },
  reviewing: { color: 'warning', text: '审批中', icon: <AuditOutlined />, barColor: '#faad14' },
  completed: { color: 'success', text: '已完成', icon: <CheckCircleOutlined />, barColor: '#52c41a' },
  rejected: { color: 'error', text: '已驳回', icon: <CloseCircleOutlined />, barColor: '#ff4d4f' },
};

const GANTT_ROW_HEIGHT = 36;
const GANTT_HEADER_HEIGHT = 50;
const DAY_WIDTH = 28;
const LEFT_PANEL_WIDTH = 650;

// ============ Phase Progress Bar ============

const PhaseProgressBar: React.FC<{ currentPhase: string }> = ({ currentPhase }) => {
  const currentIndex = PHASES.indexOf(currentPhase?.toLowerCase());

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
      {PHASES.map((phase, index) => {
        let icon = '⬜';
        let fontWeight: number = 400;
        if (index < currentIndex) {
          icon = '✅';
        } else if (index === currentIndex) {
          icon = '🔵';
          fontWeight = 600;
        }

        return (
          <React.Fragment key={phase}>
            {index > 0 && (
              <span style={{ color: index <= currentIndex ? '#1890ff' : '#d9d9d9', fontSize: 12 }}>──▶</span>
            )}
            <span style={{
              fontWeight,
              fontSize: 13,
              color: index <= currentIndex ? '#333' : '#999',
            }}>
              {icon} {phase.toUpperCase()}
            </span>
          </React.Fragment>
        );
      })}
    </div>
  );
};

// ============ Gantt Helper Types ============

interface TreeTask extends Task {
  children: TreeTask[];
  depth: number;
  expanded?: boolean;
}

// ============ Gantt Chart Component ============

const GanttChart: React.FC<{
  tasks: Task[];
  projectId: string;
  onCompleteTask: (taskId: string) => void;
  completingTask: boolean;
  onRefresh: () => void;
}> = ({ tasks, projectId, onCompleteTask: _onCompleteTask, completingTask: _completingTask, onRefresh }) => {
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set());
  const [collapsedTasks, setCollapsedTasks] = useState<Set<string>>(new Set());
  const [groupBy, setGroupBy] = useState<'phase' | 'none'>('phase');
  const timelineRef = useRef<HTMLDivElement>(null);
  const leftPanelRef = useRef<HTMLDivElement>(null);

  const handleTimelineScroll = (e: React.UIEvent<HTMLDivElement>) => {
    if (leftPanelRef.current) {
      leftPanelRef.current.scrollTop = e.currentTarget.scrollTop;
    }
  };
  const handleLeftScroll = (e: React.UIEvent<HTMLDivElement>) => {
    if (timelineRef.current) {
      timelineRef.current.scrollTop = e.currentTarget.scrollTop;
    }
  };

  const buildTree = (tasks: Task[]): TreeTask[] => {
    const map = new Map<string, TreeTask>();
    const roots: TreeTask[] = [];
    tasks.forEach(t => map.set(t.id, { ...t, children: [], depth: 0 }));
    map.forEach(node => {
      if (node.parent_task_id && map.has(node.parent_task_id)) {
        const parent = map.get(node.parent_task_id)!;
        node.depth = parent.depth + 1;
        parent.children.push(node);
      } else {
        roots.push(node);
      }
    });
    return roots;
  };

  const flattenTree = (nodes: TreeTask[]): TreeTask[] => {
    const result: TreeTask[] = [];
    const walk = (items: TreeTask[], depth: number) => {
      items.forEach(item => {
        item.depth = depth;
        result.push(item);
        if (item.children.length > 0 && !collapsedTasks.has(item.id)) {
          walk(item.children, depth + 1);
        }
      });
    };
    walk(nodes, 0);
    return result;
  };

  const groupedData = useMemo(() => {
    if (groupBy === 'none') {
      const tree = buildTree(tasks);
      return [{ phase: '', label: '全部任务', tasks: flattenTree(tree) }];
    }
    const phaseOrder = ['concept', 'evt', 'dvt', 'pvt', 'mp', ''];
    const groups = new Map<string, Task[]>();
    tasks.forEach(t => {
      const phase = (typeof t.phase === 'object' && t.phase !== null ? (t.phase as any).phase : (t.phase || '')).toLowerCase();
      if (!groups.has(phase)) groups.set(phase, []);
      groups.get(phase)!.push(t);
    });
    return phaseOrder
      .filter(p => groups.has(p))
      .map(phase => {
        const tree = buildTree(groups.get(phase)!);
        return {
          phase,
          label: phaseLabels[phase] || (phase ? phase.toUpperCase() : '未分类'),
          tasks: flattenTree(tree),
        };
      });
  }, [tasks, groupBy, collapsedTasks]);

  const { startDate, endDate, totalDays } = useMemo(() => {
    let min = dayjs().subtract(7, 'day');
    let max = dayjs().add(30, 'day');
    tasks.forEach(t => {
      if (t.start_date) { const d = dayjs(t.start_date); if (d.isBefore(min)) min = d; }
      if (t.due_date) { const d = dayjs(t.due_date); if (d.isAfter(max)) max = d; }
    });
    min = min.subtract(7, 'day').startOf('week');
    max = max.add(14, 'day').endOf('week');
    return { startDate: min, endDate: max, totalDays: max.diff(min, 'day') + 1 };
  }, [tasks]);

  const monthHeaders = useMemo(() => {
    const months: { label: string; days: number; offset: number }[] = [];
    let cursor = startDate;
    while (cursor.isBefore(endDate)) {
      const monthEnd = cursor.endOf('month');
      const end = monthEnd.isAfter(endDate) ? endDate : monthEnd;
      const days = end.diff(cursor, 'day') + 1;
      months.push({ label: cursor.format('YYYY年M月'), days, offset: cursor.diff(startDate, 'day') });
      cursor = monthEnd.add(1, 'day');
    }
    return months;
  }, [startDate, endDate]);

  const dayHeaders = useMemo(() => {
    const days: { label: string; date: dayjs.Dayjs; isWeekend: boolean; isToday: boolean }[] = [];
    for (let i = 0; i < totalDays; i++) {
      const d = startDate.add(i, 'day');
      days.push({ label: d.format('D'), date: d, isWeekend: d.day() === 0 || d.day() === 6, isToday: d.isSame(dayjs(), 'day') });
    }
    return days;
  }, [startDate, totalDays]);

  useEffect(() => {
    if (timelineRef.current) {
      const todayOffset = dayjs().diff(startDate, 'day');
      const scrollTo = Math.max(0, todayOffset * DAY_WIDTH - 200);
      timelineRef.current.scrollLeft = scrollTo;
    }
  }, [startDate]);

  const getBar = (task: Task) => {
    const start = task.start_date ? dayjs(task.start_date) : null;
    const end = task.due_date ? dayjs(task.due_date) : null;
    if (!start && !end) return null;
    const barStart = start || end!;
    const barEnd = end || start!;
    const left = barStart.diff(startDate, 'day') * DAY_WIDTH;
    const width = Math.max((barEnd.diff(barStart, 'day') + 1) * DAY_WIDTH, DAY_WIDTH);
    return { left, width };
  };

  const toggleGroup = (phase: string) => {
    setCollapsedGroups(prev => { const next = new Set(prev); if (next.has(phase)) next.delete(phase); else next.add(phase); return next; });
  };
  const toggleTask = (taskId: string) => {
    setCollapsedTasks(prev => { const next = new Set(prev); if (next.has(taskId)) next.delete(taskId); else next.add(taskId); return next; });
  };

  const rows: Array<{ type: 'group'; phase: string; label: string; count: number } | { type: 'task'; task: TreeTask }> = [];
  groupedData.forEach(group => {
    if (groupBy === 'phase') rows.push({ type: 'group', phase: group.phase, label: group.label, count: group.tasks.length });
    if (!collapsedGroups.has(group.phase) || groupBy === 'none') {
      group.tasks.forEach(t => rows.push({ type: 'task', task: t }));
    }
  });
  const totalHeight = rows.length * GANTT_ROW_HEIGHT;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid #f0f0f0' }}>
        <Space>
          <Text strong>甘特图视图</Text>
          <Tag>{tasks.length} 个任务</Tag>
        </Space>
        <Space>
          <Text type="secondary">分组:</Text>
          <Select size="small" value={groupBy} onChange={setGroupBy} style={{ width: 120 }}
            options={[{ label: '按阶段', value: 'phase' }, { label: '不分组', value: 'none' }]} />
        </Space>
      </div>

      <div style={{ display: 'flex', flex: 1, overflow: 'hidden', border: '1px solid #e8e8e8', borderRadius: 4 }}>
        {/* Left panel */}
        <div style={{ width: LEFT_PANEL_WIDTH, flexShrink: 0, borderRight: '2px solid #d9d9d9', display: 'flex', flexDirection: 'column' }}>
          <div style={{ height: GANTT_HEADER_HEIGHT, borderBottom: '1px solid #e8e8e8', display: 'flex', alignItems: 'center', padding: '0 12px', background: '#fafafa', fontWeight: 600, fontSize: 13, flexShrink: 0 }}>
            <span style={{ flex: 1 }}>任务名称</span>
            <span style={{ width: 100, textAlign: 'center' }}>负责人</span>
            <span style={{ width: 50, textAlign: 'center' }}>状态</span>
            <span style={{ width: 45, textAlign: 'center' }}>进度</span>
            <span style={{ width: 130, textAlign: 'center' }}>操作</span>
          </div>
          <div ref={leftPanelRef} onScroll={handleLeftScroll} style={{ flex: 1, overflowY: 'auto', overflowX: 'hidden' }}>
            <div style={{ minHeight: totalHeight }}>
              {rows.map((row, idx) => {
                if (row.type === 'group') {
                  const collapsed = collapsedGroups.has(row.phase);
                  return (
                    <div key={`group-${row.phase}`} style={{ height: GANTT_ROW_HEIGHT, display: 'flex', alignItems: 'center', padding: '0 12px', background: '#f5f5f5', cursor: 'pointer', borderBottom: '1px solid #f0f0f0', fontWeight: 600, fontSize: 13 }} onClick={() => toggleGroup(row.phase)}>
                      {collapsed ? <RightOutlined style={{ fontSize: 10, marginRight: 8 }} /> : <DownOutlined style={{ fontSize: 10, marginRight: 8 }} />}
                      <Tag color={phaseColors[row.phase] || 'default'} style={{ marginRight: 8 }}>{row.label}</Tag>
                      <Text type="secondary" style={{ fontSize: 12 }}>({row.count})</Text>
                    </div>
                  );
                }
                const task = row.task;
                const config = taskStatusConfig[task.status] || taskStatusConfig.pending;
                const hasChildren = task.children.length > 0;
                const isCollapsed = collapsedTasks.has(task.id);
                const isMilestone = task.task_type === 'MILESTONE';
                return (
                  <div key={task.id} style={{ height: GANTT_ROW_HEIGHT, display: 'flex', alignItems: 'center', padding: '0 12px', borderBottom: '1px solid #f7f7f7', fontSize: 12, background: idx % 2 === 0 ? '#fff' : '#fafcff' }}>
                    <div style={{ flex: 1, display: 'flex', alignItems: 'center', minWidth: 0, paddingLeft: task.depth * 20 }}>
                      {hasChildren ? (
                        <span style={{ cursor: 'pointer', marginRight: 4, width: 16, textAlign: 'center', flexShrink: 0 }} onClick={() => toggleTask(task.id)}>
                          {isCollapsed ? <RightOutlined style={{ fontSize: 9 }} /> : <DownOutlined style={{ fontSize: 9 }} />}
                        </span>
                      ) : <span style={{ width: 16, marginRight: 4, flexShrink: 0 }} />}
                      {isMilestone && <span style={{ display: 'inline-block', width: 10, height: 10, background: config.barColor, transform: 'rotate(45deg)', marginRight: 6, flexShrink: 0 }} />}
                      <Tooltip title={task.title}>
                        <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontWeight: isMilestone ? 600 : (task.task_type === 'SUBTASK' ? 400 : 500), color: task.is_critical ? '#cf1322' : undefined }}>
                          {(task.code || task.task_code) ? <Text code style={{ fontSize: 11, marginRight: 4 }}>{task.code || task.task_code}</Text> : null}
                          {task.title}
                        </span>
                      </Tooltip>
                    </div>
                    <span style={{ width: 100, flexShrink: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: '#666' }}>
                      {(task.assignee?.name || task.assignee_name) ? (
                        <Tooltip title={task.assignee?.name || task.assignee_name}>
                          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                            <Avatar size={18} src={task.assignee?.avatar_url} icon={<UserOutlined />} style={{ fontSize: 10, flexShrink: 0 }}>
                              {(task.assignee?.name || task.assignee_name)?.[0]}
                            </Avatar>
                            <span style={{ fontSize: 11, overflow: 'hidden', textOverflow: 'ellipsis' }}>{task.assignee?.name || task.assignee_name}</span>
                          </span>
                        </Tooltip>
                      ) : '-'}
                    </span>
                    <span style={{ width: 50, textAlign: 'center', flexShrink: 0 }}>
                      <Tag color={config.color} style={{ fontSize: 10, padding: '0 4px', margin: 0, lineHeight: '18px' }}>{config.text}</Tag>
                    </span>
                    <span style={{ width: 45, textAlign: 'center', flexShrink: 0, fontSize: 11, color: '#666' }}>{task.progress}%</span>
                    <span style={{ width: 130, textAlign: 'center', flexShrink: 0 }} onClick={(e) => e.stopPropagation()}>
                      <TaskActions task={task} projectId={projectId} onRefresh={onRefresh} />
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* Right timeline */}
        <div ref={timelineRef} onScroll={handleTimelineScroll} style={{ flex: 1, overflow: 'auto' }}>
          <div style={{ minWidth: totalDays * DAY_WIDTH, position: 'relative' }}>
            <div style={{ position: 'sticky', top: 0, zIndex: 10, background: '#fafafa' }}>
              <div style={{ display: 'flex', height: 24, borderBottom: '1px solid #e8e8e8' }}>
                {monthHeaders.map((m, i) => (
                  <div key={i} style={{ width: m.days * DAY_WIDTH, textAlign: 'center', fontSize: 11, fontWeight: 600, lineHeight: '24px', borderRight: '1px solid #e8e8e8', color: '#333' }}>{m.label}</div>
                ))}
              </div>
              <div style={{ display: 'flex', height: GANTT_HEADER_HEIGHT - 24, borderBottom: '1px solid #e8e8e8' }}>
                {dayHeaders.map((d, i) => (
                  <div key={i} style={{ width: DAY_WIDTH, textAlign: 'center', fontSize: 10, lineHeight: `${GANTT_HEADER_HEIGHT - 24}px`, color: d.isToday ? '#fff' : d.isWeekend ? '#bbb' : '#666', background: d.isToday ? '#1677ff' : d.isWeekend ? '#f9f9f9' : 'transparent', borderRight: '1px solid #f0f0f0', fontWeight: d.isToday ? 700 : 400 }}>{d.label}</div>
                ))}
              </div>
            </div>
            <div style={{ position: 'relative' }}>
              {dayHeaders.map((d, i) => d.isWeekend && (
                <div key={`bg-${i}`} style={{ position: 'absolute', left: i * DAY_WIDTH, top: 0, width: DAY_WIDTH, height: totalHeight, background: 'rgba(0,0,0,0.02)', zIndex: 0 }} />
              ))}
              {(() => {
                const todayOffset = dayjs().diff(startDate, 'day');
                if (todayOffset >= 0 && todayOffset <= totalDays) {
                  return <div style={{ position: 'absolute', left: todayOffset * DAY_WIDTH + DAY_WIDTH / 2, top: 0, width: 2, height: totalHeight, background: '#ff4d4f', zIndex: 5, opacity: 0.6 }} />;
                }
                return null;
              })()}
              {rows.map((row, idx) => {
                if (row.type === 'group') {
                  return <div key={`gbar-${row.phase}`} style={{ height: GANTT_ROW_HEIGHT, background: '#f5f5f5', borderBottom: '1px solid #f0f0f0' }} />;
                }
                const task = row.task;
                const bar = getBar(task);
                const config = taskStatusConfig[task.status] || taskStatusConfig.pending;
                const isMilestone = task.task_type === 'MILESTONE';
                return (
                  <div key={task.id} style={{ height: GANTT_ROW_HEIGHT, position: 'relative', borderBottom: '1px solid #f7f7f7', background: idx % 2 === 0 ? '#fff' : '#fafcff' }}>
                    {bar && !isMilestone && (
                      <Tooltip title={<div><div><strong>{task.title}</strong></div><div>{task.start_date || '?'} → {task.due_date || '?'}</div><div>进度: {task.progress}%</div>{(task.assignee?.name || task.assignee_name) && <div>负责人: {task.assignee?.name || task.assignee_name}</div>}</div>}>
                        <div style={{ position: 'absolute', left: bar.left, top: (GANTT_ROW_HEIGHT - 18) / 2, width: bar.width, height: 18, borderRadius: 3, background: config.barColor, opacity: 0.85, zIndex: 2, cursor: 'pointer', overflow: 'hidden', transition: 'opacity 0.2s' }}
                          onMouseEnter={e => (e.currentTarget.style.opacity = '1')} onMouseLeave={e => (e.currentTarget.style.opacity = '0.85')}>
                          {task.progress > 0 && task.progress < 100 && (
                            <div style={{ position: 'absolute', left: 0, top: 0, width: `${task.progress}%`, height: '100%', background: 'rgba(255,255,255,0.3)', borderRadius: '3px 0 0 3px' }} />
                          )}
                          {bar.width > 80 && (
                            <span style={{ position: 'absolute', left: 6, top: 0, lineHeight: '18px', fontSize: 10, color: '#fff', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: bar.width - 12 }}>{task.title}</span>
                          )}
                        </div>
                      </Tooltip>
                    )}
                    {bar && isMilestone && (
                      <Tooltip title={<div><div><strong>🔷 里程碑: {task.title}</strong></div><div>{task.due_date || task.start_date || '-'}</div>{(task.assignee?.name || task.assignee_name) && <div>负责人: {task.assignee?.name || task.assignee_name}</div>}</div>}>
                        <div style={{ position: 'absolute', left: bar.left + (bar.width / 2) - 8, top: (GANTT_ROW_HEIGHT - 16) / 2, width: 16, height: 16, background: config.barColor, transform: 'rotate(45deg)', zIndex: 2, cursor: 'pointer', border: '2px solid rgba(255,255,255,0.8)', boxShadow: '0 1px 3px rgba(0,0,0,0.2)' }} />
                      </Tooltip>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 16, padding: '8px 0', flexWrap: 'wrap', alignItems: 'center', borderTop: '1px solid #f0f0f0' }}>
        <Text type="secondary" style={{ fontSize: 12 }}>图例:</Text>
        {Object.entries(taskStatusConfig).map(([key, val]) => (
          <Space key={key} size={4}>
            <span style={{ display: 'inline-block', width: 14, height: 10, background: val.barColor, borderRadius: 2 }} />
            <Text style={{ fontSize: 11 }}>{val.text}</Text>
          </Space>
        ))}
        <Space size={4}>
          <span style={{ display: 'inline-block', width: 10, height: 10, background: '#1677ff', transform: 'rotate(45deg)' }} />
          <Text style={{ fontSize: 11 }}>里程碑</Text>
        </Space>
        <Space size={4}>
          <span style={{ display: 'inline-block', width: 2, height: 12, background: '#ff4d4f' }} />
          <Text style={{ fontSize: 11 }}>今天</Text>
        </Space>
      </div>
    </div>
  );
};

// ============ Overview Tab ============

const OverviewTab: React.FC<{ project: Project }> = ({ project }) => {
  return (
    <div>
      <Descriptions column={2} bordered size="small">
        <Descriptions.Item label="项目编码"><Text code>{project.code}</Text></Descriptions.Item>
        <Descriptions.Item label="项目名称"><Text strong>{project.name}</Text></Descriptions.Item>
        <Descriptions.Item label="当前阶段"><Tag color={phaseColors[project.phase]}>{project.phase?.toUpperCase()}</Tag></Descriptions.Item>
        <Descriptions.Item label="状态">
          <Badge status={statusColors[project.status] as any} text={
            project.status === 'planning' ? '规划中' :
            project.status === 'active' ? '进行中' :
            project.status === 'completed' ? '已完成' :
            project.status === 'on_hold' ? '暂停' : project.status
          } />
        </Descriptions.Item>
        <Descriptions.Item label="进度"><Progress percent={project.progress} size="small" style={{ width: 200 }} /></Descriptions.Item>
        <Descriptions.Item label="项目经理">{project.manager_name || '-'}</Descriptions.Item>
        <Descriptions.Item label="开始日期">{project.start_date ? dayjs(project.start_date).format('YYYY-MM-DD') : '-'}</Descriptions.Item>
        <Descriptions.Item label="计划结束">{project.planned_end ? dayjs(project.planned_end).format('YYYY-MM-DD') : '-'}</Descriptions.Item>
        <Descriptions.Item label="关联产品" span={2}>{project.product_name || '-'}</Descriptions.Item>
        <Descriptions.Item label="项目描述" span={2}>
          <Paragraph style={{ margin: 0 }}>{project.description || '暂无描述'}</Paragraph>
        </Descriptions.Item>
      </Descriptions>
    </div>
  );
};

// ============ BOM Tab - Full Editor ============

const BOM_STATUS_CONFIG: Record<string, { color: string; text: string }> = {
  draft: { color: 'default', text: '草稿' },
  pending_review: { color: 'processing', text: '待审批' },
  published: { color: 'success', text: '已发布' },
  rejected: { color: 'error', text: '已驳回' },
  frozen: { color: 'purple', text: '已冻结' },
};

const BOM_TYPE_CONFIG: Record<string, { label: string }> = {
  EBOM: { label: '电子BOM' },
  SBOM: { label: '结构BOM' },
  MBOM: { label: '制造BOM' },
};

const CATEGORY_OPTIONS = [
  '电子元器件', '结构件', '光学器件', '电池', '线缆/FPC', '包装材料', '标签/外观件', '其他',
];

const PROCUREMENT_OPTIONS = [
  { label: 'Buy（外购）', value: 'buy' },
  { label: 'Make（自制）', value: 'make' },
  { label: 'Phantom（虚拟件）', value: 'phantom' },
];

// Material Search Modal
const MaterialSearchModal: React.FC<{
  open: boolean;
  onClose: () => void;
  onSelect: (material: Material) => void;
}> = ({ open, onClose, onSelect }) => {
  const [search, setSearch] = useState('');
  const { data, isLoading } = useQuery({
    queryKey: ['materials-search', search],
    queryFn: () => materialsApi.list({ search: search || undefined }),
    enabled: open,
  });

  const materials = data?.materials || [];

  const columns: ColumnsType<Material> = [
    { title: '编码', dataIndex: 'code', width: 120, render: (v: string) => <Text code>{v}</Text> },
    { title: '名称', dataIndex: 'name', width: 160 },
    { title: '规格', dataIndex: 'description', width: 200, ellipsis: true },
    { title: '单位', dataIndex: 'unit', width: 60 },
    { title: '标准成本', dataIndex: 'standard_cost', width: 100, render: (v: number) => v != null ? `¥${v.toFixed(2)}` : '-' },
    {
      title: '操作', width: 80, render: (_, record) => (
        <Button size="small" type="link" onClick={() => { onSelect(record); onClose(); }}>选择</Button>
      ),
    },
  ];

  return (
    <Modal title="物料选择" open={open} onCancel={onClose} width={800} footer={null}>
      <Input.Search
        placeholder="按名称/编码/规格搜索"
        allowClear
        onSearch={setSearch}
        onChange={e => { if (!e.target.value) setSearch(''); }}
        style={{ marginBottom: 12 }}
      />
      <Table
        columns={columns}
        dataSource={materials}
        rowKey="id"
        size="small"
        loading={isLoading}
        pagination={{ pageSize: 8, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ y: 350 }}
        locale={{ emptyText: '暂无物料数据' }}
      />
    </Modal>
  );
};

const BOMTab: React.FC<{ projectId: string }> = ({ projectId }) => {
  const queryClient = useQueryClient();
  const [selectedBomId, setSelectedBomId] = useState<string | null>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [rejectModalOpen, setRejectModalOpen] = useState(false);
  const [rejectComment, setRejectComment] = useState('');
  const [materialModalOpen, setMaterialModalOpen] = useState(false);
  const [editingCell, setEditingCell] = useState<{ rowId: string; field: string } | null>(null);
  const [editingRowId, setEditingRowId] = useState<string | null>(null);
  const [compareModalOpen, setCompareModalOpen] = useState(false);
  const [compareBom1, setCompareBom1] = useState<string | undefined>(undefined);
  const [compareBom2, setCompareBom2] = useState<string | undefined>(undefined);
  const [compareResult, setCompareResult] = useState<any[] | null>(null);
  const [compareLoading, setCompareLoading] = useState(false);
  const [exportLoading, setExportLoading] = useState(false);
  const [importLoading, setImportLoading] = useState(false);
  const [drawingHistoryOpen, setDrawingHistoryOpen] = useState(false);
  const [drawingHistoryItemId, setDrawingHistoryItemId] = useState<string>('');
  const [drawingHistoryType, setDrawingHistoryType] = useState<'2D' | '3D'>('2D');
  const [drawingUploadModalOpen, setDrawingUploadModalOpen] = useState(false);
  const [drawingUploadItemId, setDrawingUploadItemId] = useState<string>('');
  const [drawingUploadType, setDrawingUploadType] = useState<'2D' | '3D'>('2D');
  const [drawingChangeDesc, setDrawingChangeDesc] = useState('');
  const [form] = Form.useForm();

  // Fetch BOM list
  const { data: bomList = [], isLoading: listLoading } = useQuery({
    queryKey: ['project-boms', projectId],
    queryFn: () => projectBomApi.list(projectId),
    retry: false,
  });

  // Fetch selected BOM detail
  const { data: bomDetail, isLoading: detailLoading } = useQuery({
    queryKey: ['project-bom-detail', projectId, selectedBomId],
    queryFn: () => projectBomApi.get(projectId, selectedBomId!),
    enabled: !!selectedBomId,
    retry: false,
  });

  // Fetch drawings for all items in selected BOM
  const { data: drawingsByBOM = {} } = useQuery({
    queryKey: ['bom-drawings', projectId, selectedBomId],
    queryFn: () => partDrawingApi.listByBOM(projectId, selectedBomId!),
    enabled: !!selectedBomId,
    retry: false,
  });

  // Auto-select first BOM
  useEffect(() => {
    if (bomList.length > 0 && !selectedBomId) {
      setSelectedBomId(bomList[0].id);
    }
  }, [bomList, selectedBomId]);

  const isEditable = bomDetail?.status === 'draft' || bomDetail?.status === 'rejected';

  // Mutations
  const createMutation = useMutation({
    mutationFn: (data: CreateProjectBOMRequest) => projectBomApi.create(projectId, data),
    onSuccess: (bom) => {
      message.success('BOM创建成功');
      setCreateModalOpen(false);
      form.resetFields();
      setSelectedBomId(bom.id);
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('创建失败'),
  });

  const submitMutation = useMutation({
    mutationFn: () => projectBomApi.submit(projectId, selectedBomId!),
    onSuccess: () => {
      message.success('已提交审批');
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('提交失败'),
  });

  const approveMutation = useMutation({
    mutationFn: () => projectBomApi.approve(projectId, selectedBomId!),
    onSuccess: () => {
      message.success('审批通过');
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('操作失败'),
  });

  const rejectMutation = useMutation({
    mutationFn: (comment: string) => projectBomApi.reject(projectId, selectedBomId!, comment),
    onSuccess: () => {
      message.success('已驳回');
      setRejectModalOpen(false);
      setRejectComment('');
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('操作失败'),
  });

  const freezeMutation = useMutation({
    mutationFn: () => projectBomApi.freeze(projectId, selectedBomId!),
    onSuccess: () => {
      message.success('已冻结');
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('操作失败'),
  });

  const addItemMutation = useMutation({
    mutationFn: (data: BOMItemRequest) => projectBomApi.addItem(projectId, selectedBomId!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('添加失败'),
  });

  const updateItemMutation = useMutation({
    mutationFn: ({ itemId, data }: { itemId: string; data: BOMItemRequest }) =>
      projectBomApi.updateItem(projectId, selectedBomId!, itemId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('更新失败'),
  });

  const deleteItemMutation = useMutation({
    mutationFn: (itemId: string) => projectBomApi.deleteItem(projectId, selectedBomId!, itemId),
    onSuccess: () => {
      message.success('已删除');
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('删除失败'),
  });

  const convertToMBOMMutation = useMutation({
    mutationFn: () => projectBomApi.convertToMBOM(projectId, selectedBomId!),
    onSuccess: () => {
      message.success('已创建MBOM副本');
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('转换失败'),
  });

  const submitToSRMMutation = useMutation({
    mutationFn: () => srmApi.createPRFromBOM({ project_id: projectId, bom_id: selectedBomId! }),
    onSuccess: (pr) => {
      message.success(`已创建采购需求 ${pr.pr_code}`);
    },
    onError: () => message.error('提交到SRM失败'),
  });

  // Export Excel handler
  const handleExportExcel = async () => {
    if (!selectedBomId) return;
    setExportLoading(true);
    try {
      await projectBomApi.exportExcel(projectId, selectedBomId);
      message.success('导出成功');
    } catch {
      message.error('导出失败');
    } finally {
      setExportLoading(false);
    }
  };

  // Import Excel handler
  const handleImportExcel = async (file: File) => {
    if (!selectedBomId) return;
    setImportLoading(true);
    try {
      const result = await projectBomApi.importExcel(projectId, selectedBomId, file);
      message.success(`导入成功：创建${result?.created ?? 0}项，匹配物料${result?.matched ?? 0}项，自动建料${result?.auto_created ?? 0}项，错误${result?.errors ?? 0}项`);
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    } catch {
      message.error('导入失败');
    } finally {
      setImportLoading(false);
    }
  };

  // Download template handler
  const handleDownloadTemplate = async () => {
    try {
      await projectBomApi.downloadTemplate(bomDetail?.bom_type);
    } catch {
      message.error('下载模板失败');
    }
  };

  // Compare BOMs handler
  const handleCompare = async () => {
    if (!compareBom1 || !compareBom2) {
      message.warning('请选择两个BOM进行对比');
      return;
    }
    setCompareLoading(true);
    try {
      const result = await projectBomApi.compareBOMs(compareBom1, compareBom2);
      setCompareResult(result);
    } catch {
      message.error('对比失败');
    } finally {
      setCompareLoading(false);
    }
  };

  // Add new empty row
  const handleAddRow = () => {
    const items = bomDetail?.items || [];
    addItemMutation.mutate({
      name: '新物料',
      quantity: 1,
      unit: 'pcs',
      item_number: items.length + 1,
      procurement_type: 'buy',
    });
  };

  // Select material from library
  const handleMaterialSelect = (material: Material) => {
    if (editingRowId) {
      updateItemMutation.mutate({
        itemId: editingRowId,
        data: {
          material_id: material.id,
          name: material.name,
          specification: material.description,
          unit: material.unit || 'pcs',
          unit_price: material.standard_cost || undefined,
          quantity: 1,
        },
      });
    } else {
      addItemMutation.mutate({
        material_id: material.id,
        name: material.name,
        specification: material.description,
        unit: material.unit || 'pcs',
        unit_price: material.standard_cost || undefined,
        quantity: 1,
        item_number: (bomDetail?.items?.length || 0) + 1,
      });
    }
    setEditingRowId(null);
  };

  // Inline cell save
  const handleCellSave = (record: ProjectBOMItem, field: string, value: any) => {
    setEditingCell(null);
    const data: any = {
      name: record.name,
      quantity: record.quantity,
      unit: record.unit,
    };
    data[field] = value;
    // Auto-compute extended_cost for quantity/unit_price changes
    if (field === 'quantity' || field === 'unit_price') {
      const qty = field === 'quantity' ? value : record.quantity;
      const price = field === 'unit_price' ? value : record.unit_price;
      if (qty != null && price != null) {
        data.extended_cost = qty * price;
      }
    }
    updateItemMutation.mutate({ itemId: record.id, data });
  };

  // Editable cell renderer
  const renderEditableCell = (value: any, record: ProjectBOMItem, field: string, type: 'text' | 'number' | 'select' = 'text', options?: { label: string; value: string }[]) => {
    if (!isEditable) return value ?? '-';
    const isEditing = editingCell?.rowId === record.id && editingCell?.field === field;

    if (isEditing) {
      if (type === 'number') {
        return (
          <InputNumber
            size="small"
            autoFocus
            defaultValue={value}
            style={{ width: '100%' }}
            onBlur={(e) => handleCellSave(record, field, parseFloat((e.target as HTMLInputElement).value) || 0)}
            onPressEnter={(e) => handleCellSave(record, field, parseFloat((e.target as HTMLInputElement).value) || 0)}
          />
        );
      }
      if (type === 'select' && options) {
        return (
          <Select
            size="small"
            autoFocus
            defaultValue={value}
            defaultOpen
            style={{ width: '100%' }}
            options={options}
            onChange={(v) => handleCellSave(record, field, v)}
            onBlur={() => setEditingCell(null)}
          />
        );
      }
      return (
        <Input
          size="small"
          autoFocus
          defaultValue={value}
          onBlur={(e) => handleCellSave(record, field, e.target.value)}
          onPressEnter={(e) => handleCellSave(record, field, (e.target as HTMLInputElement).value)}
        />
      );
    }

    return (
      <div
        style={{ cursor: 'pointer', minHeight: 22, padding: '0 2px', borderRadius: 2 }}
        className="editable-cell"
        onClick={() => setEditingCell({ rowId: record.id, field })}
      >
        {value ?? <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
      </div>
    );
  };

  // SBOM下拉选项常量
  const PROCESS_TYPE_OPTIONS = [
    { label: '注塑', value: '注塑' }, { label: 'CNC', value: 'CNC' },
    { label: '冲压', value: '冲压' }, { label: '模切', value: '模切' },
    { label: '3D打印', value: '3D打印' }, { label: '激光切割', value: '激光切割' },
    { label: 'SMT', value: 'SMT' }, { label: '手工', value: '手工' },
  ];
  const ASSEMBLY_METHOD_OPTIONS = [
    { label: '卡扣', value: '卡扣' }, { label: '螺丝', value: '螺丝' },
    { label: '胶合', value: '胶合' }, { label: '超声波焊接', value: '超声波焊接' },
    { label: '热熔', value: '热熔' }, { label: '激光焊接', value: '激光焊接' },
  ];
  const MATERIAL_TYPE_PRESETS = [
    '钛合金', '铝合金6061', '铝合金7075', '不锈钢304', '不锈钢316L',
    'PC', 'ABS', 'ABS+PC', 'PA66', 'PA66+GF30', 'PMMA', 'POM', 'TPU',
    '硅胶', 'PEEK', '碳纤维', '玻璃', '蓝宝石', '镁合金', '锌合金', '铜合金', 'TR90', 'Ultem',
  ];
  const TOLERANCE_PRESETS = [
    { label: '普通 ±0.05mm', value: '0.05' },
    { label: '精密 ±0.03mm', value: '0.03' },
    { label: '超精密 ±0.005mm', value: '0.005' },
  ];
  const formatToleranceDisplay = (v: any): string => {
    if (v == null || v === '') return '';
    const num = parseFloat(String(v));
    if (!isNaN(num)) return `±${num}mm`;
    const map: Record<string, string> = { '普通': '±0.05mm', '精密': '±0.03mm', '超精密': '±0.005mm' };
    return map[v] || String(v);
  };

  // 新版图纸上传：创建PartDrawing版本记录
  const handleDrawingVersionUpload = async (file: File) => {
    try {
      const result = await taskFormApi.uploadFile(file);
      await partDrawingApi.upload(projectId, drawingUploadItemId, {
        drawing_type: drawingUploadType,
        file_id: result.id,
        file_name: result.filename,
        file_size: file.size,
        change_description: drawingChangeDesc,
      });
      message.success('上传成功');
      setDrawingUploadModalOpen(false);
      setDrawingChangeDesc('');
      queryClient.invalidateQueries({ queryKey: ['bom-drawings', projectId, selectedBomId] });
    } catch {
      message.error('上传失败');
    }
    return false;
  };

  // 获取某item某类型的最新图纸
  const getLatestDrawing = (itemId: string, type: '2D' | '3D'): PartDrawing | undefined => {
    const itemDrawings = drawingsByBOM[itemId];
    if (!itemDrawings) return undefined;
    const list = itemDrawings[type];
    return list && list.length > 0 ? list[0] : undefined;
  };

  // 获取某item某类型的图纸数量
  const getDrawingCount = (itemId: string, type: '2D' | '3D'): number => {
    const itemDrawings = drawingsByBOM[itemId];
    if (!itemDrawings) return 0;
    return itemDrawings[type]?.length || 0;
  };

  // Dynamic table columns based on BOM type
  const getColumns = (bomType: string, editable: boolean): ColumnsType<ProjectBOMItem> => {
    // 共用列
    const commonCols: ColumnsType<ProjectBOMItem> = [
      { title: '序号', dataIndex: 'item_number', width: 55, fixed: 'left',
        render: (_, __, idx) => idx + 1 },
      { title: '物料编码', width: 120,
        render: (_, record) => {
          const code = record.material?.code;
          return (
            <Space size={4}>
              <Text code style={{ fontSize: 11 }}>{code || '-'}</Text>
              {editable && (
                <SearchOutlined
                  style={{ color: '#1677ff', cursor: 'pointer', fontSize: 12 }}
                  onClick={() => { setEditingRowId(record.id); setMaterialModalOpen(true); }}
                />
              )}
            </Space>
          );
        },
      },
      { title: '物料名称', dataIndex: 'name', width: 140,
        render: (v, record) => renderEditableCell(v, record, 'name') },
      ...(bomType !== 'SBOM' ? [{
        title: '规格描述', dataIndex: 'specification', width: 150, ellipsis: true,
        render: (v: any, record: ProjectBOMItem) => renderEditableCell(v, record, 'specification'),
      } as any] : []),
      { title: '分类', dataIndex: 'category', width: 100,
        render: (v, record) => renderEditableCell(v, record, 'category', 'select',
          CATEGORY_OPTIONS.map(c => ({ label: c, value: c }))) },
      { title: '数量', dataIndex: 'quantity', width: 70, align: 'right',
        render: (v, record) => renderEditableCell(v, record, 'quantity', 'number') },
      { title: '单位', dataIndex: 'unit', width: 55,
        render: (v, record) => renderEditableCell(v, record, 'unit') },
    ];

    // EBOM特有列
    const ebomCols: ColumnsType<ProjectBOMItem> = [
      { title: '单价(¥)', dataIndex: 'unit_price', width: 90, align: 'right',
        render: (v, record) => renderEditableCell(v != null ? v.toFixed(2) : null, record, 'unit_price', 'number') },
      { title: '小计(¥)', dataIndex: 'extended_cost', width: 90, align: 'right',
        render: (v: number | null, record) => {
          const cost = v ?? (record.quantity && record.unit_price ? record.quantity * record.unit_price : null);
          return cost != null ? <Text strong style={{ color: '#cf1322' }}>¥{cost.toFixed(2)}</Text> : '-';
        },
      },
      { title: '制造商', dataIndex: 'manufacturer', width: 110, ellipsis: true,
        render: (v, record) => renderEditableCell(v, record, 'manufacturer') },
      { title: '制造商料号', dataIndex: 'manufacturer_pn', width: 110, ellipsis: true,
        render: (v, record) => renderEditableCell(v, record, 'manufacturer_pn') },
      { title: '供应商', dataIndex: 'supplier', width: 110, ellipsis: true,
        render: (v, record) => renderEditableCell(v, record, 'supplier') },
      { title: '交期(天)', dataIndex: 'lead_time_days', width: 75, align: 'right',
        render: (v, record) => renderEditableCell(v, record, 'lead_time_days', 'number') },
      { title: '采购类型', dataIndex: 'procurement_type', width: 100,
        render: (v, record) => renderEditableCell(
          PROCUREMENT_OPTIONS.find(o => o.value === v)?.label?.split('（')[0] || v,
          record, 'procurement_type', 'select', PROCUREMENT_OPTIONS) },
      { title: '关键件', dataIndex: 'is_critical', width: 65, align: 'center',
        render: (v: boolean, record) => editable ? (
          <Checkbox checked={v} onChange={(e) => handleCellSave(record, 'is_critical', e.target.checked)} />
        ) : (v ? <Tag color="red">是</Tag> : '-'),
      },
    ];

    // SBOM特有列
    const sbomCols: ColumnsType<ProjectBOMItem> = [
      { title: '材质', dataIndex: 'material_type', width: 110,
        render: (v, record) => {
          const isEd = editable && editingCell?.rowId === record.id && editingCell?.field === 'material_type';
          if (isEd) {
            return (
              <AutoComplete
                size="small" autoFocus defaultValue={v} defaultOpen style={{ width: '100%' }}
                options={MATERIAL_TYPE_PRESETS.map(m => ({ value: m }))}
                filterOption={(input, option) => (option?.value as string)?.toLowerCase().includes(input.toLowerCase())}
                onSelect={(val) => handleCellSave(record, 'material_type', val)}
                onBlur={(e) => handleCellSave(record, 'material_type', (e.target as HTMLInputElement).value)}
              />
            );
          }
          return editable ? (
            <div style={{ cursor: 'pointer', minHeight: 22, padding: '0 2px', borderRadius: 2 }} className="editable-cell"
              onClick={() => setEditingCell({ rowId: record.id, field: 'material_type' })}>
              {v ?? <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
            </div>
          ) : (v || '-');
        },
      },
      { title: '工艺', dataIndex: 'process_type', width: 90,
        render: (v, record) => renderEditableCell(v, record, 'process_type', 'select', PROCESS_TYPE_OPTIONS) },
      { title: '2D图纸', width: 150,
        render: (_, record) => {
          const latest = getLatestDrawing(record.id, '2D');
          const count = getDrawingCount(record.id, '2D');
          return (
            <Space size={4}>
              {latest ? (
                <Tooltip title={latest.file_name}>
                  <a href={latest.file_url} target="_blank" rel="noreferrer" style={{ fontSize: 11 }}>
                    {latest.version} {latest.file_name.length > 8 ? latest.file_name.slice(0, 8) + '...' : latest.file_name}
                  </a>
                </Tooltip>
              ) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
              {editable && (
                <UploadOutlined
                  style={{ color: '#1677ff', cursor: 'pointer', fontSize: 12 }}
                  onClick={() => { setDrawingUploadItemId(record.id); setDrawingUploadType('2D'); setDrawingUploadModalOpen(true); }}
                />
              )}
              {count > 0 && (
                <Tooltip title="查看历史版本">
                  <HistoryOutlined
                    style={{ color: '#8c8c8c', cursor: 'pointer', fontSize: 12 }}
                    onClick={() => { setDrawingHistoryItemId(record.id); setDrawingHistoryType('2D'); setDrawingHistoryOpen(true); }}
                  />
                </Tooltip>
              )}
            </Space>
          );
        },
      },
      { title: '3D模型', width: 150,
        render: (_, record) => {
          const latest = getLatestDrawing(record.id, '3D');
          const count = getDrawingCount(record.id, '3D');
          return (
            <Space size={4}>
              {latest ? (
                <Tooltip title={latest.file_name}>
                  <a href={latest.file_url} target="_blank" rel="noreferrer" style={{ fontSize: 11 }}>
                    {latest.version} {latest.file_name.length > 8 ? latest.file_name.slice(0, 8) + '...' : latest.file_name}
                  </a>
                </Tooltip>
              ) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
              {editable && (
                <UploadOutlined
                  style={{ color: '#1677ff', cursor: 'pointer', fontSize: 12 }}
                  onClick={() => { setDrawingUploadItemId(record.id); setDrawingUploadType('3D'); setDrawingUploadModalOpen(true); }}
                />
              )}
              {count > 0 && (
                <Tooltip title="查看历史版本">
                  <HistoryOutlined
                    style={{ color: '#8c8c8c', cursor: 'pointer', fontSize: 12 }}
                    onClick={() => { setDrawingHistoryItemId(record.id); setDrawingHistoryType('3D'); setDrawingHistoryOpen(true); }}
                  />
                </Tooltip>
              )}
            </Space>
          );
        },
      },
      { title: '重量g', dataIndex: 'weight_grams', width: 75, align: 'right',
        render: (v, record) => renderEditableCell(v, record, 'weight_grams', 'number') },
      { title: '目标价', dataIndex: 'target_price', width: 90, align: 'right',
        render: (v, record) => renderEditableCell(v != null ? `¥${v.toFixed(2)}` : null, record, 'target_price', 'number') },
      { title: '模具费', dataIndex: 'tooling_estimate', width: 90, align: 'right',
        render: (v, record) => renderEditableCell(v != null ? `¥${v.toFixed(2)}` : null, record, 'tooling_estimate', 'number') },
      { title: '成本备注', dataIndex: 'cost_notes', width: 120, ellipsis: true,
        render: (v, record) => renderEditableCell(v, record, 'cost_notes') },
      { title: '外观件', dataIndex: 'is_appearance_part', width: 65, align: 'center',
        render: (v: boolean, record) => editable ? (
          <Checkbox checked={v} onChange={(e) => handleCellSave(record, 'is_appearance_part', e.target.checked)} />
        ) : (v ? <Tag color="blue">是</Tag> : '-'),
      },
      { title: '装配方式', dataIndex: 'assembly_method', width: 100,
        render: (v, record) => renderEditableCell(v, record, 'assembly_method', 'select', ASSEMBLY_METHOD_OPTIONS) },
      { title: '公差', dataIndex: 'tolerance_grade', width: 95,
        render: (v, record) => {
          const isEd = editable && editingCell?.rowId === record.id && editingCell?.field === 'tolerance_grade';
          if (isEd) {
            return (
              <AutoComplete
                size="small" autoFocus defaultValue={v != null ? String(v) : ''} defaultOpen style={{ width: '100%' }}
                options={TOLERANCE_PRESETS}
                onSelect={(val) => handleCellSave(record, 'tolerance_grade', val)}
                onBlur={(e) => handleCellSave(record, 'tolerance_grade', (e.target as HTMLInputElement).value)}
              />
            );
          }
          const display = formatToleranceDisplay(v);
          return editable ? (
            <div style={{ cursor: 'pointer', minHeight: 22, padding: '0 2px', borderRadius: 2 }} className="editable-cell"
              onClick={() => setEditingCell({ rowId: record.id, field: 'tolerance_grade' })}>
              {display || <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
            </div>
          ) : (display || '-');
        },
      },
    ];

    const typeCols = bomType === 'SBOM' ? sbomCols : ebomCols;
    const noteCol: ColumnsType<ProjectBOMItem> = [
      { title: '备注', dataIndex: 'notes', width: 120, ellipsis: true,
        render: (v, record) => renderEditableCell(v, record, 'notes') },
    ];

    const cols = [...commonCols, ...typeCols, ...noteCol];

    if (editable) {
      cols.push({
        title: '操作', width: 80, fixed: 'right', align: 'center',
        render: (_, record) => (
          <Space size={4}>
            <Tooltip title="从物料库选择">
              <Button size="small" type="text" icon={<SearchOutlined />}
                onClick={() => { setEditingRowId(record.id); setMaterialModalOpen(true); }} />
            </Tooltip>
            <Popconfirm title="确认删除此行？" onConfirm={() => deleteItemMutation.mutate(record.id)}>
              <Button size="small" type="text" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </Space>
        ),
      });
    }

    return cols;
  };

  const bomType = bomDetail?.bom_type || 'EBOM';
  const columns = getColumns(bomType, isEditable);

  // Stats
  const items = bomDetail?.items || [];
  const totalItems = items.length;
  const totalCost = items.reduce((sum, item) => {
    const cost = item.extended_cost ?? (item.quantity && item.unit_price ? item.quantity * item.unit_price : 0);
    return sum + (cost || 0);
  }, 0);
  const criticalCount = items.filter(i => i.is_critical).length;
  // SBOM stats
  const isSBOM = bomType === 'SBOM';
  const appearanceCount = items.filter(i => i.is_appearance_part).length;
  const totalWeight = items.reduce((sum, item) => sum + (item.weight_grams || 0), 0);
  const totalTargetPrice = items.reduce((sum, item) => {
    const price = item.target_price || 0;
    return sum + price * (item.quantity || 1);
  }, 0);
  const totalTooling = items.reduce((sum, item) => sum + (item.tooling_estimate || 0), 0);

  // Action buttons based on status
  const renderActions = () => {
    if (!bomDetail) return null;
    const s = bomDetail.status;
    return (
      <Space split={<Divider type="vertical" />}>
        <Space>
          {(s === 'draft' || s === 'rejected') && (
            <Popconfirm title="确认提交审批？" onConfirm={() => submitMutation.mutate()}>
              <Button type="primary" icon={<SendOutlined />} loading={submitMutation.isPending}>提交审批</Button>
            </Popconfirm>
          )}
          {s === 'pending_review' && (
            <>
              <Popconfirm title="确认审批通过？" onConfirm={() => approveMutation.mutate()}>
                <Button type="primary" style={{ background: '#52c41a', borderColor: '#52c41a' }}
                  icon={<CheckCircleOutlined />} loading={approveMutation.isPending}>通过</Button>
              </Popconfirm>
              <Button danger icon={<CloseCircleOutlined />} onClick={() => setRejectModalOpen(true)}>驳回</Button>
            </>
          )}
          {s === 'published' && (
            <Popconfirm title="冻结后BOM不可再修改，确认冻结？" onConfirm={() => freezeMutation.mutate()}>
              <Button icon={<LockOutlined />} loading={freezeMutation.isPending}>冻结</Button>
            </Popconfirm>
          )}
          {s === 'frozen' && <Tag color="purple" icon={<LockOutlined />}>已冻结 - 只读</Tag>}
        </Space>
        <Space>
          <Tooltip title="导出Excel">
            <Button icon={<DownloadOutlined />} loading={exportLoading} onClick={handleExportExcel}>导出Excel</Button>
          </Tooltip>
          <Upload
            accept=".xlsx,.xls,.rep"
            showUploadList={false}
            beforeUpload={(file) => { handleImportExcel(file); return false; }}
            disabled={!(s === 'draft' || s === 'rejected')}
          >
            <Tooltip title={s === 'draft' || s === 'rejected' ? '支持Excel(.xlsx)和PADS(.rep)格式' : '仅草稿/已驳回状态可导入'}>
              <Button icon={<UploadOutlined />} loading={importLoading} disabled={!(s === 'draft' || s === 'rejected')}>导入BOM</Button>
            </Tooltip>
          </Upload>
          <Tooltip title="下载导入模板">
            <Button icon={<FileExcelOutlined />} onClick={handleDownloadTemplate}>下载模板</Button>
          </Tooltip>
          {bomDetail.bom_type === 'EBOM' && (s === 'published' || s === 'frozen') && (
            <Popconfirm title="确认将此EBOM转为MBOM副本？" onConfirm={() => convertToMBOMMutation.mutate()}>
              <Button icon={<SwapOutlined />} loading={convertToMBOMMutation.isPending}>转为MBOM</Button>
            </Popconfirm>
          )}
          <Tooltip title="版本对比">
            <Button icon={<SwapOutlined />} onClick={() => { setCompareModalOpen(true); setCompareResult(null); setCompareBom1(undefined); setCompareBom2(undefined); }}>版本对比</Button>
          </Tooltip>
          {items.length > 0 && (
            <Popconfirm title="确认将此BOM提交到SRM创建采购需求？" onConfirm={() => submitToSRMMutation.mutate()}>
              <Button type="primary" icon={<ShoppingCartOutlined />} loading={submitToSRMMutation.isPending}
                style={{ background: '#722ed1', borderColor: '#722ed1' }}>
                提交到SRM
              </Button>
            </Popconfirm>
          )}
        </Space>
      </Space>
    );
  };

  return (
    <div>
      {/* Top: BOM selector + create */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <Space>
          <Text strong style={{ fontSize: 15 }}>BOM管理</Text>
          {bomList.length > 0 && (
            <Select
              value={selectedBomId || undefined}
              onChange={setSelectedBomId}
              style={{ width: 260 }}
              placeholder="选择BOM"
              loading={listLoading}
              options={bomList.map(b => ({
                label: `${b.name} (${BOM_TYPE_CONFIG[b.bom_type]?.label || b.bom_type} ${b.version})`,
                value: b.id,
              }))}
            />
          )}
        </Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalOpen(true)}>
          新建BOM
        </Button>
      </div>

      {/* BOM Info Card */}
      {bomDetail && (
        <Card size="small" style={{ marginBottom: 12 }} styles={{ body: { padding: '10px 16px' } }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Space size={16}>
              <div>
                <Text strong style={{ fontSize: 15 }}>{bomDetail.name}</Text>
                <div style={{ marginTop: 2 }}>
                  <Space size={8}>
                    <Tag>{BOM_TYPE_CONFIG[bomDetail.bom_type]?.label || bomDetail.bom_type}</Tag>
                    <Tag>{bomDetail.version}</Tag>
                    <Tag color={BOM_STATUS_CONFIG[bomDetail.status]?.color}>
                      {BOM_STATUS_CONFIG[bomDetail.status]?.text || bomDetail.status}
                    </Tag>
                  </Space>
                </div>
              </div>
              <div style={{ borderLeft: '1px solid #f0f0f0', paddingLeft: 16 }}>
                <Text type="secondary" style={{ fontSize: 12 }}>{isSBOM ? '零件数' : '物料数'}</Text>
                <div><Text strong>{totalItems}</Text></div>
              </div>
              {isSBOM ? (
                <>
                  <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>外观件</Text>
                    <div><Text strong style={{ color: appearanceCount > 0 ? '#1677ff' : undefined }}>{appearanceCount}</Text></div>
                  </div>
                  <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>总重量</Text>
                    <div><Text strong>{totalWeight.toFixed(1)}g</Text></div>
                  </div>
                  <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>目标成本</Text>
                    <div><Text strong style={{ color: '#cf1322', fontSize: 16 }}>¥{totalTargetPrice.toFixed(2)}</Text></div>
                  </div>
                  <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>模具费</Text>
                    <div><Text strong style={{ color: '#cf1322' }}>¥{totalTooling.toFixed(2)}</Text></div>
                  </div>
                </>
              ) : (
                <>
                  <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>总成本</Text>
                    <div><Text strong style={{ color: '#cf1322', fontSize: 16 }}>¥{totalCost.toFixed(2)}</Text></div>
                  </div>
                  <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>关键件</Text>
                    <div><Text strong style={{ color: criticalCount > 0 ? '#cf1322' : undefined }}>{criticalCount}</Text></div>
                  </div>
                </>
              )}
              {bomDetail.creator && (
                <div>
                  <Text type="secondary" style={{ fontSize: 12 }}>创建者</Text>
                  <div><Text>{bomDetail.creator.name}</Text></div>
                </div>
              )}
            </Space>
            {renderActions()}
          </div>
        </Card>
      )}

      {/* Loading state */}
      {(listLoading || detailLoading) && !bomDetail && (
        <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
      )}

      {/* Empty state */}
      {!listLoading && bomList.length === 0 && (
        <Empty description="暂无BOM，请新建" style={{ padding: 60 }} />
      )}

      {/* Editable Table */}
      {bomDetail && (
        <>
          <style>{`
            .bom-table .editable-cell:hover {
              background: #f5f5f5;
            }
            .bom-table .critical-row {
              background: #fff1f0 !important;
            }
            .bom-table .critical-row:hover > td {
              background: #ffccc7 !important;
            }
          `}</style>
          <Table
            className="bom-table"
            columns={columns}
            dataSource={items}
            rowKey="id"
            size="small"
            pagination={false}
            scroll={{ x: isSBOM ? 2300 : 1800, y: 450 }}
            locale={{ emptyText: '暂无物料行项，点击下方添加' }}
            rowClassName={(record) => record.is_critical ? 'critical-row' : ''}
          />

          {/* Bottom: add row button + stats */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 8, padding: '8px 0', borderTop: '1px solid #f0f0f0' }}>
            <div>
              {isEditable && (
                <Space>
                  <Button type="dashed" icon={<PlusOutlined />} onClick={handleAddRow}
                    loading={addItemMutation.isPending}>
                    添加物料
                  </Button>
                  <Button type="dashed" icon={<SearchOutlined />} onClick={() => { setEditingRowId(null); setMaterialModalOpen(true); }}>
                    从物料库选择
                  </Button>
                </Space>
              )}
            </div>
            <Space size={24}>
              <Text type="secondary">共 <Text strong>{totalItems}</Text> 项{isSBOM ? '零件' : '物料'}</Text>
              {isSBOM ? (
                <>
                  <Text type="secondary">外观件 <Text strong style={{ color: appearanceCount > 0 ? '#1677ff' : undefined }}>{appearanceCount}</Text> 项</Text>
                  <Text type="secondary">总重量 <Text strong>{totalWeight.toFixed(1)}g</Text></Text>
                  <Text>目标成本合计 <Text strong style={{ color: '#cf1322', fontSize: 18 }}>¥{totalTargetPrice.toFixed(2)}</Text></Text>
                  <Text>模具费合计 <Text strong style={{ color: '#cf1322', fontSize: 18 }}>¥{totalTooling.toFixed(2)}</Text></Text>
                </>
              ) : (
                <>
                  <Text type="secondary">关键件 <Text strong style={{ color: criticalCount > 0 ? '#cf1322' : undefined }}>{criticalCount}</Text> 项</Text>
                  <Text>
                    总成本 <Text strong style={{ color: '#cf1322', fontSize: 18 }}>¥{totalCost.toFixed(2)}</Text>
                  </Text>
                </>
              )}
            </Space>
          </div>
        </>
      )}

      {/* Create BOM Modal */}
      <Modal
        title="新建BOM"
        open={createModalOpen}
        onCancel={() => { setCreateModalOpen(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(values) => createMutation.mutate(values)}>
          <Form.Item name="name" label="BOM名称" rules={[{ required: true, message: '请输入BOM名称' }]}>
            <Input placeholder="如：EVT电子BOM" />
          </Form.Item>
          <Form.Item name="bom_type" label="BOM类型" rules={[{ required: true, message: '请选择BOM类型' }]}>
            <Select options={[
              { label: 'EBOM - 电子BOM', value: 'EBOM' },
              { label: 'SBOM - 结构BOM', value: 'SBOM' },
              { label: 'MBOM - 制造BOM', value: 'MBOM' },
            ]} />
          </Form.Item>
          <Form.Item name="version" label="版本号" initialValue="v1.0">
            <Input placeholder="如：v1.0" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="BOM描述信息" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Reject Modal */}
      <Modal
        title="驳回BOM"
        open={rejectModalOpen}
        onCancel={() => { setRejectModalOpen(false); setRejectComment(''); }}
        onOk={() => rejectMutation.mutate(rejectComment)}
        confirmLoading={rejectMutation.isPending}
        okText="确认驳回"
        okButtonProps={{ danger: true }}
      >
        <Input.TextArea
          rows={4}
          placeholder="请输入驳回原因..."
          value={rejectComment}
          onChange={(e) => setRejectComment(e.target.value)}
        />
      </Modal>

      {/* Material Search Modal */}
      <MaterialSearchModal
        open={materialModalOpen}
        onClose={() => { setMaterialModalOpen(false); setEditingRowId(null); }}
        onSelect={handleMaterialSelect}
      />

      {/* Compare BOMs Modal */}
      <Modal
        title="BOM版本对比"
        open={compareModalOpen}
        onCancel={() => setCompareModalOpen(false)}
        width={800}
        footer={compareResult ? [
          <Button key="close" onClick={() => setCompareModalOpen(false)}>关闭</Button>,
        ] : undefined}
        onOk={handleCompare}
        confirmLoading={compareLoading}
        okText="开始对比"
      >
        <Space style={{ marginBottom: 16, width: '100%' }} direction="vertical">
          <Space>
            <Text>BOM A：</Text>
            <Select
              style={{ width: 280 }}
              placeholder="选择第一个BOM"
              value={compareBom1}
              onChange={setCompareBom1}
              options={bomList.map(b => ({
                label: `${b.name} (${BOM_TYPE_CONFIG[b.bom_type]?.label || b.bom_type} ${b.version})`,
                value: b.id,
              }))}
            />
          </Space>
          <Space>
            <Text>BOM B：</Text>
            <Select
              style={{ width: 280 }}
              placeholder="选择第二个BOM"
              value={compareBom2}
              onChange={setCompareBom2}
              options={bomList.map(b => ({
                label: `${b.name} (${BOM_TYPE_CONFIG[b.bom_type]?.label || b.bom_type} ${b.version})`,
                value: b.id,
              }))}
            />
          </Space>
        </Space>
        {compareResult && (
          <Table
            dataSource={compareResult}
            rowKey={(_, idx) => String(idx)}
            size="small"
            pagination={false}
            scroll={{ y: 400 }}
            rowClassName={(record) => {
              if (record.change_type === 'added') return 'compare-row-added';
              if (record.change_type === 'removed') return 'compare-row-removed';
              if (record.change_type === 'changed') return 'compare-row-changed';
              return '';
            }}
            columns={[
              { title: '序号', width: 60, render: (_, __, idx) => idx + 1 },
              { title: '物料名称', dataIndex: 'name', width: 140 },
              { title: '规格', dataIndex: 'specification', width: 160, ellipsis: true },
              { title: '变更类型', dataIndex: 'change_type', width: 100,
                render: (v: string) => {
                  const map: Record<string, { color: string; text: string }> = {
                    added: { color: 'success', text: '新增' },
                    removed: { color: 'error', text: '删除' },
                    changed: { color: 'warning', text: '变更' },
                    unchanged: { color: 'default', text: '未变' },
                  };
                  const cfg = map[v] || { color: 'default', text: v };
                  return <Tag color={cfg.color}>{cfg.text}</Tag>;
                },
              },
              { title: '变更详情', dataIndex: 'details', ellipsis: true },
            ]}
          />
        )}
        <style>{`
          .compare-row-added { background: #f6ffed !important; }
          .compare-row-added:hover > td { background: #d9f7be !important; }
          .compare-row-removed { background: #fff1f0 !important; }
          .compare-row-removed:hover > td { background: #ffccc7 !important; }
          .compare-row-changed { background: #fffbe6 !important; }
          .compare-row-changed:hover > td { background: #fff1b8 !important; }
        `}</style>
      </Modal>

      {/* 图纸上传Modal */}
      <Modal
        title={`上传${drawingUploadType}图纸新版本`}
        open={drawingUploadModalOpen}
        onCancel={() => { setDrawingUploadModalOpen(false); setDrawingChangeDesc(''); }}
        footer={null}
        width={400}
      >
        <div style={{ marginBottom: 12 }}>
          <Text type="secondary">变更说明（可选）</Text>
          <Input.TextArea
            rows={2}
            value={drawingChangeDesc}
            onChange={(e) => setDrawingChangeDesc(e.target.value)}
            placeholder="描述本次变更内容..."
            style={{ marginTop: 4 }}
          />
        </div>
        <Upload
          showUploadList={false}
          beforeUpload={handleDrawingVersionUpload}
        >
          <Button icon={<UploadOutlined />} type="primary">选择文件并上传</Button>
        </Upload>
      </Modal>

      {/* 图纸版本历史Drawer */}
      <Drawer
        title={`${drawingHistoryType}图纸版本历史`}
        open={drawingHistoryOpen}
        onClose={() => setDrawingHistoryOpen(false)}
        width={480}
      >
        {(() => {
          const itemDrawings = drawingsByBOM[drawingHistoryItemId];
          const list = itemDrawings?.[drawingHistoryType] || [];
          if (list.length === 0) return <Empty description="暂无图纸版本" />;
          return (
            <Timeline
              items={list.map((d: PartDrawing) => ({
                key: d.id,
                color: d === list[0] ? 'blue' : 'gray',
                children: (
                  <div>
                    <Space>
                      <Tag color={d === list[0] ? 'blue' : 'default'}>{d.version}</Tag>
                      <a href={d.file_url} target="_blank" rel="noreferrer">{d.file_name}</a>
                      <Text type="secondary" style={{ fontSize: 11 }}>
                        {d.file_size ? `${(d.file_size / 1024).toFixed(0)}KB` : ''}
                      </Text>
                    </Space>
                    {d.change_description && (
                      <div style={{ marginTop: 4 }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>{d.change_description}</Text>
                      </div>
                    )}
                    <div style={{ marginTop: 2 }}>
                      <Text type="secondary" style={{ fontSize: 11 }}>
                        {d.uploader?.name || '未知'} {dayjs(d.created_at).format('MM-DD HH:mm')}
                      </Text>
                    </div>
                  </div>
                ),
              }))}
            />
          );
        })()}
      </Drawer>
    </div>
  );
};

// ============ Documents Tab ============

const DocumentsTab: React.FC<{ projectId: string }> = ({ projectId }) => {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['project-documents', projectId],
    queryFn: () => documentsApi.list({ related_type: 'project', related_id: projectId }),
    retry: false,
  });

  const columns: ColumnsType<Document> = [
    { title: '文档编号', dataIndex: 'code', key: 'code', width: 140, render: (t: string) => <Text code>{t}</Text> },
    { title: '标题', dataIndex: 'title', key: 'title', width: 200 },
    { title: '分类', dataIndex: 'category', key: 'category', width: 100, render: (_, record) => (record.category as any)?.name || (typeof record.category === 'string' ? record.category : '-') },
    { title: '版本', dataIndex: 'version', key: 'version', width: 80 },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (s: string) => <Tag color={s === 'released' ? 'success' : s === 'draft' ? 'default' : 'warning'}>{s === 'released' ? '已发布' : s === 'draft' ? '草稿' : s}</Tag>
    },
    { title: '上传者', dataIndex: 'created_by_name', key: 'created_by_name', width: 100, render: (v: string, record) => v || record.uploader?.name || '-' },
    { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 160, render: (d: string) => d ? dayjs(d).format('YYYY-MM-DD HH:mm') : '-' },
  ];

  if (isError) {
    return <Empty description="文档数据暂不可用（API开发中）" image={Empty.PRESENTED_IMAGE_SIMPLE} />;
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Text strong>图纸文档</Text>
        <Button icon={<UploadOutlined />}>上传文档</Button>
      </div>
      <Table
        columns={columns}
        dataSource={data?.items || []}
        rowKey="id"
        loading={isLoading}
        size="small"
        pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
        locale={{ emptyText: '暂无文档' }}
      />
    </div>
  );
};

// ============ Deliverables Tab ============

const DeliverablesTab: React.FC<{ projectId: string; currentPhase: string }> = ({ projectId, currentPhase }) => {
  const [selectedPhase, setSelectedPhase] = useState(currentPhase?.toLowerCase() || 'evt');

  const { data, isLoading, isError } = useQuery({
    queryKey: ['project-deliverables', projectId, selectedPhase],
    queryFn: () => deliverablesApi.list(projectId, selectedPhase),
    retry: false,
  });

  const deliverables = data?.items || [];
  const completed = deliverables.filter(d => d.status === 'approved' || d.status === 'submitted').length;
  const total = deliverables.length;
  const percent = total > 0 ? Math.round((completed / total) * 100) : 0;
  const allComplete = total > 0 && completed === total;
  const remaining = total - completed;

  const statusConfig: Record<string, { icon: string; color: string; text: string }> = {
    not_started: { icon: '⬜', color: '#999', text: '未开始' },
    in_progress: { icon: '🟡', color: '#faad14', text: '进行中' },
    submitted: { icon: '✅', color: '#52c41a', text: '已提交' },
    approved: { icon: '✅', color: '#52c41a', text: '已审批' },
    rejected: { icon: '❌', color: '#ff4d4f', text: '已驳回' },
  };

  if (isError) {
    return <Empty description="交付物数据暂不可用（API开发中）" image={Empty.PRESENTED_IMAGE_SIMPLE} />;
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space>
          <Text strong>交付物清单</Text>
          <Select
            value={selectedPhase}
            onChange={setSelectedPhase}
            style={{ width: 120 }}
            options={PHASES.map(p => ({ label: `${p.toUpperCase()} 阶段`, value: p }))}
          />
        </Space>
      </div>

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
      ) : deliverables.length === 0 ? (
        <Empty description="暂无交付物" />
      ) : (
        <>
          <Card size="small" style={{ marginBottom: 16, background: '#fafafa' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <Text>完成进度: {completed}/{total} ({percent}%)</Text>
              <Progress percent={percent} style={{ flex: 1, maxWidth: 300 }} size="small"
                status={allComplete ? 'success' : 'active'} />
            </div>
          </Card>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {deliverables.map(d => {
              const sc = statusConfig[d.status] || statusConfig.not_started;
              return (
                <div key={d.id} style={{
                  display: 'flex', alignItems: 'center', padding: '10px 16px',
                  border: '1px solid #f0f0f0', borderRadius: 6, background: '#fff',
                }}>
                  <span style={{ fontSize: 16, marginRight: 12 }}>{sc.icon}</span>
                  <div style={{ flex: 1 }}>
                    <Text strong>{d.name}</Text>
                    {d.description && <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>{d.description}</Text>}
                  </div>
                  <Text type="secondary" style={{ marginRight: 16 }}>{d.assignee_role || d.assignee_name || '-'}</Text>
                  <Tag color={sc.color === '#52c41a' ? 'success' : sc.color === '#ff4d4f' ? 'error' : sc.color === '#faad14' ? 'warning' : 'default'}>
                    {sc.text}
                  </Tag>
                </div>
              );
            })}
          </div>

          <div style={{ marginTop: 16, textAlign: 'right' }}>
            {!allComplete && (
              <Alert
                type="warning"
                showIcon
                icon={<WarningOutlined />}
                message={`还有 ${remaining} 项未完成，无法发起阶段门评审`}
                style={{ marginBottom: 12 }}
              />
            )}
            <Button type="primary" disabled={!allComplete}>
              发起阶段门评审
            </Button>
          </div>
        </>
      )}
    </div>
  );
};

// ============ ECN Tab ============

const ECNTab: React.FC<{ projectId: string; productId?: string }> = ({ productId }) => {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['project-ecns', productId],
    queryFn: () => ecnApi.list({ product_id: productId }),
    enabled: !!productId,
    retry: false,
  });

  const ecnStatusConfig: Record<string, { color: string; text: string }> = {
    draft: { color: 'default', text: '草稿' },
    pending: { color: 'processing', text: '待审批' },
    approved: { color: 'success', text: '已批准' },
    rejected: { color: 'error', text: '已驳回' },
    implemented: { color: 'purple', text: '已实施' },
  };

  const urgencyColors: Record<string, string> = {
    low: 'default',
    medium: 'blue',
    high: 'orange',
    urgent: 'red',
  };

  const columns: ColumnsType<ECN> = [
    { title: 'ECN编号', dataIndex: 'code', key: 'code', width: 140, render: (t: string) => <Text code>{t}</Text> },
    { title: '标题', dataIndex: 'title', key: 'title', width: 200 },
    { title: '变更类型', dataIndex: 'change_type', key: 'change_type', width: 100 },
    { title: '紧急度', dataIndex: 'urgency', key: 'urgency', width: 80,
      render: (u: string) => <Tag color={urgencyColors[u] || 'default'}>{u === 'high' ? '高' : u === 'medium' ? '中' : u === 'urgent' ? '紧急' : '低'}</Tag>
    },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100,
      render: (s: string) => { const cfg = ecnStatusConfig[s] || { color: 'default', text: s }; return <Tag color={cfg.color}>{cfg.text}</Tag>; }
    },
    { title: '申请人', key: 'requester', width: 100, render: (_, r) => r.requester?.name || '-' },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 160, render: (d: string) => d ? dayjs(d).format('YYYY-MM-DD HH:mm') : '-' },
  ];

  if (isError) {
    return <Empty description="ECN数据暂不可用" image={Empty.PRESENTED_IMAGE_SIMPLE} />;
  }

  if (!productId) {
    return <Empty description="该项目未关联产品，无法查看ECN" image={Empty.PRESENTED_IMAGE_SIMPLE} />;
  }

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Text strong>工程变更通知</Text>
      </div>
      <Table
        columns={columns}
        dataSource={data?.items || []}
        rowKey="id"
        loading={isLoading}
        size="small"
        pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
        locale={{ emptyText: '暂无ECN记录' }}
      />
    </div>
  );
};

// ============ Task Actions Component ============

// ============ BOM Submission Display (read-only) ============

const BOMSubmissionDisplay: React.FC<{ data: { filename: string; items: ParsedBOMItem[]; item_count: number } }> = ({ data }) => {
  const [expanded, setExpanded] = React.useState(false);

  const categoryStats = React.useMemo(() => {
    if (!data?.items?.length) return [];
    const map: Record<string, number> = {};
    for (const item of data.items) {
      const cat = item.category || '未分类';
      map[cat] = (map[cat] || 0) + 1;
    }
    return Object.entries(map).sort((a, b) => b[1] - a[1]);
  }, [data?.items]);

  const columns = [
    { title: '序号', dataIndex: 'item_number', key: 'item_number', width: 50, align: 'center' as const },
    { title: '位号', dataIndex: 'reference', key: 'reference', width: 80, ellipsis: true },
    { title: '名称', dataIndex: 'name', key: 'name', width: 100, ellipsis: true },
    { title: '规格', dataIndex: 'specification', key: 'specification', width: 120, ellipsis: true },
    { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 50, align: 'center' as const },
    { title: '单位', dataIndex: 'unit', key: 'unit', width: 45, align: 'center' as const },
    { title: '类别', dataIndex: 'category', key: 'category', width: 70, ellipsis: true },
    { title: '制造商', dataIndex: 'manufacturer', key: 'manufacturer', width: 90, ellipsis: true },
  ];

  return (
    <div>
      <Space size={8} style={{ marginBottom: 4 }}>
        <FileExcelOutlined style={{ color: '#52c41a' }} />
        <span style={{ fontSize: 13 }}>{data.filename}</span>
        <Tag color="blue">{data.item_count} 项物料</Tag>
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
            scroll={{ x: 600 }}
          />
        </div>
      )}
    </div>
  );
};

// ============ Form Submission Display ============

const FormSubmissionDisplay: React.FC<{ projectId: string; taskId: string }> = ({ projectId, taskId }) => {
  const [formDef, setFormDef] = useState<any>(null);
  const [submission, setSubmission] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [userMap, setUserMap] = useState<Record<string, string>>({});

  React.useEffect(() => {
    Promise.all([
      taskFormApi.getForm(projectId, taskId),
      taskFormApi.getSubmission(projectId, taskId),
    ]).then(([form, sub]) => {
      setFormDef(form);
      setSubmission(sub);
      // If any field is of type 'user' or 'role_assignment', fetch user list to resolve names
      if (form?.fields?.some((f: any) => f.type === 'user' || f.type === 'role_assignment')) {
        userApi.list().then((users) => {
          const map: Record<string, string> = {};
          users.forEach((u) => { map[u.id] = u.name; });
          setUserMap(map);
        });
      }
    }).finally(() => setLoading(false));
  }, [projectId, taskId]);

  if (loading) return <div style={{ color: '#999', fontSize: 12 }}>加载表单数据...</div>;
  if (!formDef || !submission) return null;

  const fields = formDef.fields || [];
  const data = submission.data || {};

  return (
    <div style={{ background: '#fafafa', padding: 12, borderRadius: 6, marginTop: 8 }}>
      <Text strong style={{ fontSize: 13, marginBottom: 8, display: 'block' }}>已提交的表单数据</Text>
      <Descriptions size="small" column={2} bordered>
        {fields.map((field: any) => {
          let value = data[field.key];
          // bom_upload: show filename + count + expandable table
          if (field.type === 'bom_upload' && value && typeof value === 'object' && value.items) {
            return (
              <Descriptions.Item key={field.key} label={field.label} span={2}>
                <BOMSubmissionDisplay data={value} />
              </Descriptions.Item>
            );
          }
          if (value === undefined || value === null) value = '-';
          else if (field.type === 'role_assignment' && typeof value === 'object' && !Array.isArray(value)) {
            const lines = Object.entries(value as Record<string, string>)
              .map(([code, uid]) => `${code}: ${userMap[uid] || uid}`)
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

const TaskActions: React.FC<{
  task: Task;
  projectId: string;
  onRefresh: () => void;
}> = ({ task, projectId, onRefresh }) => {
  const [assignModalOpen, setAssignModalOpen] = useState(false);
  const [rejectModalOpen, setRejectModalOpen] = useState(false);
  const [approvalModalOpen, setApprovalModalOpen] = useState(false);
  const [formDrawerOpen, setFormDrawerOpen] = useState(false);
  const [historyDrawerOpen, setHistoryDrawerOpen] = useState(false);
  const [historyData, setHistoryData] = useState<TaskActionLog[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [assigneeId, setAssigneeId] = useState('');
  const [feishuUserId, setFeishuUserId] = useState('');
  const [rejectComment, setRejectComment] = useState('');
  const [reviewerIds, setReviewerIds] = useState<string[]>([]);

  const handleError = (err: unknown) => {
    const axiosErr = err as any;
    const status = axiosErr?.response?.status;
    const errMsg = axiosErr?.response?.data?.error || axiosErr?.response?.data?.message || '操作失败';
    if (status === 400) {
      message.error(`前置任务未完成，${errMsg}`);
    } else {
      message.error(errMsg);
    }
  };

  const handleAssign = async () => {
    if (!assigneeId.trim()) {
      message.warning('请输入负责人ID');
      return;
    }
    setLoading(true);
    try {
      await workflowApi.assignTask(projectId, task.id, {
        assignee_id: assigneeId.trim(),
        feishu_user_id: feishuUserId.trim() || undefined,
      });
      message.success('指派成功');
      setAssignModalOpen(false);
      setAssigneeId('');
      setFeishuUserId('');
      onRefresh();
    } catch (err) {
      handleError(err);
    } finally {
      setLoading(false);
    }
  };

  const handleStart = async () => {
    setLoading(true);
    try {
      await workflowApi.startTask(projectId, task.id);
      message.success('任务已开始');
      onRefresh();
    } catch (err) {
      handleError(err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmitApproval = async () => {
    if (reviewerIds.length === 0) {
      message.warning('请选择至少一位审批人');
      return;
    }
    setLoading(true);
    try {
      await approvalApi.create({
        project_id: projectId,
        task_id: task.id,
        title: `任务审批: ${task.title}`,
        reviewer_ids: reviewerIds,
      });
      message.success('审批已提交');
      setApprovalModalOpen(false);
      setReviewerIds([]);
      onRefresh();
    } catch (err) {
      handleError(err);
    } finally {
      setLoading(false);
    }
  };

  const handleReject = async () => {
    setLoading(true);
    try {
      await workflowApi.submitReview(projectId, task.id, {
        outcome_code: 'fail_rollback',
        comment: rejectComment,
      });
      message.success('已驳回');
      setRejectModalOpen(false);
      setRejectComment('');
      onRefresh();
    } catch (err) {
      console.error('Review reject failed:', err);
      handleError(err);
    } finally {
      setLoading(false);
    }
  };

  const loadHistory = async () => {
    setHistoryLoading(true);
    try {
      const data = await workflowApi.getTaskHistory(projectId, task.id);
      setHistoryData(data);
    } catch (err) {
      handleError(err);
    } finally {
      setHistoryLoading(false);
    }
  };

  const openHistory = () => {
    setHistoryDrawerOpen(true);
    loadHistory();
  };

  const actionNameMap: Record<string, string> = {
    assign: '指派',
    start: '开始',
    complete: '完成',
    review_pass: '审批通过',
    review_reject: '审批驳回',
    review: '评审',
    rollback: '回退',
  };

  const handlePmConfirm = async () => {
    setLoading(true);
    try {
      await taskFormApi.confirmTask(projectId, task.id);
      message.success('任务已确认');
      onRefresh();
    } catch (err) {
      handleError(err);
    } finally {
      setLoading(false);
    }
  };

  const handlePmReject = async () => {
    setLoading(true);
    try {
      await taskFormApi.rejectTask(projectId, task.id);
      message.success('任务已驳回');
      onRefresh();
    } catch (err) {
      handleError(err);
    } finally {
      setLoading(false);
    }
  };

  const renderActions = () => {
    switch (task.status) {
      case 'unassigned':
        return (
          <Button size="small" type="primary" onClick={() => setAssignModalOpen(true)} loading={loading}>
            指派
          </Button>
        );
      case 'pending': {
        // 检查是否有未完成的前置任务
        const hasUnfinishedDeps = task.dependencies?.some(
          d => d.depends_on_status !== 'completed'
        );
        if (hasUnfinishedDeps) {
          return <Tag color="default" icon={<ClockCircleOutlined />}>等待前置任务</Tag>;
        }
        return (
          <Button size="small" type="primary" style={{ background: '#52c41a', borderColor: '#52c41a' }} onClick={handleStart} loading={loading}>
            开始
          </Button>
        );
      }
      case 'in_progress':
        return <Tag color="processing" icon={<ClockCircleOutlined />}>进行中</Tag>;
      case 'submitted':
        // 非流程任务(requires_approval=false)：PM 显示通过/驳回 Popconfirm
        if (!task.requires_approval) {
          return (
            <Space size={4}>
              <Popconfirm title="确认通过该任务？" onConfirm={handlePmConfirm} okText="通过" cancelText="取消">
                <Tooltip title="通过">
                  <Button size="small" type="text" icon={<CheckCircleOutlined />} style={{ color: '#52c41a' }} loading={loading} />
                </Tooltip>
              </Popconfirm>
              <Popconfirm title="确认驳回该任务？" onConfirm={handlePmReject} okText="驳回" cancelText="取消" okButtonProps={{ danger: true }}>
                <Tooltip title="驳回">
                  <Button size="small" type="text" icon={<CloseCircleOutlined />} style={{ color: '#ff4d4f' }} loading={loading} />
                </Tooltip>
              </Popconfirm>
            </Space>
          );
        }
        return <Tag color="orange" icon={<CheckCircleOutlined />}>已提交</Tag>;
      case 'reviewing':
        return (
          <Tag color="warning" icon={<AuditOutlined />}>审批中</Tag>
        );
      case 'completed':
        return <Tag color="green" icon={<CheckCircleOutlined />}>已完成</Tag>;
      case 'rejected':
        return (
          <Button size="small" style={{ color: '#fa8c16', borderColor: '#fa8c16' }} onClick={handleStart} loading={loading}>
            重新开始
          </Button>
        );
      default:
        return null;
    }
  };

  const showFormDataButton = task.status === 'submitted' || task.status === 'completed' || task.status === 'reviewing';

  return (
    <>
      <Space size={4}>
        {renderActions()}
        {showFormDataButton && (
          <Tooltip title="查看表单">
            <Button size="small" type="text" icon={<EyeOutlined />} onClick={() => setFormDrawerOpen(true)} style={{ color: '#1677ff' }} />
          </Tooltip>
        )}
        <Tooltip title="操作历史">
          <Button size="small" type="text" icon={<HistoryOutlined />} onClick={openHistory} style={{ color: '#999' }} />
        </Tooltip>
      </Space>

      {/* Assign Modal */}
      <Modal
        title={`指派任务: ${task.title}`}
        open={assignModalOpen}
        onCancel={() => { setAssignModalOpen(false); setAssigneeId(''); setFeishuUserId(''); }}
        onOk={handleAssign}
        confirmLoading={loading}
        okText="确认指派"
        cancelText="取消"
      >
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>负责人 *</Text>
          <UserSelect
            value={assigneeId || undefined}
            onChange={(val) => setAssigneeId(val as string)}
            mode="single"
            placeholder="选择负责人"
            style={{ width: '100%' }}
          />
        </div>
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>飞书用户 ID（可选）</Text>
          <Input
            placeholder="输入飞书 User ID（可选）"
            value={feishuUserId}
            onChange={(e) => setFeishuUserId(e.target.value)}
          />
        </div>
      </Modal>

      {/* Reject Modal */}
      <Modal
        title="驳回任务"
        open={rejectModalOpen}
        onCancel={() => { setRejectModalOpen(false); setRejectComment(''); }}
        onOk={handleReject}
        confirmLoading={loading}
        okText="确认驳回"
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>驳回原因</Text>
        <Input.TextArea
          rows={4}
          placeholder="请输入驳回原因..."
          value={rejectComment}
          onChange={(e) => setRejectComment(e.target.value)}
        />
      </Modal>

      {/* Approval Modal */}
      <Modal
        title="提交审批"
        open={approvalModalOpen}
        onCancel={() => { setApprovalModalOpen(false); setReviewerIds([]); }}
        onOk={handleSubmitApproval}
        confirmLoading={loading}
        okText="提交审批"
        cancelText="取消"
      >
        <div style={{ marginBottom: 8 }}>
          <Text type="secondary">任务: {task.title}</Text>
        </div>
        <div>
          <Text type="secondary" style={{ display: 'block', marginBottom: 4 }}>选择审批人 *</Text>
          <UserSelect
            value={reviewerIds}
            onChange={(val) => setReviewerIds(val as string[])}
            mode="multiple"
            placeholder="选择审批人"
            style={{ width: '100%' }}
          />
        </div>
      </Modal>

      {/* Form Data Drawer */}
      <Drawer
        title={`任务表单: ${task.title}`}
        open={formDrawerOpen}
        onClose={() => setFormDrawerOpen(false)}
        width={520}
      >
        <FormSubmissionDisplay projectId={projectId} taskId={task.id} />
      </Drawer>

      {/* History Drawer */}
      <Drawer
        title={`操作历史: ${task.title}`}
        open={historyDrawerOpen}
        onClose={() => setHistoryDrawerOpen(false)}
        width={480}
      >
        {historyLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : historyData.length === 0 ? (
          <Empty description="暂无操作记录" />
        ) : (
          <Timeline
            items={historyData.map((log) => ({
              color: log.action.includes('reject') || log.action.includes('fail') ? 'red' :
                     log.action.includes('pass') || log.action === 'complete' ? 'green' :
                     log.action === 'start' ? 'blue' : 'gray',
              children: (
                <div>
                  <div style={{ fontWeight: 500 }}>
                    {actionNameMap[log.action] || log.action}
                  </div>
                  <div style={{ fontSize: 12, color: '#666' }}>
                    {log.from_status && log.to_status && (
                      <Tag style={{ fontSize: 11 }}>
                        {(taskStatusConfig[log.from_status]?.text || log.from_status)} → {(taskStatusConfig[log.to_status]?.text || log.to_status)}
                      </Tag>
                    )}
                  </div>
                  <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                    <span>操作人: {log.operator_id}</span>
                    <span style={{ marginLeft: 12 }}>
                      {dayjs(log.created_at).format('YYYY-MM-DD HH:mm:ss')}
                    </span>
                  </div>
                  {log.comment && (
                    <div style={{ fontSize: 12, color: '#fa8c16', marginTop: 4 }}>
                      备注: {log.comment}
                    </div>
                  )}
                </div>
              ),
            }))}
          />
        )}
      </Drawer>
    </>
  );
};

// ============ Role Assignment Tab ============

// ROLE_CODES imported from @/constants/roles

const RoleAssignmentTab: React.FC<{ projectId: string }> = ({ projectId }) => {
  const queryClient = useQueryClient();
  const [assignments, setAssignments] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  // Fetch tasks to extract unique default_assignee_role values
  const { data: tasks = [] } = useQuery({
    queryKey: ['project-tasks', projectId],
    queryFn: () => projectApi.listTasks(projectId),
    enabled: !!projectId,
  });

  // Fetch task roles for label lookup
  const { data: taskRolesData = [] } = useQuery<TaskRole[]>({
    queryKey: ['task-roles'],
    queryFn: () => taskRoleApi.list(),
  });

  const roleLabelMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const r of taskRolesData) {
      map[r.code] = r.name;
    }
    for (const rc of ROLE_CODES) {
      if (!map[rc.code]) map[rc.code] = rc.label;
    }
    return map;
  }, [taskRolesData]);

  // Extract unique roles from tasks
  const uniqueRoles = useMemo(() => {
    const roles = new Set<string>();
    for (const t of tasks) {
      const role = (t as any).default_assignee_role;
      if (role) roles.add(role);
    }
    return Array.from(roles).sort();
  }, [tasks]);

  const updateAssignment = (roleCode: string, userId: string) => {
    setAssignments(prev => ({ ...prev, [roleCode]: userId }));
  };

  const handleSave = async () => {
    const validAssignments = Object.entries(assignments)
      .filter(([, userId]) => userId && userId.trim())
      .map(([role, userId]) => ({ role, user_id: userId.trim() }));

    if (validAssignments.length === 0) {
      message.warning('请至少填写一个角色的负责人');
      return;
    }

    setLoading(true);
    try {
      await projectApi.assignRoles(projectId, validAssignments);
      message.success('角色分配成功，已更新对应任务的负责人');
      queryClient.invalidateQueries({ queryKey: ['project-tasks', projectId] });
      queryClient.invalidateQueries({ queryKey: ['project', projectId] });
      queryClient.invalidateQueries({ queryKey: ['my-tasks'] });
    } catch (err) {
      const axiosErr = err as any;
      const errMsg = axiosErr?.response?.data?.message || '分配失败';
      message.error(errMsg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Text strong>角色分配</Text>
        <Button type="primary" onClick={handleSave} loading={loading}>
          保存并更新任务
        </Button>
      </div>

      <Alert
        type="info"
        showIcon
        message="为每个角色指定负责人后，将自动更新该角色下所有任务的负责人"
        style={{ marginBottom: 16 }}
      />

      {uniqueRoles.length === 0 ? (
        <Empty description="项目任务中未配置角色，请在研发流程模板中为任务分配角色" />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {uniqueRoles.map(role => (
            <Card key={role} size="small" styles={{ body: { padding: '12px 16px' } }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                <div style={{ width: 140, fontWeight: 500 }}>
                  {roleLabelMap[role] || role}
                </div>
                <Tag color="blue">{role}</Tag>
                <UserSelect
                  value={assignments[role] || undefined}
                  onChange={(val) => updateAssignment(role, val as string)}
                  mode="single"
                  placeholder="选择负责人"
                  style={{ flex: 1 }}
                />
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
};

// ============ SKU Management Tab ============

const SKUTab: React.FC<{ projectId: string }> = ({ projectId }) => {
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedSKU, setSelectedSKU] = useState<ProductSKU | null>(null);
  const [subTab, setSubTab] = useState<'cmf' | 'overrides'>('cmf');
  const [form] = Form.useForm();

  // List SKUs
  const { data: skus = [], isLoading } = useQuery<ProductSKU[]>({
    queryKey: ['project-skus', projectId],
    queryFn: () => skuApi.listSKUs(projectId),
  });

  // Get SBOM items for CMF config
  const { data: bomItems = [] } = useQuery({
    queryKey: ['project-sbom-items', projectId],
    queryFn: async () => {
      const boms = await projectBomApi.list(projectId, { bom_type: 'SBOM' });
      if (boms.length === 0) return [];
      const detail = await projectBomApi.get(projectId, boms[0].id);
      return detail.items || [];
    },
    enabled: !!selectedSKU,
  });

  // CMF configs for selected SKU
  const { data: cmfConfigs = [] } = useQuery<SKUCMFConfig[]>({
    queryKey: ['sku-cmf', projectId, selectedSKU?.id],
    queryFn: () => skuApi.getCMFConfigs(projectId, selectedSKU!.id),
    enabled: !!selectedSKU,
  });

  // BOM overrides for selected SKU
  const { data: bomOverrides = [] } = useQuery<SKUBOMOverride[]>({
    queryKey: ['sku-overrides', projectId, selectedSKU?.id],
    queryFn: () => skuApi.getBOMOverrides(projectId, selectedSKU!.id),
    enabled: !!selectedSKU,
  });

  // Create SKU mutation
  const createMutation = useMutation({
    mutationFn: (data: { name: string; code?: string; description?: string }) =>
      skuApi.createSKU(projectId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-skus', projectId] });
      setCreateOpen(false);
      form.resetFields();
      message.success('SKU创建成功');
    },
    onError: () => message.error('创建失败'),
  });

  // Delete SKU mutation
  const deleteMutation = useMutation({
    mutationFn: (skuId: string) => skuApi.deleteSKU(projectId, skuId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['project-skus', projectId] });
      if (selectedSKU) setSelectedSKU(null);
      message.success('已删除');
    },
  });

  // Save CMF configs
  const saveCMFMutation = useMutation({
    mutationFn: (configs: Array<{ bom_item_id: string; color: string; color_code: string; surface_treatment: string }>) =>
      skuApi.saveCMFConfigs(projectId, selectedSKU!.id, configs),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sku-cmf', projectId, selectedSKU?.id] });
      message.success('CMF配置已保存');
    },
    onError: () => message.error('保存失败'),
  });

  // Add BOM override
  const addOverrideMutation = useMutation({
    mutationFn: (data: { action: string; base_item_id?: string; override_name?: string; override_specification?: string; override_quantity?: number; override_unit?: string; notes?: string }) =>
      skuApi.createBOMOverride(projectId, selectedSKU!.id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sku-overrides', projectId, selectedSKU?.id] });
      message.success('已添加');
    },
  });

  // Delete BOM override
  const deleteOverrideMutation = useMutation({
    mutationFn: (overrideId: string) =>
      skuApi.deleteBOMOverride(projectId, selectedSKU!.id, overrideId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sku-overrides', projectId, selectedSKU?.id] });
      message.success('已删除');
    },
  });

  // CMF editor state
  const [cmfEdits, setCmfEdits] = useState<Record<string, { color: string; color_code: string; surface_treatment: string }>>({});

  // Initialize CMF edits when configs load
  React.useEffect(() => {
    if (cmfConfigs.length > 0) {
      const map: Record<string, { color: string; color_code: string; surface_treatment: string }> = {};
      for (const c of cmfConfigs) {
        map[c.bom_item_id] = { color: c.color, color_code: c.color_code, surface_treatment: c.surface_treatment };
      }
      setCmfEdits(map);
    }
  }, [cmfConfigs]);

  const handleCMFChange = (bomItemId: string, field: string, value: string) => {
    setCmfEdits(prev => ({
      ...prev,
      [bomItemId]: { ...prev[bomItemId] || { color: '', color_code: '', surface_treatment: '' }, [field]: value },
    }));
  };

  const handleSaveCMF = () => {
    const configs = Object.entries(cmfEdits)
      .filter(([_, v]) => v.color || v.color_code || v.surface_treatment)
      .map(([bomItemId, v]) => ({
        bom_item_id: bomItemId,
        color: v.color || '',
        color_code: v.color_code || '',
        surface_treatment: v.surface_treatment || '',
      }));
    saveCMFMutation.mutate(configs);
  };

  // Override form
  const [overrideForm] = Form.useForm();
  const [overrideModalOpen, setOverrideModalOpen] = useState(false);

  // SKU list view
  if (!selectedSKU) {
    return (
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
          <Text strong style={{ fontSize: 15 }}>配色方案 / SKU</Text>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新建SKU
          </Button>
        </div>

        {isLoading ? <Spin /> : skus.length === 0 ? (
          <Empty description={'暂无SKU，点击"新建SKU"开始'} />
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 12 }}>
            {skus.map(sku => (
              <Card
                key={sku.id}
                size="small"
                hoverable
                onClick={() => setSelectedSKU(sku)}
                styles={{ body: { padding: '12px 16px' } }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <Text strong>{sku.name}</Text>
                    {sku.code && <Tag style={{ marginLeft: 8 }}>{sku.code}</Tag>}
                  </div>
                  <Tag color={sku.status === 'active' ? 'green' : 'default'}>{sku.status === 'active' ? '启用' : '停用'}</Tag>
                </div>
                {sku.description && <Text type="secondary" style={{ fontSize: 12, marginTop: 4, display: 'block' }}>{sku.description}</Text>}
                <div style={{ marginTop: 8, display: 'flex', justifyContent: 'flex-end' }}>
                  <Popconfirm title="确认删除此SKU？" onConfirm={(e) => { e?.stopPropagation(); deleteMutation.mutate(sku.id); }}>
                    <Button size="small" type="text" danger icon={<DeleteOutlined />} onClick={(e) => e.stopPropagation()} />
                  </Popconfirm>
                </div>
              </Card>
            ))}
          </div>
        )}

        {/* Create SKU Modal */}
        <Modal
          title="新建SKU"
          open={createOpen}
          onCancel={() => { setCreateOpen(false); form.resetFields(); }}
          onOk={() => form.submit()}
          confirmLoading={createMutation.isPending}
        >
          <Form form={form} layout="vertical" onFinish={(values) => createMutation.mutate(values)}>
            <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入SKU名称' }]}>
              <Input placeholder="如：星空黑、冰川白" />
            </Form.Item>
            <Form.Item name="code" label="编码">
              <Input placeholder="SKU编码（可选）" />
            </Form.Item>
            <Form.Item name="description" label="描述">
              <Input.TextArea rows={2} placeholder="描述（可选）" />
            </Form.Item>
          </Form>
        </Modal>
      </div>
    );
  }

  // SKU detail view
  const cmfColumns = [
    { title: '序号', dataIndex: 'item_number', width: 60, align: 'center' as const },
    { title: '零件名称', dataIndex: 'name', width: 140 },
    { title: '规格', dataIndex: 'specification', width: 150, ellipsis: true },
    { title: '材质', dataIndex: 'material_type', width: 100 },
    { title: '颜色', width: 130,
      render: (_: any, record: ProjectBOMItem) => (
        <Input size="small" value={cmfEdits[record.id]?.color || ''} placeholder="颜色"
          onChange={(e) => handleCMFChange(record.id, 'color', e.target.value)} />
      ),
    },
    { title: '色号', width: 110,
      render: (_: any, record: ProjectBOMItem) => (
        <Input size="small" value={cmfEdits[record.id]?.color_code || ''} placeholder="Pantone等"
          onChange={(e) => handleCMFChange(record.id, 'color_code', e.target.value)} />
      ),
    },
    { title: '表面处理', width: 140,
      render: (_: any, record: ProjectBOMItem) => (
        <Input size="small" value={cmfEdits[record.id]?.surface_treatment || ''} placeholder="阳极氧化等"
          onChange={(e) => handleCMFChange(record.id, 'surface_treatment', e.target.value)} />
      ),
    },
  ];

  const overrideColumns = [
    { title: '操作类型', dataIndex: 'action', width: 90,
      render: (v: string) => (
        <Tag color={v === 'replace' ? 'orange' : v === 'add' ? 'green' : 'red'}>
          {v === 'replace' ? '替换' : v === 'add' ? '新增' : '移除'}
        </Tag>
      ),
    },
    { title: '基础件', dataIndex: ['base_item', 'name'], width: 140, render: (v: string) => v || '-' },
    { title: '替代件名称', dataIndex: 'override_name', width: 140, render: (v: string) => v || '-' },
    { title: '规格', dataIndex: 'override_specification', width: 150, ellipsis: true },
    { title: '数量', dataIndex: 'override_quantity', width: 70, align: 'right' as const },
    { title: '备注', dataIndex: 'notes', width: 120, ellipsis: true },
    { title: '操作', width: 60, align: 'center' as const,
      render: (_: any, record: SKUBOMOverride) => (
        <Popconfirm title="确认删除？" onConfirm={() => deleteOverrideMutation.mutate(record.id)}>
          <Button size="small" type="text" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <Button size="small" onClick={() => setSelectedSKU(null)}>&lt; 返回</Button>
        <Text strong style={{ fontSize: 15 }}>{selectedSKU.name}</Text>
        {selectedSKU.code && <Tag>{selectedSKU.code}</Tag>}
        <Tag color={selectedSKU.status === 'active' ? 'green' : 'default'}>{selectedSKU.status === 'active' ? '启用' : '停用'}</Tag>
      </div>

      <Tabs activeKey={subTab} onChange={(k) => setSubTab(k as 'cmf' | 'overrides')} items={[
        {
          key: 'cmf',
          label: 'CMF配色',
          children: (
            <div>
              {bomItems.length === 0 ? (
                <Empty description="该项目暂无结构BOM，请先创建SBOM" />
              ) : (
                <>
                  <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
                    <Button type="primary" size="small" onClick={handleSaveCMF} loading={saveCMFMutation.isPending}>
                      保存CMF配置
                    </Button>
                  </div>
                  <Table
                    columns={cmfColumns}
                    dataSource={bomItems}
                    rowKey="id"
                    size="small"
                    pagination={false}
                    scroll={{ x: 900 }}
                  />
                </>
              )}
            </div>
          ),
        },
        {
          key: 'overrides',
          label: 'BOM差异',
          children: (
            <div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
                <Button type="primary" size="small" icon={<PlusOutlined />} onClick={() => setOverrideModalOpen(true)}>
                  添加差异
                </Button>
              </div>
              <Table
                columns={overrideColumns}
                dataSource={bomOverrides}
                rowKey="id"
                size="small"
                pagination={false}
                scroll={{ x: 800 }}
                locale={{ emptyText: <Empty description="无BOM差异，与基础BOM相同" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
              />

              {/* Add Override Modal */}
              <Modal
                title="添加BOM差异"
                open={overrideModalOpen}
                onCancel={() => { setOverrideModalOpen(false); overrideForm.resetFields(); }}
                onOk={() => overrideForm.submit()}
                confirmLoading={addOverrideMutation.isPending}
              >
                <Form form={overrideForm} layout="vertical" onFinish={(values) => {
                  addOverrideMutation.mutate(values);
                  setOverrideModalOpen(false);
                  overrideForm.resetFields();
                }}>
                  <Form.Item name="action" label="操作类型" rules={[{ required: true }]}>
                    <Select options={[
                      { value: 'replace', label: '替换 - 用其他件替换基础件' },
                      { value: 'add', label: '新增 - 该SKU额外需要的件' },
                      { value: 'remove', label: '移除 - 该SKU不需要的基础件' },
                    ]} />
                  </Form.Item>
                  <Form.Item name="base_item_id" label="基础件（替换/移除时选择）">
                    <Select allowClear placeholder="选择基础BOM中的零件" options={
                      bomItems.map((item: ProjectBOMItem) => ({ value: item.id, label: `${item.item_number}. ${item.name}` }))
                    } />
                  </Form.Item>
                  <Form.Item name="override_name" label="替代件/新增件名称">
                    <Input placeholder="零件名称" />
                  </Form.Item>
                  <Form.Item name="override_specification" label="规格描述">
                    <Input placeholder="规格" />
                  </Form.Item>
                  <Form.Item name="override_quantity" label="数量" initialValue={1}>
                    <InputNumber min={0} style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="notes" label="备注">
                    <Input.TextArea rows={2} />
                  </Form.Item>
                </Form>
              </Modal>
            </div>
          ),
        },
      ]} />
    </div>
  );
};

// ============ Main ProjectDetail Page ============

const ProjectDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: project, isLoading } = useQuery({
    queryKey: ['project', id],
    queryFn: () => projectApi.get(id!),
    enabled: !!id,
  });

  const { data: tasks, isLoading: tasksLoading } = useQuery({
    queryKey: ['project-tasks', id],
    queryFn: () => projectApi.listTasks(id!),
    enabled: !!id,
  });

  const completeTaskMutation = useMutation({
    mutationFn: ({ projectId, taskId }: { projectId: string; taskId: string }) =>
      projectApi.completeTask(projectId, taskId),
    onSuccess: () => {
      message.success('任务已完成');
      queryClient.invalidateQueries({ queryKey: ['project-tasks', id] });
    },
    onError: () => message.error('操作失败'),
  });

  const refreshTasks = () => {
    queryClient.invalidateQueries({ queryKey: ['project-tasks', id] });
    queryClient.invalidateQueries({ queryKey: ['project', id] });
  };

  // SSE: 实时推送自动刷新
  useSSE({
    onTaskUpdate: useCallback((event: SSETaskEvent) => {
      if (event.project_id === id) {
        queryClient.invalidateQueries({ queryKey: ['project-tasks', id] });
        queryClient.invalidateQueries({ queryKey: ['project', id] });
      }
    }, [id, queryClient]),
    onProjectUpdate: useCallback((event: SSETaskEvent) => {
      if (event.project_id === id) {
        queryClient.invalidateQueries({ queryKey: ['project-tasks', id] });
        queryClient.invalidateQueries({ queryKey: ['project', id] });
      }
    }, [id, queryClient]),
    enabled: !!id,
  });

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  if (!project) {
    return (
      <div style={{ padding: 24 }}>
        <Empty description="项目不存在" />
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Button onClick={() => navigate('/projects')}>返回项目列表</Button>
        </div>
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      {/* Header */}
      <div style={{ marginBottom: 24 }}>
        <Button type="link" icon={<ArrowLeftOutlined />} onClick={() => navigate('/projects')} style={{ padding: 0, marginBottom: 8 }}>
          返回项目列表
        </Button>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <Title level={3} style={{ margin: 0 }}>
              {project.name}
              {project.code && <Text code style={{ marginLeft: 12, fontSize: 14 }}>{project.code}</Text>}
            </Title>
            <div style={{ marginTop: 8 }}>
              <PhaseProgressBar currentPhase={project.phase} />
            </div>
          </div>
          <Space>
            <Badge status={statusColors[project.status] as any} text={
              project.status === 'planning' ? '规划中' :
              project.status === 'active' ? '进行中' :
              project.status === 'completed' ? '已完成' :
              project.status === 'on_hold' ? '暂停' : project.status
            } />
            <Progress type="circle" percent={project.progress} size={48} />
          </Space>
        </div>
      </div>

      {/* Tabs */}
      <Card>
        <Tabs
          defaultActiveKey="overview"
          items={[
            {
              key: 'overview',
              label: '概览',
              children: <OverviewTab project={project} />,
            },
            {
              key: 'gantt',
              label: `甘特图 (${tasks?.length || 0})`,
              children: tasksLoading ? (
                <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
              ) : tasks && tasks.length > 0 ? (
                <div style={{ height: 560 }}>
                  <GanttChart
                    tasks={tasks}
                    projectId={project.id}
                    onCompleteTask={(taskId) =>
                      completeTaskMutation.mutate({ projectId: project.id, taskId })
                    }
                    completingTask={completeTaskMutation.isPending}
                    onRefresh={refreshTasks}
                  />
                </div>
              ) : (
                <Empty description="暂无任务" />
              ),
            },
            {
              key: 'bom',
              label: 'BOM管理',
              children: <BOMTab projectId={project.id} />,
            },
            {
              key: 'sku',
              label: 'SKU配色',
              children: <SKUTab projectId={project.id} />,
            },
            {
              key: 'cmf',
              label: 'CMF配色',
              children: <CMFPanel projectId={project.id} />,
            },
            {
              key: 'documents',
              label: '图纸文档',
              children: <DocumentsTab projectId={project.id} />,
            },
            {
              key: 'deliverables',
              label: '交付物',
              children: <DeliverablesTab projectId={project.id} currentPhase={project.phase} />,
            },
            {
              key: 'ecn',
              label: 'ECN',
              children: <ECNTab projectId={project.id} productId={project.product_id} />,
            },
            {
              key: 'roles',
              label: '角色指派',
              children: <RoleAssignmentTab projectId={project.id} />,
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default ProjectDetail;
