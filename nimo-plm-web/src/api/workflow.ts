import apiClient from './client';

export interface TaskActionLog {
  id: string;
  action: string;
  from_status: string;
  to_status: string;
  operator_id: string;
  operator_type: string;
  comment?: string;
  event_data?: any;
  created_at: string;
}

export interface RoleAssignment {
  role_code: string;
  user_id: string;
  feishu_user_id?: string;
}

export interface ReviewOutcome {
  outcome_code: string;
  outcome_name: string;
  outcome_type: string; // pass, fail, fail_rollback
  rollback_to_task_code?: string;
}

export const workflowApi = {
  // 指派任务
  assignTask: async (projectId: string, taskId: string, data: { assignee_id: string; feishu_user_id?: string }) => {
    const response = await apiClient.post(`/projects/${projectId}/tasks/${taskId}/assign`, data);
    return response.data;
  },

  // 开始任务
  startTask: async (projectId: string, taskId: string) => {
    const response = await apiClient.post(`/projects/${projectId}/tasks/${taskId}/start`);
    return response.data;
  },

  // 完成任务
  completeTask: async (projectId: string, taskId: string) => {
    const response = await apiClient.post(`/projects/${projectId}/tasks/${taskId}/complete`);
    return response.data;
  },

  // 提交评审
  submitReview: async (projectId: string, taskId: string, data: { outcome_code: string; comment?: string }) => {
    const response = await apiClient.post(`/projects/${projectId}/tasks/${taskId}/review`, data);
    return response.data;
  },

  // 指派阶段角色
  assignPhaseRoles: async (projectId: string, phase: string, data: { assignments: RoleAssignment[] }) => {
    const response = await apiClient.post(`/projects/${projectId}/phases/${phase}/assign-roles`, data);
    return response.data;
  },

  // 获取任务操作历史
  getTaskHistory: async (projectId: string, taskId: string): Promise<TaskActionLog[]> => {
    const response = await apiClient.get(`/projects/${projectId}/tasks/${taskId}/history`);
    return response.data.data || [];
  },
};
