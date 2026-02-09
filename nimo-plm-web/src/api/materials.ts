import apiClient from './client';
import { ApiResponse } from '@/types';

export interface Material {
  id: string;
  code: string;
  name: string;
  category_id: string;
  status: string;
  unit: string;
  description: string;
  specs: Record<string, any>;
  lead_time_days: number;
  min_order_qty: number;
  safety_stock: number;
  standard_cost: number;
  last_cost: number;
  currency: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  category?: MaterialCategory;
}

export interface MaterialCategory {
  id: string;
  code: string;
  name: string;
  parent_id: string;
  description: string;
}

export interface MaterialListResponse {
  materials: Material[];
}

export interface MaterialCategoryListResponse {
  categories: MaterialCategory[];
}

export const materialsApi = {
  list: async (params?: { category?: string; search?: string }): Promise<MaterialListResponse> => {
    const searchParams = new URLSearchParams();
    if (params?.category) searchParams.set('category', params.category);
    if (params?.search) searchParams.set('search', params.search);
    const query = searchParams.toString();
    const response = await apiClient.get<ApiResponse<MaterialListResponse>>(`/materials${query ? `?${query}` : ''}`);
    return response.data.data;
  },

  get: async (id: string): Promise<Material> => {
    const response = await apiClient.get<ApiResponse<Material>>(`/materials/${id}`);
    return response.data.data;
  },

  create: async (data: Partial<Material>): Promise<{ message: string }> => {
    const response = await apiClient.post<ApiResponse<{ message: string }>>('/materials', data);
    return response.data.data;
  },

  update: async (id: string, data: Partial<Material>): Promise<Material> => {
    const response = await apiClient.put<ApiResponse<Material>>(`/materials/${id}`, data);
    return response.data.data;
  },

  listCategories: async (): Promise<MaterialCategoryListResponse> => {
    const response = await apiClient.get<ApiResponse<MaterialCategoryListResponse>>('/material-categories');
    return response.data.data;
  },
};
