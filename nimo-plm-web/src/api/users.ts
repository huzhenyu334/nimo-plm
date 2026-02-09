import apiClient from './client';

export interface User {
  id: string;
  name: string;
  email?: string;
  avatar_url?: string;
  department_id?: string;
  feishu_open_id?: string;
  status: string;
}

export const userApi = {
  list: async (): Promise<User[]> => {
    const response = await apiClient.get('/users');
    return response.data.data?.items || response.data.data?.users || [];
  },
  search: async (q: string): Promise<User[]> => {
    const response = await apiClient.get('/users/search', { params: { q } });
    return response.data.data?.items || response.data.data || [];
  },
  get: async (id: string): Promise<User> => {
    const response = await apiClient.get(`/users/${id}`);
    return response.data.data;
  },
};
