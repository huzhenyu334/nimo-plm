import apiClient from './client';

// ========== Types ==========

export interface ProductSKU {
  id: string;
  project_id: string;
  name: string;
  code: string;
  description?: string;
  status: string;
  sort_order: number;
  created_by: string;
  created_at: string;
  updated_at: string;
  cmf_configs?: SKUCMFConfig[];
  bom_overrides?: SKUBOMOverride[];
}

export interface SKUCMFConfig {
  id: string;
  sku_id: string;
  bom_item_id: string;
  color: string;
  color_code: string;
  surface_treatment: string;
  process_params?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
  bom_item?: {
    id: string;
    name: string;
    item_number: number;
    specification?: string;
    material_type?: string;
    process_type?: string;
  };
}

export interface SKUBOMOverride {
  id: string;
  sku_id: string;
  action: 'replace' | 'add' | 'remove';
  base_item_id?: string;
  override_name?: string;
  override_specification?: string;
  override_quantity: number;
  override_unit?: string;
  override_material_type?: string;
  override_process_type?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
  base_item?: {
    id: string;
    name: string;
    item_number: number;
  };
}

export interface FullBOMItem {
  id: string;
  item_number?: number;
  name: string;
  specification?: string;
  quantity: number;
  unit: string;
  category?: string;
  material_type?: string;
  process_type?: string;
  color?: string;
  color_code?: string;
  surface_treatment?: string;
  source: 'base' | 'replaced' | 'added';
}

// ========== API ==========

export const skuApi = {
  // SKU CRUD
  listSKUs: async (projectId: string) => {
    const res = await apiClient.get(`/projects/${projectId}/skus`);
    return res.data.data?.items || [];
  },

  createSKU: async (projectId: string, data: { name: string; code?: string; description?: string; sort_order?: number }) => {
    const res = await apiClient.post(`/projects/${projectId}/skus`, data);
    return res.data.data;
  },

  updateSKU: async (projectId: string, skuId: string, data: Partial<{ name: string; code: string; description: string; status: string; sort_order: number }>) => {
    const res = await apiClient.put(`/projects/${projectId}/skus/${skuId}`, data);
    return res.data.data;
  },

  deleteSKU: async (projectId: string, skuId: string) => {
    const res = await apiClient.delete(`/projects/${projectId}/skus/${skuId}`);
    return res.data;
  },

  // CMF Config
  getCMFConfigs: async (projectId: string, skuId: string): Promise<SKUCMFConfig[]> => {
    const res = await apiClient.get(`/projects/${projectId}/skus/${skuId}/cmf`);
    return res.data.data?.items || [];
  },

  saveCMFConfigs: async (projectId: string, skuId: string, configs: Array<{
    bom_item_id: string;
    color: string;
    color_code: string;
    surface_treatment: string;
    process_params?: string;
    notes?: string;
  }>) => {
    const res = await apiClient.put(`/projects/${projectId}/skus/${skuId}/cmf`, configs);
    return res.data.data?.items || [];
  },

  // BOM Overrides
  getBOMOverrides: async (projectId: string, skuId: string): Promise<SKUBOMOverride[]> => {
    const res = await apiClient.get(`/projects/${projectId}/skus/${skuId}/bom-overrides`);
    return res.data.data?.items || [];
  },

  createBOMOverride: async (projectId: string, skuId: string, data: {
    action: string;
    base_item_id?: string;
    override_name?: string;
    override_specification?: string;
    override_quantity?: number;
    override_unit?: string;
    override_material_type?: string;
    override_process_type?: string;
    notes?: string;
  }) => {
    const res = await apiClient.post(`/projects/${projectId}/skus/${skuId}/bom-overrides`, data);
    return res.data.data;
  },

  updateBOMOverride: async (projectId: string, skuId: string, overrideId: string, data: {
    action: string;
    base_item_id?: string;
    override_name?: string;
    override_specification?: string;
    override_quantity?: number;
    override_unit?: string;
    override_material_type?: string;
    override_process_type?: string;
    notes?: string;
  }) => {
    const res = await apiClient.put(`/projects/${projectId}/skus/${skuId}/bom-overrides/${overrideId}`, data);
    return res.data.data;
  },

  deleteBOMOverride: async (projectId: string, skuId: string, overrideId: string) => {
    const res = await apiClient.delete(`/projects/${projectId}/skus/${skuId}/bom-overrides/${overrideId}`);
    return res.data;
  },

  // Full BOM (merged)
  getFullBOM: async (projectId: string, skuId: string): Promise<FullBOMItem[]> => {
    const res = await apiClient.get(`/projects/${projectId}/skus/${skuId}/full-bom`);
    return res.data.data?.items || [];
  },
};
