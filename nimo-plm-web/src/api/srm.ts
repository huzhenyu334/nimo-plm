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

export interface SamplingRequest {
  id: string;
  pr_item_id: string;
  round: number;
  supplier_id: string;
  supplier_name: string;
  sample_qty: number;
  status: string; // preparing/shipping/arrived/verifying/passed/failed
  requested_by: string;
  arrived_at?: string;
  verified_by: string;
  verified_at?: string;
  verify_result: string;
  reject_reason: string;
  approval_id: string;
  notes: string;
  created_at: string;
  updated_at: string;
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

export interface SRMProject {
  id: string;
  code: string;
  name: string;
  type: string;
  phase: string;
  status: string;
  plm_project_id?: string;
  plm_task_id?: string;
  plm_bom_id?: string;
  total_items: number;
  sourcing_count: number;
  ordered_count: number;
  received_count: number;
  passed_count: number;
  failed_count: number;
  estimated_days?: number;
  start_date?: string;
  target_date?: string;
  actual_date?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface ActivityLog {
  id: string;
  entity_type: string;
  entity_id: string;
  entity_code: string;
  action: string;
  from_status: string;
  to_status: string;
  content: string;
  attachments?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  operator_id: string;
  operator_name: string;
  created_at: string;
}

// --- Settlement Types ---

export interface Settlement {
  id: string;
  settlement_code: string;
  supplier_id: string;
  period_start?: string;
  period_end?: string;
  status: string;
  po_amount?: number;
  received_amount?: number;
  deduction?: number;
  final_amount?: number;
  currency: string;
  invoice_no: string;
  invoice_amount?: number;
  invoice_url: string;
  confirmed_by_buyer: boolean;
  confirmed_by_supplier: boolean;
  confirmed_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  notes: string;
  disputes?: SettlementDispute[];
  supplier?: Supplier;
}

export interface SettlementDispute {
  id: string;
  settlement_id: string;
  dispute_type: string;
  description: string;
  amount_diff?: number;
  status: string;
  resolution: string;
  created_at: string;
}

// --- CorrectiveAction Types ---

export interface CorrectiveAction {
  id: string;
  ca_code: string;
  inspection_id: string;
  supplier_id: string;
  problem_desc: string;
  severity: string;
  status: string;
  root_cause: string;
  corrective_action: string;
  preventive_action: string;
  response_deadline?: string;
  responded_at?: string;
  verified_at?: string;
  closed_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// --- SupplierEvaluation Types ---

export interface SupplierEvaluation {
  id: string;
  supplier_id: string;
  period: string;
  eval_type: string;
  quality_score?: number;
  delivery_score?: number;
  price_score?: number;
  service_score?: number;
  total_score?: number;
  quality_weight: number;
  delivery_weight: number;
  price_weight: number;
  service_weight: number;
  grade: string;
  total_pos: number;
  on_time_pos: number;
  quality_passed: number;
  quality_total: number;
  remarks: string;
  evaluator_id: string;
  status: string;
  created_at: string;
  updated_at: string;
  supplier?: Supplier;
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

  updatePRItemStatus: async (itemId: string, status: string): Promise<PRItem> => {
    const response = await apiClient.put<ApiResponse<PRItem>>(`/srm/pr-items/${itemId}/status`, { status });
    return response.data.data;
  },

  // --- Sampling (打样) ---
  createSampling: async (itemId: string, data: {
    supplier_id: string;
    sample_qty: number;
    notes?: string;
  }): Promise<SamplingRequest> => {
    const response = await apiClient.post<ApiResponse<SamplingRequest>>(`/srm/pr-items/${itemId}/sampling`, data);
    return response.data.data;
  },

  listSampling: async (itemId: string): Promise<SamplingRequest[]> => {
    const response = await apiClient.get<ApiResponse<{ items: SamplingRequest[] }>>(`/srm/pr-items/${itemId}/sampling`);
    return response.data.data.items;
  },

  updateSamplingStatus: async (samplingId: string, status: string): Promise<SamplingRequest> => {
    const response = await apiClient.put<ApiResponse<SamplingRequest>>(`/srm/sampling/${samplingId}/status`, { status });
    return response.data.data;
  },

