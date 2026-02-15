import React, { useState } from 'react';
import {
  Table,
  Input,
  InputNumber,
  Button,
  Typography,
  Popconfirm,
  Empty,
  Upload,
  Select,
  message,
} from 'antd';
import {
  DeleteOutlined,
  PlusOutlined,
  UploadOutlined,
  PaperClipOutlined,
} from '@ant-design/icons';
import { taskFormApi } from '@/api/taskForms';
import type { ColumnsType } from 'antd/es/table';

const { Text } = Typography;

// ========== Types ==========

export type PBOMItemRecord = Record<string, any>;

export interface PBOMEditableTableProps {
  items: PBOMItemRecord[];
  onChange: (items: PBOMItemRecord[]) => void;
  mode?: 'edit' | 'view';
  onItemSave?: (itemId: string, field: string, value: any) => void;
}

// ========== Constants ==========

const CATEGORY_OPTIONS = [
  { value: '说明书', label: '说明书' },
  { value: '外包装盒', label: '外包装盒' },
  { value: '内衬/托盘', label: '内衬/托盘' },
  { value: '充电线', label: '充电线' },
  { value: '擦镜布', label: '擦镜布' },
  { value: '保修卡', label: '保修卡' },
  { value: '标签贴纸', label: '标签贴纸' },
  { value: '其他', label: '其他' },
];

const LANG_OPTIONS = [
  { value: '通用', label: '通用' },
  { value: '中国', label: '中国' },
  { value: '美国', label: '美国' },
  { value: '日本', label: '日本' },
  { value: '韩国', label: '韩国' },
  { value: '德国', label: '德国' },
  { value: '法国', label: '法国' },
  { value: '其他', label: '其他' },
];

const formatCurrency = (v: number | null | undefined): string => {
  if (v == null || isNaN(v)) return '-';
  return `¥${v.toFixed(2)}`;
};

// ========== File cell ==========
const FileCell: React.FC<{
  fileId?: string;
  fileName?: string;
  record: PBOMItemRecord;
  onSave: (id: string, field: string, value: any) => void;
  readonly: boolean;
}> = ({ fileId, fileName, record, onSave, readonly }) => {
  const [uploading, setUploading] = useState(false);

  const handleUpload = async (file: File) => {
    setUploading(true);
    try {
      const result = await taskFormApi.uploadFile(file);
      onSave(record.id, 'design_file_id', result.id);
      onSave(record.id, 'design_file_name', result.filename || file.name);
    } catch { message.error('上传失败'); }
    finally { setUploading(false); }
    return false;
  };

  if (fileId && fileName) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12 }}>
        <PaperClipOutlined style={{ color: '#1677ff' }} />
        <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {fileName}
        </span>
        {!readonly && (
          <DeleteOutlined style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 11 }}
            onClick={() => { onSave(record.id, 'design_file_id', ''); onSave(record.id, 'design_file_name', ''); }} />
        )}
      </div>
    );
  }

  if (readonly) return <Text style={{ fontSize: 12 }}>-</Text>;

  return (
    <Upload showUploadList={false} beforeUpload={(file) => { handleUpload(file); return false; }}>
      <Button size="small" type="dashed" icon={<UploadOutlined />} loading={uploading} style={{ fontSize: 11 }}>
        上传
      </Button>
    </Upload>
  );
};

// ========== Main Component ==========

