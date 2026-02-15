import apiClient from './client';

export interface CMFVariant {
  id: string;
  bom_item_id: string;
  variant_index: number;
  material_code: string;
  color_hex: string;
  material: string;
  finish: string;
  texture: string;
  coating: string;
  pantone_code: string;
  gloss_level: string;
  reference_image_file_id: string;
  reference_image_url: string;
  process_drawing_type: string;
  process_drawings: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface AppearancePartWithCMF {
  bom_item: {
    id: string;
    name: string;
    item_number: number;
    thumbnail_url?: string;
    extended_attrs?: Record<string, any>;
  };
  cmf_variants: CMFVariant[];
  bom_id: string;
  bom_name: string;
}

export interface CreateVariantInput {
  color_hex?: string;
  finish?: string;
  texture?: string;
  coating?: string;
  pantone_code?: string;
  reference_image_file_id?: string;
  reference_image_url?: string;
  process_drawing_type?: string;
  process_drawings?: string;
  notes?: string;
}

export const cmfVariantApi = {
  // 获取外观件 + CMF变体
  getAppearanceParts: async (projectId: string): Promise<AppearancePartWithCMF[]> => {
    const res = await apiClient.get(`/projects/${projectId}/appearance-parts`);
    return res.data.data || [];
  },

  // 获取零件的CMF变体
  listVariants: async (projectId: string, itemId: string): Promise<CMFVariant[]> => {
    const res = await apiClient.get(`/projects/${projectId}/bom-items/${itemId}/cmf-variants`);
    return res.data.data || [];
  },

  // 创建CMF变体
  createVariant: async (projectId: string, itemId: string, data: CreateVariantInput): Promise<CMFVariant> => {
    const res = await apiClient.post(`/projects/${projectId}/bom-items/${itemId}/cmf-variants`, data);
    return res.data.data;
  },

  // 更新CMF变体
  updateVariant: async (projectId: string, variantId: string, data: Partial<CreateVariantInput>): Promise<CMFVariant> => {
    const res = await apiClient.put(`/projects/${projectId}/cmf-variants/${variantId}`, data);
    return res.data.data;
  },

  // 删除CMF变体
  deleteVariant: async (projectId: string, variantId: string): Promise<void> => {
    await apiClient.delete(`/projects/${projectId}/cmf-variants/${variantId}`);
  },

  // 获取SRM采购项
  getSRMItems: async (projectId: string): Promise<any[]> => {
    const res = await apiClient.get(`/projects/${projectId}/srm/items`);
    return res.data.data || [];
  },
};
