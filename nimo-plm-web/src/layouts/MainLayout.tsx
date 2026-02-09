import React, { useMemo } from 'react';
import { Outlet, useNavigate, useLocation, useSearchParams } from 'react-router-dom';
import { ProLayout, ProLayoutProps } from '@ant-design/pro-components';
import {
  HomeOutlined,
  ProjectOutlined,
  ExperimentOutlined,
  SnippetsOutlined,
  LogoutOutlined,
  UserOutlined,
  FolderOutlined,
  AuditOutlined,
  CheckSquareOutlined,
  TeamOutlined,
  DashboardOutlined,
  ShopOutlined,
  FileTextOutlined,
  ShoppingCartOutlined,
  SafetyCertificateOutlined,
  AppstoreOutlined,
} from '@ant-design/icons';
import { Dropdown, Avatar, Space, Spin } from 'antd';
import { useAuth } from '@/contexts/AuthContext';
import { useQuery } from '@tanstack/react-query';
import { projectApi } from '@/api/projects';

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const { user, logout, isLoading } = useAuth();

  // embed=1 模式：隐藏侧边栏和顶栏（用于飞书内嵌）
  const isEmbed = searchParams.get('embed') === '1';

  // embed 模式下设置页面标题
  React.useEffect(() => {
    if (isEmbed) {
      const path = location.pathname;
      if (path.includes('my-tasks')) document.title = 'nimo 任务';
      else if (path.includes('approval')) document.title = 'nimo 审批';
      else document.title = 'nimo PLM';
    }
  }, [isEmbed, location.pathname]);

  // 获取项目列表
  const { data: projectData } = useQuery({
    queryKey: ['projects'],
    queryFn: () => projectApi.list(),
    enabled: !!user,
  });

  const menuItems = useMemo(() => {
    const projects = projectData?.items || [];
    const projectChildren = projects.map((p: any) => ({
      path: `/projects/${p.id}`,
      name: `${p.name}`,
    }));

    return [
      {
        path: '/dashboard',
        name: '工作台',
        icon: <HomeOutlined />,
      },
      {
        path: '/my-tasks',
        name: '我的任务',
        icon: <CheckSquareOutlined />,
      },
      {
        path: '/projects',
        name: '项目管理',
        icon: <ProjectOutlined />,
        children: projectChildren.length > 0 ? [
          { path: '/projects', name: '全部项目', icon: <FolderOutlined /> },
          ...projectChildren,
        ] : undefined,
      },
      {
        path: '/materials',
        name: '物料选型库',
        icon: <ExperimentOutlined />,
      },
      {
        path: '/templates',
        name: '流程管理',
        icon: <SnippetsOutlined />,
      },
      {
        path: '/approvals',
        name: '审批管理',
        icon: <AuditOutlined />,
        children: [
          { path: '/approvals', name: '审批中心' },
          { path: '/approval-admin', name: '审批后台' },
        ],
      },
      {
        path: '/roles',
        name: '角色管理',
        icon: <TeamOutlined />,
      },
      {
        path: '/srm',
        name: 'SRM 采购管理',
        icon: <ShoppingCartOutlined />,
        children: [
          { path: '/srm/kanban', name: '采购看板', icon: <AppstoreOutlined /> },
          { path: '/srm/projects', name: '采购项目', icon: <ProjectOutlined /> },
          { path: '/srm/suppliers', name: '供应商', icon: <ShopOutlined /> },
          { path: '/srm/purchase-requests', name: '采购需求', icon: <FileTextOutlined /> },
          { path: '/srm/purchase-orders', name: '采购订单', icon: <ShoppingCartOutlined /> },
          { path: '/srm/inspections', name: '来料检验', icon: <SafetyCertificateOutlined /> },
          { path: '/srm', name: '采购总览', icon: <DashboardOutlined /> },
        ],
      },
    ];
  }, [projectData]);

  if (isLoading) {
    return (
      <div style={{ 
        display: 'flex', 
        justifyContent: 'center', 
        alignItems: 'center', 
        height: '100vh' 
      }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  // embed 模式：无侧边栏无顶栏，纯内容
  if (isEmbed) {
    return (
      <div style={{ padding: '16px', background: '#fff', minHeight: '100vh' }}>
        <Outlet />
      </div>
    );
  }

  // Highlight current menu
  const menuPathname = location.pathname;

  const layoutProps: ProLayoutProps = {
    title: 'nimo PLM',
    logo: '/logo.svg',
    layout: 'mix',
    splitMenus: false,
    fixedHeader: true,
    fixSiderbar: true,
    contentWidth: 'Fluid',
    route: {
      path: '/',
      routes: menuItems,
    },
    location: {
      pathname: menuPathname,
    },
    menuItemRender: (item, dom) => (
      <div onClick={() => item.path && navigate(item.path)}>{dom}</div>
    ),
    avatarProps: {
      src: user?.avatar_url,
      size: 'small',
      title: user?.name,
      render: (_, dom) => (
        <Dropdown
          menu={{
            items: [
              {
                key: 'profile',
                icon: <UserOutlined />,
                label: '个人信息',
              },
              {
                type: 'divider',
              },
              {
                key: 'logout',
                icon: <LogoutOutlined />,
                label: '退出登录',
                onClick: () => {
                  logout();
                  navigate('/login');
                },
              },
            ],
          }}
        >
          {dom}
        </Dropdown>
      ),
    },
    actionsRender: () => [
      <Space key="user">
        <span style={{ color: '#fff' }}>{user?.name}</span>
        <Avatar src={user?.avatar_url} size="small">
          {user?.name?.[0]}
        </Avatar>
      </Space>,
    ],
    token: {
      header: {
        colorBgHeader: '#001529',
        colorHeaderTitle: '#fff',
        colorTextMenu: 'rgba(255,255,255,0.75)',
        colorTextMenuSecondary: 'rgba(255,255,255,0.65)',
        colorTextMenuSelected: '#fff',
        colorBgMenuItemSelected: '#1890ff',
        colorTextMenuActive: '#fff',
        colorTextRightActionsItem: 'rgba(255,255,255,0.85)',
      },
      sider: {
        colorMenuBackground: '#fff',
        colorTextMenu: 'rgba(0,0,0,0.85)',
        colorTextMenuSelected: '#1890ff',
        colorBgMenuItemSelected: '#e6f7ff',
      },
    },
  };

  return (
    <ProLayout {...layoutProps}>
      <Outlet />
    </ProLayout>
  );
};

export default MainLayout;
