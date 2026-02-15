import React from 'react';
import { Space, Typography, Spin, Empty, Tag, Button, Input, Select, Form, Popconfirm, Collapse, App, ColorPicker, Tooltip, Image, Upload } from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined, SaveOutlined, CloseOutlined, BgColorsOutlined, UploadOutlined, PaperClipOutlined } from '@ant-design/icons';
import { taskFormApi } from '@/api/taskForms';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { cmfVariantApi, type CMFVariant, type AppearancePartWithCMF } from '@/api/cmfVariant';

const { Text } = Typography;

// ========== 常量选项（与CMFEditControl统一） ==========
const GLOSS_OPTIONS = ['高光', '半哑', '哑光', '丝光', '镜面'];

const FINISH_OPTIONS = [
  '阳极氧化', '喷涂', '电镀', 'PVD', 'IMD', 'UV转印', '丝印', '激光雷雕', '水转印',
];

const TEXTURE_OPTIONS = ['光面', '磨砂', '皮纹', '拉丝', '碳纤维纹', '木纹'];

const COATING_OPTIONS = ['UV漆', 'PU漆', '粉末涂装', '电泳', '无'];

const DRAWING_TYPE_OPTIONS = ['丝印', '移印', 'UV转印', '激光雕刻', '水转印', '热转印', '烫金', '其他'];

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

// ========== Types ==========
interface CMFVariantEditorProps {
  projectId: string;
  mode?: 'edit' | 'view';
}

