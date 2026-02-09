import apiClient from './client';
import { ApiResponse, Product, User } from '@/types';

export interface ECN {
  id: string;
  code: string;
  title: string;
  product_id: string;
  change_type: string;
  urgency: string;
  status: string;
  reason: string;
  description: string;
  impact_analysis: string;
  requested_by: string;
  requested_at: string;
  approved_by: string;
  approved_at: string;
  rejection_reason: string;
  implemented_by: string;
  implemented_at: string;
  feishu_approval_code: string;
  feishu_instance_code: string;
  created_at: string;
  updated_at: string;
  product?: Product;
  requester?: User;
  approver?: User;
  implementer?: User;
  affected_items?: ECNAffectedItem[];
  approvals?: ECNApproval[];
}

export interface ECNAffectedItem {
  id: string;
  ecn_id: string;
  item_type: string;
  item_id: string;
  before_value: Record<string, any>;
  after_value: Record<string, any>;
  change_description: string;
  created_at: string;
}

export interface ECNApproval {
  id: string;
  ecn_id: string;
  approver_id: string;
  sequence: number;
  status: string;
  comment: string;
  approved_at: string;
  approver?: User;
}

export interface ECNListResponse {
  items: ECN[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface CreateECNRequest {
  title: string;
  product_id: string;
  change_type: string;
  urgency?: string;
  reason: string;
  description?: string;
  impact_analysis?: string;
  affected_items?: Partial<ECNAffectedItem>[];
  approver_ids?: string[];
}

export const ecnApi = {
  list: async (params?: { status?: string; product_id?: string; page?: number; page_size?: number }): Promise<ECNListResponse> => {
    const searchParams = new URLSearchParams();
    if (params?.status) searchParams.set('status', params.status);
    if (params?.product_id) searchParams.set('product_id', params.product_id);
    if (params?.page) searchParams.set('page', params.page.toString());
    if (params?.page_size) searchParams.set('page_size', params.page_size.toString());
    const query = searchParams.toString();
    const response = await apiClient.get<ApiResponse<ECNListResponse>>(`/ecns${query ? `?${query}` : ''}`);
    return response.data.data;
  },

  get: async (id: string): Promise<ECN> => {
    const response = await apiClient.get<ApiResponse<ECN>>(`/ecns/${id}`);
    return response.data.data;
  },

  create: async (data: CreateECNRequest): Promise<ECN> => {
    const response = await apiClient.post<ApiResponse<ECN>>('/ecns', data);
    return response.data.data;
  },

  update: async (id: string, data: Partial<ECN>): Promise<ECN> => {
    const response = await apiClient.put<ApiResponse<ECN>>(`/ecns/${id}`, data);
    return response.data.data;
  },

  submit: async (id: string): Promise<ECN> => {
    const response = await apiClient.post<ApiResponse<ECN>>(`/ecns/${id}/submit`, {});
    return response.data.data;
  },

  approve: async (id: string, data: { comment?: string }): Promise<ECN> => {
    const response = await apiClient.post<ApiResponse<ECN>>(`/ecns/${id}/approve`, data);
    return response.data.data;
  },

  reject: async (id: string, data: { comment: string }): Promise<ECN> => {
    const response = await apiClient.post<ApiResponse<ECN>>(`/ecns/${id}/reject`, data);
    return response.data.data;
  },

  implement: async (id: string): Promise<ECN> => {
    const response = await apiClient.post<ApiResponse<ECN>>(`/ecns/${id}/implement`, {});
    return response.data.data;
  },

  listAffectedItems: async (id: string): Promise<{ items: ECNAffectedItem[] }> => {
    const response = await apiClient.get<ApiResponse<{ items: ECNAffectedItem[] }>>(`/ecns/${id}/affected-items`);
    return response.data.data;
  },

  addAffectedItem: async (id: string, data: Partial<ECNAffectedItem>): Promise<ECNAffectedItem> => {
    const response = await apiClient.post<ApiResponse<ECNAffectedItem>>(`/ecns/${id}/affected-items`, data);
    return response.data.data;
  },

  removeAffectedItem: async (id: string, itemId: string): Promise<void> => {
    await apiClient.delete(`/ecns/${id}/affected-items/${itemId}`);
  },

  listApprovals: async (id: string): Promise<{ approvals: ECNApproval[] }> => {
    const response = await apiClient.get<ApiResponse<{ approvals: ECNApproval[] }>>(`/ecns/${id}/approvals`);
    return response.data.data;
  },

  addApprover: async (id: string, data: { approver_id: string; sequence?: number }): Promise<ECNApproval> => {
    const response = await apiClient.post<ApiResponse<ECNApproval>>(`/ecns/${id}/approvers`, data);
    return response.data.data;
  },
};
