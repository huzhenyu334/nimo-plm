import apiClient from './client';
import { ApiResponse, AuthTokens, User } from '@/types';

export const authApi = {
  // 获取飞书登录 URL
  getFeishuLoginUrl: () => {
    window.location.href = '/api/v1/auth/feishu/login';
  },

  // 获取当前用户
  getCurrentUser: async (): Promise<User> => {
    const response = await apiClient.get<ApiResponse<User>>('/auth/me');
    return response.data.data;
  },

  // 刷新 Token
  refreshToken: async (refreshToken: string): Promise<AuthTokens> => {
    const response = await apiClient.post<ApiResponse<AuthTokens>>('/auth/refresh', {
      refresh_token: refreshToken,
    });
    return response.data.data;
  },

  // 登出
  logout: async (): Promise<void> => {
    await apiClient.post('/auth/logout');
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
  },
};
