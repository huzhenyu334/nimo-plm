import React from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import MainLayout from '@/layouts/MainLayout';
import { Login, Dashboard, Templates, TemplateDetail, Projects, ProjectDetail, Materials, Approvals, ApprovalAdmin, ApprovalEditor, MyTasks, RoleManagement } from '@/pages';
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
    ],
  },
  {
    path: '*',
    element: <Navigate to="/dashboard" replace />,
  },
]);
