import React, { useMemo, useCallback } from 'react';
import { Table, InputNumber, Typography, Tag, Empty, Space } from 'antd';
import {
  ExperimentOutlined,
  BuildOutlined,
  EyeOutlined,
  GiftOutlined,
  ToolOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import BOMCategoryView from './BOMCategoryView';
import {
  SUB_CATEGORY_LABELS,
  ALL_CATEGORIES,
  EBOM_CATEGORIES,
  PBOM_CATEGORIES,
  CATEGORY_SUB_CATEGORIES,
  type BOMControlConfig,
} from './bomConstants';
import { useIsMobile } from '@/hooks/useIsMobile';

const { Text } = Typography;

// ========== Types ==========

export interface MBOMControlProps {
  config: BOMControlConfig;
  value: Record<string, any>[];
  onChange: (items: Record<string, any>[]) => void;
  readonly?: boolean;
  onItemSave?: (itemId: string, field: string, value: any) => void;
  onImportExcel?: () => void;
  showMaterialCode?: boolean;
  ebomData?: Record<string, any>[];
  pbomData?: Record<string, any>[];
  editableCategories?: string[];
  onItemClick?: (item: Record<string, any>) => void;
}

// ========== Category icons ==========

const CATEGORY_ICONS: Record<string, React.ReactNode> = {
  electronic: <ExperimentOutlined style={{ color: '#595959' }} />,
  structural: <BuildOutlined style={{ color: '#595959' }} />,
  optical: <EyeOutlined style={{ color: '#595959' }} />,
  packaging: <GiftOutlined style={{ color: '#595959' }} />,
  tooling: <ToolOutlined style={{ color: '#595959' }} />,
  consumable: <ExperimentOutlined style={{ color: '#595959' }} />,
};

const CATEGORY_COLORS: Record<string, string> = {
  electronic: '#f0f5ff',
  structural: '#f6ffed',
  optical: '#fff7e6',
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

const MBOMControl: React.FC<MBOMControlProps> = ({
  config,
  value: rawValue,
  onChange,
  readonly = false,
  onItemSave,
  ebomData,
  pbomData,
  showMaterialCode = false,
  editableCategories: _editableCategories,
  onItemClick,
}) => {
  const value = Array.isArray(rawValue) ? rawValue : [];
  const isMobile = useIsMobile();
  // Merge all source data (EBOM + PBOM) into a read-only combined view
  const allSourceItems = useMemo(() => {
    const ebom = (ebomData || []).map(i => ({ ...i, _source: 'EBOM' }));
    const pbom = (pbomData || []).map(i => ({ ...i, _source: 'PBOM' }));
    return [...ebom, ...pbom];
  }, [ebomData, pbomData]);

  // The MBOM items (editable) -- these are value
  // Group by sub_category for display
  const itemsBySubCategory = useMemo(() => {
    const grouped: Record<string, Record<string, any>[]> = {};
    for (const item of value) {
      const sc = item.sub_category || 'unknown';
      if (!grouped[sc]) grouped[sc] = [];
      grouped[sc].push(item);
    }
    return grouped;
  }, [value]);

  // Determine visible categories -- MBOM shows all
  const visibleCategories = useMemo(() => {
    if (config?.visible_categories?.length > 0) {
      return config.visible_categories;
    }
    return ALL_CATEGORIES;
  }, [config]);

  const getEnabledSubCategories = useCallback((category: string): string[] => {
    if (config?.category_config?.[category]?.enabled_sub_categories?.length > 0) {
      return config.category_config[category].enabled_sub_categories;
    }
    return CATEGORY_SUB_CATEGORIES[category] || [];
  }, [config]);

  // Handle scrap rate change
  const handleScrapRateChange = useCallback((itemId: string, scrapRate: number) => {
    if (onItemSave) {
      onItemSave(itemId, 'scrap_rate', scrapRate);
    } else {
      const updated = value.map(item =>
        item.id === itemId ? { ...item, scrap_rate: scrapRate } : item
      );
      onChange(updated);
    }
  }, [value, onChange, onItemSave]);

  // Build columns for MBOM table section
  const buildColumns = useCallback((_subCategory: string): ColumnsType<Record<string, any>> => {
    const cols: ColumnsType<Record<string, any>> = [
      { title: '序号', dataIndex: 'item_number', width: 55, align: 'center' },
    ];

    if (showMaterialCode) {
      cols.push({
        title: '物料编码',
        dataIndex: 'material_code',
        width: 150,
        render: (v: string) => v ? (
          <span style={{
            background: '#f5f5f5',
            border: '1px solid #d9d9d9',
            borderRadius: 4,
            padding: '2px 8px',
            fontFamily: 'monospace',
            fontSize: 13,
            whiteSpace: 'nowrap',
          }}>
            {v}
          </span>
        ) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>,
      });
    }

    cols.push(
      { title: '名称', dataIndex: 'name', width: 120 },
      { title: '数量', dataIndex: 'quantity', width: 60, align: 'right' },
      { title: '单位', dataIndex: 'unit', width: 55 },
      { title: '供应商', dataIndex: 'supplier', width: 100, ellipsis: true },
      { title: '单价', dataIndex: 'unit_price', width: 80, align: 'right',
        render: (v) => v != null ? Number(v).toFixed(2) : '-' },
      { title: '小计', width: 80, align: 'right',
        render: (_, record) => {
          const cost = record.extended_cost ?? ((record.quantity || 0) * (record.unit_price || 0));
          return cost > 0 ? Number(cost).toFixed(2) : '-';
        },
      },
    );

    // Add scrap_rate column if editable
    if (config?.editable_scrap_rate) {
      cols.push({
        title: '损耗率(%)',
        dataIndex: 'scrap_rate',
        width: 90,
        align: 'right',
        render: (v, record) => {
          if (readonly) {
            return v != null ? `${v}%` : '-';
          }
          return (
            <InputNumber
              size="small"
              value={v}
              min={0}
              max={100}
              step={0.1}
              style={{ width: '100%' }}
              onChange={(val) => handleScrapRateChange(record.id, val ?? 0)}
            />
          );
        },
      });
    }

    // Add source indicator column
    cols.push({
      title: '来源',
      dataIndex: '_source',
      width: 70,
      align: 'center',
      render: (v) => v ? <Tag color={v === 'EBOM' ? 'blue' : 'orange'}>{v}</Tag> : '-',
    });

    cols.push({
      title: '备注', dataIndex: 'notes', width: 120, ellipsis: true,
    });

    return cols;
  }, [config, readonly, handleScrapRateChange, showMaterialCode]);

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
    const label = SUB_CATEGORY_LABELS[subCategory] || subCategory;
    const icon = CATEGORY_ICONS[category] || <ExperimentOutlined style={{ color: '#595959' }} />;
    const bgColor = CATEGORY_COLORS[category] || '#fafafa';
    const columns = buildColumns(subCategory);

    // Skip sections with no items
    if (items.length === 0) return null;

    // Flatten extended_attrs for display
    const flatItems = items.map(item => ({
      ...item,
      ...(item.extended_attrs || {}),
    }));

    return (
      <div key={`${category}-${subCategory}`} style={{ marginBottom: 16 }}>
        {/* Section header */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8, padding: '6px 12px',
          background: bgColor, borderRadius: '6px 6px 0 0', borderBottom: '1px solid #f0f0f0',
        }}>
          {icon}
          <Text strong style={{ fontSize: 13 }}>{label}</Text>
          <Tag style={{ fontSize: 11 }}>{items.length} 项</Tag>
        </div>

        <Table
          columns={columns}
          dataSource={flatItems}
          rowKey={(r) => r.id || String(Math.random())}
          size="small"
          pagination={items.length > 10 ? { pageSize: 10, size: 'small' } : false}
          scroll={{ x: 800 }}
          style={{ fontSize: 12 }}
        />
      </div>
    );
  };

  // Mobile layout — Meituan-style category view
  if (isMobile) {
    return (
      <BOMCategoryView
        bomType="MBOM"
        items={value}
        categories={visibleCategories}
        getEnabledSubCategories={getEnabledSubCategories}
        readonly={true}
        totalCost={totalCost}
        onItemClick={onItemClick}
      />
    );
  }

  return (
    <div>
      {/* Empty state */}
      {value.length === 0 && allSourceItems.length === 0 && (
        <Empty description="暂无MBOM数据，请先完成EBOM/PBOM" />
      )}

      {/* Source data summary */}
      {allSourceItems.length > 0 && value.length === 0 && (
        <div style={{
          padding: '12px 16px', background: '#fffbe6', borderRadius: 6,
          border: '1px solid #ffe58f', marginBottom: 16,
        }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            检测到 {(ebomData || []).length} 条EBOM数据和 {(pbomData || []).length} 条PBOM数据，
            MBOM将合并展示所有物料。
          </Text>
        </div>
      )}

      {/* Category sections */}
      {visibleCategories.map(category => {
        const enabledSubs = getEnabledSubCategories(category);
        return (
          <div key={category}>
            {enabledSubs.map(sc => renderSubCategorySection(category, sc))}
          </div>
        );
      })}

      {/* Cost summary */}
      {value.length > 0 && (
        <div style={{
          display: 'flex', gap: 24, padding: '10px 16px', background: '#fafafa',
          borderRadius: 6, border: '1px solid #f0f0f0', flexWrap: 'wrap', marginTop: 8,
        }}>
          <Text style={{ fontSize: 12 }}>
            EBOM部分: <Text strong>{formatCurrency(
              value.filter(i => EBOM_CATEGORIES.includes(i.category))
                .reduce((s, i) => s + ((i.extended_cost ?? ((i.quantity || 0) * (i.unit_price || 0))) || 0), 0)
            )}</Text>
          </Text>
          <Text style={{ fontSize: 12 }}>
            PBOM部分: <Text strong>{formatCurrency(
              value.filter(i => PBOM_CATEGORIES.includes(i.category))
                .reduce((s, i) => s + ((i.extended_cost ?? ((i.quantity || 0) * (i.unit_price || 0))) || 0), 0)
            )}</Text>
          </Text>
          <Text style={{ fontSize: 13 }}>
            MBOM总成本: <Text strong style={{ color: '#1677ff' }}>{formatCurrency(totalCost)}</Text>
          </Text>
          <Space>
            <Text type="secondary" style={{ fontSize: 12 }}>共 {value.length} 项</Text>
          </Space>
        </div>
      )}
    </div>
  );
};

export default MBOMControl;
