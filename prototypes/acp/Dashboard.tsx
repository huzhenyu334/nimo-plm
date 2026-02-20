/**
 * ACP — Dashboard（Agent 监控面板）
 * 路由: /
 *
 * 设计要点:
 * - 顶部全局统计栏 (Statistic × 3)
 * - Agent 卡片网格 (Row + Col, 响应式)
 * - 每张卡片: 名称/角色/模型/状态/当前操作/token/最后活跃
 * - 在线排前，离线灰色排后
 * - 10s 自动刷新
 */

import React, { useEffect, useState } from 'react';
import {
  Layout, Row, Col, Card, Statistic, Tag, Typography, Badge, Space,
  Skeleton, Empty, Alert, Tooltip, Progress, theme,
} from 'antd';
import {
  RobotOutlined, ThunderboltOutlined, ClockCircleOutlined,
  CheckCircleOutlined, SyncOutlined, ExclamationCircleOutlined,
  ApiOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { semanticColors, polling } from '@/tokens';

const { Header, Content } = Layout;
const { Text, Title } = Typography;

/* ───────── Types ───────── */

type AgentStatus = 'online' | 'offline' | 'error' | 'working';

interface AgentCard {
  id: string;
  name: string;
  role: string;
  model: string;
  status: AgentStatus;
  currentAction?: string;   // 最近一条 tool call 摘要
  tokenUsed: number;
  tokenLimit?: number;
  lastActiveAt: string;      // ISO-8601
}

interface GlobalStats {
  onlineCount: number;
  todayTasksDone: number;
  activeWorkflows: number;
}

/* ───────── Status helpers ───────── */

const statusMeta: Record<AgentStatus, { color: string; label: string; icon: React.ReactNode }> = {
  online:  { color: semanticColors.status.online,  label: '在线',  icon: <CheckCircleOutlined /> },
  working: { color: semanticColors.status.working,  label: '工作中', icon: <SyncOutlined spin /> },
  error:   { color: semanticColors.status.error,    label: '异常',  icon: <ExclamationCircleOutlined /> },
  offline: { color: semanticColors.status.offline,   label: '离线',  icon: <ClockCircleOutlined /> },
};

const roleTags: Record<string, { color: string; label: string }> = {
  coo:       { color: semanticColors.role.coo,       label: 'COO' },
  pm:        { color: semanticColors.role.pm,        label: 'PM' },
  ux:        { color: semanticColors.role.ux,        label: 'UX' },
  dev:       { color: semanticColors.role.dev,       label: 'Dev' },
  assistant: { color: semanticColors.role.assistant, label: 'Asst' },
};

/* ───────── Relative time ───────── */

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60_000) return '刚刚';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}分钟前`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}小时前`;
  return `${Math.floor(diff / 86_400_000)}天前`;
}

/* ───────── Components ───────── */

/** 全局统计栏 */
const StatsBar: React.FC<{ stats: GlobalStats; loading: boolean }> = ({ stats, loading }) => (
  <Row gutter={16} style={{ marginBottom: 24 }}>
    {[
      { title: '在线 Agent', value: stats.onlineCount, icon: <RobotOutlined />, color: semanticColors.status.online },
      { title: '今日任务完成', value: stats.todayTasksDone, icon: <CheckCircleOutlined />, color: semanticColors.status.working },
      { title: '活跃工作流', value: stats.activeWorkflows, icon: <ThunderboltOutlined />, color: '#faad14' },
    ].map((item) => (
      <Col key={item.title} xs={24} sm={8}>
        <Card
          bordered={false}
          style={{ borderLeft: `3px solid ${item.color}` }}
        >
          <Skeleton loading={loading} paragraph={false} active>
            <Statistic
              title={
                <Space>
                  {item.icon}
                  <span>{item.title}</span>
                </Space>
              }
              value={item.value}
              valueStyle={{ fontSize: 28, fontWeight: 600 }}
            />
          </Skeleton>
        </Card>
      </Col>
    ))}
  </Row>
);