const PBOMEditableTable: React.FC<PBOMEditableTableProps> = ({
  items,
  onChange,
  mode = 'view',
  onItemSave,
}) => {
  const isEdit = mode === 'edit';

  const totalCost = items.reduce((s, i) => s + (i.quantity || 0) * (i.unit_price || 0), 0);

  const handleSave = (id: string, field: string, value: any) => {
    if (onItemSave) {
      onItemSave(id, field, value);
    } else {
      const updated = items.map(i => i.id === id ? { ...i, [field]: value } : i);
      onChange(updated);
    }
  };

  const handleAdd = () => {
    const maxNum = items.reduce((m, i) => Math.max(m, i.item_number || 0), 0);
    const newItem: PBOMItemRecord = {
      id: `new-${Date.now()}`,
      item_number: maxNum + 1,
      category: '',
      name: '',
      quantity: 1,
      unit: 'pcs',
      language_code: '通用',
    };
    onChange([...items, newItem]);
  };

  const handleDelete = (id: string) => {
    onChange(items.filter(i => i.id !== id));
  };

  const columns: ColumnsType<PBOMItemRecord> = [
    { title: '序号', dataIndex: 'item_number', width: 50, align: 'center' },
    ...(!isEdit ? [{
      title: '物料编码', width: 110,
      render: (_: any, r: PBOMItemRecord) => (
        <Text code style={{ fontSize: 11 }}>{r.material?.code || '-'}</Text>
      ),
    }] : []),
    {
      title: '名称', dataIndex: 'name', width: 140,
      render: (v, r) => isEdit
        ? <Input size="small" defaultValue={v || ''} placeholder="包装项名称"
            onBlur={(e) => { if (e.target.value !== (v || '')) handleSave(r.id, 'name', e.target.value); }} />
        : (v || '-'),
    },
    {
      title: '类型', dataIndex: 'category', width: 120,
      render: (v, r) => isEdit
        ? <Select size="small" value={v || undefined} placeholder="选择类型"
            style={{ width: '100%' }} options={CATEGORY_OPTIONS}
            onChange={(val) => handleSave(r.id, 'category', val)} />
        : (v || '-'),
    },
    {
      title: '数量', dataIndex: 'quantity', width: 70, align: 'right',
      render: (v, r) => isEdit
        ? <InputNumber size="small" value={v} min={1} style={{ width: '100%' }}
            onChange={(val) => handleSave(r.id, 'quantity', val)} />
        : v,
    },
    {
      title: '单价', dataIndex: 'unit_price', width: 80, align: 'right',
      render: (v, r) => isEdit
        ? <InputNumber size="small" value={v} min={0} step={0.01} placeholder="0.00"
            style={{ width: '100%' }}
            onChange={(val) => handleSave(r.id, 'unit_price', val)} />
        : formatCurrency(v),
    },
    {
      title: '设计稿', width: 130,
      render: (_, r) => <FileCell fileId={r.design_file_id} fileName={r.design_file_name}
        record={r} onSave={handleSave} readonly={!isEdit} />,
    },
    {
      title: '语言版本', dataIndex: 'language_code', width: 100,
      render: (v, r) => isEdit
        ? <Select size="small" value={v || '通用'} style={{ width: '100%' }}
            options={LANG_OPTIONS}
            onChange={(val) => handleSave(r.id, 'language_code', val)} />
        : (v || '通用'),
    },
  ];

  if (isEdit) {
    columns.push({
      title: '', width: 40, align: 'center',
      render: (_, r) => (
        <Popconfirm title="删除此行？" onConfirm={() => handleDelete(r.id)}>
          <DeleteOutlined style={{ color: '#ff4d4f', cursor: 'pointer' }} />
        </Popconfirm>
      ),
    });
  }

  return (
    <div>
      {isEdit && (
        <div style={{ marginBottom: 12 }}>
          <Button icon={<PlusOutlined />} onClick={handleAdd}>添加包装项</Button>
        </div>
      )}

      {items.length === 0 && !isEdit ? (
        <Empty description="暂无工艺BOM(PBOM)数据" />
      ) : (
        <Table
          columns={columns}
          dataSource={items}
          rowKey="id"
          size="small"
          pagination={false}
        />
      )}

      {items.length > 0 && (
        <div style={{
          display: 'flex', gap: 24, padding: '10px 16px', marginTop: 12,
          background: '#fafafa', borderRadius: 6, border: '1px solid #f0f0f0',
        }}>
          <Text style={{ fontSize: 13 }}>
            单台包装成本: <Text strong style={{ color: '#1677ff' }}>{formatCurrency(totalCost)}</Text>
          </Text>
          <Text style={{ fontSize: 12 }} type="secondary">
            共 {items.length} 项
          </Text>
        </div>
      )}
    </div>
  );
};

export default PBOMEditableTable;
