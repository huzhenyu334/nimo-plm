import apiClient from './client';
import { ApiResponse, PaginatedResponse } from '@/types';

export interface Project {
  id: string;
  code: string;
  name: string;
  description?: string;
  product_id?: string;
  product_name?: string;
  phase: string;
  status: string;
  start_date?: string;
  planned_end?: string;
  progress: number;
  manager_id?: string;
  manager_name?: string;
  template_id?: string;
  created_at: string;
  updated_at: string;
}

export interface TaskPhase {
  id: string;
  phase: string;
  name: string;
  sequence: number;
}

export interface Task {
  id: string;
  project_id: string;
  code?: string;
  title: string;
  description?: string;
  status: string;
  priority: string;
  assignee_id?: string;
  assignee_name?: string;
  assignee?: { id: string; name: string; avatar_url?: string };
  start_date?: string;
  due_date?: string;
  progress: number;
  parent_task_id?: string;
  task_type: string;
  task_code?: string;
  phase_id?: string;
  phase?: TaskPhase | string;
  sequence?: number;
  is_critical?: boolean;
  requires_approval?: boolean;
  approval_status?: string;
  dependencies?: { id: string; task_id: string; depends_on_id: string; depends_on_status?: string; dependency_type?: string }[];
  created_at: string;
}

export const projectApi = {
  // 获取项目列表
  list: async (params?: { status?: string; page?: number; page_size?: number }): Promise<PaginatedResponse<Project>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<Project>>>('/projects', { params });
    return response.data.data;
  },

  // 获取项目详情
  get: async (id: string): Promise<Project> => {
    const response = await apiClient.get<ApiResponse<Project>>(`/projects/${id}`);
    return response.data.data;
  },

  // 创建项目
  create: async (data: Partial<Project>): Promise<Project> => {
    const response = await apiClient.post<ApiResponse<Project>>('/projects', data);
    return response.data.data;
  },

  // 更新项目
  update: async (id: string, data: Partial<Project>): Promise<Project> => {
    const response = await apiClient.put<ApiResponse<Project>>(`/projects/${id}`, data);
    return response.data.data;
  },

  // 获取项目任务
  listTasks: async (projectId: string): Promise<Task[]> => {
    const response = await apiClient.get<ApiResponse<{items: Task[], total: number}>>(`/projects/${projectId}/tasks?page_size=500`);
    return response.data.data?.items || [];
  },

  // 更新任务状态
  updateTaskStatus: async (projectId: string, taskId: string, status: string): Promise<Task> => {
    const response = await apiClient.put<ApiResponse<Task>>(`/projects/${projectId}/tasks/${taskId}/status`, { status });
    return response.data.data;
  },

  // 完成任务
  completeTask: async (projectId: string, taskId: string): Promise<Task> => {
    const response = await apiClient.post<ApiResponse<Task>>(`/projects/${projectId}/tasks/${taskId}/complete`);
    return response.data.data;
  },

  // 删除项目
  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/projects/${id}`);
  },

  // 获取我的任务
  myTasks: async (params?: { status?: string; page?: number; page_size?: number }): Promise<PaginatedResponse<Task>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<Task>>>('/my/tasks', { params });
    return response.data.data;
  },

  // 批量角色分配
  assignRoles: async (projectId: string, assignments: { role: string; user_id: string }[]): Promise<void> => {
    await apiClient.post(`/projects/${projectId}/assign-roles`, { assignments });
  },
};
