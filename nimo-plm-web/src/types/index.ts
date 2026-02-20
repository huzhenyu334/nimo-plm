// API 响应
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

// 分页
export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  items: T[];
  pagination: Pagination;
}

// 用户
export interface User {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
  roles: string[];
  permissions?: string[];
}

// 认证
export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
}

// 产品
export interface Product {
  id: string;
  code: string;
  name: string;
  description?: string;
  category_id?: string;
  category_name?: string;
  status: 'draft' | 'active' | 'deprecated' | 'obsolete';
  current_version: string;
  created_by: string;
  created_by_name?: string;
  created_at: string;
  updated_at: string;
}

export interface ProductCategory {
  id: string;
  name: string;
  parent_id?: string;
  description?: string;
}

// 物料
export interface Material {
  id: string;
  code: string;
  name: string;
  specification?: string;
  unit: string;
  category: string;
  status: 'active' | 'inactive' | 'obsolete';
  created_at: string;
  updated_at: string;
}

// BOM
export interface BOM {
  id: string;
  product_id: string;
  product_name?: string;
  version: string;
  status: 'draft' | 'released' | 'obsolete';
  items: BOMItem[];
  created_at: string;
  updated_at: string;
}

export interface BOMItem {
  id: string;
  bom_id: string;
  material_id: string;
  material_code?: string;
  material_name?: string;
  quantity: number;
  unit: string;
  position?: string;
  remark?: string;
}

// 项目
export interface Project {
  id: string;
  code: string;
  name: string;
  description?: string;
  product_id?: string;
  product_name?: string;
  phase: 'concept' | 'evt' | 'dvt' | 'pvt' | 'mp' | 'completed';
  status: 'planning' | 'active' | 'on_hold' | 'completed' | 'cancelled';
  start_date?: string;
  target_date?: string;
  actual_end_date?: string;
  manager_id?: string;
  manager_name?: string;
  progress: number;
  created_at: string;
  updated_at: string;
}

// 任务
export interface Task {
  id: string;
  project_id: string;
  title: string;
  description?: string;
  status: 'pending' | 'in_progress' | 'completed' | 'cancelled';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  assignee_id?: string;
  assignee_name?: string;
  due_date?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

// ECN 变更
export interface ECN {
  id: string;
  code: string;
  title: string;
  description?: string;
  type: 'design' | 'material' | 'process' | 'other';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  status: 'draft' | 'pending' | 'approved' | 'rejected' | 'implemented';
  product_id?: string;
  product_name?: string;
  requester_id: string;
  requester_name?: string;
  approver_id?: string;
  approver_name?: string;
  created_at: string;
  updated_at: string;
}

// 文档
export interface Document {
  id: string;
  code: string;
  title: string;
  description?: string;
  category: string;
  file_path: string;
  file_size: number;
  file_type: string;
  version: string;
  status: 'draft' | 'released' | 'obsolete';
  product_id?: string;
  product_name?: string;
  created_by: string;
  created_by_name?: string;
  created_at: string;
  updated_at: string;
}

// SRM Status Types
export type SupplierStatus = 'potential' | 'approved' | 'active' | 'suspended' | 'blacklisted';
export type SupplierCategory = 'manufacturer' | 'distributor' | 'service' | 'raw_material';
export type PRStatus = 'draft' | 'pending' | 'approved' | 'rejected' | 'cancelled';
export type PRPriority = 'low' | 'medium' | 'high' | 'urgent';
export type POStatus = 'draft' | 'submitted' | 'approved' | 'shipped' | 'received' | 'completed' | 'cancelled';
export type InspectionStatus = 'pending' | 'in_progress' | 'completed';
export type InspectionResult = 'pending' | 'pass' | 'fail' | 'conditional';
export type SettlementStatus = 'draft' | 'confirmed' | 'invoiced' | 'paid';
export type CAStatus = 'open' | 'responded' | 'verified' | 'closed';
export type CASeverity = 'low' | 'medium' | 'high' | 'critical';
export type EvaluationStatus = 'draft' | 'submitted' | 'approved';
export type EvaluationGrade = 'A' | 'B' | 'C' | 'D';

// SRM entity interfaces are defined in @/api/srm.ts
// Import them from there: import { Supplier, PurchaseOrder, ... } from '@/api/srm'