// ========== 属性行组件 ==========
const PropItem: React.FC<{ label: string; value?: string; suffix?: React.ReactNode }> = ({ label, value, suffix }) => (
  <div style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 0 }}>
    <Text type="secondary" style={{ fontSize: 12, flexShrink: 0 }}>{label}</Text>
    <Text style={{ fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
      {value || '-'}
    </Text>
    {suffix}
  </div>
);

// ========== 色块组件 ==========
const ColorSwatch: React.FC<{ hex?: string; size?: number }> = ({ hex, size = 16 }) => {
  if (!hex) return null;
  return (
    <span style={{
      display: 'inline-block', width: size, height: size, borderRadius: 3,
      backgroundColor: hex, border: '1px solid #d9d9d9', verticalAlign: 'middle',
      boxShadow: '0 1px 2px rgba(0,0,0,0.08)',
    }} />
  );
};

// ========== 单个变体卡片 (查看模式) ==========
const VariantViewCard: React.FC<{ variant: CMFVariant }> = ({ variant }) => {
  const renderImageUrl = variant.reference_image_file_id
    ? (variant.reference_image_url || `/uploads/${variant.reference_image_file_id}/image`)
    : undefined;
  const viewDrawings = parseDrawings(variant.process_drawings);

  return (
    <div style={{
      border: '1px solid #f0f0f0',
      borderRadius: 8,
      padding: '12px 16px',
      marginBottom: 8,
      background: '#fafafa',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Space size={6}>
          <Tag color="processing" style={{ margin: 0, fontSize: 11 }}>V{variant.variant_index}</Tag>
          {variant.material_code && <Text style={{ fontSize: 11, color: '#8c8c8c' }}>{variant.material_code}</Text>}
        </Space>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '4px 16px' }}>
        <PropItem label="颜色:" value={variant.color_hex} suffix={<ColorSwatch hex={variant.color_hex} />} />
        <PropItem label="色号(Pantone):" value={variant.pantone_code} />
        <PropItem label="光泽度:" value={variant.gloss_level} />
        <PropItem label="表面处理:" value={variant.finish} />
        <PropItem label="纹理:" value={variant.texture} />
        <PropItem label="涂层类型:" value={variant.coating} />
      </div>

      {/* 渲染效果图 */}
      <div style={{ marginTop: 6, fontSize: 12 }}>
        <Text type="secondary">渲染效果图: </Text>
        {renderImageUrl ? (
          <Image src={renderImageUrl} width={60} height={45}
            style={{ objectFit: 'cover', borderRadius: 4, marginLeft: 4 }}
            fallback="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNDUiPjxyZWN0IGZpbGw9IiNmNWY1ZjUiIHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiLz48L3N2Zz4=" />
        ) : <Text style={{ fontSize: 12 }}>-</Text>}
      </div>

      {/* 工艺图纸 */}
      <div style={{ marginTop: 4, fontSize: 12 }}>
        <Text type="secondary">工艺图纸: </Text>
        {variant.process_drawing_type && <Tag style={{ fontSize: 11 }}>{variant.process_drawing_type}</Tag>}
        {viewDrawings.length > 0 ? viewDrawings.map(f => (
          <div key={f.file_id} style={{ display: 'inline-flex', alignItems: 'center', gap: 4, marginLeft: 4 }}>
            <PaperClipOutlined style={{ color: '#1677ff', fontSize: 11 }} />
            <a href={f.url} target="_blank" rel="noopener noreferrer" style={{ fontSize: 12 }}>{f.file_name}</a>
          </div>
        )) : !variant.process_drawing_type && <Text style={{ fontSize: 12 }}>-</Text>}
      </div>

      {/* 备注 */}
      <div style={{ marginTop: 4, fontSize: 12 }}>
        <Text type="secondary">备注: </Text><Text style={{ fontSize: 12 }}>{variant.notes || '-'}</Text>
      </div>
    </div>
  );
};

// ========== 单个变体卡片 (编辑模式) ==========
const VariantEditCard: React.FC<{
  variant: CMFVariant;
  projectId: string;
}> = ({ variant, projectId }) => {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [editing, setEditing] = React.useState(false);
  const [form] = Form.useForm();

  const updateMutation = useMutation({
    mutationFn: (data: Partial<CMFVariant>) =>
      cmfVariantApi.updateVariant(projectId, variant.id, data),
    onSuccess: () => {
      message.success('已保存');
      queryClient.invalidateQueries({ queryKey: ['appearance-parts', projectId] });
      setEditing(false);
    },
    onError: (err: any) => message.error(err?.response?.data?.message || '保存失败'),
  });

  const deleteMutation = useMutation({
    mutationFn: () => cmfVariantApi.deleteVariant(projectId, variant.id),
    onSuccess: () => {
      message.success('已删除');
      queryClient.invalidateQueries({ queryKey: ['appearance-parts', projectId] });
    },
    onError: (err: any) => message.error(err?.response?.data?.message || '删除失败'),
  });

  const [renderUploading, setRenderUploading] = React.useState(false);
  const [drawingUploading, setDrawingUploading] = React.useState(false);
  const [renderImageId, setRenderImageId] = React.useState(variant.reference_image_file_id || '');
  const [renderImageUrl, setRenderImageUrlState] = React.useState(variant.reference_image_url || '');
  const [drawingType, setDrawingType] = React.useState(variant.process_drawing_type || '');
  const [drawings, setDrawings] = React.useState<DrawingFile[]>(parseDrawings(variant.process_drawings));

  const handleSave = () => {
    form.validateFields().then(values => {
      if (values.color_hex && typeof values.color_hex === 'object' && values.color_hex.toHexString) {
        values.color_hex = values.color_hex.toHexString();
      }
      values.reference_image_file_id = renderImageId;
      values.reference_image_url = renderImageUrl;
      values.process_drawing_type = drawingType;
      values.process_drawings = JSON.stringify(drawings);
      updateMutation.mutate(values);
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

  // ---- 编辑态 ----
  if (editing) {
    const existingRenderUrl = renderImageId ? (renderImageUrl || `/uploads/${renderImageId}/image`) : undefined;

    return (
      <div style={{
        border: '1px solid #91caff',
        borderRadius: 8,
        padding: '12px 16px',
        marginBottom: 8,
        background: '#f0f5ff',
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 10 }}>
          <Space size={6}>
            <Tag color="processing" style={{ margin: 0, fontSize: 11 }}>V{variant.variant_index}</Tag>
            {variant.material_code && <Text style={{ fontSize: 11, color: '#8c8c8c' }}>{variant.material_code}</Text>}
            <Tag color="blue" style={{ margin: 0, fontSize: 11 }}>编辑中</Tag>
          </Space>
          <Space size={4}>
            <Button size="small" icon={<CloseOutlined />} onClick={() => setEditing(false)}>取消</Button>
            <Button size="small" type="primary" icon={<SaveOutlined />}
              onClick={handleSave} loading={updateMutation.isPending}>保存</Button>
          </Space>
        </div>

        <Form form={form} size="small" layout="vertical"
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
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '4px 12px' }}>
            <Form.Item label="颜色" name="color_hex" style={{ marginBottom: 8 }}>
              <ColorPicker showText format="hex" size="small" />
            </Form.Item>

            <Form.Item label="色号(Pantone)" name="pantone_code" style={{ marginBottom: 8 }}>
              <Input placeholder="如: Black 6C" />
            </Form.Item>

            <Form.Item label="光泽度" name="gloss_level" style={{ marginBottom: 8 }}>
              <Select placeholder="选择光泽度" allowClear showSearch
                options={GLOSS_OPTIONS.map(o => ({ label: o, value: o }))} />
            </Form.Item>

            <Form.Item label="表面处理" name="finish" style={{ marginBottom: 8 }}>
              <Select placeholder="选择表面处理" allowClear showSearch
                options={FINISH_OPTIONS.map(o => ({ label: o, value: o }))} />
            </Form.Item>

            <Form.Item label="纹理" name="texture" style={{ marginBottom: 8 }}>
              <Select placeholder="选择纹理" allowClear showSearch
                options={TEXTURE_OPTIONS.map(o => ({ label: o, value: o }))} />
            </Form.Item>

            <Form.Item label="涂层类型" name="coating" style={{ marginBottom: 8 }}>
              <Select placeholder="选择涂层" allowClear showSearch
                options={COATING_OPTIONS.map(o => ({ label: o, value: o }))} />
            </Form.Item>
          </div>

          {/* 渲染效果图上传 */}
          <div style={{ marginBottom: 8 }}>
            <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>渲染效果图</Text>
            <Space size={8} align="start">
              {existingRenderUrl && (
                <div style={{ position: 'relative' }}>
                  <Image src={existingRenderUrl} width={60} height={45}
                    style={{ objectFit: 'cover', borderRadius: 4 }}
                    fallback="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNDUiPjxyZWN0IGZpbGw9IiNmNWY1ZjUiIHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiLz48L3N2Zz4=" />
                  <DeleteOutlined
                    style={{ position: 'absolute', top: -4, right: -4, color: '#ff4d4f', cursor: 'pointer', fontSize: 11, background: '#fff', borderRadius: '50%', padding: 2 }}
                    onClick={() => { setRenderImageId(''); setRenderImageUrlState(''); }}
                  />
                </div>
              )}
              <Upload showUploadList={false} accept="image/*" beforeUpload={(file) => { handleRenderUpload(file); return false; }}>
                <Button size="small" icon={<UploadOutlined />} loading={renderUploading}>上传效果图</Button>
              </Upload>
            </Space>
          </div>

          {/* 工艺图纸上传 */}
          <div style={{ marginBottom: 8 }}>
            <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>工艺图纸</Text>
            <Space size={8} style={{ marginBottom: 4 }}>
              <Select size="small" value={drawingType || undefined} placeholder="图纸类型"
                allowClear style={{ width: 120 }}
                onChange={(v) => setDrawingType(v || '')}
                options={DRAWING_TYPE_OPTIONS.map(o => ({ label: o, value: o }))} />
              <Upload showUploadList={false} accept=".pdf,.dwg,.dxf,.ai,.cdr,image/*"
                beforeUpload={(file) => { handleDrawingUpload(file); return false; }}>
                <Button size="small" icon={<UploadOutlined />} loading={drawingUploading}>上传图纸</Button>
              </Upload>
            </Space>
            {drawings.map(f => (
              <div key={f.file_id} style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: 12, marginBottom: 2 }}>
                <PaperClipOutlined style={{ color: '#1677ff' }} />
                <a href={f.url} target="_blank" rel="noopener noreferrer"
                  style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {f.file_name}
                </a>
                <DeleteOutlined style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 11 }}
                  onClick={() => setDrawings(prev => prev.filter(d => d.file_id !== f.file_id))} />
              </div>
            ))}
          </div>

          <Form.Item label="备注" name="notes" style={{ marginBottom: 0 }}>
            <Input.TextArea rows={1} placeholder="备注信息" />
          </Form.Item>
        </Form>
      </div>
    );
  }

  // ---- 展示态 ----
  const dispRenderUrl = variant.reference_image_file_id
    ? (variant.reference_image_url || `/uploads/${variant.reference_image_file_id}/image`)
    : undefined;
  const dispDrawings = parseDrawings(variant.process_drawings);

  return (
    <div style={{
      border: '1px solid #f0f0f0',
      borderRadius: 8,
      padding: '12px 16px',
      marginBottom: 8,
      background: '#fafafa',
      transition: 'all 0.2s',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Space size={6}>
          <Tag color="processing" style={{ margin: 0, fontSize: 11 }}>V{variant.variant_index}</Tag>
          {variant.material_code && <Text style={{ fontSize: 11, color: '#8c8c8c' }}>{variant.material_code}</Text>}
        </Space>
        <Space size={4}>
          <Tooltip title="编辑">
            <Button size="small" type="text" icon={<EditOutlined />} onClick={() => setEditing(true)} />
          </Tooltip>
          <Popconfirm title="确认删除此CMF方案？" onConfirm={() => deleteMutation.mutate()}>
            <Tooltip title="删除">
              <Button size="small" type="text" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '4px 16px' }}>
        <PropItem label="颜色:" value={variant.color_hex} suffix={<ColorSwatch hex={variant.color_hex} />} />
        <PropItem label="色号(Pantone):" value={variant.pantone_code} />
        <PropItem label="光泽度:" value={variant.gloss_level} />
        <PropItem label="表面处理:" value={variant.finish} />
        <PropItem label="纹理:" value={variant.texture} />
        <PropItem label="涂层类型:" value={variant.coating} />
      </div>

      {/* 渲染效果图 */}
      <div style={{ marginTop: 6, fontSize: 12 }}>
        <Text type="secondary">渲染效果图: </Text>
        {dispRenderUrl ? (
          <Image src={dispRenderUrl} width={60} height={45}
            style={{ objectFit: 'cover', borderRadius: 4, marginLeft: 4 }}
            fallback="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNDUiPjxyZWN0IGZpbGw9IiNmNWY1ZjUiIHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiLz48L3N2Zz4=" />
        ) : <Text style={{ fontSize: 12 }}>-</Text>}
      </div>

      {/* 工艺图纸 */}
      <div style={{ marginTop: 4, fontSize: 12 }}>
        <Text type="secondary">工艺图纸: </Text>
        {variant.process_drawing_type && <Tag style={{ fontSize: 11 }}>{variant.process_drawing_type}</Tag>}
        {dispDrawings.length > 0 ? dispDrawings.map(f => (
          <div key={f.file_id} style={{ display: 'inline-flex', alignItems: 'center', gap: 4, marginLeft: 4 }}>
            <PaperClipOutlined style={{ color: '#1677ff', fontSize: 11 }} />
            <a href={f.url} target="_blank" rel="noopener noreferrer" style={{ fontSize: 12 }}>{f.file_name}</a>
          </div>
        )) : !variant.process_drawing_type && <Text style={{ fontSize: 12 }}>-</Text>}
      </div>

      {/* 备注 */}
      <div style={{ marginTop: 4, fontSize: 12 }}>
        <Text type="secondary">备注: </Text><Text style={{ fontSize: 12 }}>{variant.notes || '-'}</Text>
      </div>
    </div>
  );
};

