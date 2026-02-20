/**
 * ACP — Agent 详情页
 * 路由: /agents/:id
 *
 * 布局: 左60%对话历史 + 右40%状态侧边栏
 * 5s 轮询增量刷新
 */

import React, { useEffect, useRef, useState } from 'react';
import {
  Layout, Card, Typography, Space, Tag, Badge, Statistic, Descriptions,
  List, Input, Button, Collapse, Skeleton, Empty, Alert, Divider,
  Tooltip, Spin, message as antMessage, Row, Col, Timeline,
} from 'antd';
import {
  SendOutlined, RobotOutlined, UserOutlined, ToolOutlined,
  ClockCircleOutlined, ApiOutlined, CodeOutlined, ReloadOutlined,
  ArrowLeftOutlined, FieldTimeOutlined,
} from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { semanticColors, polling, layout } from '@/tokens';

const { Content } = Layout;
const { Text, Title, Paragraph } = Typography;
const { TextArea } = Input;

/* ───────── Types ───────── */

type MessageRole = 'user' | 'assistant' | 'system';

interface ToolCall {
  id: string;
  name: string;
  paramsSummary: string;  // ≤200 char 摘要
  paramsFull: string;     // 完整参数 JSON
  result?: string;
}

interface ChatMessage {
  id: string;
  role: MessageRole;
  content: string;
  toolCalls?: ToolCall[];
  timestamp: string;
}

interface AgentInfo {
  id: string;
  name: string;
  role: string;
  model: string;
  status: 'online' | 'offline' | 'error' | 'working';
  sessionKey: string;
}

interface SessionStats {
  tokenUsed: number;
  tokenLimit?: number;
  messageCount: number;
  uptimeMinutes: number;
}

interface CronJob {
  id: string;
  name: string;
  schedule: string;
  enabled: boolean;
  lastRun?: string;
}

/* ───────── Status color ───────── */

const statusColor: Record<string, string> = {
  online:  semanticColors.status.online,
  working: semanticColors.status.working,
  error:   semanticColors.status.error,
  offline: semanticColors.status.offline,
};

/* ───────── Chat Bubble ───────── */

const ChatBubble: React.FC<{ msg: ChatMessage }> = ({ msg }) => {
  const isAssistant = msg.role === 'assistant';
  const isSystem = msg.role === 'system';

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: isAssistant ? 'flex-start' : 'flex-end',
        marginBottom: 16,
      }}
    >
      {/* 角色标签 + 时间 */}
      <Space size={8} style={{ marginBottom: 4 }}>
        {isAssistant ? (
          <Tag icon={<RobotOutlined />} color="blue">Agent</Tag>
        ) : isSystem ? (
          <Tag icon={<ApiOutlined />} color="orange">System</Tag>
        ) : (
          <Tag icon={<UserOutlined />} color="green">User</Tag>
        )}
        <Text type="tertiary" style={{ fontSize: 12 }}>
          {new Date(msg.timestamp).toLocaleTimeString('zh-CN', { hour12: false })}
        </Text>
      </Space>

      {/* 消息内容 */}
      <div
        style={{
          maxWidth: '85%',
          background: isAssistant ? '#1f1f1f' : isSystem ? 'rgba(250,173,20,0.08)' : 'rgba(22,119,255,0.12)',
          borderRadius: 8,
          padding: '10px 14px',
          border: `1px solid ${isAssistant ? '#303030' : isSystem ? 'rgba(250,173,20,0.2)' : 'rgba(22,119,255,0.2)'}`,
        }}
      >
        <Paragraph
          style={{ margin: 0, fontSize: 14, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}
          ellipsis={{ rows: 10, expandable: true, symbol: '展开' }}
        >
          {msg.content}
        </Paragraph>
      </div>

      {/* Tool Calls（仅 assistant） */}
      {msg.toolCalls && msg.toolCalls.length > 0 && (
        <div style={{ maxWidth: '85%', marginTop: 8 }}>
          <Collapse
            size="small"
            ghost
            items={msg.toolCalls.map((tc) => ({
              key: tc.id,
              label: (
                <Space>
                  <ToolOutlined style={{ color: semanticColors.status.working }} />
                  <Text code style={{ fontSize: 12 }}>{tc.name}</Text>
                  <Text type="secondary" style={{ fontSize: 12 }} ellipsis>
                    {tc.paramsSummary}
                  </Text>
                </Space>
              ),
              children: (
                <div style={{ fontSize: 12 }}>
                  <Text type="secondary">参数:</Text>
                  <pre style={{
                    background: '#141414',
                    padding: 8,
                    borderRadius: 4,
                    overflow: 'auto',
                    maxHeight: 200,
                    fontSize: 12,
                    margin: '4px 0',
                  }}>
                    {tc.paramsFull}
                  </pre>
                  {tc.result && (
                    <>
                      <Text type="secondary">结果:</Text>
                      <pre style={{
                        background: '#141414',
                        padding: 8,
                        borderRadius: 4,
                        overflow: 'auto',
                        maxHeight: 200,
                        fontSize: 12,
                        margin: '4px 0',
                      }}>
                        {tc.result.length > 500 ? tc.result.slice(0, 500) + '…' : tc.result}
                      </pre>
                    </>
                  )}
                </div>
              ),
            }))}
          />
        </div>
      )}
    </div>
  );
};

