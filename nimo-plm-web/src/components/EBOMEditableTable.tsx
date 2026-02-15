import React, { useState, useMemo } from 'react';
import {
  Table,
  Input,
  InputNumber,
  Button,
  Typography,
  Popconfirm,
  Empty,
  Upload,
  Tag,
  Select,
  message,
} from 'antd';
import {
  DeleteOutlined,
  PlusOutlined,
  UploadOutlined,
  FileExcelOutlined,
  PaperClipOutlined,
  CloudServerOutlined,
  ExperimentOutlined,
} from '@ant-design/icons';
import { taskFormApi } from '@/api/taskForms';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

// ========== Types ==========

export interface AttachmentFile {
  file_id: string;
  file_name: string;
  file_type?: string;
  url: string;
}

export type EBOMItemRecord = Record<string, any>;

export interface EBOMEditableTableProps {
  items: EBOMItemRecord[];
  onChange: (items: EBOMItemRecord[]) => void;
  mode?: 'edit' | 'view';
  onItemSave?: (itemId: string, field: string, value: any) => void;
  onImportExcel?: () => void;
}

// ========== Constants ==========

const PCB_SURFACE_OPTIONS = ['沉金(ENIG)', '喷锡(HASL)', '无铅喷锡', 'OSP', '沉银', '沉锡', '电镀硬金'];
const PCB_MATERIAL_OPTIONS = ['FR4', 'FR4 高Tg', '铝基板', 'CEM-3', 'Rogers', 'PI(柔性板)'];

// ========== Helpers ==========

const parseAttachments = (v: any): AttachmentFile[] => {
  if (!v) return [];
  if (Array.isArray(v)) return v;
  try { return JSON.parse(v); } catch { return []; }
};

const formatCurrency = (v: number | null | undefined): string => {
  if (v == null || isNaN(v)) return '-';
  return `¥${v.toFixed(2)}`;
};

// ========== Inline edit cell ==========
const EditableCell: React.FC<{
  value: any;
  field: string;
  record: EBOMItemRecord;
  onSave: (id: string, field: string, value: any) => void;
  type?: 'text' | 'number' | 'select' | 'textarea';
  options?: { label: string; value: string }[];
  placeholder?: string;
  width?: number;
}> = ({ value, field, record, onSave, type = 'text', options, placeholder, width }) => {
  if (type === 'number') {
    return (
      <InputNumber
        size="small"
        value={value}
        placeholder={placeholder}
        style={{ width: width || '100%' }}
        onChange={(v) => onSave(record.id, field, v)}
      />
    );
  }
  if (type === 'select' && options) {
    return (
      <Select
        size="small"
        value={value || undefined}
        placeholder={placeholder}
        style={{ width: width || '100%' }}
        allowClear
        showSearch
        options={options}
        onChange={(v) => onSave(record.id, field, v)}
      />
    );
  }
  if (type === 'textarea') {
    return (
      <Input.TextArea
        size="small"
        value={value || ''}
        placeholder={placeholder}
        rows={1}
        style={{ width: width || '100%' }}
        onBlur={(e) => { if (e.target.value !== (value || '')) onSave(record.id, field, e.target.value); }}
        onChange={() => {}}
        defaultValue={value || ''}
      />
    );
  }
  return (
    <Input
      size="small"
      defaultValue={value || ''}
      placeholder={placeholder}
      style={{ width: width || '100%' }}
      onBlur={(e) => { if (e.target.value !== (value || '')) onSave(record.id, field, e.target.value); }}
    />
  );
};

// ========== Attachment cell ==========
const AttachmentCell: React.FC<{
  record: EBOMItemRecord;
  onSave: (id: string, field: string, value: any) => void;
  readonly: boolean;
}> = ({ record, onSave, readonly }) => {
  const [uploading, setUploading] = useState(false);
  const files = parseAttachments(record.attachments);

  const handleUpload = async (file: File) => {
    setUploading(true);
    try {
      const result = await taskFormApi.uploadFile(file);
      const newFile: AttachmentFile = {
        file_id: result.id,
        file_name: result.filename || file.name,
        file_type: file.name.split('.').pop() || '',
        url: result.url,
      };
      const updated = [...files, newFile];
      onSave(record.id, 'attachments', JSON.stringify(updated));
    } catch {
      message.error('上传失败');
    } finally {
      setUploading(false);
    }
    return false; // prevent default upload
  };

  const handleDelete = (fileId: string) => {
    const updated = files.filter(f => f.file_id !== fileId);
    onSave(record.id, 'attachments', JSON.stringify(updated));
  };

  return (
    <div>
      {files.map(f => (
        <div key={f.file_id} style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, marginBottom: 2 }}>
          <PaperClipOutlined style={{ color: '#1677ff' }} />
          <a href={f.url} target="_blank" rel="noopener noreferrer" style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {f.file_name}
          </a>
          {!readonly && (
            <Popconfirm title="删除此文件？" onConfirm={() => handleDelete(f.file_id)}>
              <DeleteOutlined style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 11 }} />
            </Popconfirm>
          )}
        </div>
      ))}
      {!readonly && (
        <Upload
          showUploadList={false}
          beforeUpload={(file) => { handleUpload(file); return false; }}
        >
          <Button size="small" type="dashed" icon={<UploadOutlined />} loading={uploading} style={{ fontSize: 11 }}>
            上传
          </Button>
        </Upload>
      )}
    </div>
  );
};

