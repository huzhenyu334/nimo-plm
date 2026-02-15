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
  technical_plan: string;
  planned_date: string;
  completion_rate: number;
  approval_mode: string;
  sop_impact: Record<string, any>;
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
  tasks?: ECNTask[];
}

export interface ECNAffectedItem {
  id: string;
  ecn_id: string;
  item_type: string;
  item_id: string;
  material_code: string;
  material_name: string;
  affected_bom_ids: Record<string, any>;
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
  decision: string;
  comment: string;
  decided_at: string;
  created_at: string;
  approver?: User;
}

export interface ECNTask {
  id: string;
  ecn_id: string;
  type: string;
  title: string;
  description: string;
  assignee_id: string;
  due_date: string;
  status: string;
  completed_at: string;
  completed_by: string;
  metadata: Record<string, any>;
  sort_order: number;
  created_at: string;
  updated_at: string;
  assignee?: User;
}

export interface ECNHistory {
  id: string;
  ecn_id: string;
  action: string;
  user_id: string;
  detail: Record<string, any>;
  created_at: string;
  user?: User;
}

export interface ECNStats {
  pending_approval: number;
  executing: number;
  month_created: number;
  month_closed: number;
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
  technical_plan?: string;
  planned_date?: string;
  approval_mode?: string;
  sop_impact?: Record<string, any>;
  affected_items?: Partial<ECNAffectedItem>[];
  approver_ids?: string[];
}

export const ecnApi = {
  list: async (params?: { status?: string; product_id?: string; keyword?: string; change_type?: string; urgency?: string; page?: number; page_size?: number }): Promise<ECNListResponse> => {
    const searchParams = new URLSearchParams();
    if (params?.status) searchParams.set('status', params.status);
    if (params?.product_id) searchParams.set('product_id', params.product_id);
    if (params?.keyword) searchParams.set('keyword', params.keyword);
    if (params?.change_type) searchParams.set('change_type', params.change_type);
    if (params?.urgency) searchParams.set('urgency', params.urgency);
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

  reject: async (id: string, data: { reason: string }): Promise<ECN> => {
    const response = await apiClient.post<ApiResponse<ECN>>(`/ecns/${id}/reject`, data);
    return response.data.data;
  },

  implement: async (id: string): Promise<ECN> => {
    const response = await apiClient.post<ApiResponse<ECN>>(`/ecns/${id}/implement`, {});
    return response.data.data;
  },

  // 统计
  getStats: async (): Promise<ECNStats> => {
    const response = await apiClient.get<ApiResponse<ECNStats>>('/ecns/stats');
    return response.data.data;
  },

  getMyPending: async (): Promise<{ items: ECN[] }> => {
    const response = await apiClient.get<ApiResponse<{ items: ECN[] }>>('/ecns/my-pending');
    return response.data.data;
  },

  // 受影响项
  listAffectedItems: async (id: string): Promise<ECNAffectedItem[]> => {
    const response = await apiClient.get<ApiResponse<ECNAffectedItem[]>>(`/ecns/${id}/affected-items`);
    return response.data.data;
  },

  addAffectedItem: async (id: string, data: Partial<ECNAffectedItem>): Promise<ECNAffectedItem> => {
    const response = await apiClient.post<ApiResponse<ECNAffectedItem>>(`/ecns/${id}/affected-items`, data);
    return response.data.data;
  },

  updateAffectedItem: async (id: string, itemId: string, data: Partial<ECNAffectedItem>): Promise<ECNAffectedItem> => {
    const response = await apiClient.put<ApiResponse<ECNAffectedItem>>(`/ecns/${id}/affected-items/${itemId}`, data);
    return response.data.data;
  },

  removeAffectedItem: async (id: string, itemId: string): Promise<void> => {
    await apiClient.delete(`/ecns/${id}/affected-items/${itemId}`);
  },

  // 审批
  listApprovals: async (id: string): Promise<ECNApproval[]> => {
    const response = await apiClient.get<ApiResponse<ECNApproval[]>>(`/ecns/${id}/approvals`);
    return response.data.data;
  },

  addApprover: async (id: string, data: { approver_id: string; sequence?: number }): Promise<ECNApproval> => {
    const response = await apiClient.post<ApiResponse<ECNApproval>>(`/ecns/${id}/approvers`, data);
    return response.data.data;
  },

  // 执行任务
  listTasks: async (id: string): Promise<{ items: ECNTask[] }> => {
    const response = await apiClient.get<ApiResponse<{ items: ECNTask[] }>>(`/ecns/${id}/tasks`);
    return response.data.data;
  },

  createTask: async (id: string, data: Partial<ECNTask>): Promise<ECNTask> => {
    const response = await apiClient.post<ApiResponse<ECNTask>>(`/ecns/${id}/tasks`, data);
    return response.data.data;
  },

  updateTask: async (id: string, taskId: string, data: Partial<ECNTask>): Promise<ECNTask> => {
    const response = await apiClient.put<ApiResponse<ECNTask>>(`/ecns/${id}/tasks/${taskId}`, data);
    return response.data.data;
  },

  // BOM变更应用
  applyBOMChanges: async (id: string): Promise<void> => {
    await apiClient.post(`/ecns/${id}/apply-bom-changes`, {});
  },

  // 操作历史
  listHistory: async (id: string): Promise<{ items: ECNHistory[] }> => {
    const response = await apiClient.get<ApiResponse<{ items: ECNHistory[] }>>(`/ecns/${id}/history`);
    return response.data.data;
  },
};
