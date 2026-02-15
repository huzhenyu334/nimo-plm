import apiClient from './client';
import { ApiResponse } from '@/types';

// 项目BOM相关类型
export interface ProjectBOM {
  id: string;
  project_id: string;
  phase_id?: string;
  task_id?: string;
  name: string;
  bom_type: 'EBOM' | 'PBOM' | 'MBOM';
  source_bom_id?: string;
  source_version?: string;
  version: string;
  version_major: number;
  version_minor: number;
  status: 'draft' | 'released' | 'obsolete' | 'pending_review' | 'published' | 'rejected' | 'frozen';
  description?: string;
  release_note?: string;
  released_at?: string;
  released_by?: string;
  submitted_by?: string;
  submitted_at?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  review_comment?: string;
  approved_by?: string;
  approved_at?: string;
  frozen_at?: string;
  frozen_by?: string;
  total_items: number;
  estimated_cost?: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  // Relations
  phase?: { id: string; phase: string; name?: string };
  creator?: { id: string; name: string; avatar_url?: string };
  submitter?: { id: string; name: string };
  reviewer?: { id: string; name: string };
}

export interface ProjectBOMItem {
  id: string;
  bom_id: string;
  item_number: number;
  parent_item_id?: string;
  level: number;
  material_id?: string;
  category: string;
  sub_category?: string;
  name: string;
  specification?: string;
  quantity: number;
  unit: string;
  reference?: string;
  manufacturer?: string;
  manufacturer_pn?: string;
  supplier?: string;
  supplier_pn?: string;
  unit_price?: number;
  extended_cost?: number;
  lead_time_days?: number;
  notes?: string;
  drawing_no?: string;
  is_critical: boolean;
  is_alternative: boolean;
  alternative_for?: string;
  attachments?: string;
  thumbnail_url?: string;
  // 动态扩展属性（由属性模板决定）
  extended_attrs?: Record<string, any>;
  // 工艺关联
  process_step_id?: string;
  scrap_rate?: number;
  effective_date?: string;
  expire_date?: string;
  created_at: string;
  updated_at: string;
  // Relations
  material?: { id: string; name: string; code: string; specification?: string };
  children?: ProjectBOMItem[];
}