/* ───────── Sidebar ───────── */

const Sidebar: React.FC<{
  agent: AgentInfo | null;
  sessionStats: SessionStats | null;
  cronJobs: CronJob[];
  onSendMessage: (text: string) => void;
  sending: boolean;
  loading: boolean;
}> = ({ agent, sessionStats, cronJobs, onSendMessage, sending, loading }) => {
  const [inputVal, setInputVal] = useState('');

  const handleSend = () => {
    const text = inputVal.trim();
    if (!text) return;
    onSendMessage(text);
    setInputVal('');
  };

  if (loading) return <Skeleton active paragraph={{ rows: 12 }} />;
  if (!agent) return <Empty description="Agent 信息不可用" />;

  const sc = statusColor[agent.status] ?? '#888';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, height: '100%' }}>
      {/* 基本信息 */}
      <Card bordered={false} size="small" title="基本信息">
        <Descriptions column={1} size="small" colon={false}>
          <Descriptions.Item label="ID">
            <Text code>{agent.id}</Text>
          </Descriptions.Item>
          <Descriptions.Item label="模型">
            <Text>{agent.model}</Text>
          </Descriptions.Item>
          <Descriptions.Item label="角色">{agent.role}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <Badge color={sc} text={<Text style={{ color: sc }}>{agent.status}</Text>} />
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Session 统计 */}
      <Card bordered={false} size="small" title="Session 统计">
        {sessionStats ? (
          <Row gutter={[12, 12]}>
            <Col span={12}>
              <Statistic title="Token 用量" value={sessionStats.tokenUsed} suffix="tok" valueStyle={{ fontSize: 18 }} />
            </Col>
            <Col span={12}>
              <Statistic title="消息数" value={sessionStats.messageCount} valueStyle={{ fontSize: 18 }} />
            </Col>
            <Col span={24}>
              <Statistic
                title="运行时长"
                value={sessionStats.uptimeMinutes >= 60
                  ? `${Math.floor(sessionStats.uptimeMinutes / 60)}h ${sessionStats.uptimeMinutes % 60}m`
                  : `${sessionStats.uptimeMinutes}m`
                }
                valueStyle={{ fontSize: 18 }}
              />
            </Col>
          </Row>
        ) : (
          <Text type="secondary">暂无统计</Text>
        )}
      </Card>

      {/* Cron Jobs */}
      <Card bordered={false} size="small" title="定时任务" style={{ flex: 1, overflow: 'auto' }}>
        {cronJobs.length === 0 ? (
          <Text type="secondary">暂无定时任务</Text>
        ) : (
          <List
            size="small"
            dataSource={cronJobs}
            renderItem={(job) => (
              <List.Item style={{ padding: '6px 0' }}>
                <Space direction="vertical" size={0} style={{ width: '100%' }}>
                  <Space>
                    <FieldTimeOutlined />
                    <Text style={{ fontSize: 13 }}>{job.name}</Text>
                    <Tag color={job.enabled ? 'green' : 'default'} style={{ fontSize: 11 }}>
                      {job.enabled ? '启用' : '禁用'}
                    </Tag>
                  </Space>
                  <Text type="secondary" style={{ fontSize: 12 }}>{job.schedule}</Text>
                </Space>
              </List.Item>
            )}
          />
        )}
      </Card>

      {/* 发消息输入框 */}
      <Card bordered={false} size="small" title="发送消息">
        <Space.Compact style={{ width: '100%' }}>
          <TextArea
            value={inputVal}
            onChange={(e) => setInputVal(e.target.value)}
            placeholder="输入消息发送给 Agent..."
            autoSize={{ minRows: 2, maxRows: 4 }}
            onPressEnter={(e) => {
              if (!e.shiftKey) { e.preventDefault(); handleSend(); }
            }}
            style={{ borderRadius: '8px 8px 0 0' }}
          />
        </Space.Compact>
        <Button
          type="primary"
          icon={<SendOutlined />}
          loading={sending}
          onClick={handleSend}
          block
          style={{ marginTop: 8 }}
        >
          发送
        </Button>
      </Card>
    </div>
  );
};

