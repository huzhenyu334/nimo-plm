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
  bom_items?: SKUBOMItem[];
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

export interface SKUBOMItem {
  id: string;
  sku_id: string;
  bom_item_id: string;
  cmf_variant_id?: string;
  quantity: number;
  notes?: string;
  created_at: string;
  updated_at: string;
  bom_item?: {
    id: string;
    name: string;
    item_number: number;
    specification?: string;
    material_type?: string;
  };
  cmf_variant?: {
    id: string;
    variant_index: number;
    material_code: string;
    color_hex: string;
    material: string;
    finish: string;
    texture: string;
    coating: string;
    pantone_code: string;
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
  is_appearance_part?: boolean;
  color?: string;
  color_code?: string;
  color_hex?: string;
  material_code?: string;
  finish?: string;
  texture?: string;
  coating?: string;
  surface_treatment?: string;
  cmf_variant_id?: string;
  cmf_variant?: {
    id: string;
    variant_index: number;
    material_code: string;
    color_hex: string;
    material: string;
    finish: string;
    texture: string;
    coating: string;
    pantone_code: string;
  };
}

// ========== API ==========

export const skuApi = {
  // SKU CRUD
  listSKUs: async (projectId: string) => {
    const res = await apiClient.get(`/projects/${projectId}/skus`);
    return res.data.data?.items || [];
  },

  createSKU: async (projectId: string, data: {
    name: string;
    code?: string;
    description?: string;
    sort_order?: number;
    bom_items?: Array<{ bom_item_id: string; cmf_variant_id?: string; quantity?: number }>;
  }) => {
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

  // BOM Items (checkbox selection from PBOM)
  getBOMItems: async (projectId: string, skuId: string): Promise<SKUBOMItem[]> => {
    const res = await apiClient.get(`/projects/${projectId}/skus/${skuId}/bom-items`);
    return res.data.data?.items || [];
  },

  saveBOMItems: async (projectId: string, skuId: string, items: Array<{
    bom_item_id: string;
    cmf_variant_id?: string;
    quantity?: number;
    notes?: string;
  }>) => {
    const res = await apiClient.put(`/projects/${projectId}/skus/${skuId}/bom-items`, items);
    return res.data.data?.items || [];
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

  // Full BOM (merged: selected PBOM items + CMF)
  getFullBOM: async (projectId: string, skuId: string): Promise<FullBOMItem[]> => {
    const res = await apiClient.get(`/projects/${projectId}/skus/${skuId}/full-bom`);
    return res.data.data?.items || [];
  },
};
