import apiClient from '@/api/client';

export interface Role {
  id: string;
  code: string;
  name: string;
  sort_order: number;
}

// 默认角色（fallback，API 不可用时使用）
export const DEFAULT_ROLE_CODES: Role[] = [
  { id: '', code: 'project_manager', name: '项目经理', sort_order: 1 },
  { id: '', code: 'hardware_engineer', name: '硬件工程师', sort_order: 2 },
  { id: '', code: 'software_engineer', name: '软件工程师', sort_order: 3 },
  { id: '', code: 'mechanical_engineer', name: '结构工程师', sort_order: 4 },
  { id: '', code: 'quality_engineer', name: '品质工程师', sort_order: 5 },
  { id: '', code: 'reviewer', name: '评审人', sort_order: 6 },
];

// 兼容旧接口
export const ROLE_CODES = DEFAULT_ROLE_CODES.map(r => ({ code: r.code, label: r.name }));

// API 方法
export const roleApi = {
  list: async (): Promise<Role[]> => {
    const response = await apiClient.get('/roles');
    return response.data.data?.items || [];
  },
  create: async (data: { code: string; name: string; sort_order?: number }): Promise<Role> => {
    const response = await apiClient.post('/roles', data);
    return response.data.data;
  },
  update: async (id: string, data: { code?: string; name?: string; sort_order?: number }): Promise<Role> => {
    const response = await apiClient.put(`/roles/${id}`, data);
    return response.data.data;
  },
  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/roles/${id}`);
  },
};

// 任务角色 API（用于模板任务分配，区别于权限角色）
export interface TaskRole {
  id: string;
  code: string;
  name: string;
  is_system: boolean;
  sort_order: number;
}

export const taskRoleApi = {
  list: async (): Promise<TaskRole[]> => {
    const response = await apiClient.get('/task-roles');
    return response.data.data?.items || [];
  },
};

// 飞书角色（部门）API
export interface FeishuRole {
  code: string;
  name: string;
}

export const feishuRoleApi = {
  list: async (): Promise<FeishuRole[]> => {
    const response = await apiClient.get('/feishu/roles');
    return response.data.data?.items || [];
  },
};
