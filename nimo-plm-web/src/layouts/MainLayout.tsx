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
  AuditOutlined,
  CheckSquareOutlined,
  TeamOutlined,
  DashboardOutlined,
  ShopOutlined,
  FileTextOutlined,
  ShoppingCartOutlined,
  SafetyCertificateOutlined,
  AccountBookOutlined,
  ToolOutlined,
  StarOutlined,
  SwapOutlined,
  RightOutlined,
  ArrowLeftOutlined,
  PartitionOutlined,
  DatabaseOutlined,
  InboxOutlined,
  FolderOpenOutlined,
} from '@ant-design/icons';
import { Dropdown, Avatar, Space, Spin } from 'antd';
import { useAuth } from '@/contexts/AuthContext';
import { useIsMobile } from '@/hooks/useIsMobile';

// Page title mapping for mobile header
const pageTitles: Record<string, string> = {
  '/dashboard': '工作台',
  '/my-tasks': '我的任务',
  '/projects': '研发项目',
  '/approvals': '审批中心',
  '/materials': '物料选型库',
  '/material-search': '物料查询',
  '/templates': '流程管理',
  '/roles': '角色管理',
  '/bom-management': 'BOM管理',
  '/bom-ecn': 'BOM ECN管理',
  '/documents': '文档管理',
  '/ecn': 'ECN变更管理',
  '/srm': 'SRM采购管理',
  '/srm/suppliers': '供应商',
  '/srm/purchase-requests': '采购需求',
  '/srm/purchase-orders': '采购订单',
  '/srm/inspections': '来料检验',
  '/srm/inventory': '库存管理',
  '/srm/settlements': '对账结算',
  '/srm/corrective-actions': '8D改进',
  '/srm/evaluations': '供应商评价',
  '/srm/equipment': '通用设备',
};

// Bottom tab configuration
const bottomTabs = [
  { path: '/dashboard', label: '工作台', icon: <HomeOutlined /> },
  { path: '/projects', label: '项目', icon: <ProjectOutlined /> },
  { path: '/bom-management', label: 'BOM', icon: <PartitionOutlined /> },
  { path: '/my-tasks', label: '任务', icon: <CheckSquareOutlined /> },
  { path: '/__my__', label: '我的', icon: <UserOutlined /> },
];

// Grouped "More" menu items
const moreMenuGroups = [
  {
    title: '项目管理',
    items: [
      { path: '/bom-management', label: 'BOM管理', icon: <PartitionOutlined /> },
      { path: '/bom-ecn', label: 'BOM ECN管理', icon: <SwapOutlined /> },
      { path: '/templates', label: '流程管理', icon: <SnippetsOutlined /> },
      { path: '/ecn', label: 'ECN变更管理', icon: <SwapOutlined /> },
    ],
  },
  {
    title: '采购管理',
    items: [
      { path: '/srm', label: 'SRM采购管理', icon: <ShoppingCartOutlined /> },
      { path: '/srm/suppliers', label: '供应商', icon: <ShopOutlined /> },
      { path: '/srm/purchase-requests', label: '采购需求', icon: <FileTextOutlined /> },
      { path: '/srm/purchase-orders', label: '采购订单', icon: <ShoppingCartOutlined /> },
      { path: '/srm/inspections', label: '来料检验', icon: <SafetyCertificateOutlined /> },
      { path: '/srm/inventory', label: '库存管理', icon: <InboxOutlined /> },
      { path: '/srm/settlements', label: '对账结算', icon: <AccountBookOutlined /> },
      { path: '/srm/evaluations', label: '供应商评价', icon: <StarOutlined /> },
    ],
  },
  {
    title: '系统设置',
    items: [
      { path: '/material-search', label: '物料查询', icon: <DatabaseOutlined /> },
      { path: '/materials', label: '物料选型库', icon: <ExperimentOutlined /> },
      { path: '/roles', label: '角色管理', icon: <TeamOutlined /> },
    ],
  },
];

