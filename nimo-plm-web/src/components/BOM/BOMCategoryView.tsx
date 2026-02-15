import React, { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { Tag, Empty, Button } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import {
  CATEGORY_LABELS,
  SUB_CATEGORY_LABELS,
  CATEGORY_SUB_CATEGORIES,
} from './bomConstants';

// ========== Types ==========

export interface BOMCategoryViewProps {
  bomType: 'EBOM' | 'PBOM' | 'MBOM';
  items: Record<string, any>[];
  categories: string[];
  getEnabledSubCategories: (category: string) => string[];
  readonly: boolean;
  totalCost: number;
  onAddRow?: (category: string, subCategory: string) => void;
  onItemClick?: (item: Record<string, any>) => void;
}

// ========== Helpers ==========

const formatCurrency = (v: number | null | undefined): string => {
  if (v == null || isNaN(v)) return '-';
  return `\u00a5${v.toFixed(2)}`;
};

// ========== Component ==========

const BOMCategoryView: React.FC<BOMCategoryViewProps> = ({
  bomType,
  items,
  categories,
  getEnabledSubCategories,
  readonly,
  totalCost,
  onAddRow,
  onItemClick,
}) => {
  // Build category → sub-category tree with items
  const categoryTree = useMemo(() => {
    const itemsBySub: Record<string, Record<string, any>[]> = {};
    for (const item of items) {
      const sc = item.sub_category || 'unknown';
      if (!itemsBySub[sc]) itemsBySub[sc] = [];
      itemsBySub[sc].push(item);
    }

    const tree: {
      key: string;
      label: string;
      isParent: boolean;
      parentKey?: string;
      subCategory?: string;
      category?: string;
      count: number;
    }[] = [];

    for (const cat of categories) {
      const enabledSubs = getEnabledSubCategories(cat);
      const allSubs = (CATEGORY_SUB_CATEGORIES[cat] || []).filter(sc => enabledSubs.includes(sc));
      const catItems = items.filter(i => i.category === cat);

      // Skip empty categories in readonly mode
      if (readonly && catItems.length === 0) continue;

      // If only one sub-category, show it directly as a flat entry (no parent)
      if (allSubs.length === 1) {
        const sc = allSubs[0];
        const scItems = itemsBySub[sc] || [];
        if (readonly && scItems.length === 0) continue;
        tree.push({
          key: `${cat}-${sc}`,
          label: SUB_CATEGORY_LABELS[sc] || sc,
          isParent: false,
          category: cat,
          subCategory: sc,
          count: scItems.length,
        });
      } else {
        // Parent entry
        tree.push({
          key: cat,
          label: CATEGORY_LABELS[cat] || cat,
          isParent: true,
          count: catItems.length,
        });
        // Child entries
        for (const sc of allSubs) {
          const scItems = itemsBySub[sc] || [];
          if (readonly && scItems.length === 0) continue;
          tree.push({
            key: `${cat}-${sc}`,
            label: SUB_CATEGORY_LABELS[sc] || sc,
            isParent: false,
            parentKey: cat,
            category: cat,
            subCategory: sc,
            count: scItems.length,
          });
        }
      }
    }

    return { tree, itemsBySub };
  }, [items, categories, getEnabledSubCategories, readonly]);

  // Active sub-category key
  const selectableItems = categoryTree.tree.filter(t => !t.isParent);
  const [activeKey, setActiveKey] = useState<string>(selectableItems[0]?.key || '');
  const [expandedParents, setExpandedParents] = useState<Record<string, boolean>>(() => {
    // Default: expand the parent of the first selectable item
    const first = selectableItems[0];
    if (first?.parentKey) return { [first.parentKey]: true };
    return {};
  });

  // Refs for scrollspy
  const rightPanelRef = useRef<HTMLDivElement>(null);
  const sectionRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const isScrollingByClick = useRef(false);

  // Expand parent when clicking a parent item
  const handleParentClick = useCallback((parentKey: string) => {
    setExpandedParents(prev => {
      const next = { ...prev, [parentKey]: !prev[parentKey] };
      // If expanding, auto-select first child
      if (next[parentKey]) {
        const firstChild = categoryTree.tree.find(t => !t.isParent && t.parentKey === parentKey);
        if (firstChild) {
          setActiveKey(firstChild.key);
          // Scroll to section
          setTimeout(() => {
            const el = sectionRefs.current[firstChild.key];
            if (el && rightPanelRef.current) {
              isScrollingByClick.current = true;
              el.scrollIntoView({ behavior: 'smooth', block: 'start' });
              setTimeout(() => { isScrollingByClick.current = false; }, 600);
            }
          }, 50);
        }
      }
      return next;
    });
  }, [categoryTree.tree]);

  // Click a child item in left nav
  const handleChildClick = useCallback((key: string) => {
    setActiveKey(key);
    const el = sectionRefs.current[key];
    if (el && rightPanelRef.current) {
      isScrollingByClick.current = true;
      el.scrollIntoView({ behavior: 'smooth', block: 'start' });
      setTimeout(() => { isScrollingByClick.current = false; }, 600);
    }
  }, []);

  // Scrollspy: observe which section is visible
  useEffect(() => {
    const panel = rightPanelRef.current;
    if (!panel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (isScrollingByClick.current) return;
        for (const entry of entries) {
          if (entry.isIntersecting) {
            const key = entry.target.getAttribute('data-section-key');
            if (key) {
              setActiveKey(key);
              // Auto-expand parent
              const item = categoryTree.tree.find(t => t.key === key);
              if (item?.parentKey) {
                setExpandedParents(prev => ({ ...prev, [item.parentKey!]: true }));
              }
            }
            break;
          }
        }
      },
      { root: panel, rootMargin: '-10% 0px -80% 0px', threshold: 0 }
    );

    for (const key of Object.keys(sectionRefs.current)) {
      const el = sectionRefs.current[key];
      if (el) observer.observe(el);
    }

    return () => observer.disconnect();
  }, [categoryTree.tree]);

  // Empty state
  if (items.length === 0 && readonly) {
    return (
      <div style={{ padding: 16 }}>
        <Empty description={`暂无${bomType}数据`} />
      </div>
    );
  }

  return (
    <div className="bom-catview">
      {/* Summary bar */}
      <div className="bom-catview-summary">
        <span className="bom-catview-summary-type">{bomType}</span>
        <span className="bom-catview-summary-count">{items.length}项</span>
        <span className="bom-catview-summary-cost">{formatCurrency(totalCost)}</span>
      </div>

      <div className="bom-catview-body">
        {/* Left category nav */}
        <div className="bom-catview-nav">
          {categoryTree.tree.map(node => {
            if (node.isParent) {
              // Check if any child is active
              const hasActiveChild = categoryTree.tree.some(
                t => !t.isParent && t.parentKey === node.key && t.key === activeKey
              );
              return (
                <div
                  key={node.key}
                  className={`bom-catview-nav-item is-parent ${hasActiveChild ? 'has-active' : ''}`}
                  onClick={() => handleParentClick(node.key)}
                >
                  {node.label}
                  <span className="bom-catview-nav-count">{node.count}</span>
                </div>
              );
            }

            // Child: only show if parent is expanded (or no parent)
            if (node.parentKey && !expandedParents[node.parentKey]) return null;

            return (
              <div
                key={node.key}
                className={`bom-catview-nav-item ${node.parentKey ? 'is-child' : ''} ${activeKey === node.key ? 'active' : ''}`}
                onClick={() => handleChildClick(node.key)}
              >
                {node.label}
                {node.count > 0 && <span className="bom-catview-nav-badge">{node.count}</span>}
              </div>
            );
          })}
        </div>

        {/* Right content panel */}
        <div className="bom-catview-content" ref={rightPanelRef}>
          {selectableItems.map(node => {
            const scItems = categoryTree.itemsBySub[node.subCategory!] || [];
            return (
              <div
                key={node.key}
                ref={el => { sectionRefs.current[node.key] = el; }}
                data-section-key={node.key}
                className="bom-catview-section"
              >
                {/* Section header */}
                <div className="bom-catview-section-header">
                  <span className="bom-catview-section-title">{node.label}</span>
                  <span className="bom-catview-section-count">{scItems.length}项</span>
                  {!readonly && onAddRow && node.category && node.subCategory && (
                    <Button
                      type="link"
                      size="small"
                      icon={<PlusOutlined />}
                      className="bom-catview-add-btn"
                      onClick={() => onAddRow(node.category!, node.subCategory!)}
                    >
                      添加
                    </Button>
                  )}
                </div>

                {/* Item cards */}
                {scItems.length === 0 ? (
                  <div className="bom-catview-empty">暂无物料</div>
                ) : (
                  scItems.map((item, idx) => {
                    const cost = item.extended_cost ?? ((item.quantity || 0) * (item.unit_price || 0));
                    return (
                      <div
                        key={item.id || idx}
                        className="bom-catview-card"
                        onClick={() => onItemClick?.(item)}
                      >
                        <div className="bom-catview-card-top">
                          {item.material_code && (
                            <span className="bom-catview-card-code">{item.material_code}</span>
                          )}
                          {item._source && (
                            <Tag color={item._source === 'EBOM' ? 'blue' : 'orange'} style={{ margin: 0, fontSize: 10, lineHeight: '16px', padding: '0 4px' }}>
                              {item._source}
                            </Tag>
                          )}
                        </div>
                        <div className="bom-catview-card-name">
                          {item.name || `#${item.item_number || idx + 1}`}
                        </div>
                        <div className="bom-catview-card-bottom">
                          <span className="bom-catview-card-qty">
                            {item.quantity || 0}{item.unit || 'pcs'}
                          </span>
                          {item.unit_price > 0 && (
                            <span className="bom-catview-card-price">
                              {'\u00a5'}{Number(item.unit_price).toFixed(2)}
                            </span>
                          )}
                          {cost > 0 && (
                            <span className="bom-catview-card-cost">
                              {'\u00a5'}{Number(cost).toFixed(2)}
                            </span>
                          )}
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};

export default BOMCategoryView;
