import React, { useState, useMemo } from 'react';
import {
  Table,
  Input,
  InputNumber,
  Select,
  Checkbox,
  Button,
  Typography,
  Popconfirm,
  Empty,
  Upload,
  Space,
  message,
  Tooltip,
} from 'antd';
import {
  DeleteOutlined,
  UploadOutlined,
  CloseCircleOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { taskFormApi } from '@/api/taskForms';
import type { ColumnsType } from 'antd/es/table';
import type { CategoryAttrTemplate } from '@/api/projectBom';
import { COMMON_FIELDS } from './bomConstants';
import { useIsMobile } from '@/hooks/useIsMobile';

const { Text } = Typography;

const PAGE_SIZE = 10;

// ========== Types ==========

export interface DynamicBOMTableProps {
  subCategory: string;
  items: Record<string, any>[];
  onChange: (items: Record<string, any>[]) => void;
  fieldOrder?: string[];
  templates: CategoryAttrTemplate[];
  readonly?: boolean;
  onItemSave?: (itemId: string, field: string, value: any) => void;
  showMaterialCode?: boolean;
}

// ========== Helpers ==========

const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)}KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
};

const COMMON_FIELD_CONFIG: Record<string, { title: string; width: number; type: 'text' | 'number'; align?: 'left' | 'center' | 'right' }> = {
  name: { title: '名称', width: 120, type: 'text' },
  quantity: { title: '数量', width: 60, type: 'number', align: 'right' },
  unit: { title: '单位', width: 55, type: 'text' },
  supplier: { title: '供应商', width: 100, type: 'text' },
  unit_price: { title: '单价', width: 80, type: 'number', align: 'right' },
  extended_cost: { title: '小计', width: 80, type: 'number', align: 'right' },
  notes: { title: '备注', width: 120, type: 'text' },
};

// ========== Component ==========

