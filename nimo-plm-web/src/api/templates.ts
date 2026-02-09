import apiClient from './client';
import { ApiResponse } from '@/types';

// 模板类型定义
export interface ProjectTemplate {
  id: string;
  code: string;
  name: string;
  description?: string;
  template_type: 'SYSTEM' | 'CUSTOM';
  product_type?: string;
  phases: string[];
  estimated_days: number;
  is_active: boolean;
  version: number;
  status: 'draft' | 'published';
  published_at?: string;
  parent_template_id?: string;
  base_code?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  tasks?: TemplateTask[];
  dependencies?: TemplateTaskDependency[];
}

export interface TemplateVersion {
  id: string;
  code: string;
  name: string;
  version: number;
  status: 'draft' | 'published';
  published_at?: string;
  task_count: number;
  created_at: string;
}

export interface TemplateTask {
  id: string;
  template_id: string;
  task_code: string;
  name: string;
  description?: string;
  phase: string;
  parent_task_code?: string;
  task_type: 'MILESTONE' | 'TASK' | 'SUBTASK';
  default_assignee_role?: string;
  estimated_days: number;
  is_critical: boolean;
  requires_approval: boolean;
  approval_type?: string;
  auto_create_feishu_task: boolean;
  feishu_approval_code?: string;
  sort_order: number;
  dependencies?: TemplateTaskDependency[];
}

export interface TemplateTaskDependency {
  id: string;
  template_id: string;
  task_code: string;
  depends_on_task_code: string;
  dependency_type: 'FS' | 'SS' | 'FF' | 'SF';
  lag_days: number;
}

export interface CreateProjectFromTemplateInput {
  template_id: string;
  project_name: string;
  project_code: string;
  product_id?: string;
  start_date: string;
  pm_user_id: string;
  skip_weekends?: boolean;
  role_assignments?: Record<string, string>;
  codename_id?: string;
}

export const templateApi = {
  // 获取模板列表
  list: async (type?: string, productType?: string): Promise<ProjectTemplate[]> => {
    const params: Record<string, string> = {};
    if (type) params.type = type;
    if (productType) params.product_type = productType;
    const response = await apiClient.get<ApiResponse<ProjectTemplate[]>>('/templates', { params });
    return response.data.data;
  },

  // 获取模板详情（含任务）
  get: async (id: string): Promise<ProjectTemplate> => {
    const response = await apiClient.get<ApiResponse<ProjectTemplate>>(`/templates/${id}`);
    return response.data.data;
  },

  // 创建模板
  create: async (data: Partial<ProjectTemplate>): Promise<ProjectTemplate> => {
    const response = await apiClient.post<ApiResponse<ProjectTemplate>>('/templates', data);
    return response.data.data;
  },

  // 更新模板
  update: async (id: string, data: Partial<ProjectTemplate>): Promise<ProjectTemplate> => {
    const response = await apiClient.put<ApiResponse<ProjectTemplate>>(`/templates/${id}`, data);
    return response.data.data;
  },

  // 删除模板
  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/templates/${id}`);
  },

  // 复制模板
  duplicate: async (id: string, newCode: string, newName: string): Promise<ProjectTemplate> => {
    const response = await apiClient.post<ApiResponse<ProjectTemplate>>(`/templates/${id}/duplicate`, {
      new_code: newCode,
      new_name: newName,
    });
    return response.data.data;
  },

  // 从模板创建项目
  createProjectFromTemplate: async (data: CreateProjectFromTemplateInput): Promise<unknown> => {
    const response = await apiClient.post<ApiResponse<unknown>>('/projects/create-from-template', data);
    return response.data.data;
  },

  // 发布流程
  publish: async (id: string): Promise<ProjectTemplate> => {
    const response = await apiClient.post<ApiResponse<ProjectTemplate>>(`/templates/${id}/publish`);
    return response.data.data;
  },

  // 升级版本（从已发布版本创建新草稿）
  upgrade: async (id: string): Promise<ProjectTemplate> => {
    const response = await apiClient.post<ApiResponse<ProjectTemplate>>(`/templates/${id}/upgrade`);
    return response.data.data;
  },

  // 撤销草稿，回退到上一个已发布版本
  revert: async (id: string): Promise<ProjectTemplate> => {
    const response = await apiClient.post<ApiResponse<ProjectTemplate>>(`/templates/${id}/revert`);
    return response.data.data;
  },

  // 获取版本历史
  listVersions: async (id: string): Promise<TemplateVersion[]> => {
    const response = await apiClient.get<ApiResponse<TemplateVersion[]>>(`/templates/${id}/versions`);
    return response.data.data || [];
  },

  // 批量保存任务
  batchSaveTasks: async (
    templateId: string,
    tasks: Omit<TemplateTask, 'id' | 'template_id'>[],
    version?: string
  ): Promise<{ task_count: number; version: string }> => {
    const response = await apiClient.put<
      ApiResponse<{ task_count: number; version: string }>
    >(`/templates/${templateId}/tasks/batch`, {
      tasks,
      version,
    });
    return response.data.data;
  },
};
