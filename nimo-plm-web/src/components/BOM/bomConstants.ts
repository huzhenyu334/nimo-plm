// BOM configuration constants for template designer

export interface BOMControlConfig {
  bom_type: 'EBOM' | 'PBOM' | 'MBOM';
  visible_categories: string[];
  category_config: Record<string, {
    enabled_sub_categories: string[];
    sub_category_order: string[];
    field_order: Record<string, string[]>;
  }>;
  // PBOM only
  show_route_editor?: boolean;
  // MBOM only
  editable_scrap_rate?: boolean;
  show_freeze_button?: boolean;
}

export const CATEGORY_LABELS: Record<string, string> = {
  electronic: '电子',
  structural: '结构',
  optical: '光学',
  packaging: '包装',
  tooling: '工装',
  consumable: '辅料',
};

export const SUB_CATEGORY_LABELS: Record<string, string> = {
  component: '元器件',
  pcb: 'PCB',
  connector: '连接器',
  cable: '线缆',
  structural_part: '结构件',
  fastener: '紧固件',
  light_engine: '光机',
  waveguide: '波导',
  lens: '镜片',
  lightguide: '导光板',
  box: '彩盒',
  document: '说明书/卡片',
  cushion: '内衬/缓冲',
  mold: '模具',
  fixture: '治具/检具',
  consumable: '辅料',
};

// Category -> sub-category mapping
export const CATEGORY_SUB_CATEGORIES: Record<string, string[]> = {
  electronic: ['component', 'pcb', 'connector', 'cable'],
  structural: ['structural_part', 'fastener'],
  optical: ['light_engine', 'waveguide', 'lens', 'lightguide'],
  packaging: ['box', 'document', 'cushion'],
  tooling: ['mold', 'fixture'],
  consumable: ['consumable'],
};

export const EBOM_CATEGORIES = ['electronic', 'structural', 'optical'];
export const PBOM_CATEGORIES = ['packaging', 'tooling', 'consumable'];
export const ALL_CATEGORIES = [...EBOM_CATEGORIES, ...PBOM_CATEGORIES];

// Common fields always shown in BOM tables, not configurable
export const COMMON_FIELDS = [
  'name',
  'quantity',
  'unit',
  'supplier',
  'unit_price',
  'extended_cost',
  'notes',
];
