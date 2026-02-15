import React, { useState, useMemo } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useIsMobile } from '@/hooks/useIsMobile';
import {
  Card,
  Button,
  Tag,
  Typography,
  Badge,
  Row,
  Col,
  Input,
  App,
  Empty,
  Spin,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
  SearchOutlined,
  CalendarOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { projectApi, Project } from '@/api/projects';

const { Title, Text } = Typography;

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

const statusConfig: Record<string, { color: string; text: string }> = {
  planning: { color: 'default', text: '规划中' },
  active: { color: 'processing', text: '进行中' },
  on_hold: { color: 'warning', text: '暂停' },
  completed: { color: 'success', text: '已完成' },
  cancelled: { color: 'error', text: '已取消' },
};

const Projects: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { modal, message } = App.useApp();
  const isMobile = useIsMobile();
  const [searchText, setSearchText] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['projects'],
    queryFn: () => projectApi.list(),
  });

  const projects = useMemo(() => {
    const items = data?.items || [];
    if (!searchText) return items;
    const lower = searchText.toLowerCase();
    return items.filter((p: Project) =>
      p.name?.toLowerCase().includes(lower) ||
      p.code?.toLowerCase().includes(lower)
    );
  }, [data, searchText]);

  const handleDelete = (e: React.MouseEvent, record: Project) => {
    e.stopPropagation();
    modal.confirm({
      title: '删除项目',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除项目「${record.name}」吗？此操作不可恢复。`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await projectApi.delete(record.id);
          message.success('项目已删除');
          queryClient.invalidateQueries({ queryKey: ['projects'] });
        } catch {
          message.error('删除失败');
        }
      },
    });
  };

  return (
    <div style={{ padding: isMobile ? 0 : 24 }}>
      {/* Header */}
      {!isMobile && (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <Title level={3} style={{ margin: 0 }}>研发项目</Title>
          <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
            <Input
              placeholder="搜索项目名称或编号"
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 240 }}
              allowClear
            />
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => navigate('/templates')}
            >
              创建项目
            </Button>
          </div>
        </div>
      )}

      {/* Mobile search bar */}
      {isMobile && (
        <div className="ds-search-bar">
          <Input
            placeholder="搜索项目"
            prefix={<SearchOutlined />}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            allowClear
            style={{ borderRadius: 20, height: 40, background: '#f5f5f5', border: 'none' }}
          />
        </div>
      )}

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
      ) : projects.length === 0 ? (
        <div style={{ padding: isMobile ? 16 : 0 }}>
          <Empty description={searchText ? '没有匹配的项目' : '暂无项目'} />
        </div>
      ) : isMobile ? (
        <div className="ds-page-content">
          {projects.map((project: Project) => {
            const status = statusConfig[project.status] || { color: 'default', text: project.status };
            const tagClass = status.color === 'processing' ? 'ds-tag-processing' :
              status.color === 'success' ? 'ds-tag-success' :
              status.color === 'warning' ? 'ds-tag-warning' :
              status.color === 'error' ? 'ds-tag-danger' : 'ds-tag-default';
            return (
              <div key={project.id} className="ds-list-card" onClick={() => navigate(`/projects/${project.id}`)}>
                <div className="ds-card-header">
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div className="ds-card-title">{project.name}</div>
                    <div className="ds-card-subtitle" style={{ fontFamily: 'monospace' }}>{project.code}</div>
                  </div>
                  <span className={`ds-tag ${tagClass}`}>{status.text}</span>
                </div>
                <div className="ds-card-meta">
                  {project.phase && (
                    <Tag color={phaseColors[project.phase]} style={{ margin: 0 }}>{project.phase?.toUpperCase()}</Tag>
                  )}
                  {project.manager_name && (
                    <span><UserOutlined /> {project.manager_name}</span>
                  )}
                  {project.start_date && (
                    <span><CalendarOutlined /> {new Date(project.start_date).toLocaleDateString('zh-CN')}</span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <div>
          <Row gutter={[20, 20]}>
            {projects.map((project: Project) => {
              const status = statusConfig[project.status] || { color: 'default', text: project.status };
              return (
                <Col key={project.id} xs={24} sm={12} md={8} lg={6}>
                  <Card
                    hoverable
                    onClick={() => navigate(`/projects/${project.id}`)}
                    style={{ height: '100%', borderRadius: 8 }}
                    bodyStyle={{ padding: 20 }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12 }}>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <Text strong style={{ fontSize: 16, display: 'block' }} ellipsis={{ tooltip: project.name }}>
                          {project.name}
                        </Text>
                        <Text type="secondary" style={{ fontSize: 12, fontFamily: 'monospace' }}>
                          {project.code}
                        </Text>
                      </div>
                      <Button
                        type="text"
                        size="small"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={(e) => handleDelete(e, project)}
                        style={{ marginLeft: 8, flexShrink: 0 }}
                      />
                    </div>
                    <div style={{ display: 'flex', gap: 6, marginBottom: 12, flexWrap: 'wrap', alignItems: 'center' }}>
                      <Badge status={status.color as any} text={<Text style={{ fontSize: 12 }}>{status.text}</Text>} />
                      {project.phase && (
                        <Tag color={phaseColors[project.phase]} style={{ margin: 0 }}>{project.phase?.toUpperCase()}</Tag>
                      )}
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, flexWrap: 'wrap' }}>
                      {project.manager_name && (
                        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                          <UserOutlined style={{ fontSize: 12, color: '#999' }} />
                          <Text type="secondary" style={{ fontSize: 12 }}>{project.manager_name}</Text>
                        </div>
                      )}
                      {project.start_date && (
                        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                          <CalendarOutlined style={{ fontSize: 12, color: '#999' }} />
                          <Text type="secondary" style={{ fontSize: 12 }}>
                            {new Date(project.start_date).toLocaleDateString('zh-CN')}
                          </Text>
                        </div>
                      )}
                    </div>
                  </Card>
                </Col>
              );
            })}
          </Row>
        </div>
      )}

      {/* Mobile FAB for creating project */}
      {isMobile && (
        <div className="ds-fab" onClick={() => navigate('/templates')}>
          <PlusOutlined />
        </div>
      )}
    </div>
  );
};

export default Projects;
