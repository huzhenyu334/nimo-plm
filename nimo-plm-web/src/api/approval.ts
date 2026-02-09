import apiClient from './client';

export interface ApprovalReviewer {
  id: string;
  user_id: string;
  status: string;
  comment?: string;
  decided_at?: string;
  user?: { id: string; name: string; avatar_url?: string };
}

export interface ApprovalRequest {
  id: string;
  project_id: string;
  task_id: string;
  title: string;
  description?: string;
  type: string;
  status: string;
  form_data?: any;
  result?: string;
  result_comment?: string;
  requested_by: string;
  created_at: string;
  updated_at: string;
  reviewers?: ApprovalReviewer[];
  requester?: { id: string; name: string };
  task?: { id: string; title: string };
  project?: { id: string; name: string };
}

export const approvalApi = {
  create: async (data: { project_id: string; task_id: string; title: string; description?: string; reviewer_ids: string[] }) => {
    const response = await apiClient.post('/approvals', data);
    return response.data.data;
  },
  list: async (params?: { status?: string; my_pending?: boolean }) => {
    const response = await apiClient.get('/approvals', { params });
    return response.data.data;
  },
  get: async (id: string) => {
    const response = await apiClient.get(`/approvals/${id}`);
    return response.data.data;
  },
  approve: async (id: string, comment?: string) => {
    const response = await apiClient.post(`/approvals/${id}/approve`, { comment });
    return response.data;
  },
  reject: async (id: string, comment: string) => {
    const response = await apiClient.post(`/approvals/${id}/reject`, { comment });
    return response.data;
  },
};