/** 单个 Agent 卡片 */
const AgentCardItem: React.FC<{ agent: AgentCard }> = ({ agent }) => {
  const navigate = useNavigate();
  const meta = statusMeta[agent.status];
  const roleTag = roleTags[agent.role] ?? { color: '#888', label: agent.role };
  const isInactive = agent.status === 'offline';

  return (
    <Card
      hoverable
      bordered={false}
      onClick={() => navigate(`/agents/${agent.id}`)}
      style={{
        opacity: isInactive ? 0.55 : 1,
        cursor: 'pointer',
        height: '100%',
      }}
      styles={{
        body: { padding: 20, display: 'flex', flexDirection: 'column', gap: 12 },
      }}
    >
      {/* Row 1: 名称 + 状态 badge */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space size={8}>
          <RobotOutlined style={{ fontSize: 18, color: meta.color }} />
          <Text strong style={{ fontSize: 16 }}>{agent.name}</Text>
          <Tag color={roleTag.color} style={{ marginLeft: 4 }}>{roleTag.label}</Tag>
        </Space>
        <Badge
          status={agent.status === 'working' ? 'processing' : agent.status === 'online' ? 'success' : agent.status === 'error' ? 'error' : 'default'}
          text={<Text style={{ color: meta.color, fontSize: 12 }}>{meta.label}</Text>}
        />
      </div>

      {/* Row 2: 模型 */}
      <Text type="secondary" style={{ fontSize: 12 }}>
        <ApiOutlined style={{ marginRight: 4 }} />
        {agent.model}
      </Text>

      {/* Row 3: 当前操作 */}
      <div
        style={{
          background: 'rgba(255,255,255,0.04)',
          borderRadius: 6,
          padding: '8px 12px',
          minHeight: 36,
        }}
      >
        {agent.currentAction ? (
          <Text ellipsis style={{ fontSize: 13 }}>
            <ThunderboltOutlined style={{ marginRight: 4, color: semanticColors.status.working }} />
            {agent.currentAction}
          </Text>
        ) : (
          <Text type="secondary" style={{ fontSize: 13 }}>暂无操作</Text>
        )}
      </div>

      {/* Row 4: Token + 最后活跃 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 'auto' }}>
        <Tooltip title={`${agent.tokenUsed.toLocaleString()} tokens`}>
          <Space size={4}>
            <Text type="secondary" style={{ fontSize: 12 }}>Token</Text>
            <Progress
              percent={agent.tokenLimit ? Math.min((agent.tokenUsed / agent.tokenLimit) * 100, 100) : 0}
              size="small"
              showInfo={false}
              strokeColor={semanticColors.status.working}
              style={{ width: 80, margin: 0 }}
            />
            <Text style={{ fontSize: 12 }}>{(agent.tokenUsed / 1000).toFixed(1)}k</Text>
          </Space>
        </Tooltip>
        <Text type="tertiary" style={{ fontSize: 12 }}>
          <ClockCircleOutlined style={{ marginRight: 4 }} />
          {relativeTime(agent.lastActiveAt)}
        </Text>
      </div>
    </Card>
  );
};

/* ───────── Page ───────── */

const Dashboard: React.FC = () => {
  const [agents, setAgents] = useState<AgentCard[]>([]);
  const [stats, setStats] = useState<GlobalStats>({ onlineCount: 0, todayTasksDone: 0, activeWorkflows: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = async () => {
    try {
      const res = await fetch('/api/agents');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      // 在线排前，离线排后
      const sorted = [...data.agents].sort((a: AgentCard, b: AgentCard) => {
        const order: Record<AgentStatus, number> = { working: 0, online: 1, error: 2, offline: 3 };
        return order[a.status] - order[b.status];
      });
      setAgents(sorted);
      setStats(data.stats);
      setError(null);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const timer = setInterval(fetchData, polling.dashboardMs);
    return () => clearInterval(timer);
  }, []);

  return (
    <Content style={{ padding: 24, maxWidth: 1440, margin: '0 auto' }}>
      <Title level={4} style={{ marginBottom: 24 }}>Agent 监控面板</Title>

      {/* 错误提示 */}
      {error && (
        <Alert
          type="error"
          showIcon
          message="数据加载失败"
          description={error}
          closable
          style={{ marginBottom: 16 }}
        />
      )}

      {/* 全局统计 */}
      <StatsBar stats={stats} loading={loading} />

      {/* Agent 卡片网格 */}
      {loading ? (
        <Row gutter={[16, 16]}>
          {[1, 2, 3, 4].map((i) => (
            <Col key={i} xs={24} sm={12} lg={8} xl={6}>
              <Card bordered={false}>
                <Skeleton active paragraph={{ rows: 4 }} />
              </Card>
            </Col>
          ))}
        </Row>
      ) : agents.length === 0 ? (
        <Card bordered={false}>
          <Empty description="暂无 Agent" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {agents.map((agent) => (
            <Col key={agent.id} xs={24} sm={12} lg={8} xl={6}>
              <AgentCardItem agent={agent} />
            </Col>
          ))}
        </Row>
      )}
    </Content>
  );
};

export default Dashboard;
