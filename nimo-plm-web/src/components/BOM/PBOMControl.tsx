import React, { useState, useMemo, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button, Typography, Tag, Empty, Space, Spin } from 'antd';
import {
  PlusOutlined,
  GiftOutlined,
  ToolOutlined,
  ExperimentOutlined,
} from '@ant-design/icons';
import { projectBomApi } from '@/api/projectBom';
import type { CategoryAttrTemplate } from '@/api/projectBom';
import DynamicBOMTable from './DynamicBOMTable';
import BOMCategoryView from './BOMCategoryView';
import AddMaterialModal from './AddMaterialModal';
import {
  CATEGORY_LABELS,
  SUB_CATEGORY_LABELS,
  PBOM_CATEGORIES,
  CATEGORY_SUB_CATEGORIES,
  type BOMControlConfig,
} from './bomConstants';
import { useIsMobile } from '@/hooks/useIsMobile';

const { Text } = Typography;

// ========== Types ==========

export interface PBOMControlProps {
  config: BOMControlConfig;
  value: Record<string, any>[];
  onChange: (items: Record<string, any>[]) => void;
  readonly?: boolean;
  onItemSave?: (itemId: string, field: string, value: any) => void;
  onImportExcel?: () => void;
  showMaterialCode?: boolean;
  editableCategories?: string[];
  onItemClick?: (item: Record<string, any>) => void;
  onMobileAddRow?: (category: string, subCategory: string) => void;
}

// ========== Category icons ==========

const CATEGORY_ICONS: Record<string, React.ReactNode> = {
  packaging: <GiftOutlined style={{ color: '#595959' }} />,
  tooling: <ToolOutlined style={{ color: '#595959' }} />,
  consumable: <ExperimentOutlined style={{ color: '#595959' }} />,
};

const CATEGORY_COLORS: Record<string, string> = {
  packaging: '#fff7e6',
  tooling: '#f9f0ff',
  consumable: '#f0f5ff',
};

// ========== Helpers ==========

const formatCurrency = (v: number | null | undefined): string => {
  if (v == null || isNaN(v)) return '-';
  return `\u00a5${v.toFixed(2)}`;
};

// ========== Component ==========

