import React from 'react';
import { Card, Row, Col, Statistic, Typography, List, Tag, Progress, Badge, Space, Button, Empty } from 'antd';
import {
  ProjectOutlined,
  AuditOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  BellOutlined,
  RightOutlined,
  ExperimentOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/contexts/AuthContext';
import { useSSE } from '@/hooks/useSSE';
import { projectApi, Project } from '@/api/projects';
import dayjs from 'dayjs';

const { Title, Text } = Typography;

const phaseColors: Record<string, string> = {
  concept: 'purple',
  evt: 'blue',
  dvt: 'cyan',
  pvt: 'orange',
  mp: 'green',
};

// Mock 待办数据（后端开发中，先用mock）
const mockTodoItems = [
  {
    id: '1',
    type: 'bom_review',
    title: 'EVT 工程BOM v1.1 待审批',
    project: 'nimo Air 2',
    priority: 'high',
    dueDate: '2026-02-08',
    icon: <AuditOutlined style={{ color: '#faad14' }} />,
  },
  {
    id: '2',
    type: 'task_due',
    title: '天线设计评审 即将到期',
    project: 'nimo Air 2',
    priority: 'medium',
    dueDate: '2026-02-10',
    icon: <ClockCircleOutlined style={{ color: '#1890ff' }} />,
  },
  {
    id: '3',
    type: 'gate_review',
    title: 'G1 阶段门评审通知',
    project: 'nimo Pro',
    priority: 'high',
    dueDate: '2026-02-12',
    icon: <BellOutlined style={{ color: '#ff4d4f' }} />,
  },
  {
    id: '4',
    type: 'task_due',
    title: 'BOM 成本核算 即将到期',
    project: 'nimo Lite',
    priority: 'low',
    dueDate: '2026-02-15',
    icon: <ClockCircleOutlined style={{ color: '#1890ff' }} />,
  },
];

const priorityColors: Record<string, string> = {
  low: 'default',
  medium: 'blue',
  high: 'orange',
  urgent: 'red',
};

const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // SSE: 实时推送自动刷新
  useSSE({
    onTaskUpdate: React.useCallback(() => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    }, [queryClient]),
    onProjectUpdate: React.useCallback(() => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
    }, [queryClient]),
  });

  // Fetch real project data
  const { data: projectsData, isLoading: projectsLoading } = useQuery({
    queryKey: ['projects'],
    queryFn: () => projectApi.list({ page_size: 50 }),
  });

  const projects = projectsData?.items || [];
  const activeProjects = projects.filter(p => p.status === 'active');
  const completedProjects = projects.filter(p => p.status === 'completed');

  // Compute stats from real data
  const stats = {
    totalProjects: projects.length,
    activeProjects: activeProjects.length,
    completedProjects: completedProjects.length,
    pendingTodos: mockTodoItems.length,
  };

  return (
    <div style={{ padding: 24 }}>
      {/* Welcome */}
      <div style={{ marginBottom: 24 }}>
        <Title level={3} style={{ margin: 0 }}>
          🏠 工作台
        </Title>
        <Text type="secondary">
          👋 欢迎回来，{user?.name}！今天是 {dayjs().format('YYYY年M月D日 dddd')}
        </Text>
      </div>

      {/* Stats cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable onClick={() => navigate('/projects')}>
            <Statistic
              title="进行中项目"
              value={stats.activeProjects}
              prefix={<ProjectOutlined style={{ color: '#1890ff' }} />}
              suffix={<Text type="secondary" style={{ fontSize: 14 }}>/ {stats.totalProjects}</Text>}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="我的待办"
              value={stats.pendingTodos}
              prefix={<AuditOutlined style={{ color: '#faad14' }} />}
              valueStyle={{ color: stats.pendingTodos > 0 ? '#faad14' : undefined }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable>
            <Statistic
              title="已完成项目"
              value={stats.completedProjects}
              prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable onClick={() => navigate('/materials')}>
            <Statistic
              title="物料选型库"
              value={'查看'}
              prefix={<ExperimentOutlined style={{ color: '#722ed1' }} />}
              valueStyle={{ fontSize: 16 }}
            />
          </Card>
        </Col>
      </Row>

      {/* Main content */}
      <Row gutter={[16, 16]}>
        {/* 我的待办 */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <AuditOutlined />
                <span>我的待办</span>
                <Badge count={mockTodoItems.length} style={{ backgroundColor: '#faad14' }} />
              </Space>
            }
            extra={<Button type="link" size="small">查看全部</Button>}
          >
            <List
              dataSource={mockTodoItems}
              renderItem={(item) => (
                <List.Item
                  style={{ cursor: 'pointer', padding: '10px 0' }}
                  actions={[
                    <Tag color={priorityColors[item.priority]} key="priority">
                      {item.priority === 'high' ? '高' : item.priority === 'medium' ? '中' : '低'}
                    </Tag>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={item.icon}
                    title={item.title}
                    description={
                      <Space size={8}>
                        <Tag style={{ fontSize: 11 }}>{item.project}</Tag>
                        <Text type="secondary" style={{ fontSize: 12 }}>截止: {item.dueDate}</Text>
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          </Card>
        </Col>

        {/* 我参与的项目 */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <ProjectOutlined />
                <span>我参与的项目</span>
              </Space>
            }
            extra={
              <Button type="link" size="small" onClick={() => navigate('/projects')}>
                查看全部 <RightOutlined />
              </Button>
            }
          >
            {projectsLoading ? (
              <div style={{ textAlign: 'center', padding: 40 }}>加载中...</div>
            ) : projects.length === 0 ? (
              <Empty description="暂无项目" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            ) : (
              <List
                dataSource={projects.slice(0, 6)}
                renderItem={(project: Project) => (
                  <List.Item
                    style={{ cursor: 'pointer', padding: '10px 0' }}
                    onClick={() => navigate(`/projects/${project.id}`)}
                    actions={[<RightOutlined key="go" style={{ color: '#ccc' }} />]}
                  >
                    <div style={{ width: '100%', paddingRight: 8 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                        <Space>
                          <Text strong>{project.name}</Text>
                          <Text code style={{ fontSize: 11 }}>{project.code}</Text>
                        </Space>
                        <Tag color={phaseColors[project.phase?.toLowerCase()] || 'default'}>
                          {project.phase?.toUpperCase()}
                        </Tag>
                      </div>
                      <Progress
                        percent={project.progress}
                        size="small"
                        strokeColor={
                          project.progress >= 80 ? '#52c41a' :
                          project.progress >= 50 ? '#1890ff' :
                          project.progress >= 20 ? '#faad14' : '#ff4d4f'
                        }
                      />
                    </div>
                  </List.Item>
                )}
              />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
