import React, { useState, useCallback, useRef, useEffect, useMemo } from 'react';
import {
  Input,
  Select,
  Button,
  Space,
  Typography,
  Spin,
  Empty,
  Image,
  Tag,
  Tooltip,
  ColorPicker,
  Form,
  App,
  Upload,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  BgColorsOutlined,
  UploadOutlined,
  PaperClipOutlined,
  ArrowLeftOutlined,
  RightOutlined,
} from '@ant-design/icons';
import { taskFormApi } from '@/api/taskForms';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { cmfVariantApi, type AppearancePartWithCMF } from '@/api/cmfVariant';
import { useIsMobile } from '@/hooks/useIsMobile';
import EditableTable, { type EditableColumn } from './EditableTable';

const { Text } = Typography;

// ========== Option Constants ==========

const GLOSS_OPTIONS = ['高光', '半哑', '哑光', '丝光', '镜面'];
const FINISH_OPTIONS = ['阳极氧化', '喷涂', '电镀', 'PVD', 'IMD', 'UV转印', '丝印', '激光雷雕', '水转印'];
const TEXTURE_OPTIONS = ['光面', '磨砂', '皮纹', '拉丝', '碳纤维纹', '木纹'];
const COATING_OPTIONS = ['UV漆', 'PU漆', '粉末涂装', '电泳', '无'];
const DRAWING_TYPE_OPTIONS = ['丝印', '移印', 'UV转印', '激光雕刻', '水转印', '热转印', '烫金', '其他'];

const GLOSS_OPTS = GLOSS_OPTIONS.map(o => ({ label: o, value: o }));
const FINISH_OPTS = FINISH_OPTIONS.map(o => ({ label: o, value: o }));
const TEXTURE_OPTS = TEXTURE_OPTIONS.map(o => ({ label: o, value: o }));
const COATING_OPTS = COATING_OPTIONS.map(o => ({ label: o, value: o }));

// Fields to compare for auto-save diff
const CMF_FIELDS = [
  'color_hex', 'pantone_code', 'gloss_level', 'finish', 'texture', 'coating',
  'reference_image_file_id', 'reference_image_url',
  'process_drawing_type', 'process_drawings', 'notes',
];

interface DrawingFile {
  file_id: string;
  file_name: string;
  url: string;
}

const parseDrawings = (v: any): DrawingFile[] => {
  if (!v) return [];
  if (Array.isArray(v)) return v;
  try { return JSON.parse(v); } catch { return []; }
};

// ========== Props ==========

interface CMFEditControlProps {
  projectId: string;
  taskId?: string;
  readonly?: boolean;
}