const PBOMControl: React.FC<PBOMControlProps> = ({
  config,
  value: rawValue,
  onChange,
  readonly = false,
  onItemSave,
  onImportExcel,
  showMaterialCode = false,
  editableCategories,
  onItemClick,
  onMobileAddRow,
}) => {
  const value = Array.isArray(rawValue) ? rawValue : [];
  const isMobile = useIsMobile();

  // Fetch templates via react-query (cached globally, 5min staleTime)
  const { data: allTemplates = [], isLoading: templatesLoading } = useQuery({
    queryKey: ['bom-attr-templates'],
    queryFn: () => projectBomApi.listTemplates(),
    staleTime: 5 * 60 * 1000,
  });

  const isCategoryEditable = useCallback((category: string): boolean => {
    if (readonly) return false;
    if (!editableCategories) return true;
    return editableCategories.includes(category);
  }, [readonly, editableCategories]);

  // Determine which categories and sub-categories to show
  const visibleCategories = useMemo(() => {
    if (config?.visible_categories?.length > 0) {
      return config.visible_categories.filter(c => PBOM_CATEGORIES.includes(c));
    }
    return PBOM_CATEGORIES;
  }, [config]);

  const getEnabledSubCategories = useCallback((category: string): string[] => {
    if (config?.category_config?.[category]?.enabled_sub_categories?.length > 0) {
      return config.category_config[category].enabled_sub_categories;
    }
    return CATEGORY_SUB_CATEGORIES[category] || [];
  }, [config]);

  const getSubCategoryOrder = useCallback((category: string): string[] => {
    if (config?.category_config?.[category]?.sub_category_order?.length > 0) {
      return config.category_config[category].sub_category_order;
    }
    return getEnabledSubCategories(category);
  }, [config, getEnabledSubCategories]);

  const getFieldOrder = useCallback((category: string, subCategory: string): string[] | undefined => {
    return config?.category_config?.[category]?.field_order?.[subCategory];
  }, [config]);

  // Group items by sub_category
  const itemsBySubCategory = useMemo(() => {
    const grouped: Record<string, Record<string, any>[]> = {};
    for (const item of value) {
      const sc = item.sub_category || 'unknown';
      if (!grouped[sc]) grouped[sc] = [];
      grouped[sc].push(item);
    }
    return grouped;
  }, [value]);

  // Get templates for a specific sub_category
  const getTemplatesForSubCategory = useCallback((subCategory: string): CategoryAttrTemplate[] => {
    return allTemplates.filter(t =>
      t.sub_category === subCategory && t.bom_type === 'PBOM'
    );
  }, [allTemplates]);

  // Handle item changes for a specific sub_category
  const handleSubCategoryChange = useCallback((subCategory: string, newItems: Record<string, any>[]) => {
    const otherItems = value.filter(item => item.sub_category !== subCategory);
    onChange([...otherItems, ...newItems]);
  }, [value, onChange]);

  // Modal state for "先搜后建" add material flow
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [addModalTarget, setAddModalTarget] = useState<{ category: string; subCategory: string }>({ category: '', subCategory: '' });

  // Open modal for adding material
  const handleOpenAddModal = useCallback((category: string, subCategory: string) => {
    setAddModalTarget({ category, subCategory });
    setAddModalOpen(true);
  }, []);

  // Add item helper (used by modal callbacks)
  const addNewItem = useCallback((itemData: Record<string, any>) => {
    const maxNum = value.reduce((m, i) => Math.max(m, i.item_number || 0), 0);
    const newItem: Record<string, any> = {
      id: 'new-' + Date.now(),
      item_number: maxNum + 1,
      category: addModalTarget.category,
      sub_category: addModalTarget.subCategory,
      quantity: 1,
      unit: 'pcs',
      extended_attrs: {},
      ...itemData,
    };
    onChange([...value, newItem]);
    setAddModalOpen(false);
  }, [value, onChange, addModalTarget]);

  // Legacy direct add (skip modal — for mobile or fallback)
  const handleAddRow = useCallback((category: string, subCategory: string) => {
    const maxNum = value.reduce((m, i) => Math.max(m, i.item_number || 0), 0);
    const newItem: Record<string, any> = {
      id: 'new-' + Date.now(),
      item_number: maxNum + 1,
      category,
      sub_category: subCategory,
      name: '',
      quantity: 1,
      unit: 'pcs',
      extended_attrs: {},
    };
    onChange([...value, newItem]);
  }, [value, onChange]);

  // Calculate total cost
  const totalCost = useMemo(() => {
    return value.reduce((sum, item) => {
      const cost = item.extended_cost ?? ((item.quantity || 0) * (item.unit_price || 0));
      return sum + (cost || 0);
    }, 0);
  }, [value]);

  // Render a section for a sub_category
  const renderSubCategorySection = (category: string, subCategory: string) => {
    const items = itemsBySubCategory[subCategory] || [];
    const templates = getTemplatesForSubCategory(subCategory);
    const fieldOrder = getFieldOrder(category, subCategory);
    const label = SUB_CATEGORY_LABELS[subCategory] || subCategory;
    const icon = CATEGORY_ICONS[category] || <ExperimentOutlined style={{ color: '#595959' }} />;
    const bgColor = CATEGORY_COLORS[category] || '#fafafa';
    const catEditable = isCategoryEditable(category);

    // In readonly mode, skip sections with no items
    if (readonly && items.length === 0) return null;

    return (
      <div key={`${category}-${subCategory}`} style={{ marginBottom: 16, opacity: !catEditable && !readonly ? 0.6 : 1 }}>
        {/* Section header */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8, padding: '6px 12px',
          background: bgColor, borderRadius: '6px 6px 0 0', borderBottom: '1px solid #f0f0f0',
        }}>
          {icon}
          <Text strong style={{ fontSize: 13 }}>{label}</Text>
          <Tag style={{ fontSize: 11 }}>{items.length} 项</Tag>
        </div>

        {/* Table */}
        <DynamicBOMTable
          subCategory={subCategory}
          items={items}
          onChange={(newItems) => handleSubCategoryChange(subCategory, newItems)}
          fieldOrder={fieldOrder}
          templates={templates}
          readonly={readonly || !catEditable}
          onItemSave={onItemSave}
          showMaterialCode={showMaterialCode}
          bomType="PBOM"
          category={category}
        />

        {/* Add material button */}
        {catEditable && (
          <div style={{ padding: '4px 12px' }}>
            <Button
              type="dashed"
              size="small"
              icon={<PlusOutlined />}
              onClick={() => handleOpenAddModal(category, subCategory)}
            >
              添加物料
            </Button>
          </div>
        )}
      </div>
    );
  };

  // Wait for templates before rendering tables
  if (templatesLoading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>;
  }

  // Mobile layout — Meituan-style category view
  if (isMobile) {
    return (
      <BOMCategoryView
        bomType="PBOM"
        items={value}
        categories={visibleCategories}
        getEnabledSubCategories={getEnabledSubCategories}
        readonly={readonly}
        totalCost={totalCost}
        onAddRow={readonly ? undefined : (onMobileAddRow || handleAddRow)}
        onItemClick={onItemClick}
      />
    );
  }

  return (
    <div>
      {/* Import button */}
      {!readonly && onImportExcel && (
        <div style={{ marginBottom: 12 }}>
          <Button size="small" onClick={onImportExcel}>导入Excel</Button>
        </div>
      )}

      {/* Empty state */}
      {value.length === 0 && readonly && (
        <Empty description="暂无PBOM数据" />
      )}

      {/* Category sections */}
      {visibleCategories.map(category => {
        const subCategories = getSubCategoryOrder(category);
        const enabledSubs = getEnabledSubCategories(category);

        return (
          <div key={category}>
            {subCategories
              .filter(sc => enabledSubs.includes(sc))
              .map(sc => renderSubCategorySection(category, sc))}
          </div>
        );
      })}

      {/* Process route placeholder */}
      {config?.show_route_editor && (
        <div style={{
          marginTop: 16, padding: '16px', background: '#fafafa',
          borderRadius: 6, border: '1px dashed #d9d9d9', textAlign: 'center',
        }}>
          <Text type="secondary">工艺路线编辑器（待开发）</Text>
        </div>
      )}

      {/* Cost summary */}
      {value.length > 0 && (
        <div style={{
          display: 'flex', gap: 24, padding: '10px 16px', background: '#fafafa',
          borderRadius: 6, border: '1px solid #f0f0f0', flexWrap: 'wrap', marginTop: 8,
        }}>
          {visibleCategories.map(category => {
            const categoryItems = value.filter(item => item.category === category);
            const cost = categoryItems.reduce((sum, item) => {
              const c = item.extended_cost ?? ((item.quantity || 0) * (item.unit_price || 0));
              return sum + (c || 0);
            }, 0);
            return (
              <Text key={category} style={{ fontSize: 12 }}>
                {CATEGORY_LABELS[category] || category}: <Text strong>{formatCurrency(cost)}</Text>
              </Text>
            );
          })}
          <Text style={{ fontSize: 13 }}>
            PBOM总成本: <Text strong style={{ color: '#1677ff' }}>{formatCurrency(totalCost)}</Text>
          </Text>
          <Space>
            <Text type="secondary" style={{ fontSize: 12 }}>共 {value.length} 项</Text>
          </Space>
        </div>
      )}
      {/* 添加物料弹窗 */}
      <AddMaterialModal
        open={addModalOpen}
        onClose={() => setAddModalOpen(false)}
        onSelect={addNewItem}
        onCreate={addNewItem}
        onSkip={() => {
          handleAddRow(addModalTarget.category, addModalTarget.subCategory);
          setAddModalOpen(false);
        }}
        category={addModalTarget.category}
        subCategory={addModalTarget.subCategory}
      />
    </div>
  );
};

export default PBOMControl;