// Flat list for path matching
const allMorePaths = moreMenuGroups.flatMap(g => g.items.map(i => i.path));

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const { user, logout, isLoading } = useAuth();
  const isMobile = useIsMobile();
  // embed=1 mode
  const isEmbed = searchParams.get('embed') === '1';

  React.useEffect(() => {
    if (isEmbed) {
      const path = location.pathname;
      if (path.includes('my-tasks')) document.title = 'nimo 任务';
      else if (path.includes('approval')) document.title = 'nimo 审批';
      else document.title = 'nimo PLM';
    }
  }, [isEmbed, location.pathname]);

  const menuItems = useMemo(() => {
    return [
      { path: '/dashboard', name: '工作台', icon: <HomeOutlined /> },
      { path: '/my-tasks', name: '我的任务', icon: <CheckSquareOutlined /> },
      { path: '/projects', name: '项目管理', icon: <ProjectOutlined /> },
      { path: '/bom-management', name: 'BOM管理', icon: <PartitionOutlined /> },
      { path: '/bom-ecn', name: 'BOM ECN管理', icon: <SwapOutlined /> },
      { path: '/material-search', name: '物料查询', icon: <DatabaseOutlined /> },
      { path: '/materials', name: '物料选型库', icon: <ExperimentOutlined /> },
      { path: '/templates', name: '流程管理', icon: <SnippetsOutlined /> },
      { path: '/approvals', name: '审批管理', icon: <AuditOutlined /> },
      { path: '/documents', name: '文档管理', icon: <FolderOpenOutlined /> },
      { path: '/ecn', name: 'ECN变更管理', icon: <SwapOutlined /> },
      { path: '/roles', name: '角色管理', icon: <TeamOutlined /> },
      {
        path: '/srm',
        name: 'SRM 采购管理',
        icon: <ShoppingCartOutlined />,
        children: [
          { path: '/srm', name: '采购总览', icon: <DashboardOutlined /> },
          { path: '/srm/suppliers', name: '供应商', icon: <ShopOutlined /> },
          { path: '/srm/purchase-requests', name: '采购需求', icon: <FileTextOutlined /> },
          { path: '/srm/purchase-orders', name: '采购订单', icon: <ShoppingCartOutlined /> },
          { path: '/srm/inspections', name: '来料检验', icon: <SafetyCertificateOutlined /> },
          { path: '/srm/inventory', name: '库存管理', icon: <InboxOutlined /> },
          { path: '/srm/settlements', name: '对账结算', icon: <AccountBookOutlined /> },
          { path: '/srm/corrective-actions', name: '8D改进', icon: <ToolOutlined /> },
          { path: '/srm/evaluations', name: '供应商评价', icon: <StarOutlined /> },
          { path: '/srm/equipment', name: '通用设备', icon: <ToolOutlined /> },
        ],
      },
    ];
  }, []);

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  if (isEmbed) {
    return (
      <div style={{ padding: '16px', background: '#fff', minHeight: '100vh' }}>
        <Outlet />
      </div>
    );
  }

  // ===== Mobile Layout =====
  if (isMobile) {
    const currentPath = location.pathname;
    const isMyPage = currentPath === '/__my__';
    // Determine if we're on a sub-page (need back button)
    const isSubPage = !isMyPage && !bottomTabs.some((t) => t.path !== '/__my__' && t.path === currentPath) &&
      !allMorePaths.includes(currentPath);
    // Get page title
    let pageTitle = isMyPage ? '我的' : (pageTitles[currentPath] || 'nimo PLM');
    if (currentPath.startsWith('/projects/') && currentPath !== '/projects') {
      pageTitle = '项目详情';
    }
    if (currentPath.startsWith('/templates/') && currentPath !== '/templates') {
      pageTitle = '流程详情';
    }

    // Active tab matching - special handling for /__my__ and /bom-management
    let activeTab = '';
    if (isMyPage) {
      activeTab = '/__my__';
    } else if (currentPath.startsWith('/bom-management')) {
      activeTab = '/bom-management';
    } else {
      activeTab = bottomTabs.find((t) => t.path !== '/__my__' && t.path !== '/bom-management' && currentPath.startsWith(t.path))?.path || '';
    }

    return (
      <div className="mobile-content">
        {/* Mobile Header */}
        <div className="mobile-header">
          {isSubPage ? (
            <div className="mobile-header-back" onClick={() => navigate(-1)}>
              <ArrowLeftOutlined />
            </div>
          ) : (
            <div className="mobile-header-action" />
          )}
          <div className="mobile-header-title">{pageTitle}</div>
          <div className="mobile-header-action" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Avatar
              size={28}
              src={user?.avatar_url}
              style={{ background: '#1677ff', fontSize: 12 }}
            >
              {user?.name?.[0]}
            </Avatar>
          </div>
        </div>

        {/* Main Content */}
        {isMyPage ? (
          <div style={{ padding: '0 0 16px' }}>
            {/* User info */}
            <div style={{ background: '#fff', padding: '20px 16px', display: 'flex', alignItems: 'center', gap: 12, marginBottom: 8 }}>
              <Avatar size={48} src={user?.avatar_url} style={{ background: '#1677ff', fontSize: 20 }}>
                {user?.name?.[0]}
              </Avatar>
              <div>
                <div style={{ fontSize: 17, fontWeight: 600 }}>{user?.name || '用户'}</div>
                <div style={{ fontSize: 13, color: '#999' }}>{user?.email || ''}</div>
              </div>
            </div>

            {/* Quick entries */}
            <div style={{ background: '#fff', marginBottom: 8 }}>
              <div
                className="mobile-my-menu-item"
                onClick={() => navigate('/my-tasks')}
              >
                <CheckSquareOutlined style={{ fontSize: 20, color: '#1677ff' }} />
                <span style={{ flex: 1 }}>我的任务</span>
                <RightOutlined style={{ fontSize: 12, color: '#ccc' }} />
              </div>
              <div
                className="mobile-my-menu-item"
                onClick={() => navigate('/approvals')}
              >
                <AuditOutlined style={{ fontSize: 20, color: '#faad14' }} />
                <span style={{ flex: 1 }}>审批中心</span>
                <RightOutlined style={{ fontSize: 12, color: '#ccc' }} />
              </div>
            </div>

            {/* Grouped more menu */}
            <div style={{ background: '#fff', marginBottom: 8 }}>
              {moreMenuGroups.map((group) => (
                <div key={group.title}>
                  <div className="mobile-my-group-title">{group.title}</div>
                  {group.items.map((item) => (
                    <div
                      key={item.path}
                      className="mobile-my-menu-item"
                      onClick={() => navigate(item.path)}
                    >
                      <span style={{ fontSize: 20, color: '#666' }}>{item.icon}</span>
                      <span style={{ flex: 1 }}>{item.label}</span>
                      <RightOutlined style={{ fontSize: 12, color: '#ccc' }} />
                    </div>
                  ))}
                </div>
              ))}
            </div>

            {/* Logout */}
            <div style={{ background: '#fff' }}>
              <div
                className="mobile-my-menu-item"
                style={{ color: '#ff4d4f', justifyContent: 'center' }}
                onClick={() => { logout(); navigate('/login'); }}
              >
                <LogoutOutlined style={{ fontSize: 20 }} />
                <span>退出登录</span>
              </div>
            </div>
          </div>
        ) : (
          <Outlet />
        )}

        {/* Bottom Tab Bar */}
        <div className="mobile-bottom-nav">
          {bottomTabs.map((tab) => (
            <div
              key={tab.path}
              className={`mobile-bottom-nav-item ${activeTab === tab.path ? 'active' : ''}`}
              onClick={() => {
                if (tab.path === '/__my__') {
                  // "我的" just renders inline, no real navigation needed
                  navigate('/__my__');
                } else {
                  navigate(tab.path);
                }
              }}
            >
              {tab.icon}
              <span>{tab.label}</span>
            </div>
          ))}
        </div>
      </div>
    );
  }

  // ===== Desktop Layout =====
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
              { key: 'profile', icon: <UserOutlined />, label: '个人信息' },
              { type: 'divider' },
              {
                key: 'logout',
                icon: <LogoutOutlined />,
                label: '退出登录',
                onClick: () => { logout(); navigate('/login'); },
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