// ========== Part Section (pure onChange, no mutations) ==========
const PartSection: React.FC<{
  bomItem: { id: string; name: string; item_number: number; thumbnail_url?: string };
  variants: Record<string, any>[];
  onVariantsChange: (newVariants: Record<string, any>[]) => void;
  onAddVariant: () => void;
  readonly: boolean;
}> = ({ bomItem, variants, onVariantsChange, onAddVariant, readonly }) => {
  const { message } = App.useApp();

  const cmfColumns: EditableColumn[] = useMemo(() => [
    {
      key: 'variant_index', title: '版本', width: 50, editable: false,
      render: (v: any) => <Tag color="processing" style={{ margin: 0, fontSize: 11 }}>V{v}</Tag>,
    },
    { key: 'color_hex', title: '颜色', width: 100, type: 'color' as const },
    { key: 'pantone_code', title: '色号(Pantone)', width: 100 },
    { key: 'gloss_level', title: '光泽度', width: 80, type: 'select' as const, options: GLOSS_OPTS },
    { key: 'finish', title: '表面处理', width: 90, type: 'select' as const, options: FINISH_OPTS },
    { key: 'texture', title: '纹理', width: 80, type: 'select' as const, options: TEXTURE_OPTS },
    { key: 'coating', title: '涂层类型', width: 90, type: 'select' as const, options: COATING_OPTS },
    {
      key: 'reference_image_file_id', title: '效果图', width: 60,
      render: (_: any, record: Record<string, any>) => {
        const url = record.reference_image_file_id
          ? (record.reference_image_url || `/uploads/${record.reference_image_file_id}/image`)
          : undefined;
        if (url) {
          return <Image src={url} width={40} height={30}
            style={{ objectFit: 'cover', borderRadius: 3 }}
            fallback="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNDAiIGhlaWdodD0iMzAiPjxyZWN0IGZpbGw9IiNmNWY1ZjUiIHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiLz48L3N2Zz4=" />;
        }
        if (readonly) return <Text type="secondary" style={{ fontSize: 11 }}>-</Text>;
        return (
          <Upload showUploadList={false} accept="image/*"
            beforeUpload={(file) => {
              taskFormApi.uploadFile(file).then((result) => {
                onVariantsChange(variants.map(v =>
                  v.id === record.id
                    ? { ...v, reference_image_file_id: result.id, reference_image_url: result.url }
                    : v
                ));
              }).catch(() => message.error('上传失败'));
              return false;
            }}>
            <Button size="small" type="text" icon={<UploadOutlined />} style={{ fontSize: 11, padding: '0 4px' }} />
          </Upload>
        );
      },
    },
    {
      key: 'process_drawings', title: '图纸', width: 60,
      render: (_: any, record: Record<string, any>) => {
        const drawings = parseDrawings(record.process_drawings);
        if (drawings.length > 0) {
          return (
            <Space size={2} direction="vertical">
              {record.process_drawing_type && <Tag style={{ fontSize: 10, margin: 0 }}>{record.process_drawing_type}</Tag>}
              {drawings.map((f: DrawingFile) => (
                <Tooltip key={f.file_id} title={f.file_name}>
                  <a href={f.url} target="_blank" rel="noopener noreferrer" style={{ fontSize: 11 }}>
                    <PaperClipOutlined style={{ color: '#1677ff' }} />
                  </a>
                </Tooltip>
              ))}
            </Space>
          );
        }
        if (readonly) return <Text type="secondary" style={{ fontSize: 11 }}>-</Text>;
        return (
          <Upload showUploadList={false} accept=".pdf,.dwg,.dxf,.ai,.cdr,image/*"
            beforeUpload={(file) => {
              taskFormApi.uploadFile(file).then((result) => {
                const newDrawings = [...drawings, { file_id: result.id, file_name: result.filename || file.name, url: result.url }];
                onVariantsChange(variants.map(v =>
                  v.id === record.id
                    ? { ...v, process_drawings: JSON.stringify(newDrawings) }
                    : v
                ));
              }).catch(() => message.error('上传失败'));
              return false;
            }}>
            <Button size="small" type="text" icon={<UploadOutlined />} style={{ fontSize: 11, padding: '0 4px' }} />
          </Upload>
        );
      },
    },
    { key: 'notes', title: '备注', width: 120 },
  ], [readonly, variants, onVariantsChange, message]);

  return (
    <div style={{ marginBottom: 12 }}>
      {/* Section header */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '4px 12px',
        background: '#fff7e6', borderRadius: '4px 4px 0 0',
        borderBottom: '1px solid #f0f0f0',
      }}>
        {bomItem.thumbnail_url && (
          <img src={bomItem.thumbnail_url} alt="" width={24} height={18}
            style={{ objectFit: 'contain', borderRadius: 3, background: '#f5f5f5' }} />
        )}
        <Text style={{ fontSize: 13, fontWeight: 500 }}>#{bomItem.item_number} {bomItem.name}</Text>
        <Tag style={{ fontSize: 11 }}>{variants.length} 方案</Tag>
        <div style={{ flex: 1 }} />
        {!readonly && (
          <Button type="dashed" size="small" icon={<PlusOutlined />} onClick={onAddVariant}>
            添加方案
          </Button>
        )}
      </div>

      {/* Table */}
      {variants.length > 0 ? (
        <EditableTable
          columns={cmfColumns}
          items={variants}
          onChange={onVariantsChange}
          readonly={readonly}
          rowKey="id"
          deleteConfirmText="确认删除此CMF方案？"
        />
      ) : (
        <div style={{ padding: '16px 0', textAlign: 'center', color: '#999', fontSize: 12, border: '1px solid #f0f0f0', borderTop: 0, borderRadius: '0 0 4px 4px' }}>
          暂无CMF方案
        </div>
      )}
    </div>
  );
};