/* ───────── Page ───────── */

const AgentDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [agent, setAgent] = useState<AgentInfo | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [sessionStats, setSessionStats] = useState<SessionStats | null>(null);
  const [cronJobs, setCronJobs] = useState<CronJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);

  const chatEndRef = useRef<HTMLDivElement>(null);

  /** 拉取数据 */
  const fetchData = async (initial = false) => {
    try {
      const [agentRes, historyRes, cronRes] = await Promise.all([
        fetch(`/api/agents/${id}`),
        fetch(`/api/agents/${id}/history?limit=50`),
        fetch(`/api/agents/${id}/cron-jobs`),
      ]);
      if (!agentRes.ok) throw new Error(`Agent API ${agentRes.status}`);
      const agentData = await agentRes.json();
      setAgent(agentData.agent);
      setSessionStats(agentData.stats);
      setMessages(await historyRes.json().then((d: any) => d.messages ?? []));
      setCronJobs(await cronRes.json().then((d: any) => d.jobs ?? []));
      setError(null);
    } catch (e: any) {
      setError(e.message);
    } finally {
      if (initial) setLoading(false);
    }
  };

  useEffect(() => {
    fetchData(true);
    const timer = setInterval(() => fetchData(false), polling.detailMs);
    return () => clearInterval(timer);
  }, [id]);

  /** 自动滚动 */
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  /** 发送消息 */
  const handleSend = async (text: string) => {
    setSending(true);
    try {
      const res = await fetch(`/api/agents/${id}/send`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: text }),
      });
      if (!res.ok) throw new Error(`发送失败: ${res.status}`);
      // 乐观插入
      setMessages((prev) => [
        ...prev,
        { id: `temp-${Date.now()}`, role: 'user', content: text, timestamp: new Date().toISOString() },
      ]);
      antMessage.success('已发送');
    } catch (e: any) {
      antMessage.error(e.message);
    } finally {
      setSending(false);
    }
  };

  return (
    <Content style={{ padding: 24, maxWidth: 1440, margin: '0 auto', height: 'calc(100vh - 56px)' }}>
      {/* 面包屑返回 */}
      <Space style={{ marginBottom: 16, cursor: 'pointer' }} onClick={() => navigate('/')}>
        <ArrowLeftOutlined />
        <Text>返回监控面板</Text>
      </Space>

      {error && (
        <Alert type="error" showIcon message="加载失败" description={error} closable style={{ marginBottom: 16 }} />
      )}

      <div style={{ display: 'flex', gap: 16, height: 'calc(100% - 48px)' }}>
        {/* ── 左侧: 对话历史 60% ── */}
        <Card
          bordered={false}
          title={
            <Space>
              <RobotOutlined />
              <span>{agent?.name ?? id}</span>
              {agent && <Badge color={statusColor[agent.status]} />}
              <Text type="secondary" style={{ fontSize: 12 }}>对话历史</Text>
            </Space>
          }
          extra={<Button size="small" icon={<ReloadOutlined />} onClick={() => fetchData()} />}
          style={{ flex: 6, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
          styles={{ body: { flex: 1, overflow: 'auto', padding: '16px 20px' } }}
        >
          {loading ? (
            <Skeleton active paragraph={{ rows: 8 }} />
          ) : messages.length === 0 ? (
            <Empty description="暂无对话记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          ) : (
            <>
              {messages.map((msg) => (
                <ChatBubble key={msg.id} msg={msg} />
              ))}
              <div ref={chatEndRef} />
            </>
          )}
        </Card>

        {/* ── 右侧: 状态侧边栏 40% ── */}
        <div style={{ flex: 4, overflow: 'auto' }}>
          <Sidebar
            agent={agent}
            sessionStats={sessionStats}
            cronJobs={cronJobs}
            onSendMessage={handleSend}
            sending={sending}
            loading={loading}
          />
        </div>
      </div>
    </Content>
  );
};

export default AgentDetail;
