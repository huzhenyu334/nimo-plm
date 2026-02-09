import apiClient from './client';

export interface FormField {
  key: string;
  type: 'text' | 'textarea' | 'number' | 'money' | 'select' | 'multiselect' | 'date' | 'daterange' | 'user' | 'attachment' | 'table' | 'description';
  label: string;
  required?: boolean;
  placeholder?: string;
  options?: string[];
  prefix?: string;
  multiple?: boolean;
  columns?: { key: string; label: string; type: string }[];
}

export interface FlowNode {
  type: 'submit' | 'approve' | 'end';
  name: string;
  config: {
    submitter?: string;
    approver_type?: 'supervisor' | 'dept_leader' | 'designated' | 'self_select' | 'submitter' | 'role';
    approver_ids?: string[];
    multi_approve?: 'all' | 'any' | 'sequential';
    select_range?: string;
    when_self?: string;
    cc_users?: string[];
  };
}

export interface ApprovalDefinition {
  id: string;
  code: string;
  name: string;
  description?: string;
  icon: string;
  group_name: string;
  form_schema: FormField[];
  flow_schema: { nodes: FlowNode[] };
  visibility: string;
  status: string;
  created_at: string;
}

export interface ApprovalGroup {
  id: string;
  name: string;
  sort_order: number;
}

export interface ApprovalInstance {
  id: string;
  definition_id: string;
  definition_name: string;
  status: string;
  form_data: Record<string, any>;
  submitted_by: string;
  requester?: { id: string; name: string; avatar_url?: string };
  current_step: number;
  steps: ApprovalStep[];
  created_at: string;
  updated_at: string;
  definition?: ApprovalDefinition;
}

export interface ApprovalStep {
  node_index: number;
  node_name: string;
  node_type: string;
  status: string;
  approvers: StepApprover[];
  started_at?: string;
  completed_at?: string;
}

export interface StepApprover {
  user_id: string;
  user?: { id: string; name: string; avatar_url?: string };
  status: string;
  comment?: string;
  decided_at?: string;
}

export const approvalDefinitionApi = {
  list: async (): Promise<{ groups: { name: string; definitions: ApprovalDefinition[] }[] }> => {
    const res = await apiClient.get('/approval-definitions');
    return res.data.data;
  },
  get: async (id: string): Promise<ApprovalDefinition> => {
    const res = await apiClient.get(`/approval-definitions/${id}`);
    return res.data.data;
  },
  create: async (data: Partial<ApprovalDefinition>): Promise<ApprovalDefinition> => {
    const res = await apiClient.post('/approval-definitions', data);
    return res.data.data;
  },
  update: async (id: string, data: Partial<ApprovalDefinition>): Promise<ApprovalDefinition> => {
    const res = await apiClient.put(`/approval-definitions/${id}`, data);
    return res.data.data;
  },
  delete: async (id: string) => {
    const res = await apiClient.delete(`/approval-definitions/${id}`);
    return res.data;
  },
  publish: async (id: string) => {
    const res = await apiClient.post(`/approval-definitions/${id}/publish`);
    return res.data;
  },
  unpublish: async (id: string) => {
    const res = await apiClient.post(`/approval-definitions/${id}/unpublish`);
    return res.data;
  },
  submit: async (id: string, data: { form_data: Record<string, any>; approver_ids?: string[] }) => {
    const res = await apiClient.post(`/approval-definitions/${id}/submit`, data);
    return res.data.data;
  },
};

export const approvalGroupApi = {
  list: async (): Promise<ApprovalGroup[]> => {
    const res = await apiClient.get('/approval-groups');
    return res.data.data?.items || res.data.data || [];
  },
  create: async (name: string): Promise<ApprovalGroup> => {
    const res = await apiClient.post('/approval-groups', { name });
    return res.data.data;
  },
  delete: async (id: string) => {
    const res = await apiClient.delete(`/approval-groups/${id}`);
    return res.data;
  },
};

export const approvalInstanceApi = {
  list: async (params?: { status?: string; type?: string }): Promise<ApprovalInstance[]> => {
    const res = await apiClient.get('/approvals', { params });
    return res.data.data?.items || res.data.data || [];
  },
  get: async (id: string): Promise<ApprovalInstance> => {
    const res = await apiClient.get(`/approvals/${id}`);
    return res.data.data;
  },
  approve: async (id: string, comment?: string) => {
    const res = await apiClient.post(`/approvals/${id}/approve`, { comment });
    return res.data;
  },
  reject: async (id: string, comment: string) => {
    const res = await apiClient.post(`/approvals/${id}/reject`, { comment });
    return res.data;
  },
};
