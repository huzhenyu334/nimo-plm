import React, { useState } from 'react';
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
  AutoComplete,
  Tooltip,
} from 'antd';
import { DeleteOutlined, PlusOutlined, UploadOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { taskFormApi } from '@/api/taskForms';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

// ========== Option Constants ==========

export const CATEGORY_OPTIONS = [
  '电子元器件', '结构件', '光学器件', '电池', '线缆/FPC', '包装材料', '标签/外观件', '其他',
];

export const PROCUREMENT_OPTIONS = [
  { label: 'Buy（外购）', value: 'buy' },
  { label: 'Make（自制）', value: 'make' },
  { label: 'Phantom（虚拟件）', value: 'phantom' },
];

export const PROCESS_TYPE_OPTIONS = [
  { label: '注塑', value: '注塑' }, { label: 'CNC', value: 'CNC' },
  { label: '冲压', value: '冲压' }, { label: '模切', value: '模切' },
  { label: '3D打印', value: '3D打印' }, { label: '激光切割', value: '激光切割' },
  { label: 'SMT', value: 'SMT' }, { label: '手工', value: '手工' },
];

export const ASSEMBLY_METHOD_OPTIONS = [
  { label: '卡扣', value: '卡扣' }, { label: '螺丝', value: '螺丝' },
  { label: '胶合', value: '胶合' }, { label: '超声波焊接', value: '超声波焊接' },
  { label: '热熔', value: '热熔' }, { label: '激光焊接', value: '激光焊接' },
];

export const TOLERANCE_GRADE_OPTIONS = [
  { label: '普通', value: '普通' }, { label: '精密', value: '精密' }, { label: '超精密', value: '超精密' },
];

export const MATERIAL_TYPE_PRESETS = [
  '钛合金', '铝合金6061', '铝合金7075', '不锈钢304', '不锈钢316L',
  'PC', 'ABS', 'ABS+PC', 'PA66', 'PA66+GF30', 'PMMA', 'POM', 'TPU',
  '硅胶', 'PEEK', '碳纤维', '玻璃', '蓝宝石', '镁合金', '锌合金', '铜合金', 'TR90', 'Ultem',
];

export const TOLERANCE_PRESETS = [
  { label: '普通 ±0.05mm', value: '0.05' },
  { label: '精密 ±0.02mm', value: '0.02' },
  { label: '超精密 ±0.005mm', value: '0.005' },
];

// 格式化公差显示
const formatTolerance = (v: any): string => {
  if (v == null || v === '') return '';
  const num = parseFloat(String(v));
  if (!isNaN(num)) return `±${num}mm`;
  // 兼容旧数据
  const map: Record<string, string> = { '普通': '±0.05mm', '精密': '±0.03mm', '超精密': '±0.005mm' };
  return map[v] || String(v);
};

// 格式化文件大小
const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)}KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
};

// ========== Types ==========

export type BOMItemRecord = Record<string, any>;

export interface BOMEditableTableProps {
  bomType: 'EBOM' | 'SBOM';
  items: BOMItemRecord[];
  onChange: (items: BOMItemRecord[]) => void;
  showAddDelete?: boolean;
}

// ========== Component ==========

const PAGE_SIZE = 10;