const DynamicBOMTable: React.FC<DynamicBOMTableProps> = ({
  subCategory: _subCategory,
  items,
  onChange,
  fieldOrder,
  templates,
  readonly = false,
  onItemSave,
  showMaterialCode = false,
}) => {
  const [editingCell, setEditingCell] = useState<{ rowIdx: number; field: string } | null>(null);
  const [currentPage, setCurrentPage] = useState(1);

  // Flatten extended_attrs onto items for display
  const flatItems = useMemo(() => items.map(item => ({
    ...item,
    ...(item.extended_attrs || {}),
  })), [items]);

  // Convert page-relative index to global array index
  const toGlobalIdx = (pageIdx: number) =>
    items.length > PAGE_SIZE ? (currentPage - 1) * PAGE_SIZE + pageIdx : pageIdx;

  // Determine which extended fields to show and their order
  const orderedTemplates = useMemo(() => {
    const showable = templates.filter(t => t.show_in_table);
    if (fieldOrder && fieldOrder.length > 0) {
      // Order by fieldOrder, then remaining by sort_order
      const ordered: CategoryAttrTemplate[] = [];
      const remaining = [...showable];
      for (const key of fieldOrder) {
        // Skip common fields in fieldOrder -- they are handled separately
        if (COMMON_FIELDS.includes(key)) continue;
        const idx = remaining.findIndex(t => t.field_key === key);
        if (idx >= 0) {
          ordered.push(remaining.splice(idx, 1)[0]);
        }
      }
      // Add remaining templates sorted by sort_order
      remaining.sort((a, b) => a.sort_order - b.sort_order);
      return [...ordered, ...remaining];
    }
    return showable.sort((a, b) => a.sort_order - b.sort_order);
  }, [templates, fieldOrder]);

  // ========== Cell save ==========

  const handleCellSave = (idx: number, field: string, value: any) => {
    const gi = toGlobalIdx(idx);
    // Skip save if value unchanged
    if (flatItems[gi] && flatItems[gi][field] === value) {
      setEditingCell(null);
      return;
    }

    // Determine if this is an extended attr
    const isExtendedField = !COMMON_FIELDS.includes(field)
      && field !== 'item_number'
      && templates.some(t => t.field_key === field);

    if (onItemSave && items[gi]?.id) {
      onItemSave(items[gi].id, field, value);
    } else {
      if (isExtendedField) {
        const newItems = items.map((item, i) => {
          if (i !== gi) return item;
          return {
            ...item,
            extended_attrs: { ...(item.extended_attrs || {}), [field]: value },
          };
        });
        onChange(newItems);
      } else {
        const newItems = items.map((item, i) => i === gi ? { ...item, [field]: value } : item);
        onChange(newItems);
      }
    }
    setEditingCell(null);
  };

  const handleDeleteRow = (idx: number) => {
    const gi = toGlobalIdx(idx);
    onChange(items.filter((_, i) => i !== gi));
  };

  // ========== Click-to-edit cell renderer ==========

  const renderCell = (
    value: any, idx: number, field: string,
    type: 'text' | 'number' | 'select' | 'checkbox' = 'text',
    options?: { label: string; value: string }[],
  ) => {
    const gi = toGlobalIdx(idx);

    // Checkbox: always show inline
    if (type === 'checkbox') {
      return (
        <Checkbox
          checked={!!value}
          disabled={readonly}
          onChange={(e) => handleCellSave(idx, field, e.target.checked)}
        />
      );
    }

    // Format display value — guard against GORM relation objects (React error #31)
    let displayValue: any = value;
    if (displayValue != null && typeof displayValue === 'object') {
      displayValue = null;
    }
    if (type === 'select' && options && displayValue) {
      displayValue = options.find(o => o.value === displayValue)?.label || displayValue;
    }
    if (type === 'number' && (displayValue != null && displayValue !== '') && (field === 'unit_price' || field === 'extended_cost')) {
      displayValue = Number(displayValue).toFixed(2);
    }

    // Readonly mode
    if (readonly) {
      return (
        <div style={{ minHeight: 22, padding: '0 2px' }}>
          {displayValue ?? <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
        </div>
      );
    }

    const isEditing = editingCell?.rowIdx === gi && editingCell?.field === field;

    if (isEditing) {
      if (type === 'number') {
        return (
          <InputNumber
            size="small"
            autoFocus
            defaultValue={typeof value === 'string' ? parseFloat(value) || 0 : value}
            style={{ width: '100%' }}
            onBlur={(e) => handleCellSave(idx, field, parseFloat((e.target as HTMLInputElement).value) || 0)}
            onPressEnter={(e) => handleCellSave(idx, field, parseFloat((e.target as HTMLInputElement).value) || 0)}
          />
        );
      }
      if (type === 'select' && options) {
        return (
          <Select
            size="small"
            autoFocus
            defaultValue={value}
            defaultOpen
            style={{ width: '100%' }}
            options={options}
            onChange={(v) => handleCellSave(idx, field, v)}
            onBlur={() => setEditingCell(null)}
          />
        );
      }
      return (
        <Input
          size="small"
          autoFocus
          defaultValue={value}
          onBlur={(e) => handleCellSave(idx, field, e.target.value)}
          onPressEnter={(e) => handleCellSave(idx, field, (e.target as HTMLInputElement).value)}
        />
      );
    }

    // Display mode: click to edit
    return (
      <div
        style={{ cursor: 'pointer', minHeight: 22, padding: '0 2px', borderRadius: 2 }}
        className="editable-cell"
        onClick={() => setEditingCell({ rowIdx: gi, field })}
      >
        {displayValue ?? <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
      </div>
    );
  };

  // ========== File cell renderer ==========

  const renderFileCell = (record: Record<string, any>, idx: number, fieldKey: string) => {
    const gi = toGlobalIdx(idx);
    const fileValue = record[fieldKey];
    const hasFile = fileValue && typeof fileValue === 'object' && fileValue.file_id;

    if (readonly) {
      return hasFile ? (
        <Tooltip title={`${fileValue.file_name}${fileValue.file_size ? ' ' + formatFileSize(fileValue.file_size) : ''}`}>
          <a href={`/uploads/${fileValue.file_id}/${fileValue.file_name}`} target="_blank" rel="noreferrer"
            style={{ fontSize: 11, maxWidth: 100, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {fileValue.file_name}
          </a>
          {fileValue.file_size > 0 && <Text type="secondary" style={{ fontSize: 10, marginLeft: 4 }}>{formatFileSize(fileValue.file_size)}</Text>}
        </Tooltip>
      ) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>;
    }

    return (
      <Space size={4} style={{ width: '100%' }}>
        {hasFile ? (
          <Tooltip title={`${fileValue.file_name}${fileValue.file_size ? ' ' + formatFileSize(fileValue.file_size) : ''}`}>
            <a href={`/uploads/${fileValue.file_id}/${fileValue.file_name}`} target="_blank" rel="noreferrer"
              style={{ fontSize: 11, maxWidth: 70, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', verticalAlign: 'middle' }}>
              {fileValue.file_name}
            </a>
            {fileValue.file_size > 0 && <Text type="secondary" style={{ fontSize: 10 }}>{formatFileSize(fileValue.file_size)}</Text>}
          </Tooltip>
        ) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
        <Upload
          showUploadList={false}
          customRequest={() => {}}
          beforeUpload={(file) => {
            taskFormApi.uploadFile(file).then((result) => {
              const fileData = { file_id: result.id, file_name: result.filename, file_size: file.size };
              const newItems = items.map((it, i) => {
                if (i !== gi) return it;
                return {
                  ...it,
                  extended_attrs: { ...(it.extended_attrs || {}), [fieldKey]: fileData },
                };
              });
              onChange(newItems);
              message.success('上传成功');
            }).catch(() => {
              message.error('上传失败');
            });
            return Upload.LIST_IGNORE;
          }}
        >
          <UploadOutlined style={{ color: '#1677ff', cursor: 'pointer', fontSize: 12 }} />
        </Upload>
        {hasFile && (
          <CloseCircleOutlined
            style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 11 }}
            onClick={() => {
              const newItems = items.map((it, i) => {
                if (i !== gi) return it;
                const newAttrs = { ...(it.extended_attrs || {}) };
                delete newAttrs[fieldKey];
                return { ...it, extended_attrs: newAttrs };
              });
              onChange(newItems);
            }}
          />
        )}
      </Space>
    );
  };

  // ========== Thumbnail cell renderer ==========

  const renderThumbnailCell = (record: Record<string, any>, fieldKey: string) => {
    const url = record[fieldKey];
    if (url && typeof url === 'string') {
      return (
        <img
          src={url}
          width={64}
          height={64}
          style={{ objectFit: 'contain', background: '#fff', borderRadius: 2 }}
        />
      );
    }
    return (
      <div style={{ width: 64, height: 64, background: '#f5f5f5', display: 'flex', alignItems: 'center', justifyContent: 'center', borderRadius: 4 }}>
        <EyeOutlined style={{ color: '#d9d9d9', fontSize: 16 }} />
      </div>
    );
  };

  // ========== Build columns ==========

  const columns: ColumnsType<Record<string, any>> = useMemo(() => {
    const cols: ColumnsType<Record<string, any>> = [];

    // Always first: item_number
    cols.push({
      title: '序号',
      dataIndex: 'item_number',
      width: 55,
      align: 'center',
      render: (v, _, idx) => renderCell(v, idx, 'item_number', 'number'),
    });

    // Material code column (shown in readonly BOM management view)
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

    // Common fields
    for (const fieldKey of COMMON_FIELDS) {
      const config = COMMON_FIELD_CONFIG[fieldKey];
      if (!config) continue;

      if (fieldKey === 'extended_cost') {
        // Extended cost is computed, not editable
        cols.push({
          title: config.title,
          dataIndex: fieldKey,
          width: config.width,
          align: config.align as any,
          render: (v, record) => {
            const cost = v ?? ((record.quantity || 0) * (record.unit_price || 0));
            return cost > 0 ? Number(cost).toFixed(2) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>;
          },
        });
      } else {
        cols.push({
          title: config.title,
          dataIndex: fieldKey,
          width: config.width,
          align: config.align as any,
          ellipsis: fieldKey === 'supplier' || fieldKey === 'notes',
          render: (v, _, idx) => renderCell(v, idx, fieldKey, config.type),
        });
      }
    }

    // Extended template fields
    for (const tmpl of orderedTemplates) {
      const colType = tmpl.field_type === 'number' ? 'number'
        : tmpl.field_type === 'select' ? 'select'
        : tmpl.field_type === 'boolean' ? 'checkbox'
        : 'text';
      const selectOpts = tmpl.field_type === 'select' && tmpl.options?.values
        ? (tmpl.options.values as string[]).map((v: string) => ({ label: v, value: v }))
        : undefined;
      const title = tmpl.unit ? `${tmpl.field_name}(${tmpl.unit})` : tmpl.field_name;

      if (tmpl.field_type === 'file') {
        cols.push({
          title,
          dataIndex: tmpl.field_key,
          width: 140,
          render: (_, record, idx) => renderFileCell(record, idx, tmpl.field_key),
        });
      } else if (tmpl.field_type === 'thumbnail') {
        cols.push({
          title,
          dataIndex: tmpl.field_key,
          width: 80,
          align: 'center',
          render: (_, record) => renderThumbnailCell(record, tmpl.field_key),
        });
      } else {
        cols.push({
          title,
          dataIndex: tmpl.field_key,
          width: colType === 'checkbox' ? 60 : (colType === 'number' ? 80 : 100),
          align: (colType === 'checkbox' ? 'center' : colType === 'number' ? 'right' : undefined) as any,
          ellipsis: colType === 'text',
          render: (v: any, _: any, idx: number) => renderCell(v, idx, tmpl.field_key, colType as any, selectOpts),
        });
      }
    }

    // Delete column
    if (!readonly) {
      cols.push({
        title: '', width: 40, align: 'center', fixed: 'right',
        render: (_, _record, idx) => (
          <Popconfirm title="确认删除此行？" onConfirm={() => handleDeleteRow(idx)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        ),
      });
    }

    return cols;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [orderedTemplates, readonly, editingCell, items, currentPage, showMaterialCode]);

  // Calculate scroll width
  const scrollX = columns.reduce((sum, c) => sum + ((c.width as number) || 100), 0);

  const isMobile = useIsMobile();

  // Mobile card view
  if (isMobile) {
    if (flatItems.length === 0) {
      return <Empty description="暂无物料" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ padding: '16px 0' }} />;
    }
    return (
      <div className="bom-mobile-card-list">
        {flatItems.map((item, idx) => {
          const cost = item.extended_cost ?? ((item.quantity || 0) * (item.unit_price || 0));
          return (
            <div key={item.id || idx} className="bom-item-card">
              <div className="bom-item-card-header">
                <span className="bom-item-card-name">{item.name || `#${item.item_number || idx + 1}`}</span>
                {item.material_code && <span className="bom-item-card-code">{item.material_code}</span>}
              </div>
              <div className="bom-item-card-meta">
                <span className="bom-item-card-meta-item">
                  <span className="bom-item-card-meta-label">数量</span>
                  <span className="bom-item-card-meta-value">{item.quantity || 0} {item.unit || ''}</span>
                </span>
                {item.unit_price > 0 && (
                  <span className="bom-item-card-meta-item">
                    <span className="bom-item-card-meta-label">单价</span>
                    <span className="bom-item-card-meta-value">{'\u00a5'}{Number(item.unit_price).toFixed(2)}</span>
                  </span>
                )}
                {cost > 0 && (
                  <span className="bom-item-card-meta-item">
                    <span className="bom-item-card-meta-label">小计</span>
                    <span className="bom-item-card-cost">{'\u00a5'}{Number(cost).toFixed(2)}</span>
                  </span>
                )}
                {item.supplier && (
                  <span className="bom-item-card-meta-item">
                    <span className="bom-item-card-meta-label">供应商</span>
                    <span className="bom-item-card-meta-value">{item.supplier}</span>
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    );
  }

  return (
    <div>
      <Table
        columns={columns}
        dataSource={flatItems}
        rowKey={(r, idx) => r.id || String(idx)}
        size="small"
        pagination={items.length > PAGE_SIZE ? {
          pageSize: PAGE_SIZE,
          size: 'small',
          showTotal: (t: number) => `共 ${t} 条`,
          current: currentPage,
          onChange: (p) => setCurrentPage(p),
        } : false}
        scroll={{ x: scrollX }}
        style={{ fontSize: 12 }}
        locale={{ emptyText: <Empty description="暂无物料" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
      />
      <style>{`
        .editable-cell:hover {
          background: #f0f5ff;
        }
      `}</style>
    </div>
  );
};

export default DynamicBOMTable;