// ========== Main Component ==========

const EBOMEditableTable: React.FC<EBOMEditableTableProps> = ({
  items,
  onChange,
  mode = 'view',
  onItemSave,
  onImportExcel,
}) => {
  const isEdit = mode === 'edit';

  // Split items by type
  const { components, pcbs, services } = useMemo(() => {
    const components: EBOMItemRecord[] = [];
    const pcbs: EBOMItemRecord[] = [];
    const services: EBOMItemRecord[] = [];
    for (const item of items) {
      const t = item.item_type || 'component';
      if (t === 'pcb') pcbs.push(item);
      else if (t === 'service') services.push(item);
      else components.push(item);
    }
    return { components, pcbs, services };
  }, [items]);

  // Cost summary
  const componentCost = components.reduce((s, i) => s + (i.quantity || 0) * (i.unit_price || 0), 0);
  const pcbCost = pcbs.reduce((s, i) => s + (i.quantity || 0) * (i.unit_price || 0), 0);
  const serviceCost = services.reduce((s, i) => s + (i.quantity || 0) * (i.unit_price || 0), 0);
  const totalCost = componentCost + pcbCost + serviceCost;

  // Save handler
  const handleSave = (id: string, field: string, value: any) => {
    if (onItemSave) {
      onItemSave(id, field, value);
    } else {
      const updated = items.map(i => i.id === id ? { ...i, [field]: value } : i);
      onChange(updated);
    }
  };

  // Add item
  const handleAdd = (itemType: string) => {
    const maxNum = items.reduce((m, i) => Math.max(m, i.item_number || 0), 0);
    const newItem: EBOMItemRecord = {
      id: `new-${Date.now()}`,
      item_number: maxNum + 1,
      item_type: itemType,
      name: itemType === 'pcb' ? 'PCB裸板' : itemType === 'service' ? 'SMT贴片' : '',
      quantity: 1,
      unit: itemType === 'service' ? '次' : 'pcs',
      attachments: '[]',
    };
    onChange([...items, newItem]);
  };

  // Delete item
  const handleDelete = (id: string) => {
    onChange(items.filter(i => i.id !== id));
  };

  // ========== Component columns ==========
  const componentCols: ColumnsType<EBOMItemRecord> = [
    { title: '#', dataIndex: 'item_number', width: 45, align: 'center' },
    ...(!isEdit ? [{
      title: '物料编码', width: 110,
      render: (_: any, r: EBOMItemRecord) => (
        <Text code style={{ fontSize: 11 }}>{r.material?.code || '-'}</Text>
      ),
    }] : []),
    {
      title: '名称', dataIndex: 'name', width: 140,
      render: (v, r) => isEdit ? <EditableCell value={v} field="name" record={r} onSave={handleSave} placeholder="元器件名称" /> : v,
    },
    {
      title: '数量', dataIndex: 'quantity', width: 60, align: 'right',
      render: (v, r) => isEdit ? <EditableCell value={v} field="quantity" record={r} onSave={handleSave} type="number" /> : v,
    },
    {
      title: '位号', dataIndex: 'designator', width: 120, ellipsis: true,
      render: (v, r) => isEdit ? <EditableCell value={v} field="designator" record={r} onSave={handleSave} placeholder="R1,R2..." /> : (v || '-'),
    },
    {
      title: '封装', dataIndex: 'package', width: 80,
      render: (v, r) => isEdit ? <EditableCell value={v} field="package" record={r} onSave={handleSave} placeholder="0402" /> : (v || '-'),
    },
    {
      title: '规格参数', dataIndex: 'specification', width: 120, ellipsis: true,
      render: (v, r) => isEdit ? <EditableCell value={v} field="specification" record={r} onSave={handleSave} placeholder="10kΩ 1%" /> : (v || '-'),
    },
    {
      title: '单价', dataIndex: 'unit_price', width: 80, align: 'right',
      render: (v, r) => isEdit
        ? <EditableCell value={v} field="unit_price" record={r} onSave={handleSave} type="number" placeholder="0.00" />
        : formatCurrency(v),
    },
  ];

  // ========== PCB columns ==========
  const pcbCols: ColumnsType<EBOMItemRecord> = [
    { title: '#', dataIndex: 'item_number', width: 45, align: 'center' },
    ...(!isEdit ? [{
      title: '物料编码', width: 110,
      render: (_: any, r: EBOMItemRecord) => (
        <Text code style={{ fontSize: 11 }}>{r.material?.code || '-'}</Text>
      ),
    }] : []),
    {
      title: '名称', dataIndex: 'name', width: 120,
      render: (v, r) => isEdit ? <EditableCell value={v} field="name" record={r} onSave={handleSave} placeholder="PCB名称" /> : v,
    },
    {
      title: '数量', dataIndex: 'quantity', width: 55, align: 'right',
      render: (v, r) => isEdit ? <EditableCell value={v} field="quantity" record={r} onSave={handleSave} type="number" /> : v,
    },
    {
      title: '层数', dataIndex: 'pcb_layers', width: 55, align: 'center',
      render: (v, r) => isEdit ? <EditableCell value={v} field="pcb_layers" record={r} onSave={handleSave} type="number" placeholder="4" /> : (v ?? '-'),
    },
    {
      title: '板厚', dataIndex: 'pcb_thickness', width: 70,
      render: (v, r) => isEdit ? <EditableCell value={v} field="pcb_thickness" record={r} onSave={handleSave} placeholder="1.6mm" /> : (v || '-'),
    },
    {
      title: '板材', dataIndex: 'pcb_material', width: 80,
      render: (v, r) => isEdit
        ? <EditableCell value={v} field="pcb_material" record={r} onSave={handleSave} type="select"
            options={PCB_MATERIAL_OPTIONS.map(o => ({ label: o, value: o }))} placeholder="FR4" />
        : (v || '-'),
    },
    {
      title: '尺寸', dataIndex: 'pcb_dimensions', width: 85,
      render: (v, r) => isEdit ? <EditableCell value={v} field="pcb_dimensions" record={r} onSave={handleSave} placeholder="50x30mm" /> : (v || '-'),
    },
    {
      title: '表面工艺', dataIndex: 'pcb_surface_finish', width: 100,
      render: (v, r) => isEdit
        ? <EditableCell value={v} field="pcb_surface_finish" record={r} onSave={handleSave} type="select"
            options={PCB_SURFACE_OPTIONS.map(o => ({ label: o, value: o }))} placeholder="沉金" />
        : (v || '-'),
    },
    {
      title: '单价', dataIndex: 'unit_price', width: 75, align: 'right',
      render: (v, r) => isEdit
        ? <EditableCell value={v} field="unit_price" record={r} onSave={handleSave} type="number" placeholder="0.00" />
        : formatCurrency(v),
    },
    {
      title: '图纸', width: 120,
      render: (_, r) => <AttachmentCell record={r} onSave={handleSave} readonly={!isEdit} />,
    },
  ];

  // ========== Service columns ==========
  const serviceCols: ColumnsType<EBOMItemRecord> = [
    { title: '#', dataIndex: 'item_number', width: 45, align: 'center' },
    ...(!isEdit ? [{
      title: '物料编码', width: 110,
      render: (_: any, r: EBOMItemRecord) => (
        <Text code style={{ fontSize: 11 }}>{r.material?.code || '-'}</Text>
      ),
    }] : []),
    {
      title: '名称', dataIndex: 'name', width: 120,
      render: (v, r) => isEdit ? <EditableCell value={v} field="name" record={r} onSave={handleSave} placeholder="贴片服务" /> : v,
    },
    {
      title: '数量', dataIndex: 'quantity', width: 55, align: 'right',
      render: (v, r) => isEdit ? <EditableCell value={v} field="quantity" record={r} onSave={handleSave} type="number" /> : v,
    },
    {
      title: '加工工艺', dataIndex: 'process_requirements', width: 160,
      render: (v, r) => isEdit ? <EditableCell value={v} field="process_requirements" record={r} onSave={handleSave} placeholder="回流焊+电测" /> : (v || '-'),
    },
    {
      title: '钢网要求', dataIndex: 'service_type', width: 100,
      render: (v, r) => isEdit ? <EditableCell value={v} field="service_type" record={r} onSave={handleSave} placeholder="有/无" /> : (v || '-'),
    },
    {
      title: '单价', dataIndex: 'unit_price', width: 80, align: 'right',
      render: (v, r) => isEdit
        ? <EditableCell value={v} field="unit_price" record={r} onSave={handleSave} type="number" placeholder="0.00" />
        : formatCurrency(v),
    },
    {
      title: '图纸', width: 120,
      render: (_, r) => <AttachmentCell record={r} onSave={handleSave} readonly={!isEdit} />,
    },
  ];

  // Add delete column in edit mode
  const addDeleteCol = (cols: ColumnsType<EBOMItemRecord>): ColumnsType<EBOMItemRecord> => {
    if (!isEdit) return cols;
    return [
      ...cols,
      {
        title: '', width: 40, align: 'center' as const,
        render: (_: any, r: EBOMItemRecord) => (
          <Popconfirm title="删除此行？" onConfirm={() => handleDelete(r.id)}>
            <DeleteOutlined style={{ color: '#ff4d4f', cursor: 'pointer' }} />
          </Popconfirm>
        ),
      },
    ];
  };

  // ========== Section renderer ==========
  const renderSection = (
    title: string,
    icon: React.ReactNode,
    data: EBOMItemRecord[],
    columns: ColumnsType<EBOMItemRecord>,
    bgColor: string,
    count?: number,
  ) => {
    if (data.length === 0 && !isEdit) return null;
    return (
      <div style={{ marginBottom: 16 }}>
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8, padding: '6px 12px',
          background: bgColor, borderRadius: '6px 6px 0 0', borderBottom: '1px solid #f0f0f0',
        }}>
          {icon}
          <Text strong style={{ fontSize: 13 }}>{title}</Text>
          <Tag style={{ fontSize: 11 }}>{count ?? data.length} 项</Tag>
        </div>
        {data.length === 0 ? (
          <div style={{ padding: '16px 0', textAlign: 'center', background: bgColor, borderRadius: '0 0 6px 6px' }}>
            <Text type="secondary" style={{ fontSize: 12 }}>暂无{title}</Text>
          </div>
        ) : (
          <Table
            columns={addDeleteCol(columns)}
            dataSource={data}
            rowKey="id"
            size="small"
            pagination={false}
            style={{ background: bgColor }}
            rowClassName={() => 'ebom-row'}
          />
        )}
      </div>
    );
  };

  return (
    <div>
      {/* Toolbar */}
      {isEdit && (
        <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap' }}>
          {onImportExcel && (
            <Button size="small" icon={<FileExcelOutlined />} onClick={onImportExcel}>
              导入Excel
            </Button>
          )}
          <Button size="small" icon={<PlusOutlined />} onClick={() => handleAdd('component')}>
            添加元器件
          </Button>
          <Button size="small" icon={<PlusOutlined />} style={{ borderColor: '#91caff', color: '#1677ff' }}
            onClick={() => handleAdd('pcb')}>
            添加PCB
          </Button>
        </div>
      )}

      {/* Empty state */}
      {items.length === 0 && !isEdit && (
        <Empty description="暂无工程BOM数据" />
      )}

      {/* Sections */}
      {renderSection(
        '元器件',
        <ExperimentOutlined style={{ color: '#595959' }} />,
        components,
        componentCols,
        '#ffffff',
        components.length,
      )}
      {renderSection(
        'PCB',
        <CloudServerOutlined style={{ color: '#1677ff' }} />,
        pcbs,
        pcbCols,
        '#f0f8ff',
      )}
      {renderSection(
        '贴片服务',
        <UploadOutlined style={{ color: '#d48806' }} />,
        services,
        serviceCols,
        '#fffdf0',
      )}

      {/* Cost summary */}
      {items.length > 0 && (
        <div style={{
          display: 'flex', gap: 24, padding: '10px 16px', background: '#fafafa',
          borderRadius: 6, border: '1px solid #f0f0f0', flexWrap: 'wrap',
        }}>
          <Text style={{ fontSize: 12 }}>
            元器件: <Text strong>{formatCurrency(componentCost)}</Text>
          </Text>
          <Text style={{ fontSize: 12 }}>
            PCB: <Text strong>{formatCurrency(pcbCost)}</Text>
          </Text>
          <Text style={{ fontSize: 12 }}>
            贴片: <Text strong>{formatCurrency(serviceCost)}</Text>
          </Text>
          <Text style={{ fontSize: 13 }}>
            PCBA总成本: <Text strong style={{ color: '#1677ff' }}>{formatCurrency(totalCost)}</Text>
          </Text>
        </div>
      )}
    </div>
  );
};

export default EBOMEditableTable;
