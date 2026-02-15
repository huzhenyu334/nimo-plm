import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Checkbox, Collapse, Typography, Button, Spin, Empty, Switch, Tooltip } from 'antd';
import { UpOutlined, DownOutlined } from '@ant-design/icons';
import { projectBomApi, CategoryAttrTemplate } from '@/api/projectBom';
import {
  BOMControlConfig,
  CATEGORY_LABELS,
  SUB_CATEGORY_LABELS,
  CATEGORY_SUB_CATEGORIES,
  EBOM_CATEGORIES,
  PBOM_CATEGORIES,
  ALL_CATEGORIES,
  COMMON_FIELDS,
} from './bomConstants';

const { Text } = Typography;

interface BOMConfigPanelProps {
  bomType: 'EBOM' | 'PBOM' | 'MBOM';
  config: Partial<BOMControlConfig> | Record<string, any>;
  onChange: (config: BOMControlConfig) => void;
}

/** Normalize a partial/empty config into a full BOMControlConfig */
function normalizeConfig(raw: Partial<BOMControlConfig> | Record<string, any>, bomType: 'EBOM' | 'PBOM' | 'MBOM'): BOMControlConfig {
  return {
    bom_type: (raw as any).bom_type || bomType,
    visible_categories: Array.isArray((raw as any).visible_categories) ? (raw as any).visible_categories : [],
    category_config: (raw as any).category_config || {},
    show_route_editor: (raw as any).show_route_editor,
    editable_scrap_rate: (raw as any).editable_scrap_rate,
    show_freeze_button: (raw as any).show_freeze_button,
  };
}

// Grouped templates by category+sub_category
interface TemplateGroup {
  category: string;
  sub_category: string;
  fields: CategoryAttrTemplate[];
}

