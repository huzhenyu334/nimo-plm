import apiClient from './client';
import { ApiResponse } from '@/types';

export interface Codename {
  id: string;
  codename: string;
  codename_type: 'platform' | 'product';
  generation: number;
  theme: string;
  description: string;
  is_used: boolean;
  // legacy fields
  name?: string;
  type?: string;
  available?: boolean;
  assigned_to?: string;
  created_at?: string;
}

export const codenameApi = {
  list: async (type?: string, available?: boolean): Promise<Codename[]> => {
    const params: any = {};
    if (type) params.type = type;
    if (available) params.available = 'true';
    const response = await apiClient.get<ApiResponse<any>>('/codenames', { params });
    // Support both array and { items: [] } response formats
    const data = response.data.data;
    if (Array.isArray(data)) return data;
    return data?.items || [];
  },
};

// Keep backward-compatible export
export const codenamesApi = codenameApi;