  requestSamplingVerify: async (samplingId: string, data: {
    approver_open_id: string;
    initiator_open_id: string;
  }): Promise<SamplingRequest> => {
    const response = await apiClient.post<ApiResponse<SamplingRequest>>(`/srm/sampling/${samplingId}/request-verify`, data);
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

  // --- SRM Projects ---
  listProjects: async (params?: {
    status?: string;
    type?: string;
    phase?: string;
    plm_project_id?: string;
    search?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<SRMProject>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<SRMProject>>>('/srm/projects', { params });
    return response.data.data;
  },

  getProject: async (id: string): Promise<SRMProject> => {
    const response = await apiClient.get<ApiResponse<SRMProject>>(`/srm/projects/${id}`);
    return response.data.data;
  },

  getProjectProgress: async (id: string): Promise<SRMProject> => {
    const response = await apiClient.get<ApiResponse<SRMProject>>(`/srm/projects/${id}/progress`);
    return response.data.data;
  },

  // --- Activity Logs ---
  listProjectActivities: async (projectId: string, params?: {
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<ActivityLog>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<ActivityLog>>>(`/srm/projects/${projectId}/activities`, { params });
    return response.data.data;
  },

  listActivities: async (entityType: string, entityId: string, params?: {
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<ActivityLog>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<ActivityLog>>>(`/srm/activities/${entityType}/${entityId}`, { params });
    return response.data.data;
  },

  // --- Settlements ---
  listSettlements: async (params?: {
    supplier_id?: string;
    status?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<Settlement>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<Settlement>>>('/srm/settlements', { params });
    return response.data.data;
  },

  getSettlement: async (id: string): Promise<Settlement> => {
    const response = await apiClient.get<ApiResponse<Settlement>>(`/srm/settlements/${id}`);
    return response.data.data;
  },

  createSettlement: async (data: {
    supplier_id: string;
    period_start?: string;
    period_end?: string;
    deduction?: number;
    currency?: string;
    notes?: string;
  }): Promise<Settlement> => {
    const response = await apiClient.post<ApiResponse<Settlement>>('/srm/settlements', data);
    return response.data.data;
  },

  updateSettlement: async (id: string, data: Partial<Settlement>): Promise<Settlement> => {
    const response = await apiClient.put<ApiResponse<Settlement>>(`/srm/settlements/${id}`, data);
    return response.data.data;
  },

  deleteSettlement: async (id: string): Promise<void> => {
    await apiClient.delete(`/srm/settlements/${id}`);
  },

  confirmSettlementBuyer: async (id: string): Promise<Settlement> => {
    const response = await apiClient.post<ApiResponse<Settlement>>(`/srm/settlements/${id}/confirm-buyer`);
    return response.data.data;
  },

  confirmSettlementSupplier: async (id: string): Promise<Settlement> => {
    const response = await apiClient.post<ApiResponse<Settlement>>(`/srm/settlements/${id}/confirm-supplier`);
    return response.data.data;
  },

  addSettlementDispute: async (id: string, data: {
    dispute_type: string;
    description?: string;
    amount_diff?: number;
  }): Promise<SettlementDispute> => {
    const response = await apiClient.post<ApiResponse<SettlementDispute>>(`/srm/settlements/${id}/disputes`, data);
    return response.data.data;
  },

  updateSettlementDispute: async (settlementId: string, disputeId: string, data: Partial<SettlementDispute>): Promise<SettlementDispute> => {
    const response = await apiClient.put<ApiResponse<SettlementDispute>>(`/srm/settlements/${settlementId}/disputes/${disputeId}`, data);
    return response.data.data;
  },

  generateSettlement: async (data: {
    supplier_id: string;
    period_start: string;
    period_end: string;
  }): Promise<Settlement> => {
    const response = await apiClient.post<ApiResponse<Settlement>>('/srm/settlements/generate', data);
    return response.data.data;
  },

  // --- Corrective Actions ---
  listCorrectiveActions: async (params?: {
    supplier_id?: string;
    status?: string;
    severity?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<CorrectiveAction>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<CorrectiveAction>>>('/srm/corrective-actions', { params });
    return response.data.data;
  },

  getCorrectiveAction: async (id: string): Promise<CorrectiveAction> => {
    const response = await apiClient.get<ApiResponse<CorrectiveAction>>(`/srm/corrective-actions/${id}`);
    return response.data.data;
  },

  createCorrectiveAction: async (data: {
    inspection_id?: string;
    supplier_id: string;
    problem_desc: string;
    severity: string;
    response_deadline?: string;
  }): Promise<CorrectiveAction> => {
    const response = await apiClient.post<ApiResponse<CorrectiveAction>>('/srm/corrective-actions', data);
    return response.data.data;
  },

  updateCorrectiveAction: async (id: string, data: Partial<CorrectiveAction>): Promise<CorrectiveAction> => {
    const response = await apiClient.put<ApiResponse<CorrectiveAction>>(`/srm/corrective-actions/${id}`, data);
    return response.data.data;
  },

  respondCorrectiveAction: async (id: string, data: {
    root_cause: string;
    corrective_action: string;
    preventive_action?: string;
  }): Promise<CorrectiveAction> => {
    const response = await apiClient.post<ApiResponse<CorrectiveAction>>(`/srm/corrective-actions/${id}/respond`, data);
    return response.data.data;
  },

  verifyCorrectiveAction: async (id: string): Promise<CorrectiveAction> => {
    const response = await apiClient.post<ApiResponse<CorrectiveAction>>(`/srm/corrective-actions/${id}/verify`);
    return response.data.data;
  },

  closeCorrectiveAction: async (id: string): Promise<CorrectiveAction> => {
    const response = await apiClient.post<ApiResponse<CorrectiveAction>>(`/srm/corrective-actions/${id}/close`);
    return response.data.data;
  },

  // --- Evaluations ---
  listEvaluations: async (params?: {
    supplier_id?: string;
    status?: string;
    eval_type?: string;
    period?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<SupplierEvaluation>> => {
    const response = await apiClient.get<ApiResponse<PaginatedResponse<SupplierEvaluation>>>('/srm/evaluations', { params });
    return response.data.data;
  },

  getEvaluation: async (id: string): Promise<SupplierEvaluation> => {
    const response = await apiClient.get<ApiResponse<SupplierEvaluation>>(`/srm/evaluations/${id}`);
    return response.data.data;
  },

  createEvaluation: async (data: {
    supplier_id: string;
    period: string;
    eval_type?: string;
    quality_score?: number;
    delivery_score?: number;
    price_score?: number;
    service_score?: number;
    remarks?: string;
  }): Promise<SupplierEvaluation> => {
    const response = await apiClient.post<ApiResponse<SupplierEvaluation>>('/srm/evaluations', data);
    return response.data.data;
  },

  updateEvaluation: async (id: string, data: Partial<SupplierEvaluation>): Promise<SupplierEvaluation> => {
    const response = await apiClient.put<ApiResponse<SupplierEvaluation>>(`/srm/evaluations/${id}`, data);
    return response.data.data;
  },

  submitEvaluation: async (id: string): Promise<SupplierEvaluation> => {
    const response = await apiClient.post<ApiResponse<SupplierEvaluation>>(`/srm/evaluations/${id}/submit`);
    return response.data.data;
  },

  approveEvaluation: async (id: string): Promise<SupplierEvaluation> => {
    const response = await apiClient.post<ApiResponse<SupplierEvaluation>>(`/srm/evaluations/${id}/approve`);
    return response.data.data;
  },

  autoGenerateEvaluation: async (data: {
    supplier_id: string;
    period: string;
    eval_type?: string;
  }): Promise<SupplierEvaluation> => {
    const response = await apiClient.post<ApiResponse<SupplierEvaluation>>('/srm/evaluations/auto-generate', data);
    return response.data.data;
  },

  getSupplierEvaluationHistory: async (supplierId: string): Promise<SupplierEvaluation[]> => {
    const response = await apiClient.get<ApiResponse<SupplierEvaluation[]>>(`/srm/evaluations/supplier/${supplierId}`);
    return response.data.data;
  },
};
