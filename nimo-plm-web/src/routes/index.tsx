import React from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import MainLayout from '@/layouts/MainLayout';
import { Login, Dashboard, Templates, TemplateDetail, Projects, ProjectDetail, Materials, Approvals, ApprovalAdmin, ApprovalEditor, MyTasks, RoleManagement } from '@/pages';
import SRMDashboard from '@/pages/srm/SRMDashboard';
import Suppliers from '@/pages/srm/Suppliers';
import PurchaseRequests from '@/pages/srm/PurchaseRequests';
import PurchaseOrders from '@/pages/srm/PurchaseOrders';
import Inspections from '@/pages/srm/Inspections';
import SRMProjects from '@/pages/srm/Projects';
import KanbanBoard from '@/pages/srm/KanbanBoard';
import Settlements from '@/pages/srm/Settlements';
import CorrectiveActions from '@/pages/srm/CorrectiveActions';
import Evaluations from '@/pages/srm/Evaluations';
import SRMProjectDetail from '@/pages/srm/ProjectDetail';
import SRMEquipment from '@/pages/srm/Equipment';
import ECNList from '@/pages/ECN';
import ECNDetail from '@/pages/ECN/ECNDetail';
import ECNForm from '@/pages/ECN/ECNForm';
import BOMManagement from '@/pages/BOMManagement';
import BOMManagementDetail from '@/pages/BOMManagementDetail';
import { useAuth } from '@/contexts/AuthContext';

// 受保护路由组件
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return null;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <Login />,
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <MainLayout />
      </ProtectedRoute>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="/dashboard" replace />,
      },
      {
        path: 'dashboard',
        element: <Dashboard />,
      },
      {
        path: 'projects',
        element: <Projects />,
      },
      {
        path: 'projects/:id',
        element: <ProjectDetail />,
      },
      {
        path: 'materials',
        element: <Materials />,
      },
      {
        path: 'templates',
        element: <Templates />,
      },
      {
        path: 'templates/:id',
        element: <TemplateDetail />,
      },
      {
        path: 'approvals',
        element: <Approvals />,
      },
      {
        path: 'approval-admin',
        element: <ApprovalAdmin />,
      },
      {
        path: 'approval-editor/:id',
        element: <ApprovalEditor />,
      },
      {
        path: 'my-tasks',
        element: <MyTasks />,
      },
      {
        path: 'roles',
        element: <RoleManagement />,
      },
      {
        path: '__my__',
        element: <></>,
      },
      {
        path: 'bom-management',
        element: <BOMManagement />,
      },
      {
        path: 'bom-management/:projectId',
        element: <BOMManagementDetail />,
      },
      {
        path: 'ecn',
        element: <ECNList />,
      },
      {
        path: 'ecn/new',
        element: <ECNForm />,
      },
      {
        path: 'ecn/:id',
        element: <ECNDetail />,
      },
      {
        path: 'ecn/:id/edit',
        element: <ECNForm />,
      },
      {
        path: 'srm',
        element: <SRMDashboard />,
      },
      {
        path: 'srm/suppliers',
        element: <Suppliers />,
      },
      {
        path: 'srm/purchase-requests',
        element: <PurchaseRequests />,
      },
      {
        path: 'srm/purchase-orders',
        element: <PurchaseOrders />,
      },
      {
        path: 'srm/inspections',
        element: <Inspections />,
      },
      {
        path: 'srm/projects',
        element: <SRMProjects />,
      },
      {
        path: 'srm/kanban',
        element: <KanbanBoard />,
      },
      {
        path: 'srm/settlements',
        element: <Settlements />,
      },
      {
        path: 'srm/corrective-actions',
        element: <CorrectiveActions />,
      },
      {
        path: 'srm/evaluations',
        element: <Evaluations />,
      },
      {
        path: 'srm/projects/:id',
        element: <SRMProjectDetail />,
      },
      {
        path: 'srm/equipment',
        element: <SRMEquipment />,
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/dashboard" replace />,
  },
]);
