import apiClient from './client';
import { ApiResponse } from '@/types';

export interface Deliverable {
  id: string;
  project_id: string;
  phase: string;
  name: string;
  description?: string;
  status: 'not_started' | 'in_progress' | 'submitted' | 'approved' | 'rejected';
  assignee_role?: string;
  assignee_id?: string;
  assignee_name?: string;
  due_date?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface DeliverableListResponse {
  items: Deliverable[];
  total: number;
}

export const deliverablesApi = {
  // 获取交付物列表
  list: async (projectId: string, phase?: string): Promise<DeliverableListResponse> => {
    const params = phase ? `?phase=${phase}` : '';
    const response = await apiClient.get<ApiResponse<DeliverableListResponse>>(
      `/projects/${projectId}/deliverables${params}`
    );
    return response.data.data;
  },

  // 更新交付物状态
  update: async (projectId: string, deliverableId: string, data: Partial<Deliverable>): Promise<Deliverable> => {
    const response = await apiClient.put<ApiResponse<Deliverable>>(
      `/projects/${projectId}/deliverables/${deliverableId}`, data
    );
    return response.data.data;
  },
};
