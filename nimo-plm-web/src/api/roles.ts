import apiClient from './client';

export interface Role {
  id: string;
  code: string;
  name: string;
  description?: string;
  is_system: boolean;
  status: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface RoleMember {
  user_id: string;
  name: string;
  email: string;
  avatar_url: string;
  department_name: string;
}

export interface DepartmentUser {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
}

export interface DepartmentTreeNode {
  id: string;
  name: string;
  parent_id: string;
  children: DepartmentTreeNode[];
  users: DepartmentUser[];
}

export const roleApi = {
  list: async (): Promise<Role[]> => {
    const res = await apiClient.get('/roles');
    return res.data.data?.items || [];
  },
  create: async (data: { code: string; name: string; sort_order?: number }): Promise<Role> => {
    const res = await apiClient.post('/roles', data);
    return res.data.data;
  },
  update: async (id: string, data: { code?: string; name?: string; sort_order?: number }): Promise<Role> => {
    const res = await apiClient.put(`/roles/${id}`, data);
    return res.data.data;
  },
  delete: async (id: string) => {
    const res = await apiClient.delete(`/roles/${id}`);
    return res.data;
  },
  listMembers: async (roleId: string): Promise<RoleMember[]> => {
    const res = await apiClient.get(`/roles/${roleId}/members`);
    return res.data.data?.items || [];
  },
  addMembers: async (roleId: string, userIds: string[]) => {
    const res = await apiClient.post(`/roles/${roleId}/members`, { user_ids: userIds });
    return res.data;
  },
  removeMembers: async (roleId: string, userIds: string[]) => {
    const res = await apiClient.delete(`/roles/${roleId}/members`, { data: { user_ids: userIds } });
    return res.data;
  },
};

export const departmentApi = {
  tree: async (): Promise<DepartmentTreeNode[]> => {
    const res = await apiClient.get('/departments');
    return res.data.data?.items || [];
  },
};
