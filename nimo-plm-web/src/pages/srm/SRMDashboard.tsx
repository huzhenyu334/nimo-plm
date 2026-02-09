import React from 'react';
import { Card, Row, Col, Statistic, Progress, Space, Tag, Empty } from 'antd';
import {
  ShopOutlined,
  FileTextOutlined,
  ShoppingCartOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { srmApi } from '@/api/srm';
import { projectApi, Project } from '@/api/projects';

const SRMDashboard: React.FC = () => {
  const { data: supplierData } = useQuery({
    queryKey: ['srm-suppliers-count'],
    queryFn: () => srmApi.listSuppliers({ page: 1, page_size: 1 }),
  });

  const { data: prData } = useQuery({
    queryKey: ['srm-prs-active'],
    queryFn: () => srmApi.listPRs({ status: 'in_progress', page: 1, page_size: 1 }),
  });

  const { data: poData } = useQuery({
    queryKey: ['srm-pos-active'],
    queryFn: () => srmApi.listPOs({ status: 'in_progress', page: 1, page_size: 1 }),
  });

  const { data: inspData } = useQuery({
    queryKey: ['srm-inspections-pending'],
    queryFn: () => srmApi.listInspections({ status: 'pending', page: 1, page_size: 1 }),
  });

  const { data: projectData } = useQuery({
    queryKey: ['projects'],
    queryFn: () => projectApi.list({ page_size: 100 }),
  });

  const activeProjects = (projectData?.items || []).filter(
    (p: Project) => p.status === 'active'
  );

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>采购看板</h2>

      {/* Stats Cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic
              title="供应商总数"
              value={supplierData?.pagination?.total || 0}
              prefix={<ShopOutlined style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic
              title="进行中PR"
              value={prData?.pagination?.total || 0}
              prefix={<FileTextOutlined style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic
              title="进行中PO"
              value={poData?.pagination?.total || 0}
              prefix={<ShoppingCartOutlined style={{ color: '#fa8c16' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card>
            <Statistic
              title="待检验"
              value={inspData?.pagination?.total || 0}
              prefix={<SafetyCertificateOutlined style={{ color: '#eb2f96' }} />}
            />
          </Card>
        </Col>
      </Row>

      {/* Sampling Progress */}
      <Card title="打样进度跟踪">
        {activeProjects.length === 0 ? (
          <Empty description="暂无进行中的项目" />
        ) : (
          <Row gutter={[16, 16]}>
            {activeProjects.map((project: Project) => (
              <Col xs={24} sm={12} lg={8} key={project.id}>
                <ProjectSamplingCard project={project} />
              </Col>
            ))}
          </Row>
        )}
      </Card>
    </div>
  );
};

const phaseColors: Record<string, string> = {
  concept: 'purple',
  evt: 'blue',
  dvt: 'cyan',
  pvt: 'orange',
  mp: 'green',
};

const ProjectSamplingCard: React.FC<{ project: Project }> = ({ project }) => {
  const { data: progress } = useQuery({
    queryKey: ['srm-sampling-progress', project.id],
    queryFn: () => srmApi.getSamplingProgress(project.id),
  });

  const pct = progress?.progress_pct ?? 0;
  const total = progress?.total_items ?? 0;
  const passed = progress?.passed_items ?? 0;
  const ordered = progress?.ordered_items ?? 0;
  const received = progress?.received_items ?? 0;
  const inspected = progress?.inspected_items ?? 0;

  return (
    <Card size="small" hoverable>
      <Space direction="vertical" style={{ width: '100%' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <strong>{project.name}</strong>
          <Tag color={phaseColors[project.phase] || 'default'}>{project.phase?.toUpperCase()}</Tag>
        </div>
        <Progress percent={Math.round(pct)} size="small" />
        <Row gutter={8}>
          <Col span={8}>
            <Statistic title="总物料" value={total} valueStyle={{ fontSize: 16 }} />
          </Col>
          <Col span={8}>
            <Statistic title="已下单" value={ordered} valueStyle={{ fontSize: 16, color: '#1890ff' }} />
          </Col>
          <Col span={8}>
            <Statistic title="已通过" value={passed} valueStyle={{ fontSize: 16, color: '#52c41a' }} />
          </Col>
        </Row>
        <div style={{ fontSize: 12, color: '#999' }}>
          已收货 {received} / 已检验 {inspected}
        </div>
      </Space>
    </Card>
  );
};

export default SRMDashboard;
