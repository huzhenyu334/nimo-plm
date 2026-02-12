import { apiClient } from './client';

export interface PartDrawing {
  id: string;
  bom_item_id: string;
  drawing_type: '2D' | '3D';
  version: string;
  file_id: string;
  file_name: string;
  file_size: number;
  file_url?: string;
  change_description?: string;
  change_reason?: string;
  uploaded_by: string;
  created_at: string;
  uploader?: { id: string; name: string; avatar_url?: string };
}

export interface DrawingsByType {
  '2D': PartDrawing[];
  '3D': PartDrawing[];
}

// 按BOMItem ID分组的图纸
export type DrawingsByBOM = Record<string, DrawingsByType>;

export interface CreatePartDrawingRequest {
  drawing_type: '2D' | '3D';
  file_id: string;
  file_name: string;
  file_size?: number;
  change_description?: string;
  change_reason?: string;
}

export const partDrawingApi = {
  // 获取单个BOMItem的图纸列表
  list: async (projectId: string, itemId: string): Promise<DrawingsByType> => {
    const res = await apiClient.get(`/projects/${projectId}/bom-items/${itemId}/drawings`);
    return res.data.data;
  },

  // 上传新版本图纸
  upload: async (projectId: string, itemId: string, data: CreatePartDrawingRequest): Promise<PartDrawing> => {
    const res = await apiClient.post(`/projects/${projectId}/bom-items/${itemId}/drawings`, data);
    return res.data.data;
  },

  // 删除图纸
  delete: async (projectId: string, itemId: string, drawingId: string): Promise<void> => {
    await apiClient.delete(`/projects/${projectId}/bom-items/${itemId}/drawings/${drawingId}`);
  },

  // 批量获取BOM所有item的图纸
  listByBOM: async (projectId: string, bomId: string): Promise<DrawingsByBOM> => {
    const res = await apiClient.get(`/projects/${projectId}/boms/${bomId}/drawings`);
    return res.data.data;
  },
};
