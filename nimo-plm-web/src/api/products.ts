import apiClient from './client';
import { ApiResponse, PaginatedResponse, Product, ProductCategory } from '@/types';

export interface ProductListParams {
  page?: number;
  page_size?: number;
  status?: string;
  category_id?: string;
  search?: string;
}

export interface CreateProductInput {
  code: string;
  name: string;
  description?: string;
  category_id?: string;
}

export interface UpdateProductInput {
  name?: string;
  description?: string;
  category_id?: string;
  status?: string;
}

export const productApi = {
  // 获取产品列表
  list: async (params?: ProductListParams): Promise<PaginatedResponse<Product>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<Product>>>('/products', { params });
    return response.data.data;
  },

  // 获取单个产品
  get: async (id: string): Promise<Product> => {
    const response = await apiClient.get<ApiResponse<Product>>(`/products/${id}`);
    return response.data.data;
  },

  // 创建产品
  create: async (data: CreateProductInput): Promise<Product> => {
    const response = await apiClient.post<ApiResponse<Product>>('/products', data);
    return response.data.data;
  },

  // 更新产品
  update: async (id: string, data: UpdateProductInput): Promise<Product> => {
    const response = await apiClient.put<ApiResponse<Product>>(`/products/${id}`, data);
    return response.data.data;
  },

  // 删除产品
  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/products/${id}`);
  },

  // 发布产品
  release: async (id: string): Promise<Product> => {
    const response = await apiClient.post<ApiResponse<Product>>(`/products/${id}/release`);
    return response.data.data;
  },

  // 获取产品分类
  listCategories: async (): Promise<ProductCategory[]> => {
    const response = await apiClient.get<ApiResponse<ProductCategory[]>>('/product-categories');
    return response.data.data;
  },
};
