import React from 'react';
import {
  Collapse,
  Input,
  Select,
  Button,
  Space,
  Typography,
  Spin,
  Empty,
  Popconfirm,
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
  SaveOutlined,
  CloseOutlined,
  EditOutlined,
  UploadOutlined,
  PaperClipOutlined,
  ArrowLeftOutlined,
  RightOutlined,
} from '@ant-design/icons';
import { taskFormApi } from '@/api/taskForms';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { cmfVariantApi, type CMFVariant, type AppearancePartWithCMF } from '@/api/cmfVariant';
import { useIsMobile } from '@/hooks/useIsMobile';

const { Text } = Typography;

// ========== Option Constants (与CMFVariantEditor统一) ==========

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

// ========== Props ==========

interface CMFEditControlProps {
  projectId: string;
  taskId: string;
  readonly?: boolean;
}

// ========== 属性行组件（只读时使用） ==========
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

// ========== 单个变体卡片（查看模式） ==========
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

// ========== 单个变体卡片（编辑模式） ==========
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

// ========== 移动端: CMF变体全屏编辑面板 ==========
const CMFMobileEditPanel: React.FC<{
  variant: CMFVariant;
  projectId: string;
  readonly: boolean;
  onClose: () => void;
}> = ({ variant, projectId, readonly, onClose }) => {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [visible, setVisible] = React.useState(false);
  const [form] = Form.useForm();

  const [renderUploading, setRenderUploading] = React.useState(false);
  const [drawingUploading, setDrawingUploading] = React.useState(false);
  const [renderImageId, setRenderImageId] = React.useState(variant.reference_image_file_id || '');
  const [renderImageUrl, setRenderImageUrlState] = React.useState(variant.reference_image_url || '');
  const [drawingType, setDrawingType] = React.useState(variant.process_drawing_type || '');
  const [drawings, setDrawings] = React.useState<DrawingFile[]>(parseDrawings(variant.process_drawings));

  React.useEffect(() => {
    requestAnimationFrame(() => setVisible(true));
  }, []);

  const handleClose = () => {
    setVisible(false);
    setTimeout(onClose, 300);
  };

  const updateMutation = useMutation({
    mutationFn: (data: Partial<CMFVariant>) =>
      cmfVariantApi.updateVariant(projectId, variant.id, data),
    onSuccess: () => {
      message.success('已保存');
      queryClient.invalidateQueries({ queryKey: ['appearance-parts', projectId] });
      handleClose();
    },
    onError: (err: any) => message.error(err?.response?.data?.message || '保存失败'),
  });

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
            loading={updateMutation.isPending}
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
  variant: CMFVariant;
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
      {/* Color swatch / thumbnail */}
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

      {/* Info */}
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

      {/* Arrow */}
      <RightOutlined style={{ color: '#bfbfbf', fontSize: 12, flexShrink: 0 }} />
    </div>
  );
};

// ========== 主组件 ==========

const CMFEditControl: React.FC<CMFEditControlProps> = ({ projectId, taskId: _taskId, readonly = false }) => {
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();
  const [editingVariant, setEditingVariant] = React.useState<CMFVariant | null>(null);

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
    return <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载CMF数据..." /></div>;
  }

  if (parts.length === 0) {
    return <Empty description="未找到外观件。请先在EBOM中将零件标记为外观件。" />;
  }

  // ===== Mobile layout: compact cards =====
  if (isMobile) {
    return (
      <div>
        <div style={{ marginBottom: 10, display: 'flex', alignItems: 'center', gap: 6 }}>
          <BgColorsOutlined style={{ color: '#1677ff' }} />
          <Text type="secondary" style={{ fontSize: 12 }}>CMF方案</Text>
        </div>

        {parts.map((part: AppearancePartWithCMF) => {
          const item = part.bom_item;
          const variants = part.cmf_variants || [];

          return (
            <div key={item.id} style={{ marginBottom: 16 }}>
              {/* Part header */}
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

              {/* Variant summary cards */}
              {variants.map(v => (
                <MobileVariantSummaryCard
                  key={v.id}
                  variant={v}
                  onClick={() => setEditingVariant(v)}
                />
              ))}

              {/* Add button */}
              {!readonly && (
                <Button
                  type="dashed"
                  icon={<PlusOutlined />}
                  onClick={() => createMutation.mutate({ itemId: item.id })}
                  loading={createMutation.isPending}
                  block
                  style={{ borderRadius: 10, marginTop: 4 }}
                >
                  添加方案
                </Button>
              )}
            </div>
          );
        })}

        {/* Slide-in edit panel */}
        {editingVariant && (
          <CMFMobileEditPanel
            variant={editingVariant}
            projectId={projectId}
            readonly={readonly}
            onClose={() => setEditingVariant(null)}
          />
        )}
      </div>
    );
  }

  // ===== Desktop layout: existing Collapse =====
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
            readonly
              ? <VariantViewCard key={v.id} variant={v} />
              : <VariantEditCard key={v.id} variant={v} projectId={projectId} />
          ))}
          {!readonly && (
            <Button
              type="dashed"
              icon={<PlusOutlined />}
              onClick={() => createMutation.mutate({ itemId: item.id })}
              loading={createMutation.isPending}
              style={{ width: '100%', marginTop: 4, borderRadius: 8 }}
            >
              添加CMF方案
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

export default CMFEditControl;