const BOMEditableTable: React.FC<BOMEditableTableProps> = ({
  bomType,
  items,
  onChange,
  showAddDelete = true,
}) => {
  const [editingCell, setEditingCell] = useState<{ rowIdx: number; field: string } | null>(null);
  const [currentPage, setCurrentPage] = useState(1);

  // Convert page-relative index to global array index
  const toGlobalIdx = (pageIdx: number) =>
    items.length > PAGE_SIZE ? (currentPage - 1) * PAGE_SIZE + pageIdx : pageIdx;

  const handleCellSave = (idx: number, field: string, value: any) => {
    const gi = toGlobalIdx(idx);
    // Skip save if value unchanged
    if (items[gi] && items[gi][field] === value) {
      setEditingCell(null);
      return;
    }
    const newItems = items.map((item, i) => i === gi ? { ...item, [field]: value } : item);
    onChange(newItems);
    setEditingCell(null);
  };

  const handleDeleteRow = (idx: number) => {
    const gi = toGlobalIdx(idx);
    onChange(items.filter((_, i) => i !== gi));
  };

  const handleAddRow = () => {
    const nextNum = items.length > 0 ? Math.max(...items.map(i => i.item_number || 0)) + 1 : 1;
    const newItem: BOMItemRecord = {
      item_number: nextNum,
      name: '新零件',
      quantity: 1,
      unit: 'pcs',
      weight_grams: 0,
      target_price: 0,
      tooling_estimate: 0,
      tolerance_grade: '0.05',
    };
    const newItems = [...items, newItem];
    onChange(newItems);
    // Auto-navigate to last page
    if (newItems.length > PAGE_SIZE) {
      setCurrentPage(Math.ceil(newItems.length / PAGE_SIZE));
    }
  };

  // Click-to-edit cell renderer
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
          onChange={(e) => handleCellSave(idx, field, e.target.checked)}
        />
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
    let displayValue = value;
    if (type === 'select' && options && value) {
      displayValue = options.find(o => o.value === value)?.label?.split('（')[0] || value;
    }
    if (type === 'number' && (value != null && value !== '') && field === 'target_price') displayValue = `¥${Number(value).toFixed(2)}`;
    else if (type === 'number' && field === 'tooling_estimate') displayValue = `¥${Number(value || 0).toFixed(2)}`;
    else if (type === 'number' && (value != null && value !== '') && field === 'unit_price') displayValue = Number(value).toFixed(2);

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

  // AutoComplete renderer for material_type
  const renderMaterialTypeCell = (value: any, idx: number) => {
    const gi = toGlobalIdx(idx);
    const isEditing = editingCell?.rowIdx === gi && editingCell?.field === 'material_type';
    if (isEditing) {
      return (
        <AutoComplete
          size="small"
          autoFocus
          defaultValue={value}
          defaultOpen
          style={{ width: '100%' }}
          options={MATERIAL_TYPE_PRESETS.map(m => ({ value: m }))}
          filterOption={(input, option) =>
            (option?.value as string)?.toLowerCase().includes(input.toLowerCase())
          }
          onSelect={(v) => handleCellSave(idx, 'material_type', v)}
          onBlur={(e) => handleCellSave(idx, 'material_type', (e.target as HTMLInputElement).value)}
        />
      );
    }
    return (
      <div
        style={{ cursor: 'pointer', minHeight: 22, padding: '0 2px', borderRadius: 2 }}
        className="editable-cell"
        onClick={() => setEditingCell({ rowIdx: gi, field: 'material_type' })}
      >
        {value ?? <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
      </div>
    );
  };

  // AutoComplete renderer for tolerance_grade (±mm)
  const renderToleranceCell = (value: any, idx: number) => {
    const gi = toGlobalIdx(idx);
    const isEditing = editingCell?.rowIdx === gi && editingCell?.field === 'tolerance_grade';
    if (isEditing) {
      return (
        <AutoComplete
          size="small"
          autoFocus
          defaultValue={value != null ? String(value) : ''}
          defaultOpen
          style={{ width: '100%' }}
          options={TOLERANCE_PRESETS}
          onSelect={(v) => handleCellSave(idx, 'tolerance_grade', v)}
          onBlur={(e) => {
            const raw = (e.target as HTMLInputElement).value;
            handleCellSave(idx, 'tolerance_grade', raw);
          }}
        />
      );
    }
    const display = formatTolerance(value);
    return (
      <div
        style={{ cursor: 'pointer', minHeight: 22, padding: '0 2px', borderRadius: 2 }}
        className="editable-cell"
        onClick={() => setEditingCell({ rowIdx: gi, field: 'tolerance_grade' })}
      >
        {display || <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
      </div>
    );
  };

  // Drawing file upload cell with filename + size display
  const renderDrawingUploadCell = (record: BOMItemRecord, idx: number, fileIdField: string, fileNameField: string, fileSizeField: string) => {
    const gi = toGlobalIdx(idx);
    const fileName = record?.[fileNameField];
    const fileSize = record?.[fileSizeField];
    const fileId = record?.[fileIdField];

    return (
      <Space size={4} style={{ width: '100%' }}>
        {fileName ? (
          <Tooltip title={`${fileName}${fileSize ? ' ' + formatFileSize(fileSize) : ''}`}>
            <a href={fileId ? `/uploads/${fileId}/${fileName}` : '#'} target="_blank" rel="noreferrer"
              style={{ fontSize: 11, maxWidth: 70, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', verticalAlign: 'middle' }}>
              {fileName}
            </a>
            {fileSize > 0 && <Text type="secondary" style={{ fontSize: 10 }}>{formatFileSize(fileSize)}</Text>}
          </Tooltip>
        ) : <Text type="secondary" style={{ fontSize: 11 }}>-</Text>}
        <Upload
          showUploadList={false}
          customRequest={() => {}}
          beforeUpload={(file) => {
            console.log('[BOM Upload] started:', file.name, 'gi:', gi, 'field:', fileNameField);
            taskFormApi.uploadFile(file).then((result) => {
              console.log('[BOM Upload] result:', JSON.stringify(result));
              const newItems = items.map((it, i) => i === gi ? {
                ...it,
                [fileIdField]: result.id,
                [fileNameField]: result.filename,
                [fileSizeField]: file.size,
              } : it);
              console.log('[BOM Upload] updated item:', JSON.stringify(newItems[gi]));
              onChange(newItems);
              message.success('上传成功');
            }).catch((err) => {
              console.error('[BOM Upload] failed:', err);
              message.error('上传失败');
            });
            return Upload.LIST_IGNORE;
          }}
        >
          <UploadOutlined style={{ color: '#1677ff', cursor: 'pointer', fontSize: 12 }} />
        </Upload>
        {fileName && (
          <CloseCircleOutlined
            style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 11 }}
            onClick={() => {
              const newItems = items.map((it, i) => i === gi ? {
                ...it, [fileIdField]: undefined, [fileNameField]: undefined, [fileSizeField]: undefined,
              } : it);
              onChange(newItems);
            }}
          />
        )}
      </Space>
    );
  };

  // Build columns based on bomType
  const commonCols: ColumnsType<BOMItemRecord> = [
    { title: '序号', dataIndex: 'item_number', width: 55, align: 'center',
      render: (v, _, idx) => renderCell(v, idx, 'item_number', 'number') },
  ];

  const ebomCols: ColumnsType<BOMItemRecord> = [
    { title: '位号', dataIndex: 'reference', width: 80,
      render: (v, _, idx) => renderCell(v, idx, 'reference') },
    { title: '名称', dataIndex: 'name', width: 120,
      render: (v, _, idx) => renderCell(v, idx, 'name') },
    { title: '规格', dataIndex: 'specification', width: 140, ellipsis: true,
      render: (v, _, idx) => renderCell(v, idx, 'specification') },
    { title: '数量', dataIndex: 'quantity', width: 60, align: 'right',
      render: (v, _, idx) => renderCell(v, idx, 'quantity', 'number') },
    { title: '单位', dataIndex: 'unit', width: 55,
      render: (v, _, idx) => renderCell(v, idx, 'unit') },
    { title: '类别', dataIndex: 'category', width: 100,
      render: (v, _, idx) => renderCell(v, idx, 'category', 'select',
        CATEGORY_OPTIONS.map(c => ({ label: c, value: c }))) },
    { title: '单价', dataIndex: 'unit_price', width: 80, align: 'right',
      render: (v, _, idx) => renderCell(v, idx, 'unit_price', 'number') },
    { title: '制造商', dataIndex: 'manufacturer', width: 100, ellipsis: true,
      render: (v, _, idx) => renderCell(v, idx, 'manufacturer') },
    { title: '制造商料号', dataIndex: 'manufacturer_pn', width: 100, ellipsis: true,
      render: (v, _, idx) => renderCell(v, idx, 'manufacturer_pn') },
    { title: '供应商', dataIndex: 'supplier', width: 100, ellipsis: true,
      render: (v, _, idx) => renderCell(v, idx, 'supplier') },
    { title: '交期(天)', dataIndex: 'lead_time_days', width: 75, align: 'right',
      render: (v, _, idx) => renderCell(v, idx, 'lead_time_days', 'number') },
    { title: '采购类型', dataIndex: 'procurement_type', width: 100,
      render: (v, _, idx) => renderCell(v, idx, 'procurement_type', 'select', PROCUREMENT_OPTIONS) },
    { title: '关键件', dataIndex: 'is_critical', width: 60, align: 'center',
      render: (v, _, idx) => renderCell(v, idx, 'is_critical', 'checkbox') },
  ];

  const sbomCols: ColumnsType<BOMItemRecord> = [
    { title: '名称', dataIndex: 'name', width: 120,
      render: (v, _, idx) => renderCell(v, idx, 'name') },
    { title: '数量', dataIndex: 'quantity', width: 60, align: 'right',
      render: (v, _, idx) => renderCell(v, idx, 'quantity', 'number') },
    { title: '单位', dataIndex: 'unit', width: 55,
      render: (v, _, idx) => renderCell(v, idx, 'unit') },
    { title: '材质', dataIndex: 'material_type', width: 110,
      render: (v, _, idx) => renderMaterialTypeCell(v, idx) },
    { title: '工艺类型', dataIndex: 'process_type', width: 90,
      render: (v, _, idx) => renderCell(v, idx, 'process_type', 'select', PROCESS_TYPE_OPTIONS) },
    { title: '2D图纸', dataIndex: 'drawing_2d_file_name', width: 140,
      render: (_, record, idx) => renderDrawingUploadCell(record, idx, 'drawing_2d_file_id', 'drawing_2d_file_name', 'drawing_2d_file_size'),
    },
    { title: '3D模型', dataIndex: 'drawing_3d_file_name', width: 140,
      render: (_, record, idx) => renderDrawingUploadCell(record, idx, 'drawing_3d_file_id', 'drawing_3d_file_name', 'drawing_3d_file_size'),
    },
    { title: '重量(g)', dataIndex: 'weight_grams', width: 75, align: 'right',
      render: (v, _, idx) => renderCell(v, idx, 'weight_grams', 'number') },
    { title: '目标价', dataIndex: 'target_price', width: 85, align: 'right',
      render: (v, _, idx) => renderCell(v, idx, 'target_price', 'number') },
    { title: '模具费', dataIndex: 'tooling_estimate', width: 85, align: 'right',
      render: (v, _, idx) => renderCell(v, idx, 'tooling_estimate', 'number') },
    { title: '外观件', dataIndex: 'is_appearance_part', width: 60, align: 'center',
      render: (v, _, idx) => renderCell(v, idx, 'is_appearance_part', 'checkbox') },
    { title: '装配方式', dataIndex: 'assembly_method', width: 90,
      render: (v, _, idx) => renderCell(v, idx, 'assembly_method', 'select', ASSEMBLY_METHOD_OPTIONS) },
    { title: '公差', dataIndex: 'tolerance_grade', width: 95,
      render: (v, _, idx) => renderToleranceCell(v, idx) },
    { title: '备注', dataIndex: 'notes', width: 120, ellipsis: true,
      render: (v, _, idx) => renderCell(v, idx, 'notes') },
  ];

  const typeCols = bomType === 'SBOM' ? sbomCols : ebomCols;
  const columns: ColumnsType<BOMItemRecord> = [...commonCols, ...typeCols];

  if (showAddDelete) {
    columns.push({
      title: '', width: 40, align: 'center', fixed: 'right',
      render: (_, _record, idx) => (
        <Popconfirm title="确认删除此行？" onConfirm={() => handleDeleteRow(idx)}>
          <Button type="text" size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    });
  }

  const scrollX = bomType === 'SBOM' ? 1500 : 1100;

  return (
    <div>
      {showAddDelete && (
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
          <Button type="dashed" size="small" icon={<PlusOutlined />} onClick={handleAddRow}>添加行</Button>
        </div>
      )}
      <Table
        columns={columns}
        dataSource={items}
        rowKey={(_, idx) => String(idx)}
        size="small"
        pagination={items.length > PAGE_SIZE ? {
          pageSize: PAGE_SIZE, size: 'small', showTotal: (t: number) => `共 ${t} 条`,
          current: currentPage, onChange: (p) => setCurrentPage(p),
        } : false}
        scroll={{ x: scrollX }}
        style={{ fontSize: 12 }}
        locale={{ emptyText: <Empty description={'暂无物料，点击"添加行"或"导入模板"开始'} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
      />
      <style>{`
        .editable-cell:hover {
          background: #f0f5ff;
        }
      `}</style>
    </div>
  );
};

export default BOMEditableTable;
