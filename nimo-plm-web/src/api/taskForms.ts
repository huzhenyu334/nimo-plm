import apiClient from './client';
import { ApiResponse } from '@/types';

export interface TaskFormField {
  key: string;
  label: string;
  type: 'text' | 'textarea' | 'number' | 'select' | 'multiselect' | 'date' | 'file' | 'checkbox' | 'user' | 'role_assignment' | 'bom_upload' | 'ebom_control' | 'pbom_control' | 'mbom_control' | 'cmf' | 'tooling_list' | 'consumable_list' | 'procurement_control';
  required: boolean;
  placeholder?: string;
  description?: string;
  options?: string[];
  accept?: string;
  multiple?: boolean;
  source_task_code?: string; // CMF字段：指定从哪个任务的EBOM提取外观件
  source_field_keys?: string[]; // 采购控件：引用源任务的哪些字段key
  bom_categories?: string[]; // 采购控件：BOM大类筛选
  bom_sub_categories?: string[]; // 采购控件：BOM子类筛选
  bom_type?: 'EBOM' | 'PBOM' | 'MBOM'; // BOM上传字段：BOM类型（向后兼容）
  config?: Record<string, any>; // BOM控件配置（ebom_control/pbom_control/mbom_control）
}

export interface TaskForm {
  id: string;
  task_id: string;
  name: string;
  description?: string;
  fields: TaskFormField[];
}

export interface TemplateTaskForm {
  id: string;
  template_id: string;
  task_code: string;
  name: string;
  fields: TaskFormField[];
}

export interface TaskFormSubmission {
  id: string;
  form_id: string;
  task_id: string;
  data: Record<string, any>;
  files: any[];
  submitted_by: string;
  submitted_at: string;
  version: number;
}

export const taskFormApi = {
  // 获取任务表单定义
  getForm: async (projectId: string, taskId: string): Promise<TaskForm | null> => {
    try {
      const response = await apiClient.get<ApiResponse<TaskForm>>(`/projects/${projectId}/tasks/${taskId}/form`);
      return response.data.data;
    } catch {
      return null;
    }
  },

  // 创建/更新任务表单
  upsertForm: async (projectId: string, taskId: string, data: { name: string; fields: TaskFormField[] }): Promise<TaskForm> => {
    const response = await apiClient.put<ApiResponse<TaskForm>>(`/projects/${projectId}/tasks/${taskId}/form`, data);
    return response.data.data;
  },

  // 获取表单提交内容
  getSubmission: async (projectId: string, taskId: string): Promise<TaskFormSubmission | null> => {
    try {
      const response = await apiClient.get<ApiResponse<TaskFormSubmission>>(`/projects/${projectId}/tasks/${taskId}/form/submission`);
      return response.data.data;
    } catch {
      return null;
    }
  },

  // 完成任务（工程师用）
  completeMyTask: async (taskId: string, formData?: Record<string, any>): Promise<void> => {
    await apiClient.post(`/my/tasks/${taskId}/complete`, { form_data: formData });
  },

  // 确认任务（项目经理用）
  confirmTask: async (projectId: string, taskId: string): Promise<void> => {
    await apiClient.post(`/projects/${projectId}/tasks/${taskId}/confirm`);
  },

  // 驳回任务（项目经理用）
  rejectTask: async (projectId: string, taskId: string, reason?: string): Promise<void> => {
    await apiClient.post(`/projects/${projectId}/tasks/${taskId}/reject`, { reason });
  },

  // 获取模板任务表单列表
  getTemplateTaskForms: async (templateId: string): Promise<TemplateTaskForm[]> => {
    try {
      const response = await apiClient.get<ApiResponse<TemplateTaskForm[]>>(`/templates/${templateId}/task-forms`);
      return response.data.data || [];
    } catch {
      return [];
    }
  },

  // 保存模板任务表单
  saveTemplateTaskForm: async (templateId: string, data: { task_code: string; name: string; fields: TaskFormField[] }): Promise<void> => {
    await apiClient.post(`/templates/${templateId}/task-forms`, data);
  },

  // 上传文件
  uploadFile: async (file: File): Promise<{ id: string; url: string; filename: string; size: number; thumbnail_url?: string }> => {
    const formData = new FormData();
    formData.append('files', file);
    const response = await apiClient.post<ApiResponse<any[]>>('/upload', formData);
    return response.data.data[0];
  },

  // 保存表单草稿
  saveDraft: async (taskId: string, formData: Record<string, any>): Promise<void> => {
    await apiClient.put(`/my/tasks/${taskId}/form-draft`, { form_data: formData });
  },

  // 获取表单草稿
  getDraft: async (taskId: string): Promise<TaskFormSubmission | null> => {
    try {
      const response = await apiClient.get<ApiResponse<TaskFormSubmission | null>>(`/my/tasks/${taskId}/form-draft`);
      return response.data.data;
    } catch {
      return null;
    }
  },

  // BOM解析预览（不保存）
  parseBOM: async (file: File): Promise<{ items: ParsedBOMItem[] }> => {
    const formData = new FormData();
    formData.append('file', file);
    const response = await apiClient.post<ApiResponse<{ items: ParsedBOMItem[] }>>('/bom/parse', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return response.data.data;
  },
};

export interface ParsedBOMItem {
  item_number: number;
  reference: string;
  name: string;
  specification: string;
  quantity: number;
  unit: string;
  category: string;
  manufacturer: string;
  manufacturer_pn: string;
}