// ========== 移动端: CMF变体全屏编辑面板 ==========
const CMFMobileEditPanel: React.FC<{
  variant: Record<string, any>;
  readonly: boolean;
  onSave: (variantId: string, data: Record<string, any>) => void;
  onClose: () => void;
}> = ({ variant, readonly, onSave, onClose }) => {
  const { message } = App.useApp();
  const [visible, setVisible] = useState(false);
  const [form] = Form.useForm();

  const [renderUploading, setRenderUploading] = useState(false);
  const [drawingUploading, setDrawingUploading] = useState(false);
  const [renderImageId, setRenderImageId] = useState(variant.reference_image_file_id || '');
  const [renderImageUrl, setRenderImageUrlState] = useState(variant.reference_image_url || '');
  const [drawingType, setDrawingType] = useState(variant.process_drawing_type || '');
  const [drawings, setDrawings] = useState<DrawingFile[]>(parseDrawings(variant.process_drawings));

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true));
  }, []);

  const handleClose = () => {
    setVisible(false);
    setTimeout(onClose, 300);
  };

  const handleSave = () => {
    form.validateFields().then(values => {
      if (values.color_hex && typeof values.color_hex === 'object' && values.color_hex.toHexString) {
        values.color_hex = values.color_hex.toHexString();
      }
      values.reference_image_file_id = renderImageId;
      values.reference_image_url = renderImageUrl;
      values.process_drawing_type = drawingType;
      values.process_drawings = JSON.stringify(drawings);
      onSave(variant.id, values);
      handleClose();
    });
  };

  const handleRenderUpload = async (file: File) => {
    setRenderUploading(true);
    try {
      const result = await taskFormApi.uploadFile(file);
      setRenderImageId(result.id);
      setRenderImageUrlState(result.url);
    } catch { message.error('上传失败'); }
    finally { setRenderUploading(false); }
    return false;
  };

  const handleDrawingUpload = async (file: File) => {
    setDrawingUploading(true);
    try {
      const result = await taskFormApi.uploadFile(file);
      setDrawings(prev => [...prev, { file_id: result.id, file_name: result.filename || file.name, url: result.url }]);
    } catch { message.error('上传失败'); }
    finally { setDrawingUploading(false); }
    return false;
  };

  const existingRenderUrl = renderImageId ? (renderImageUrl || `/uploads/${renderImageId}/image`) : undefined;

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
      }}>
        <ArrowLeftOutlined
          onClick={handleClose}
          style={{ fontSize: 18, cursor: 'pointer', padding: 4 }}
        />
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 16, fontWeight: 600 }}>
            {readonly ? '查看CMF' : '编辑CMF'}
          </div>
          <div style={{ fontSize: 12, color: '#999' }}>
            V{variant.variant_index} {variant.material_code || ''}
          </div>
        </div>
        {!readonly && (
          <Button
            type="primary"
            onClick={handleSave}
            style={{ borderRadius: 8, fontWeight: 500 }}
          >
            保存
          </Button>
        )}
      </div>

      {/* Form body */}
      <div style={{
        flex: 1, overflow: 'auto', padding: 16,
        WebkitOverflowScrolling: 'touch',
      }}>
        <Form form={form} layout="vertical"
          initialValues={{
            color_hex: variant.color_hex || '',
            pantone_code: variant.pantone_code || '',
            gloss_level: variant.gloss_level || undefined,
            finish: variant.finish || undefined,
            texture: variant.texture || undefined,
            coating: variant.coating || undefined,
            notes: variant.notes || '',
          }}
        >
          {/* 颜色 section */}
          <div style={mfSectionStyle}>
            <div style={mfSectionTitleStyle}>颜色信息</div>
            <Form.Item label="颜色" name="color_hex" style={{ marginBottom: 14 }}>
              <ColorPicker showText format="hex" disabled={readonly} />
            </Form.Item>
            <Form.Item label="色号 (Pantone)" name="pantone_code" style={{ marginBottom: 14 }}>
              <Input placeholder="如: Black 6C" size="large" style={{ borderRadius: 10 }} disabled={readonly} />
            </Form.Item>
          </div>

          {/* 表面处理 section */}
          <div style={mfSectionStyle}>
            <div style={mfSectionTitleStyle}>表面处理</div>
            <Form.Item label="光泽度" name="gloss_level" style={{ marginBottom: 14 }}>
              <Select placeholder="选择光泽度" allowClear size="large"
                disabled={readonly}
                options={GLOSS_OPTIONS.map(o => ({ label: o, value: o }))} />
            </Form.Item>
            <Form.Item label="表面处理" name="finish" style={{ marginBottom: 14 }}>
              <Select placeholder="选择表面处理" allowClear size="large"
                disabled={readonly}
                options={FINISH_OPTIONS.map(o => ({ label: o, value: o }))} />
            </Form.Item>
            <Form.Item label="纹理" name="texture" style={{ marginBottom: 14 }}>
              <Select placeholder="选择纹理" allowClear size="large"
                disabled={readonly}
                options={TEXTURE_OPTIONS.map(o => ({ label: o, value: o }))} />
            </Form.Item>
            <Form.Item label="涂层类型" name="coating" style={{ marginBottom: 14 }}>
              <Select placeholder="选择涂层" allowClear size="large"
                disabled={readonly}
                options={COATING_OPTIONS.map(o => ({ label: o, value: o }))} />
            </Form.Item>
          </div>

          {/* 渲染图 & 图纸 */}
          <div style={mfSectionStyle}>
            <div style={mfSectionTitleStyle}>附件</div>
            <div style={{ marginBottom: 14 }}>
              <Text style={{ fontSize: 13, color: '#666', display: 'block', marginBottom: 6 }}>渲染效果图</Text>
              {existingRenderUrl && (
                <div style={{ position: 'relative', display: 'inline-block', marginBottom: 8 }}>
                  <Image src={existingRenderUrl} width={100} height={75}
                    style={{ objectFit: 'cover', borderRadius: 8 }}
                    fallback="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNDUiPjxyZWN0IGZpbGw9IiNmNWY1ZjUiIHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiLz48L3N2Zz4=" />
                  {!readonly && (
                    <DeleteOutlined
                      style={{ position: 'absolute', top: -6, right: -6, color: '#ff4d4f', cursor: 'pointer', fontSize: 14, background: '#fff', borderRadius: '50%', padding: 3, boxShadow: '0 1px 4px rgba(0,0,0,0.15)' }}
                      onClick={() => { setRenderImageId(''); setRenderImageUrlState(''); }}
                    />
                  )}
                </div>
              )}
              {!readonly && (
                <Upload showUploadList={false} accept="image/*" beforeUpload={(file) => { handleRenderUpload(file); return false; }}>
                  <Button icon={<UploadOutlined />} loading={renderUploading} block style={{ borderRadius: 10 }}>上传效果图</Button>
                </Upload>
              )}
            </div>

            <div style={{ marginBottom: 14 }}>
              <Text style={{ fontSize: 13, color: '#666', display: 'block', marginBottom: 6 }}>工艺图纸</Text>
              {!readonly && (
                <Space size={8} style={{ marginBottom: 8, width: '100%' }}>
                  <Select value={drawingType || undefined} placeholder="图纸类型"
                    allowClear style={{ flex: 1 }}
                    onChange={(v) => setDrawingType(v || '')}
                    options={DRAWING_TYPE_OPTIONS.map(o => ({ label: o, value: o }))} />
                  <Upload showUploadList={false} accept=".pdf,.dwg,.dxf,.ai,.cdr,image/*"
                    beforeUpload={(file) => { handleDrawingUpload(file); return false; }}>
                    <Button icon={<UploadOutlined />} loading={drawingUploading}>上传</Button>
                  </Upload>
                </Space>
              )}
              {variant.process_drawing_type && !drawingType && (
                <Tag style={{ marginBottom: 4 }}>{variant.process_drawing_type}</Tag>
              )}
              {drawings.map(f => (
                <div key={f.file_id} style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, marginBottom: 4, padding: '4px 0' }}>
                  <PaperClipOutlined style={{ color: '#1677ff' }} />
                  <a href={f.url} target="_blank" rel="noopener noreferrer"
                    style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {f.file_name}
                  </a>
                  {!readonly && (
                    <DeleteOutlined style={{ color: '#ff4d4f', cursor: 'pointer' }}
                      onClick={() => setDrawings(prev => prev.filter(d => d.file_id !== f.file_id))} />
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* 备注 */}
          <div style={mfSectionStyle}>
            <Form.Item label="备注" name="notes" style={{ marginBottom: 0 }}>
              <Input.TextArea rows={3} placeholder="备注信息" style={{ borderRadius: 10 }} disabled={readonly} />
            </Form.Item>
          </div>

          <div style={{ height: 40 }} />
        </Form>
      </div>
    </div>
  );
};