// ========== 主组件 ==========
const CMFVariantEditor: React.FC<CMFVariantEditorProps> = ({ projectId, mode = 'edit' }) => {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const isView = mode === 'view';

  const { data: parts = [], isLoading } = useQuery({
    queryKey: ['appearance-parts', projectId],
    queryFn: () => cmfVariantApi.getAppearanceParts(projectId),
  });

  const createMutation = useMutation({
    mutationFn: ({ itemId }: { itemId: string }) =>
      cmfVariantApi.createVariant(projectId, itemId, {}),
    onSuccess: () => {
      message.success('已添加CMF方案');
      queryClient.invalidateQueries({ queryKey: ['appearance-parts', projectId] });
    },
    onError: (err: any) => message.error(err?.response?.data?.message || '添加失败'),
  });

  if (isLoading) {
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>;
  }

  if (parts.length === 0) {
    return <Empty description="暂无外观件。请在EBOM中标记外观件 (is_appearance_part)" />;
  }

  const collapseItems = parts.map((part: AppearancePartWithCMF) => {
    const item = part.bom_item;
    const variants = part.cmf_variants || [];

    return {
      key: item.id,
      label: (
        <Space size={8}>
          {item.thumbnail_url && (
            <img src={item.thumbnail_url} alt="" width={28} height={21}
              style={{ objectFit: 'contain', borderRadius: 3, background: '#f5f5f5' }} />
          )}
          <Text strong style={{ fontSize: 13 }}>#{item.item_number} {item.name}</Text>
          {item.extended_attrs?.material_type && <Tag style={{ fontSize: 11 }}>{item.extended_attrs.material_type}</Tag>}
          <Tag color="blue" style={{ fontSize: 11 }}>{variants.length} 方案</Tag>
        </Space>
      ),
      children: (
        <div>
          {variants.map(v => (
            isView
              ? <VariantViewCard key={v.id} variant={v} />
              : <VariantEditCard key={v.id} variant={v} projectId={projectId} />
          ))}
          {!isView && (
            <Button type="dashed" block icon={<PlusOutlined />}
              onClick={() => createMutation.mutate({ itemId: item.id })}
              loading={createMutation.isPending}
              style={{ marginTop: 4, borderRadius: 8 }}
            >
              增加CMF方案
            </Button>
          )}
        </div>
      ),
    };
  });

  return (
    <div>
      <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
        <BgColorsOutlined style={{ color: '#1677ff' }} />
        <Text type="secondary" style={{ fontSize: 12 }}>
          为外观件定义CMF（颜色/材质/表面处理）方案。
        </Text>
      </div>
      <Collapse
        defaultActiveKey={parts.map((p: AppearancePartWithCMF) => p.bom_item.id)}
        items={collapseItems}
        style={{ background: '#fff' }}
      />
    </div>
  );
};

export default CMFVariantEditor;
