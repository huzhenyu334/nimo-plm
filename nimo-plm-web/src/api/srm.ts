import apiClient from './client';
import { ApiResponse, PaginatedResponse } from '@/types';

// ============ Types ============

export interface Supplier {
  id: string;
  code: string;
  name: string;
  short_name: string;
  category: string;
  level: string;
  status: string;
  country: string;
  province: string;
  city: string;
  address: string;
  website: string;
  business_scope: string;
  annual_revenue?: number;
  employee_count?: number;
  factory_area?: number;
  certifications?: string[];
  bank_name: string;
  bank_account: string;
  tax_id: string;
  payment_terms: string;
  tags?: string[];
  tech_capability: string;
  cooperation: string;
  capacity_limit: string;
  quality_score?: number;
  delivery_score?: number;
  price_score?: number;
  overall_score?: number;
  created_by: string;
  approved_by?: string;
  approved_at?: string;
  created_at: string;
  updated_at: string;
  notes: string;
  contacts?: SupplierContact[];
}

export interface SupplierContact {
  id: string;
  supplier_id: string;
  name: string;
  title: string;
  phone: string;
  email: string;
  wechat: string;
  is_primary: boolean;
  created_at: string;
}

export interface PurchaseRequest {
  id: string;
  pr_code: string;
  title: string;
  type: string;
  priority: string;
  status: string;
  project_id?: string;
  bom_id?: string;
  phase: string;
  required_date?: string;
  requested_by: string;
  approved_by?: string;
  approved_at?: string;
  created_at: string;
  updated_at: string;
  notes: string;
  items?: PRItem[];
}

