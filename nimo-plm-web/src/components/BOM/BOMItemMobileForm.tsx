import React, { useState, useEffect, useRef } from 'react';
import { Input, InputNumber, Select, Button, message } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { projectBomApi } from '@/api/projectBom';
import type { CategoryAttrTemplate } from '@/api/projectBom';
import { SUB_CATEGORY_LABELS } from './bomConstants';
import type { BOMControlConfig } from './bomConstants';

export interface BOMItemMobileFormProps {
  item?: Record<string, any>;
  category: string;
  subCategory: string;
  config: BOMControlConfig;
  onSave: (item: Record<string, any>) => void;
  onClose: () => void;
  bomType: string;
}

const UNIT_OPTIONS = [
  { label: 'pcs', value: 'pcs' },
  { label: 'set', value: 'set' },
  { label: 'kg', value: 'kg' },
  { label: 'm', value: 'm' },
  { label: 'roll', value: 'roll' },
  { label: 'pair', value: 'pair' },
  { label: 'lot', value: 'lot' },
];

const BOMItemMobileForm: React.FC<BOMItemMobileFormProps> = ({
  item,
  category,
  subCategory,
  config: _config,
  onSave,
  onClose,
  bomType,
}) => {
  const isEdit = !!item;
  const [visible, setVisible] = useState(false);
  const [saving, setSaving] = useState(false);
  const [templates, setTemplates] = useState<CategoryAttrTemplate[]>([]);

  // Form state
  const [form, setForm] = useState<Record<string, any>>(() => {
    if (item) {
      return {
        name: item.name || '',
        item_number: item.item_number || '',
        quantity: item.quantity || 1,
        unit: item.unit || 'pcs',
        unit_price: item.unit_price || undefined,
        specification: item.specification || '',
        manufacturer: item.manufacturer || '',
        supplier: item.supplier || '',
        notes: item.notes || '',
        ...(item.extended_attrs || {}),
      };
    }
    return {
      name: '',
      item_number: '',
      quantity: 1,
      unit: 'pcs',
      unit_price: undefined,
      specification: '',
      manufacturer: '',
      supplier: '',
      notes: '',
    };
  });

  const formRef = useRef<HTMLDivElement>(null);

  // Slide in on mount
  useEffect(() => {
    requestAnimationFrame(() => setVisible(true));
  }, []);

  // Fetch attribute templates for this sub-category
  useEffect(() => {
    projectBomApi.listTemplates({ sub_category: subCategory }).then(list => {
      // MBOM merges EBOM+PBOM, so show templates from both
      setTemplates(bomType === 'MBOM'
        ? list.filter(t => t.bom_type === 'EBOM' || t.bom_type === 'PBOM')
        : list.filter(t => t.bom_type === bomType));
    }).catch(() => {});
  }, [subCategory, bomType]);

  const handleClose = () => {
    setVisible(false);
    setTimeout(onClose, 300);
  };

  const updateField = (key: string, value: any) => {
    setForm(prev => ({ ...prev, [key]: value }));
  };

  const handleSave = async () => {
    if (!form.name?.trim()) {
      message.warning('请输入物料名称');
      return;
    }
    setSaving(true);
    try {
      // Separate base fields from extended attrs
      const baseFields = ['name', 'item_number', 'quantity', 'unit', 'unit_price', 'specification', 'manufacturer', 'supplier', 'notes'];
      const base: Record<string, any> = {};
      const extended: Record<string, any> = {};

      for (const [k, v] of Object.entries(form)) {
        if (baseFields.includes(k)) {
          base[k] = v;
        } else {
          extended[k] = v;
        }
      }

      const result: Record<string, any> = {
        ...base,
        category,
        sub_category: subCategory,
        extended_attrs: extended,
      };

      if (isEdit && item) {
        result.id = item.id;
      }

      onSave(result);
      handleClose();
    } catch {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const renderField = (
    key: string,
    label: string,
    type: 'text' | 'number' | 'select' | 'textarea' = 'text',
    options?: { label: string; value: string }[],
  ) => (
    <div key={key} style={fieldContainerStyle}>
      <div style={fieldLabelStyle}>{label}</div>
      {type === 'number' ? (
        <InputNumber
          value={form[key]}
          onChange={v => updateField(key, v)}
          style={{ width: '100%', borderRadius: 10 }}
          size="large"
          placeholder={`请输入${label}`}
        />
      ) : type === 'select' && options ? (
        <Select
          value={form[key]}
          onChange={v => updateField(key, v)}
          style={{ width: '100%' }}
          size="large"
          options={options}
          placeholder={`请选择${label}`}
        />
      ) : type === 'textarea' ? (
        <Input.TextArea
          value={form[key]}
          onChange={e => updateField(key, e.target.value)}
          rows={3}
          style={{ borderRadius: 10 }}
          placeholder={`请输入${label}`}
        />
      ) : (
        <Input
          value={form[key]}
          onChange={e => updateField(key, e.target.value)}
          size="large"
          style={{ borderRadius: 10 }}
          placeholder={`请输入${label}`}
        />
      )}
    </div>
  );

  // Render a template-defined dynamic field
  const renderTemplateField = (tmpl: CategoryAttrTemplate) => {
    const key = tmpl.field_key;
    const label = tmpl.field_name;
    if (tmpl.field_type === 'number') {
      return renderField(key, label + (tmpl.unit ? ` (${tmpl.unit})` : ''), 'number');
    }
    if (tmpl.field_type === 'select' && tmpl.options) {
      const opts = Array.isArray(tmpl.options)
        ? tmpl.options.map((o: any) => ({ label: String(o), value: String(o) }))
        : Object.entries(tmpl.options).map(([k, v]) => ({ label: String(v), value: k }));
      return renderField(key, label, 'select', opts);
    }
    if (tmpl.field_type === 'boolean') {
      return renderField(key, label, 'select', [
        { label: '是', value: 'true' },
        { label: '否', value: 'false' },
      ]);
    }
    return renderField(key, label, 'text');
  };

  const subCatLabel = SUB_CATEGORY_LABELS[subCategory] || subCategory;

  return (
    <div style={{
      position: 'fixed',
      top: 0, left: 0, right: 0, bottom: 0,
      zIndex: 1100,
      background: '#fff',
      transform: visible ? 'translateX(0)' : 'translateX(100%)',
      transition: 'transform 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
      display: 'flex',
      flexDirection: 'column',
    }}>
      {/* Header */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 12,
        padding: '12px 16px',
        borderBottom: '1px solid #f0f0f0',
        background: '#fff',
        position: 'sticky', top: 0, zIndex: 1,
      }}>
        <ArrowLeftOutlined
          onClick={handleClose}
          style={{ fontSize: 18, cursor: 'pointer', padding: 4 }}
        />
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 16, fontWeight: 600 }}>
            {isEdit ? '编辑物料' : '添加物料'}
          </div>
          <div style={{ fontSize: 12, color: '#999' }}>{subCatLabel}</div>
        </div>
        <Button
          type="primary"
          onClick={handleSave}
          loading={saving}
          style={{ borderRadius: 8, fontWeight: 500 }}
        >
          保存
        </Button>
      </div>

      {/* Form body */}
      <div ref={formRef} style={{
        flex: 1, overflow: 'auto', padding: '16px',
        WebkitOverflowScrolling: 'touch',
      }}>
        {/* Base fields */}
        <div style={sectionStyle}>
          <div style={sectionTitleStyle}>基本信息</div>
          {renderField('name', '名称')}
          {renderField('specification', '规格型号')}
          <div style={{ display: 'flex', gap: 12 }}>
            <div style={{ flex: 1 }}>
              {renderField('quantity', '数量', 'number')}
            </div>
            <div style={{ flex: 1 }}>
              {renderField('unit', '单位', 'select', UNIT_OPTIONS)}
            </div>
          </div>
          {renderField('unit_price', '单价 (¥)', 'number')}
        </div>

        {/* Supplier info */}
        <div style={sectionStyle}>
          <div style={sectionTitleStyle}>供应信息</div>
          {renderField('manufacturer', '制造商')}
          {renderField('supplier', '供应商')}
        </div>

        {/* Extended fields from templates */}
        {templates.length > 0 && (
          <div style={sectionStyle}>
            <div style={sectionTitleStyle}>扩展属性</div>
            {templates
              .sort((a, b) => a.sort_order - b.sort_order)
              .map(tmpl => renderTemplateField(tmpl))}
          </div>
        )}

        {/* Notes */}
        <div style={sectionStyle}>
          {renderField('notes', '备注', 'textarea')}
        </div>

        {/* Bottom spacing for scroll */}
        <div style={{ height: 40 }} />
      </div>
    </div>
  );
};

// Styles
const sectionStyle: React.CSSProperties = {
  marginBottom: 20,
  background: '#fafafa',
  borderRadius: 12,
  padding: '16px',
};

const sectionTitleStyle: React.CSSProperties = {
  fontSize: 14,
  fontWeight: 600,
  color: '#333',
  marginBottom: 12,
};

const fieldContainerStyle: React.CSSProperties = {
  marginBottom: 14,
};

const fieldLabelStyle: React.CSSProperties = {
  fontSize: 13,
  color: '#666',
  marginBottom: 6,
};

export default BOMItemMobileForm;
