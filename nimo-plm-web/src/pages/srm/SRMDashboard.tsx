import React from 'react';
import { Card, Row, Col, Statistic, Progress, Tag, Empty, Timeline, Button } from 'antd';
import {
  ProjectOutlined,
  AppstoreOutlined,
  WarningOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { srmApi, SRMProject, ActivityLog } from '@/api/srm';
import dayjs from 'dayjs';

const phaseColors: Record<string, string> = {
  concept: 'purple', evt: 'blue', dvt: 'cyan', pvt: 'orange', mp: 'green',
};

const SRMDashboard: React.FC = () => {
  const navigate = useNavigate();

  // Load active projects
  const { data: projectData } = useQuery({
    queryKey: ['srm-projects-dashboard'],
    queryFn: () => srmApi.listProjects({ status: 'active', page_size: 100 }),
  });

  // Load all projects for stats
  const { data: allProjectData } = useQuery({
    queryKey: ['srm-projects-all-stats'],
    queryFn: () => srmApi.listProjects({ page_size: 1 }),
  });

  // Load PRs for total items + overdue count
  const { data: prData } = useQuery({
    queryKey: ['srm-prs-dashboard-stats'],
    queryFn: () => srmApi.listPRs({ page_size: 200 }),
  });

  const activeProjects = projectData?.items || [];
  const totalProjects = allProjectData?.pagination?.total || 0;

  // Compute stats from active projects
  const stats = React.useMemo(() => {
    let totalItems = 0;
    let overdueItems = 0;
    let passedItems = 0;

    activeProjects.forEach((p: SRMProject) => {
      totalItems += p.total_items || 0;
      passedItems += p.passed_count || 0;
    });

    // Check PR items for overdue
    const prs = prData?.items || [];
    const now = dayjs();
    prs.forEach((pr) => {
      (pr.items || []).forEach((item) => {
        if (item.expected_date && item.status !== 'passed' && item.status !== 'failed') {
          if (dayjs(item.expected_date).isBefore(now, 'day')) {
            overdueItems++;
          }
        }
      });
    });

    const passRate = totalItems > 0 ? Math.round((passedItems / totalItems) * 100) : 0;
    return { activeCount: activeProjects.length, totalItems, overdueItems, passRate };
  }, [activeProjects, prData]);

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>采购总览</h2>

      {/* Stat cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic
              title="活跃项目"
              value={stats.activeCount}
              suffix={<span style={{ fontSize: 14, color: '#999' }}>/ {totalProjects}</span>}
              prefix={<ProjectOutlined style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic
              title="采购物料"
              value={stats.totalItems}
              prefix={<AppstoreOutlined style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic
              title="超期物料"
              value={stats.overdueItems}
              valueStyle={stats.overdueItems > 0 ? { color: '#ff4d4f' } : undefined}
              prefix={<WarningOutlined style={{ color: stats.overdueItems > 0 ? '#ff4d4f' : '#999' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic
              title="检验通过率"
              value={stats.passRate}
              suffix="%"
              prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
      </Row>

      {/* Active projects */}
      <Card
        title="进行中的项目"
        extra={
          <Button type="link" onClick={() => navigate('/srm/projects')}>
            查看全部
          </Button>
        }
        style={{ marginBottom: 24 }}
      >
        {activeProjects.length === 0 ? (
          <Empty description="暂无进行中的项目" />
        ) : (
          <Row gutter={[16, 16]}>
            {activeProjects.map((project: SRMProject) => (
              <Col xs={24} sm={12} lg={8} key={project.id}>
                <Card
                  size="small"
                  hoverable
                  onClick={() => navigate(`/srm/kanban?project=${project.id}`)}
                  style={{ cursor: 'pointer' }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                    <strong style={{ fontSize: 14 }}>{project.name}</strong>
                    <div>
                      {project.phase && (
                        <Tag color={phaseColors[project.phase] || 'default'} style={{ marginRight: 4 }}>
                          {project.phase.toUpperCase()}
                        </Tag>
                      )}
                      <Tag color={project.type === 'sample' ? 'blue' : 'green'}>
                        {project.type === 'sample' ? '打样' : '量产'}
                      </Tag>
                    </div>
                  </div>
                  <div style={{ fontSize: 12, color: '#999', marginBottom: 8 }}>{project.code}</div>
                  <Progress
                    percent={project.total_items > 0 ? Math.round((project.passed_count / project.total_items) * 100) : 0}
                    size="small"
                    strokeColor="#52c41a"
                  />
                  <Row gutter={8} style={{ marginTop: 8 }}>
                    <Col span={6}>
                      <Statistic title="总计" value={project.total_items} valueStyle={{ fontSize: 14 }} />
                    </Col>
                    <Col span={6}>
                      <Statistic title="已下单" value={project.ordered_count} valueStyle={{ fontSize: 14, color: '#1890ff' }} />
                    </Col>
                    <Col span={6}>
                      <Statistic title="已收货" value={project.received_count} valueStyle={{ fontSize: 14 }} />
                    </Col>
                    <Col span={6}>
                      <Statistic title="已通过" value={project.passed_count} valueStyle={{ fontSize: 14, color: '#52c41a' }} />
                    </Col>
                  </Row>
                  {project.target_date && (
                    <div style={{ fontSize: 12, color: '#999', marginTop: 8 }}>
                      目标: {dayjs(project.target_date).format('YYYY-MM-DD')}
                    </div>
                  )}
                </Card>
              </Col>
            ))}
          </Row>
        )}
      </Card>

      {/* Recent activity */}
      <RecentActivity projects={activeProjects} />
    </div>
  );
};

const RecentActivity: React.FC<{ projects: SRMProject[] }> = ({ projects }) => {
  const firstProjectId = projects[0]?.id;

  const { data: activityData } = useQuery({
    queryKey: ['srm-recent-activities', firstProjectId],
    queryFn: () => srmApi.listProjectActivities(firstProjectId!, { page_size: 15 }),
    enabled: !!firstProjectId,
  });

  const activities = activityData?.items || [];

  return (
    <Card title="最近操作记录">
      {activities.length === 0 ? (
        <Empty description="暂无操作记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <Timeline
          items={activities.map((log: ActivityLog) => ({
            children: (
              <div>
                <div style={{ fontSize: 13 }}>
                  {log.entity_code && (
                    <Tag style={{ fontSize: 11, marginRight: 6 }}>{log.entity_code}</Tag>
                  )}
                  {log.content}
                </div>
                <div style={{ fontSize: 12, color: '#999' }}>
                  {log.operator_name} · {dayjs(log.created_at).format('MM-DD HH:mm')}
                </div>
              </div>
            ),
          }))}
        />
      )}
    </Card>
  );
};

export default SRMDashboard;