export interface PRItem {
  id: string;
  pr_id: string;
  material_id?: string;
  material_code: string;
  material_name: string;
  specification: string;
  category: string;
  quantity: number;
  unit: string;
  status: string;
  supplier_id?: string;
  unit_price?: number;
  total_amount?: number;
  expected_date?: string;
  actual_date?: string;
  inspection_result: string;
  sort_order: number;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface PurchaseOrder {
  id: string;
  po_code: string;
  supplier_id: string;
  pr_id?: string;
  type: string;
  status: string;
  total_amount?: number;
  currency: string;
  expected_date?: string;
  actual_date?: string;
  shipping_address: string;
  payment_terms: string;
  created_by: string;
  approved_by?: string;
  approved_at?: string;
  created_at: string;
  updated_at: string;
  notes: string;
  items?: POItem[];
  supplier?: Supplier;
}

export interface POItem {
  id: string;
  po_id: string;
  pr_item_id?: string;
  material_id?: string;
  material_code: string;
  material_name: string;
  specification: string;
  quantity: number;
  unit: string;
  unit_price?: number;
  total_amount?: number;
  received_qty: number;
  status: string;
  sort_order: number;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface Inspection {
  id: string;
  inspection_code: string;
  po_id?: string;
  po_item_id?: string;
  supplier_id?: string;
  material_id?: string;
  material_code: string;
  material_name: string;
  quantity?: number;
  sample_qty?: number;
  status: string;
  result: string;
  inspection_items?: unknown;
  report_url: string;
  inspector_id?: string;
  inspected_at?: string;
  created_at: string;
  updated_at: string;
  notes: string;
}

export interface SamplingProgress {
  project_id: string;
  total_items: number;
  ordered_items: number;
  received_items: number;
  inspected_items: number;
  passed_items: number;
  progress_pct: number;
}

// ============ API ============

export const srmApi = {
  // --- Suppliers ---
  listSuppliers: async (params?: {
    search?: string;
    category?: string;
    level?: string;
    status?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<Supplier>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<Supplier>>>('/srm/suppliers', { params });
    return response.data.data;
  },

  getSupplier: async (id: string): Promise<Supplier> => {
    const response = await apiClient.get<ApiResponse<Supplier>>(`/srm/suppliers/${id}`);
    return response.data.data;
  },

  createSupplier: async (data: Partial<Supplier>): Promise<Supplier> => {
    const response = await apiClient.post<ApiResponse<Supplier>>('/srm/suppliers', data);
    return response.data.data;
  },

  updateSupplier: async (id: string, data: Partial<Supplier>): Promise<Supplier> => {
    const response = await apiClient.put<ApiResponse<Supplier>>(`/srm/suppliers/${id}`, data);
    return response.data.data;
  },

  deleteSupplier: async (id: string): Promise<void> => {
    await apiClient.delete(`/srm/suppliers/${id}`);
  },

  listContacts: async (supplierId: string): Promise<SupplierContact[]> => {
    const response = await apiClient.get<ApiResponse<{ items: SupplierContact[] }>>(`/srm/suppliers/${supplierId}/contacts`);
    return response.data.data.items;
  },

  createContact: async (supplierId: string, data: Partial<SupplierContact>): Promise<SupplierContact> => {
    const response = await apiClient.post<ApiResponse<SupplierContact>>(`/srm/suppliers/${supplierId}/contacts`, data);
    return response.data.data;
  },

  deleteContact: async (supplierId: string, contactId: string): Promise<void> => {
    await apiClient.delete(`/srm/suppliers/${supplierId}/contacts/${contactId}`);
  },

  // --- Purchase Requests ---
  listPRs: async (params?: {
    project_id?: string;
    status?: string;
    type?: string;
    search?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<PurchaseRequest>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<PurchaseRequest>>>('/srm/purchase-requests', { params });
    return response.data.data;
  },

  getPR: async (id: string): Promise<PurchaseRequest> => {
    const response = await apiClient.get<ApiResponse<PurchaseRequest>>(`/srm/purchase-requests/${id}`);
    return response.data.data;
  },

  createPR: async (data: {
    title: string;
    type: string;
    priority?: string;
    project_id?: string;
    phase?: string;
    required_date?: string;
    notes?: string;
    items?: Array<{
      material_name: string;
      material_code?: string;
      specification?: string;
      quantity: number;
      unit?: string;
    }>;
  }): Promise<PurchaseRequest> => {
    const response = await apiClient.post<ApiResponse<PurchaseRequest>>('/srm/purchase-requests', data);
    return response.data.data;
  },

  updatePR: async (id: string, data: Partial<PurchaseRequest>): Promise<PurchaseRequest> => {
    const response = await apiClient.put<ApiResponse<PurchaseRequest>>(`/srm/purchase-requests/${id}`, data);
    return response.data.data;
  },

  approvePR: async (id: string): Promise<PurchaseRequest> => {
    const response = await apiClient.post<ApiResponse<PurchaseRequest>>(`/srm/purchase-requests/${id}/approve`);
    return response.data.data;
  },

  createPRFromBOM: async (data: { project_id: string; bom_id: string; phase?: string }): Promise<PurchaseRequest> => {
    const response = await apiClient.post<ApiResponse<PurchaseRequest>>('/srm/purchase-requests/from-bom', data);
    return response.data.data;
  },

  assignSupplier: async (prId: string, itemId: string, data: { supplier_id: string; unit_price?: number; expected_date?: string }): Promise<PRItem> => {
    const response = await apiClient.put<ApiResponse<PRItem>>(`/srm/purchase-requests/${prId}/items/${itemId}/assign-supplier`, data);
    return response.data.data;
  },

  generatePOs: async (prId: string): Promise<PurchaseOrder[]> => {
    const response = await apiClient.post<ApiResponse<PurchaseOrder[]>>(`/srm/purchase-requests/${prId}/generate-pos`);
    return response.data.data;
  },

  // --- Purchase Orders ---
  listPOs: async (params?: {
    supplier_id?: string;
    status?: string;
    type?: string;
    search?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<PurchaseOrder>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<PurchaseOrder>>>('/srm/purchase-orders', { params });
    return response.data.data;
  },

  getPO: async (id: string): Promise<PurchaseOrder> => {
    const response = await apiClient.get<ApiResponse<PurchaseOrder>>(`/srm/purchase-orders/${id}`);
    return response.data.data;
  },

  createPO: async (data: {
    supplier_id: string;
    pr_id?: string;
    type: string;
    expected_date?: string;
    shipping_address?: string;
    payment_terms?: string;
    notes?: string;
    items?: Array<{
      material_name: string;
      material_code?: string;
      specification?: string;
      quantity: number;
      unit?: string;
      unit_price?: number;
    }>;
  }): Promise<PurchaseOrder> => {
    const response = await apiClient.post<ApiResponse<PurchaseOrder>>('/srm/purchase-orders', data);
    return response.data.data;
  },

  updatePO: async (id: string, data: Partial<PurchaseOrder>): Promise<PurchaseOrder> => {
    const response = await apiClient.put<ApiResponse<PurchaseOrder>>(`/srm/purchase-orders/${id}`, data);
    return response.data.data;
  },

  approvePO: async (id: string): Promise<PurchaseOrder> => {
    const response = await apiClient.post<ApiResponse<PurchaseOrder>>(`/srm/purchase-orders/${id}/approve`);
    return response.data.data;
  },

  receivePOItem: async (poId: string, itemId: string, receivedQty: number): Promise<void> => {
    await apiClient.post(`/srm/purchase-orders/${poId}/items/${itemId}/receive`, { received_qty: receivedQty });
  },

  // --- Inspections ---
  listInspections: async (params?: {
    supplier_id?: string;
    status?: string;
    result?: string;
    po_id?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<Inspection>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<Inspection>>>('/srm/inspections', { params });
    return response.data.data;
  },

  getInspection: async (id: string): Promise<Inspection> => {
    const response = await apiClient.get<ApiResponse<Inspection>>(`/srm/inspections/${id}`);
    return response.data.data;
  },

  updateInspection: async (id: string, data: Partial<Inspection>): Promise<Inspection> => {
    const response = await apiClient.put<ApiResponse<Inspection>>(`/srm/inspections/${id}`, data);
    return response.data.data;
  },

  completeInspection: async (id: string, data: { result: string; inspection_items?: unknown; notes?: string }): Promise<Inspection> => {
    const response = await apiClient.post<ApiResponse<Inspection>>(`/srm/inspections/${id}/complete`, data);
    return response.data.data;
  },

  // --- Dashboard ---
  getSamplingProgress: async (projectId: string): Promise<SamplingProgress> => {
    const response = await apiClient.get<ApiResponse<SamplingProgress>>('/srm/dashboard/sampling-progress', {
      params: { project_id: projectId },
    });
    return response.data.data;
  },
};