export interface LangVariant {
  id: string;
  bom_item_id: string;
  variant_index: number;
  material_code?: string;
  language_code: string;
  language_name: string;
  design_file_id?: string;
  design_file_name?: string;
  design_file_url?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface ProjectBOMDetail extends ProjectBOM {
  items: ProjectBOMItem[];
}

export interface CreateProjectBOMRequest {
  name: string;
  bom_type: string;
  phase_id?: string;
  version?: string;
  description?: string;
}

export interface BOMItemRequest {
  material_id?: string;
  parent_item_id?: string;
  level?: number;
  category?: string;
  sub_category?: string;
  name: string;
  specification?: string;
  quantity: number;
  unit?: string;
  reference?: string;
  manufacturer?: string;
  manufacturer_pn?: string;
  supplier?: string;
  supplier_pn?: string;
  unit_price?: number;
  lead_time_days?: number;
  is_critical?: boolean;
  is_alternative?: boolean;
  is_appearance_part?: boolean;
  notes?: string;
  drawing_no?: string;
  item_number?: number;
  attachments?: string;
  // 动态扩展属性
  extended_attrs?: Record<string, any>;
}

// 属性模板
export interface CategoryAttrTemplate {
  id: string;
  category: string;
  sub_category: string;
  bom_type: 'EBOM' | 'PBOM';
  field_key: string;
  field_name: string;
  field_type: 'text' | 'number' | 'select' | 'boolean' | 'date' | 'file' | 'thumbnail';
  unit?: string;
  required: boolean;
  options?: Record<string, any>;
  validation?: Record<string, any>;
  default_value?: string;
  sort_order: number;
  show_in_table: boolean;
  created_at: string;
  updated_at: string;
}

// 工艺路线
export interface ProcessRoute {
  id: string;
  project_id: string;
  bom_id?: string;
  name: string;
  description?: string;
  version: string;
  status: string;
  created_by?: string;
  steps?: ProcessStep[];
  created_at: string;
  updated_at: string;
}

export interface ProcessStep {
  id: string;
  route_id: string;
  step_number: number;
  name: string;
  description?: string;
  equipment?: string;
  cycle_time_seconds?: number;
  setup_time_minutes?: number;
  quality_checks?: string;
  sort_order: number;
  materials?: ProcessStepMaterial[];
  created_at: string;
  updated_at: string;
}

export interface ProcessStepMaterial {
  id: string;
  step_id: string;
  material_id?: string;
  bom_item_id?: string;
  name: string;
  quantity: number;
  unit: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

// 分类树节点
export interface CategoryTreeNode {
  category: string;
  sub_category: string;
  count: number;
}

export const projectBomApi = {
  // 获取项目BOM列表
  list: async (projectId: string, params?: { phase?: string; bom_type?: string; status?: string }): Promise<ProjectBOM[]> => {
    const searchParams = new URLSearchParams();
    if (params?.phase) searchParams.set('phase', params.phase);
    if (params?.bom_type) searchParams.set('bom_type', params.bom_type);
    if (params?.status) searchParams.set('status', params.status);
    const query = searchParams.toString();
    const response = await apiClient.get<ApiResponse<ProjectBOM[]>>(
      `/projects/${projectId}/boms${query ? `?${query}` : ''}`
    );
    return response.data.data;
  },

  // 创建BOM
  create: async (projectId: string, data: CreateProjectBOMRequest): Promise<ProjectBOM> => {
    const response = await apiClient.post<ApiResponse<ProjectBOM>>(
      `/projects/${projectId}/boms`, data
    );
    return response.data.data;
  },

  // 获取BOM详情（含items）
  get: async (projectId: string, bomId: string): Promise<ProjectBOMDetail> => {
    const response = await apiClient.get<ApiResponse<ProjectBOMDetail>>(
      `/projects/${projectId}/boms/${bomId}`
    );
    return response.data.data;
  },

  // 更新BOM基本信息
  update: async (projectId: string, bomId: string, data: { name?: string; description?: string; version?: string }): Promise<ProjectBOM> => {
    const response = await apiClient.put<ApiResponse<ProjectBOM>>(
      `/projects/${projectId}/boms/${bomId}`, data
    );
    return response.data.data;
  },

  // 提交审批
  submit: async (projectId: string, bomId: string): Promise<ProjectBOM> => {
    const response = await apiClient.post<ApiResponse<ProjectBOM>>(
      `/projects/${projectId}/boms/${bomId}/submit`, {}
    );
    return response.data.data;
  },

  // 审批通过
  approve: async (projectId: string, bomId: string, comment?: string): Promise<ProjectBOM> => {
    const response = await apiClient.post<ApiResponse<ProjectBOM>>(
      `/projects/${projectId}/boms/${bomId}/approve`, { comment }
    );
    return response.data.data;
  },

  // 审批驳回
  reject: async (projectId: string, bomId: string, comment: string): Promise<ProjectBOM> => {
    const response = await apiClient.post<ApiResponse<ProjectBOM>>(
      `/projects/${projectId}/boms/${bomId}/reject`, { comment }
    );
    return response.data.data;
  },

  // 冻结
  freeze: async (projectId: string, bomId: string): Promise<ProjectBOM> => {
    const response = await apiClient.post<ApiResponse<ProjectBOM>>(
      `/projects/${projectId}/boms/${bomId}/freeze`, {}
    );
    return response.data.data;
  },

  // 添加物料行项
  addItem: async (projectId: string, bomId: string, data: BOMItemRequest): Promise<ProjectBOMItem> => {
    const response = await apiClient.post<ApiResponse<ProjectBOMItem>>(
      `/projects/${projectId}/boms/${bomId}/items`, data
    );
    return response.data.data;
  },

  // 批量添加物料行项
  batchAddItems: async (projectId: string, bomId: string, items: BOMItemRequest[]): Promise<{ created: number }> => {
    const response = await apiClient.post<ApiResponse<{ created: number }>>(
      `/projects/${projectId}/boms/${bomId}/items/batch`, { items }
    );
    return response.data.data;
  },

  // 更新物料行项
  updateItem: async (projectId: string, bomId: string, itemId: string, data: BOMItemRequest): Promise<ProjectBOMItem> => {
    const response = await apiClient.put<ApiResponse<ProjectBOMItem>>(
      `/projects/${projectId}/boms/${bomId}/items/${itemId}`, data
    );
    return response.data.data;
  },

  // 删除物料行项
  deleteItem: async (projectId: string, bomId: string, itemId: string): Promise<void> => {
    await apiClient.delete(`/projects/${projectId}/boms/${bomId}/items/${itemId}`);
  },

  // 拖拽排序
  reorderItems: async (projectId: string, bomId: string, itemIds: string[]): Promise<void> => {
    await apiClient.post(`/projects/${projectId}/boms/${bomId}/reorder`, { item_ids: itemIds });
  },

  // Excel导出（返回blob下载）
  exportExcel: async (projectId: string, bomId: string): Promise<void> => {
    const response = await apiClient.get(`/projects/${projectId}/boms/${bomId}/export`, { responseType: 'blob' });
    const url = window.URL.createObjectURL(new Blob([response.data]));
    const a = document.createElement('a');
    a.href = url;
    a.download = `BOM_${bomId}.xlsx`;
    a.click();
    window.URL.revokeObjectURL(url);
  },

  // Excel导入（multipart/form-data）
  importExcel: async (projectId: string, bomId: string, file: File): Promise<any> => {
    const formData = new FormData();
    formData.append('file', file);
    const response = await apiClient.post(`/projects/${projectId}/boms/${bomId}/import`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return response.data.data;
  },

  // 下载导入模板
  downloadTemplate: async (bomType?: string): Promise<void> => {
    const query = bomType ? `?bom_type=${bomType}` : '';
    const response = await apiClient.get(`/bom-template${query}`, { responseType: 'blob' });
    const url = window.URL.createObjectURL(new Blob([response.data]));
    const a = document.createElement('a');
    a.href = url;
    a.download = `BOM导入模板${bomType ? `_${bomType}` : ''}.xlsx`;
    a.click();
    window.URL.revokeObjectURL(url);
  },

  // EBOM转MBOM
  convertToMBOM: async (projectId: string, bomId: string): Promise<any> => {
    const response = await apiClient.post(`/projects/${projectId}/boms/${bomId}/convert-to-mbom`, {});
    return response.data.data;
  },

  // EBOM转PBOM
  convertToPBOM: async (projectId: string, bomId: string): Promise<any> => {
    const response = await apiClient.post(`/projects/${projectId}/boms/${bomId}/convert-to-pbom`, {});
    return response.data.data;
  },

  // 发布BOM
  release: async (projectId: string, bomId: string, releaseNote?: string): Promise<ProjectBOM> => {
    const response = await apiClient.post<ApiResponse<ProjectBOM>>(
      `/projects/${projectId}/boms/${bomId}/release`, { release_note: releaseNote || '' }
    );
    return response.data.data;
  },

  // 从上游BOM创建（EBOM→PBOM, PBOM→MBOM）
  createFrom: async (projectId: string, sourceBomId: string, targetType: string): Promise<ProjectBOM> => {
    const response = await apiClient.post<ApiResponse<ProjectBOM>>(
      `/projects/${projectId}/boms/create-from`, { source_bom_id: sourceBomId, target_type: targetType }
    );
    return response.data.data;
  },

  // BOM版本对比
  compareBOMs: async (bomId1: string, bomId2: string): Promise<any> => {
    const response = await apiClient.get(`/bom-compare?bom1=${bomId1}&bom2=${bomId2}`);
    return response.data.data;
  },

  // 分类树
  getCategoryTree: async (projectId: string, bomId: string): Promise<CategoryTreeNode[]> => {
    const response = await apiClient.get<ApiResponse<CategoryTreeNode[]>>(
      `/projects/${projectId}/boms/${bomId}/category-tree`
    );
    return response.data.data;
  },

  // ========== 属性模板 ==========
  listTemplates: async (params?: { category?: string; sub_category?: string }): Promise<CategoryAttrTemplate[]> => {
    const searchParams = new URLSearchParams();
    if (params?.category) searchParams.set('category', params.category);
    if (params?.sub_category) searchParams.set('sub_category', params.sub_category);
    const query = searchParams.toString();
    const response = await apiClient.get<ApiResponse<CategoryAttrTemplate[]>>(
      `/bom-attr-templates${query ? `?${query}` : ''}`
    );
    return response.data.data;
  },

  createTemplate: async (data: Omit<CategoryAttrTemplate, 'id' | 'created_at' | 'updated_at'>): Promise<CategoryAttrTemplate> => {
    const response = await apiClient.post<ApiResponse<CategoryAttrTemplate>>('/bom-attr-templates', data);
    return response.data.data;
  },

  updateTemplate: async (id: string, data: Partial<CategoryAttrTemplate>): Promise<CategoryAttrTemplate> => {
    const response = await apiClient.put<ApiResponse<CategoryAttrTemplate>>(`/bom-attr-templates/${id}`, data);
    return response.data.data;
  },

  deleteTemplate: async (id: string): Promise<void> => {
    await apiClient.delete(`/bom-attr-templates/${id}`);
  },

  seedTemplates: async (): Promise<void> => {
    await apiClient.post('/bom-attr-templates/seed', {});
  },

  // ========== 工艺路线 ==========
  listRoutes: async (projectId: string): Promise<ProcessRoute[]> => {
    const response = await apiClient.get<ApiResponse<ProcessRoute[]>>(`/projects/${projectId}/routes`);
    return response.data.data;
  },

  getRoute: async (projectId: string, routeId: string): Promise<ProcessRoute> => {
    const response = await apiClient.get<ApiResponse<ProcessRoute>>(`/projects/${projectId}/routes/${routeId}`);
    return response.data.data;
  },

  createRoute: async (projectId: string, bomId: string, data: { name: string; description?: string }): Promise<ProcessRoute> => {
    const response = await apiClient.post<ApiResponse<ProcessRoute>>(
      `/projects/${projectId}/boms/${bomId}/routes`, data
    );
    return response.data.data;
  },

  updateRoute: async (projectId: string, routeId: string, data: { name?: string; description?: string; status?: string }): Promise<ProcessRoute> => {
    const response = await apiClient.put<ApiResponse<ProcessRoute>>(
      `/projects/${projectId}/routes/${routeId}`, data
    );
    return response.data.data;
  },

  // ========== 工序 ==========
  createStep: async (projectId: string, routeId: string, data: {
    name: string; step_number: number; description?: string; equipment?: string;
    cycle_time_seconds?: number; setup_time_minutes?: number; quality_checks?: string; sort_order?: number;
  }): Promise<ProcessStep> => {
    const response = await apiClient.post<ApiResponse<ProcessStep>>(
      `/projects/${projectId}/routes/${routeId}/steps`, data
    );
    return response.data.data;
  },

  updateStep: async (projectId: string, routeId: string, stepId: string, data: Partial<ProcessStep>): Promise<ProcessStep> => {
    const response = await apiClient.put<ApiResponse<ProcessStep>>(
      `/projects/${projectId}/routes/${routeId}/steps/${stepId}`, data
    );
    return response.data.data;
  },

  deleteStep: async (projectId: string, routeId: string, stepId: string): Promise<void> => {
    await apiClient.delete(`/projects/${projectId}/routes/${routeId}/steps/${stepId}`);
  },

  // ========== 工序物料 ==========
  createStepMaterial: async (projectId: string, routeId: string, stepId: string, data: {
    name: string; material_id?: string; bom_item_id?: string; quantity?: number; unit?: string; notes?: string;
  }): Promise<ProcessStepMaterial> => {
    const response = await apiClient.post<ApiResponse<ProcessStepMaterial>>(
      `/projects/${projectId}/routes/${routeId}/steps/${stepId}/materials`, data
    );
    return response.data.data;
  },

  deleteStepMaterial: async (projectId: string, routeId: string, stepId: string, materialId: string): Promise<void> => {
    await apiClient.delete(`/projects/${projectId}/routes/${routeId}/steps/${stepId}/materials/${materialId}`);
  },

  // BOM权限
  getBOMPermissions: async (projectId: string): Promise<BOMPermissions> => {
    const res = await apiClient.get<ApiResponse<BOMPermissions>>(`/projects/${projectId}/bom-permissions`);
    return res.data.data;
  },
};

export interface BOMPermissions {
  can_edit_categories: string[];
  can_view_all: boolean;
  can_release: boolean;
}
