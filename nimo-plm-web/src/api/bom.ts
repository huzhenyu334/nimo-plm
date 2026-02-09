import apiClient from './client';
import { ApiResponse } from '@/types';
import { Material } from './materials';

export interface BOMItem {
  id: string;
  bom_header_id: string;
  parent_item_id: string;
  material_id: string;
  level: number;
  sequence: number;
  quantity: number;
  unit: string;
  position: string;
  reference: string;
  notes: string;
  unit_cost: number;
  extended_cost: number;
  created_at: string;
  updated_at: string;
  material?: Material;
}

export interface BOMHeader {
  id: string;
  product_id: string;
  version: string;
  status: string;
  description: string;
  total_items: number;
  total_cost: number;
  max_level: number;
  released_by: string;
  released_at: string;
  release_notes: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  items?: BOMItem[];
}

export interface BOMVersionsResponse {
  versions: BOMHeader[];
}

export const bomApi = {
  get: async (productId: string): Promise<BOMHeader> => {
    const response = await apiClient.get<ApiResponse<BOMHeader>>(`/products/${productId}/bom`);
    return response.data.data;
  },

  listVersions: async (productId: string): Promise<BOMVersionsResponse> => {
    const response = await apiClient.get<ApiResponse<BOMVersionsResponse>>(`/products/${productId}/bom/versions`);
    return response.data.data;
  },

  addItem: async (productId: string, data: Partial<BOMItem>): Promise<BOMItem> => {
    const response = await apiClient.post<ApiResponse<BOMItem>>(`/products/${productId}/bom/items`, data);
    return response.data.data;
  },

  updateItem: async (productId: string, itemId: string, data: Partial<BOMItem>): Promise<BOMItem> => {
    const response = await apiClient.put<ApiResponse<BOMItem>>(`/products/${productId}/bom/items/${itemId}`, data);
    return response.data.data;
  },

  deleteItem: async (productId: string, itemId: string): Promise<void> => {
    await apiClient.delete(`/products/${productId}/bom/items/${itemId}`);
  },

  release: async (productId: string, data: { version: string; release_notes?: string }): Promise<BOMHeader> => {
    const response = await apiClient.post<ApiResponse<BOMHeader>>(`/products/${productId}/bom/release`, data);
    return response.data.data;
  },

  compare: async (productId: string, params: { version1: string; version2: string }): Promise<any> => {
    const response = await apiClient.get(`/products/${productId}/bom/compare?version1=${params.version1}&version2=${params.version2}`);
    return response.data;
  },
};