const BOMConfigPanel: React.FC<BOMConfigPanelProps> = ({ bomType, config: rawConfig, onChange }) => {
  const config = useMemo(() => normalizeConfig(rawConfig, bomType), [rawConfig, bomType]);
  const [templates, setTemplates] = useState<CategoryAttrTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  // Fix 4: Controlled Collapse activeKey to prevent checkbox clicks from toggling collapse
  const [activeKeys, setActiveKeys] = useState<string[]>([...config.visible_categories]);
  // Ref for accessing latest config in effects without adding it as a dependency
  const configRef = useRef(config);
  configRef.current = config;

  // Which categories are available for this bom type
  const availableCategories = useMemo(() => {
    if (bomType === 'EBOM') return EBOM_CATEGORIES;
    if (bomType === 'PBOM') return PBOM_CATEGORIES;
    return ALL_CATEGORIES; // MBOM shows all
  }, [bomType]);

  // Fetch templates when visible_categories change
  useEffect(() => {
    if (config.visible_categories.length === 0) {
      setTemplates([]);
      return;
    }

    let cancelled = false;
    const fetchTemplates = async () => {
      setLoading(true);
      try {
        // Fetch templates for all visible categories in parallel
        const promises = config.visible_categories.map((cat) =>
          projectBomApi.listTemplates({ category: cat }).catch(() => [] as CategoryAttrTemplate[])
        );
        const results = await Promise.all(promises);
        if (!cancelled) {
          setTemplates(results.flat());
        }
      } catch {
        // silently ignore
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    fetchTemplates();
    return () => { cancelled = true; };
  }, [config.visible_categories]);

  // Fix 3: Auto-populate field_order when templates arrive (handles async template loading)
  useEffect(() => {
    if (templates.length === 0) return;
    const currentConfig = configRef.current;
    if (currentConfig.visible_categories.length === 0) return;

    let changed = false;
    const newCategoryConfig = { ...currentConfig.category_config };

    for (const cat of currentConfig.visible_categories) {
      const catConfig = newCategoryConfig[cat];
      if (!catConfig) continue;

      const subs = catConfig.enabled_sub_categories || [];
      let fieldOrderChanged = false;
      const newFieldOrder = { ...catConfig.field_order };

      for (const sc of subs) {
        // Only auto-populate if field_order is missing or empty for this sub_category
        if (!newFieldOrder[sc] || newFieldOrder[sc].length === 0) {
          const scTemplates = templates.filter(
            (t) => t.category === cat && t.sub_category === sc && !COMMON_FIELDS.includes(t.field_key)
          );
          if (scTemplates.length > 0) {
            newFieldOrder[sc] = scTemplates.sort((a, b) => a.sort_order - b.sort_order).map((t) => t.field_key);
            fieldOrderChanged = true;
          }
        }
      }

      if (fieldOrderChanged) {
        newCategoryConfig[cat] = { ...catConfig, field_order: newFieldOrder };
        changed = true;
      }
    }

    if (changed) {
      onChange({ ...currentConfig, category_config: newCategoryConfig });
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [templates]); // Only trigger on template changes; reads config via ref

  // Group templates by category + sub_category
  const templateGroups = useMemo(() => {
    const map = new Map<string, TemplateGroup>();
    for (const t of templates) {
      const key = `${t.category}::${t.sub_category}`;
      if (!map.has(key)) {
        map.set(key, { category: t.category, sub_category: t.sub_category, fields: [] });
      }
      map.get(key)!.fields.push(t);
    }
    // Sort fields within each group by sort_order
    for (const group of map.values()) {
      group.fields.sort((a, b) => a.sort_order - b.sort_order);
    }
    return map;
  }, [templates]);

  // Helper: get or create category_config entry
  const getCatConfig = useCallback((category: string) => {
    return config.category_config[category] || {
      enabled_sub_categories: [],
      sub_category_order: [],
      field_order: {},
    };
  }, [config.category_config]);

  // Fix 3: Toggle category visibility — cascade to all sub-categories and fields
  const handleCategoryToggle = useCallback((checkedCategories: string[]) => {
    const newCategoryConfig = { ...config.category_config };

    // Clean up removed categories
    for (const cat of config.visible_categories) {
      if (!checkedCategories.includes(cat)) {
        delete newCategoryConfig[cat];
      }
    }

    // Auto-populate only newly added categories with all sub-categories + all fields
    for (const cat of checkedCategories) {
      if (!config.visible_categories.includes(cat)) {
        const subs = CATEGORY_SUB_CATEGORIES[cat] || [];
        const fieldOrder: Record<string, string[]> = {};
        // Try to populate from currently loaded templates (may be empty for new categories)
        for (const sc of subs) {
          const scTemplates = templates.filter(
            (t) => t.category === cat && t.sub_category === sc && !COMMON_FIELDS.includes(t.field_key)
          );
          if (scTemplates.length > 0) {
            fieldOrder[sc] = scTemplates.sort((a, b) => a.sort_order - b.sort_order).map((t) => t.field_key);
          }
        }
        newCategoryConfig[cat] = {
          enabled_sub_categories: [...subs],
          sub_category_order: [...subs],
          field_order: fieldOrder,
        };
        // Auto-expand newly added category in Collapse
        setActiveKeys(prev => prev.includes(cat) ? prev : [...prev, cat]);
      }
    }

    onChange({
      ...config,
      visible_categories: checkedCategories,
      category_config: newCategoryConfig,
    });
  }, [config, onChange, templates]);

  // Toggle sub-category (Fix 4: keep sub_category_order stable on uncheck)
  const handleSubCategoryToggle = useCallback((category: string, subCategory: string, checked: boolean) => {
    const catConfig = getCatConfig(category);

    if (checked) {
      const newEnabled = [...catConfig.enabled_sub_categories, subCategory];
      // Add to order only if not already present
      const newOrder = catConfig.sub_category_order.includes(subCategory)
        ? [...catConfig.sub_category_order]
        : [...catConfig.sub_category_order, subCategory];
      // Auto-select all fields for this sub_category
      const scTemplates = templates.filter(
        (t) => t.category === category && t.sub_category === subCategory && !COMMON_FIELDS.includes(t.field_key)
      );
      const newFieldOrder = { ...catConfig.field_order };
      if (scTemplates.length > 0) {
        newFieldOrder[subCategory] = scTemplates.sort((a, b) => a.sort_order - b.sort_order).map((t) => t.field_key);
      }
      onChange({
        ...config,
        category_config: {
          ...config.category_config,
          [category]: { ...catConfig, enabled_sub_categories: newEnabled, sub_category_order: newOrder, field_order: newFieldOrder },
        },
      });
    } else {
      const newEnabled = catConfig.enabled_sub_categories.filter((sc) => sc !== subCategory);
      // Fix 4: Keep sub_category_order unchanged on uncheck (preserve position)
      const newFieldOrder = { ...catConfig.field_order };
      delete newFieldOrder[subCategory];
      onChange({
        ...config,
        category_config: {
          ...config.category_config,
          [category]: { ...catConfig, enabled_sub_categories: newEnabled, field_order: newFieldOrder },
        },
      });
    }
  }, [config, getCatConfig, onChange, templates]);

  // Toggle a field in field_order
  const handleFieldToggle = useCallback((category: string, subCategory: string, fieldKey: string, checked: boolean) => {
    const catConfig = getCatConfig(category);
    const currentOrder = catConfig.field_order[subCategory] || [];
    let newOrder: string[];

    if (checked) {
      newOrder = [...currentOrder, fieldKey];
    } else {
      newOrder = currentOrder.filter((k) => k !== fieldKey);
    }

    onChange({
      ...config,
      category_config: {
        ...config.category_config,
        [category]: {
          ...catConfig,
          field_order: { ...catConfig.field_order, [subCategory]: newOrder },
        },
      },
    });
  }, [config, getCatConfig, onChange]);

  // Move field up/down in order
  const handleFieldMove = useCallback((category: string, subCategory: string, fieldKey: string, direction: 'up' | 'down') => {
    const catConfig = getCatConfig(category);
    const currentOrder = [...(catConfig.field_order[subCategory] || [])];
    const idx = currentOrder.indexOf(fieldKey);
    if (idx < 0) return;

    const swapIdx = direction === 'up' ? idx - 1 : idx + 1;
    if (swapIdx < 0 || swapIdx >= currentOrder.length) return;

    [currentOrder[idx], currentOrder[swapIdx]] = [currentOrder[swapIdx], currentOrder[idx]];

    onChange({
      ...config,
      category_config: {
        ...config.category_config,
        [category]: {
          ...catConfig,
          field_order: { ...catConfig.field_order, [subCategory]: currentOrder },
        },
      },
    });
  }, [config, getCatConfig, onChange]);

  // Move sub-category up/down in order
  const handleSubCategoryMove = useCallback((category: string, subCategory: string, direction: 'up' | 'down') => {
    const catConfig = getCatConfig(category);
    const order = [...catConfig.sub_category_order];
    const idx = order.indexOf(subCategory);
    if (idx < 0) return;

    const swapIdx = direction === 'up' ? idx - 1 : idx + 1;
    if (swapIdx < 0 || swapIdx >= order.length) return;

    [order[idx], order[swapIdx]] = [order[swapIdx], order[idx]];

    onChange({
      ...config,
      category_config: {
        ...config.category_config,
        [category]: { ...catConfig, sub_category_order: order },
      },
    });
  }, [config, getCatConfig, onChange]);

  // Render field list for a sub-category (Fix 4: stable display order)
  const renderFieldList = (category: string, subCategory: string) => {
    const groupKey = `${category}::${subCategory}`;
    const group = templateGroups.get(groupKey);
    const catConfig = getCatConfig(category);
    const fieldOrder = catConfig.field_order[subCategory] || [];

    if (!group || group.fields.length === 0) {
      return (
        <div style={{ padding: '4px 0 4px 24px' }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            暂无属性模板字段
          </Text>
        </div>
      );
    }

    // Filter out common fields that are always shown
    const configurableFields = group.fields.filter((f) => !COMMON_FIELDS.includes(f.field_key));

    if (configurableFields.length === 0) {
      return (
        <div style={{ padding: '4px 0 4px 24px' }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            所有字段均为通用字段，无需配置
          </Text>
        </div>
      );
    }

    // Fix 4: Stable display order — always use sort_order from templates
    const allFields = [...configurableFields].sort((a, b) => a.sort_order - b.sort_order);

    return (
      <div style={{ padding: '4px 0 4px 24px' }}>
        {allFields.map((field) => {
          const isEnabled = fieldOrder.includes(field.field_key);
          const orderIdx = fieldOrder.indexOf(field.field_key);

          return (
            <div
              key={field.field_key}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '2px 0',
                borderBottom: '1px solid #f5f5f5',
              }}
            >
              <Checkbox
                checked={isEnabled}
                onChange={(e) => {
                  e.stopPropagation();
                  handleFieldToggle(category, subCategory, field.field_key, e.target.checked);
                }}
              />
              <Text style={{ flex: 1, fontSize: 13 }}>
                {field.field_name}
                {field.unit && (
                  <Text type="secondary" style={{ fontSize: 11, marginLeft: 4 }}>
                    ({field.unit})
                  </Text>
                )}
                {field.required && (
                  <Text type="danger" style={{ fontSize: 11, marginLeft: 2 }}>*</Text>
                )}
              </Text>
              <Text type="secondary" style={{ fontSize: 11, width: 40, textAlign: 'center' }}>
                {field.field_type}
              </Text>
              {isEnabled && (
                <span style={{ display: 'inline-flex', gap: 2 }}>
                  <Tooltip title="上移">
                    <Button
                      type="text"
                      size="small"
                      icon={<UpOutlined style={{ fontSize: 10 }} />}
                      disabled={orderIdx === 0}
                      onClick={(e) => { e.stopPropagation(); handleFieldMove(category, subCategory, field.field_key, 'up'); }}
                      style={{ width: 20, height: 20, padding: 0 }}
                    />
                  </Tooltip>
                  <Tooltip title="下移">
                    <Button
                      type="text"
                      size="small"
                      icon={<DownOutlined style={{ fontSize: 10 }} />}
                      disabled={orderIdx === fieldOrder.length - 1}
                      onClick={(e) => { e.stopPropagation(); handleFieldMove(category, subCategory, field.field_key, 'down'); }}
                      style={{ width: 20, height: 20, padding: 0 }}
                    />
                  </Tooltip>
                </span>
              )}
            </div>
          );
        })}
      </div>
    );
  };

  // Render sub-categories for a category (Fix 4: stable display order)
  const renderSubCategories = (category: string) => {
    // Fix 4: Always use CATEGORY_SUB_CATEGORIES as stable display order
    const subCategories = CATEGORY_SUB_CATEGORIES[category] || [];
    const catConfig = getCatConfig(category);
    const enabledSubs = catConfig.enabled_sub_categories;

    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {subCategories.map((sc) => {
          const isEnabled = enabledSubs.includes(sc);
          const orderIdx = catConfig.sub_category_order.indexOf(sc);

          return (
            <div key={sc}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  padding: '4px 0',
                  background: isEnabled ? '#fafafa' : 'transparent',
                  borderRadius: 4,
                  paddingLeft: 4,
                }}
              >
                <Checkbox
                  checked={isEnabled}
                  onChange={(e) => {
                    e.stopPropagation();
                    handleSubCategoryToggle(category, sc, e.target.checked);
                  }}
                />
                <Text style={{ flex: 1, fontSize: 13, fontWeight: isEnabled ? 500 : 400 }}>
                  {SUB_CATEGORY_LABELS[sc] || sc}
                </Text>
                {isEnabled && orderIdx >= 0 && (
                  <span style={{ display: 'inline-flex', gap: 2, marginRight: 4 }}>
                    <Tooltip title="上移">
                      <Button
                        type="text"
                        size="small"
                        icon={<UpOutlined style={{ fontSize: 10 }} />}
                        disabled={orderIdx <= 0}
                        onClick={(e) => { e.stopPropagation(); handleSubCategoryMove(category, sc, 'up'); }}
                        style={{ width: 20, height: 20, padding: 0 }}
                      />
                    </Tooltip>
                    <Tooltip title="下移">
                      <Button
                        type="text"
                        size="small"
                        icon={<DownOutlined style={{ fontSize: 10 }} />}
                        disabled={orderIdx < 0 || orderIdx === catConfig.sub_category_order.length - 1}
                        onClick={(e) => { e.stopPropagation(); handleSubCategoryMove(category, sc, 'down'); }}
                        style={{ width: 20, height: 20, padding: 0 }}
                      />
                    </Tooltip>
                  </span>
                )}
              </div>
              {isEnabled && renderFieldList(category, sc)}
            </div>
          );
        })}
      </div>
    );
  };

  // Build Collapse items for visible categories
  const collapseItems = config.visible_categories
    .filter((cat) => availableCategories.includes(cat))
    .map((cat) => ({
      key: cat,
      label: (
        <Text strong style={{ fontSize: 13 }}>
          {CATEGORY_LABELS[cat] || cat}
        </Text>
      ),
      children: renderSubCategories(cat),
    }));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* Category selection */}
      <div>
        <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>
          可见分类
        </Text>
        <Checkbox.Group
          value={config.visible_categories}
          onChange={(vals) => handleCategoryToggle(vals as string[])}
          style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}
        >
          {availableCategories.map((cat) => (
            <Checkbox key={cat} value={cat}>
              {CATEGORY_LABELS[cat] || cat}
            </Checkbox>
          ))}
        </Checkbox.Group>
      </div>

      {/* Category details */}
      {config.visible_categories.length > 0 && (
        <div>
          {loading ? (
            <div style={{ textAlign: 'center', padding: 16 }}>
              <Spin size="small" />
              <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
                加载属性模板...
              </Text>
            </div>
          ) : (
            <>
              {collapseItems.length > 0 ? (
                <Collapse
                  size="small"
                  items={collapseItems}
                  activeKey={activeKeys}
                  onChange={(keys) => setActiveKeys(keys as string[])}
                  style={{ background: '#fff' }}
                />
              ) : (
                <Empty description="无可用分类" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </>
          )}
        </div>
      )}

      {/* BOM-type specific options */}
      {bomType === 'PBOM' && (
        <div style={{ borderTop: '1px solid #f0f0f0', paddingTop: 8 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Switch
              size="small"
              checked={!!config.show_route_editor}
              onChange={(checked) => onChange({ ...config, show_route_editor: checked })}
            />
            <Text style={{ fontSize: 13 }}>显示工艺路线编辑器</Text>
          </div>
        </div>
      )}

      {bomType === 'MBOM' && (
        <div style={{ borderTop: '1px solid #f0f0f0', paddingTop: 8, display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Switch
              size="small"
              checked={!!config.editable_scrap_rate}
              onChange={(checked) => onChange({ ...config, editable_scrap_rate: checked })}
            />
            <Text style={{ fontSize: 13 }}>允许编辑损耗率</Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Switch
              size="small"
              checked={!!config.show_freeze_button}
              onChange={(checked) => onChange({ ...config, show_freeze_button: checked })}
            />
            <Text style={{ fontSize: 13 }}>显示冻结/发布按钮</Text>
          </div>
        </div>
      )}
    </div>
  );
};

export default BOMConfigPanel;
