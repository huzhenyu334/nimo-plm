import React from 'react';
import { Card, Row, Col, Statistic, Typography, List, Tag, Progress, Badge, Space, Button, Empty, Skeleton } from 'antd';
import {
  ProjectOutlined,
  AuditOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  RightOutlined,
  ExperimentOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/contexts/AuthContext';
import { useSSE } from '@/hooks/useSSE';
import { projectApi, Project } from '@/api/projects';
import { materialsApi } from '@/api/materials';
import apiClient from '@/api/client';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';

dayjs.locale('zh-cn');

const { Title, Text } = Typography;

const phaseColors: Record<string, string> = {
  concept: 'purple',
  evt: 'blue',
  dvt: 'cyan',
  pvt: 'orange',
  mp: 'green',
};

const priorityColors: Record<string, string> = {
  low: 'default',
  medium: 'blue',
  high: 'orange',
  urgent: 'red',
};

const taskTypeIcons: Record<string, React.ReactNode> = {
  gate_review: <AuditOutlined style={{ color: '#faad14' }} />,
  task: <ClockCircleOutlined style={{ color: '#1890ff' }} />,
};

const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // SSE: 实时推送自动刷新
  useSSE({
    onTaskUpdate: React.useCallback(() => {
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['my-tasks'] });
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

  // Fetch my tasks
  const { data: tasksData, isLoading: tasksLoading } = useQuery({
    queryKey: ['my-tasks'],
    queryFn: async () => {
      const res = await apiClient.get<{ data: { items: any[]; total: number } }>('/my/tasks', { params: { page_size: 10 } });
      return res.data.data;
    },
  });

  // Fetch materials count
  const { data: materialsData } = useQuery({
    queryKey: ['materials-count'],
    queryFn: () => materialsApi.list(),
  });

  const projects = projectsData?.items || [];
  const activeProjects = projects.filter(p => p.status === 'active');
  const completedProjects = projects.filter(p => p.status === 'completed');
  const todoItems = tasksData?.items || [];
  const materialsCount = materialsData?.materials?.length ?? '-';

  // Compute stats from real data
  const stats = {
    totalProjects: projects.length,
    activeProjects: activeProjects.length,
    completedProjects: completedProjects.length,
    pendingTodos: tasksData?.total ?? 0,
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
          <Card hoverable onClick={() => navigate('/my-tasks')}>
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
              value={materialsCount}
              prefix={<ExperimentOutlined style={{ color: '#722ed1' }} />}
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
                <Badge count={stats.pendingTodos} style={{ backgroundColor: '#faad14' }} />
              </Space>
            }
            extra={<Button type="link" size="small" onClick={() => navigate('/my-tasks')}>查看全部</Button>}
          >
            {tasksLoading ? (
              <Skeleton active paragraph={{ rows: 4 }} />
            ) : todoItems.length === 0 ? (
              <Empty description="暂无待办" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            ) : (
              <List
                dataSource={todoItems}
                renderItem={(item: any) => (
                  <List.Item
                    style={{ cursor: 'pointer', padding: '10px 0' }}
                    onClick={() => navigate(`/projects/${item.project_id}`)}
                    actions={[
                      <Tag color={priorityColors[item.priority] || 'default'} key="priority">
                        {item.priority === 'urgent' ? '紧急' : item.priority === 'high' ? '高' : item.priority === 'medium' ? '中' : '低'}
                      </Tag>,
                    ]}
                  >
                    <List.Item.Meta
                      avatar={taskTypeIcons[item.task_type] || <ClockCircleOutlined style={{ color: '#1890ff' }} />}
                      title={item.title}
                      description={
                        <Space size={8}>
                          {item.project?.name && <Tag style={{ fontSize: 11 }}>{item.project.name}</Tag>}
                          {item.due_date && (
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              截止: {dayjs(item.due_date).format('YYYY-MM-DD')}
                            </Text>
                          )}
                        </Space>
                      }
                    />
                  </List.Item>
                )}
              />
            )}
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
              <Skeleton active paragraph={{ rows: 4 }} />
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
