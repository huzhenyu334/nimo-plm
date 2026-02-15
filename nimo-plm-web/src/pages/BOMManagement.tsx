import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Input, Tag, Badge, Progress, Empty, Spin, Typography } from 'antd';
import { SearchOutlined, RightOutlined } from '@ant-design/icons';
import { projectApi, Project } from '@/api/projects';
import { useIsMobile } from '@/hooks/useIsMobile';

const { Text } = Typography;

const phaseLabels: Record<string, string> = {
  concept: '概念', evt: 'EVT', dvt: 'DVT', pvt: 'PVT', mp: 'MP',
  CONCEPT: '概念', EVT: 'EVT', DVT: 'DVT', PVT: 'PVT', MP: 'MP',
};

const phaseColors: Record<string, string> = {
  concept: 'purple', evt: 'blue', dvt: 'cyan', pvt: 'orange', mp: 'green',
  CONCEPT: 'purple', EVT: 'blue', DVT: 'DVT', PVT: 'orange', MP: 'green',
};

const statusLabels: Record<string, string> = {
  planning: '规划中', active: '进行中', completed: '已完成', on_hold: '暂停',
};

const statusColors: Record<string, string> = {
  planning: 'default', active: 'processing', completed: 'success', on_hold: 'warning',
};

const BOMManagement: React.FC = () => {
  const navigate = useNavigate();
  const isMobile = useIsMobile();
  const [search, setSearch] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['projects', 'bom-management'],
    queryFn: () => projectApi.list({ page: 1, page_size: 100 }),
  });

  const projects: Project[] = (data as any)?.items || data || [];

  const filtered = projects.filter(p =>
    !search || p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.code?.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div style={{ padding: isMobile ? 0 : 24 }}>
      {!isMobile && (
        <div style={{ marginBottom: 16 }}>
          <Text strong style={{ fontSize: 20 }}>BOM管理</Text>
          <Text type="secondary" style={{ marginLeft: 12, fontSize: 14 }}>选择项目进行BOM编辑</Text>
        </div>
      )}

      {/* Search */}
      <div style={{ padding: isMobile ? '8px 16px' : '0 0 16px 0' }}>
        <Input
          placeholder="搜索项目..."
          prefix={<SearchOutlined />}
          value={search}
          onChange={e => setSearch(e.target.value)}
          allowClear
          style={isMobile ? { borderRadius: 20, height: 40, background: '#f5f5f5', border: 'none' } : { width: 320 }}
        />
      </div>

      {/* Loading */}
      {isLoading && (
        <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
      )}

      {/* Empty */}
      {!isLoading && filtered.length === 0 && (
        <Empty description="暂无项目" style={{ padding: 60 }} />
      )}

      {/* Project list */}
      <div style={{ padding: isMobile ? '0 12px' : 0, display: 'grid', gridTemplateColumns: isMobile ? '1fr' : 'repeat(auto-fill, minmax(340px, 1fr))', gap: isMobile ? 8 : 16 }}>
        {filtered.map(project => (
          <div
            key={project.id}
            onClick={() => navigate(`/bom-management/${project.id}`)}
            style={{
              background: '#fff',
              borderRadius: 12,
              padding: isMobile ? '14px 16px' : '16px 20px',
              cursor: 'pointer',
              border: isMobile ? 'none' : '1px solid #f0f0f0',
              boxShadow: isMobile ? 'none' : '0 1px 2px rgba(0,0,0,0.04)',
              borderBottom: isMobile ? '1px solid #f0f0f0' : undefined,
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              transition: 'background 0.15s',
            }}
          >
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                <Text strong style={{ fontSize: 15, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {project.name}
                </Text>
                {project.code && (
                  <Text type="secondary" style={{ fontSize: 12, flexShrink: 0 }}>{project.code}</Text>
                )}
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <Tag color={phaseColors[project.phase]}>{phaseLabels[project.phase] || project.phase}</Tag>
                <Badge status={statusColors[project.status] as any} text={statusLabels[project.status] || project.status} />
              </div>
            </div>
            <Progress type="circle" percent={project.progress} size={36} />
            <RightOutlined style={{ color: '#ccc', fontSize: 12 }} />
          </div>
        ))}
      </div>
    </div>
  );
};

export default BOMManagement;