const mfSectionStyle: React.CSSProperties = {
  marginBottom: 20,
  background: '#fafafa',
  borderRadius: 12,
  padding: 16,
};

const mfSectionTitleStyle: React.CSSProperties = {
  fontSize: 14,
  fontWeight: 600,
  color: '#333',
  marginBottom: 12,
};

// ========== 移动端: CMF摘要卡片 ==========
const MobileVariantSummaryCard: React.FC<{
  variant: Record<string, any>;
  onClick: () => void;
}> = ({ variant, onClick }) => {
  const renderImageUrl = variant.reference_image_file_id
    ? (variant.reference_image_url || `/uploads/${variant.reference_image_file_id}/image`)
    : undefined;

  return (
    <div
      onClick={onClick}
      style={{
        display: 'flex', alignItems: 'center', gap: 12,
        padding: '12px 14px',
        background: '#fff',
        borderRadius: 10,
        border: '1px solid #f0f0f0',
        marginBottom: 8,
        cursor: 'pointer',
        transition: 'background 0.15s',
      }}
    >
      {renderImageUrl ? (
        <img src={renderImageUrl} alt="" width={44} height={44}
          style={{ objectFit: 'cover', borderRadius: 8, background: '#f5f5f5', flexShrink: 0 }} />
      ) : variant.color_hex ? (
        <div style={{
          width: 44, height: 44, borderRadius: 8, flexShrink: 0,
          backgroundColor: variant.color_hex,
          border: '1px solid #e8e8e8',
          boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
        }} />
      ) : (
        <div style={{
          width: 44, height: 44, borderRadius: 8, flexShrink: 0,
          background: '#f5f5f5', display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <BgColorsOutlined style={{ color: '#bfbfbf', fontSize: 18 }} />
        </div>
      )}

      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
          <Tag color="processing" style={{ margin: 0, fontSize: 10, lineHeight: '16px', padding: '0 4px' }}>
            V{variant.variant_index}
          </Tag>
          {variant.material_code && (
            <Text style={{ fontSize: 11, color: '#8c8c8c' }}>{variant.material_code}</Text>
          )}
        </div>
        <div style={{ fontSize: 13, color: '#333', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {variant.color_hex || '未设颜色'}
          {variant.pantone_code ? ` / ${variant.pantone_code}` : ''}
        </div>
        <div style={{ fontSize: 11, color: '#999', marginTop: 2 }}>
          {[variant.gloss_level, variant.finish, variant.texture].filter(Boolean).join(' / ') || '未设置表面处理'}
        </div>
      </div>

      <RightOutlined style={{ color: '#bfbfbf', fontSize: 12, flexShrink: 0 }} />
    </div>
  );
};

// ========== 主组件 ==========

const CMFEditControl: React.FC<CMFEditControlProps> = ({ projectId, taskId: _taskId, readonly = false }) => {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();
  const [editingVariant, setEditingVariant] = useState<Record<string, any> | null>(null);

  // Local state — single source of truth for UI
  const [localVariants, setLocalVariants] = useState<Record<string, any>[]>([]);
  const [initialized, setInitialized] = useState(false);
  const serverVariantsRef = useRef<Record<string, any>[]>([]);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const syncingRef = useRef(false);

  // Fetch server data
  const { data: parts = [], isLoading } = useQuery({
    queryKey: ['appearance-parts', projectId],
    queryFn: () => cmfVariantApi.getAppearanceParts(projectId),
  });

  // Sync server data → local state (skip during save to avoid overwriting local edits)
  // Sync server data → local state (same pattern as BOM)
  useEffect(() => {
    if (syncingRef.current) return;
    if (!parts.length) {
      serverVariantsRef.current = [];
      setLocalVariants([]);
      setInitialized(true);
      return;
    }
    const allVariants: Record<string, any>[] = parts.flatMap(p =>
      (p.cmf_variants || []).map(v => ({ ...v, bom_item_id: p.bom_item.id }))
    );
    serverVariantsRef.current = allVariants;
    setLocalVariants(allVariants);
    setInitialized(true);
  }, [parts]);

  // Get variants for a specific bom_item
  const getVariantsForItem = useCallback((itemId: string) =>
    localVariants.filter(v => v.bom_item_id === itemId),
  [localVariants]);

  // Handle variants change for a specific bom_item
  const handleVariantsChange = useCallback((itemId: string, newItemVariants: Record<string, any>[]) => {
    setLocalVariants(prev => [
      ...prev.filter(v => v.bom_item_id !== itemId),
      ...newItemVariants,
    ]);
  }, []);

  // Add variant — instant local, auto-save creates on server
  const handleAddVariant = useCallback((itemId: string) => {
    const itemVariants = localVariants.filter(v => v.bom_item_id === itemId);
    const maxIdx = itemVariants.reduce((m, v) => Math.max(m, v.variant_index || 0), 0);
    setLocalVariants(prev => [...prev, {
      id: 'new-' + Date.now(),
      bom_item_id: itemId,
      variant_index: maxIdx + 1,
      color_hex: '', pantone_code: '', gloss_level: '', finish: '', texture: '',
      coating: '', reference_image_file_id: '', reference_image_url: '',
      process_drawing_type: '', process_drawings: '', notes: '', material_code: '',
    }]);
  }, [localVariants]);

  // Mobile save — update local state
  const handleMobileSave = useCallback((variantId: string, data: Record<string, any>) => {
    setLocalVariants(prev => prev.map(v =>
      v.id === variantId ? { ...v, ...data } : v
    ));
  }, []);

  // ========== Auto-save: debounced diff & sync ==========
  useEffect(() => {
    if (readonly) return;
    if (syncingRef.current) return;

    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);

    saveTimerRef.current = setTimeout(async () => {
      if (syncingRef.current) return;
      syncingRef.current = true;

      try {
        const server = serverVariantsRef.current;
        const local = localVariants;
        const serverIds = new Set(server.map(v => v.id));
        const localIds = new Set(local.map(v => v.id));

        // 1. Deleted variants (in server but not in local)
        const deletedIds = [...serverIds].filter(id => !localIds.has(id));
        for (const id of deletedIds) {
          await cmfVariantApi.deleteVariant(projectId, id);
        }

        // 2. New variants (id starts with 'new-')
        const newVariants = local.filter(v => typeof v.id === 'string' && v.id.startsWith('new-'));
        const idMapping: Record<string, Record<string, any>> = {};
        for (const v of newVariants) {
          const payload: Record<string, any> = {};
          for (const f of CMF_FIELDS) {
            if (v[f]) payload[f] = v[f];
          }
          const created = await cmfVariantApi.createVariant(projectId, v.bom_item_id, payload);
          idMapping[v.id] = created;
        }

        // 3. Updated variants (exist in both, check for changes)
        const existing = local.filter(v =>
          !(typeof v.id === 'string' && v.id.startsWith('new-')) && serverIds.has(v.id)
        );
        for (const v of existing) {
          const sv = server.find(s => s.id === v.id);
          if (!sv) continue;
          const updateData: Record<string, any> = {};
          let hasUpdate = false;
          for (const f of CMF_FIELDS) {
            if (String(v[f] ?? '') !== String(sv[f] ?? '')) {
              updateData[f] = v[f] ?? '';
              hasUpdate = true;
            }
          }
          if (hasUpdate) {
            await cmfVariantApi.updateVariant(projectId, v.id, updateData);
          }
        }

        // Update refs — replace temp ids with server data
        const updated = local.map(v => {
          if (idMapping[v.id]) return { ...idMapping[v.id], bom_item_id: v.bom_item_id };
          return v;
        });
        serverVariantsRef.current = updated;
        // Only update local state if temp IDs were replaced (matching BOM pattern)
        if (Object.keys(idMapping).length > 0) {
          setLocalVariants(updated);
        }

        // Refresh server cache in background (won't overwrite local because syncingRef is still true)
        await queryClient.invalidateQueries({ queryKey: ['appearance-parts', projectId] });

      } catch {
        message.error('保存失败');
        queryClient.invalidateQueries({ queryKey: ['appearance-parts', projectId] });
      } finally {
        syncingRef.current = false;
      }
    }, 1500);

    return () => {
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    };
  }, [localVariants, projectId, readonly, queryClient, message]);

  if (isLoading || !initialized) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载CMF数据..." /></div>;
  }

  if (parts.length === 0) {
    return <Empty description="未找到外观件。请先在EBOM中将零件标记为外观件。" />;
  }

  // ===== Mobile layout =====
  if (isMobile) {
    return (
      <div>
        <div style={{ marginBottom: 10, display: 'flex', alignItems: 'center', gap: 6 }}>
          <BgColorsOutlined style={{ color: '#1677ff' }} />
          <Text type="secondary" style={{ fontSize: 12 }}>CMF方案</Text>
        </div>

        {parts.map((part: AppearancePartWithCMF) => {
          const item = part.bom_item;
          const variants = getVariantsForItem(item.id);

          return (
            <div key={item.id} style={{ marginBottom: 16 }}>
              <div style={{
                display: 'flex', alignItems: 'center', gap: 8,
                padding: '8px 0', marginBottom: 4,
              }}>
                {item.thumbnail_url && (
                  <img src={item.thumbnail_url} alt="" width={24} height={18}
                    style={{ objectFit: 'contain', borderRadius: 3, background: '#f5f5f5' }} />
                )}
                <Text strong style={{ fontSize: 13, flex: 1 }}>
                  #{item.item_number} {item.name}
                </Text>
                <Tag color="blue" style={{ fontSize: 10, margin: 0 }}>{variants.length}方案</Tag>
              </div>

              {variants.map(v => (
                <MobileVariantSummaryCard
                  key={v.id}
                  variant={v}
                  onClick={() => setEditingVariant(v)}
                />
              ))}

              {!readonly && (
                <Button
                  type="dashed"
                  icon={<PlusOutlined />}
                  onClick={() => handleAddVariant(item.id)}
                  block
                  style={{ borderRadius: 10, marginTop: 4 }}
                >
                  添加方案
                </Button>
              )}
            </div>
          );
        })}

        {editingVariant && (
          <CMFMobileEditPanel
            variant={editingVariant}
            readonly={readonly}
            onSave={handleMobileSave}
            onClose={() => setEditingVariant(null)}
          />
        )}
      </div>
    );
  }

  // ===== Desktop layout: EditableTable sections =====
  return (
    <div>
      <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
        <BgColorsOutlined style={{ color: '#1677ff' }} />
        <Text strong style={{ fontSize: 14 }}>CMF方案</Text>
        <Tag style={{ fontSize: 11 }}>
          {localVariants.length} 方案
        </Tag>
      </div>

      {parts.map((part: AppearancePartWithCMF) => (
        <PartSection
          key={part.bom_item.id}
          bomItem={part.bom_item}
          variants={getVariantsForItem(part.bom_item.id)}
          onVariantsChange={(newVariants) => handleVariantsChange(part.bom_item.id, newVariants)}
          onAddVariant={() => handleAddVariant(part.bom_item.id)}
          readonly={readonly}
        />
      ))}
    </div>
  );
};

export default CMFEditControl;
