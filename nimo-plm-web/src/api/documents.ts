import apiClient from './client';
import { ApiResponse, User } from '@/types';

export interface Document {
  id: string;
  code: string;
  title: string;
  category_id: string;
  status: string;
  file_name: string;
  file_size: number;
  file_type: string;
  storage_path: string;
  version: string;
  description: string;
  related_type: string;
  related_id: string;
  tags: string[];
  uploaded_by: string;
  released_by: string;
  released_at: string;
  created_at: string;
  updated_at: string;
  category?: DocumentCategory;
  uploader?: User;
  releaser?: User;
}

export interface DocumentCategory {
  id: string;
  code: string;
  name: string;
  parent_id: string;
  description: string;
  children?: DocumentCategory[];
}

export interface DocumentVersion {
  id: string;
  document_id: string;
  version: string;
  file_name: string;
  file_size: number;
  file_path: string;
  change_summary: string;
  created_by: string;
  created_at: string;
  creator?: User;
}

export interface DocumentListResponse {
  items: Document[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface DocumentCategoryListResponse {
  categories: DocumentCategory[];
}

export const documentsApi = {
  list: async (params?: { category?: string; search?: string; related_type?: string; related_id?: string; page?: number; page_size?: number }): Promise<DocumentListResponse> => {
    const searchParams = new URLSearchParams();
    if (params?.category) searchParams.set('category', params.category);
    if (params?.search) searchParams.set('search', params.search);
    if (params?.related_type) searchParams.set('related_type', params.related_type);
    if (params?.related_id) searchParams.set('related_id', params.related_id);
    if (params?.page) searchParams.set('page', params.page.toString());
    if (params?.page_size) searchParams.set('page_size', params.page_size.toString());
    const query = searchParams.toString();
    const response = await apiClient.get<ApiResponse<DocumentListResponse>>(`/documents${query ? `?${query}` : ''}`);
    return response.data.data;
  },

  get: async (id: string): Promise<Document> => {
    const response = await apiClient.get<ApiResponse<Document>>(`/documents/${id}`);
    return response.data.data;
  },

  upload: async (file: File, data: { title: string; category_id?: string; description?: string; related_type?: string; related_id?: string }): Promise<Document> => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('title', data.title);
    if (data.category_id) formData.append('category_id', data.category_id);
    if (data.description) formData.append('description', data.description);
    if (data.related_type) formData.append('related_type', data.related_type);
    if (data.related_id) formData.append('related_id', data.related_id);
    
    const response = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/documents`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
      },
      body: formData,
    });
    if (!response.ok) throw new Error('Upload failed');
    const json = await response.json();
    return json.data;
  },

  update: async (id: string, data: Partial<Document>): Promise<Document> => {
    const response = await apiClient.put<ApiResponse<Document>>(`/documents/${id}`, data);
    return response.data.data;
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/documents/${id}`);
  },

  download: async (id: string): Promise<Blob> => {
    const token = localStorage.getItem('access_token');
    const response = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/documents/${id}/download`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });
    if (!response.ok) throw new Error('Download failed');
    return response.blob();
  },

  release: async (id: string): Promise<Document> => {
    const response = await apiClient.post<ApiResponse<Document>>(`/documents/${id}/release`, {});
    return response.data.data;
  },

  obsolete: async (id: string): Promise<Document> => {
    const response = await apiClient.post<ApiResponse<Document>>(`/documents/${id}/obsolete`, {});
    return response.data.data;
  },

  listVersions: async (id: string): Promise<DocumentVersion[]> => {
    const response = await apiClient.get<ApiResponse<DocumentVersion[]>>(`/documents/${id}/versions`);
    return response.data.data;
  },

  uploadNewVersion: async (id: string, file: File, changeSummary?: string): Promise<Document> => {
    const formData = new FormData();
    formData.append('file', file);
    if (changeSummary) formData.append('change_summary', changeSummary);
    
    const response = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/documents/${id}/versions`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
      },
      body: formData,
    });
    if (!response.ok) throw new Error('Upload version failed');
    const json = await response.json();
    return json.data;
  },

  downloadVersion: async (id: string, versionId: string): Promise<Blob> => {
    const token = localStorage.getItem('access_token');
    const response = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/api/v1/documents/${id}/versions/${versionId}/download`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });
    if (!response.ok) throw new Error('Download failed');
    return response.blob();
  },

  listCategories: async (): Promise<DocumentCategoryListResponse> => {
    const response = await apiClient.get<ApiResponse<DocumentCategoryListResponse>>('/document-categories');
    return response.data.data;
  },
};
