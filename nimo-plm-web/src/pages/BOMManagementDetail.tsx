import React, { useState, useCallback, useRef, useMemo, useEffect } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Button, Tabs, Select, Typography, Spin, Empty, message,
  Modal, Form, Input,
} from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { projectApi } from '@/api/projects';
import { projectBomApi } from '@/api/projectBom';
import { EBOMControl, PBOMControl, MBOMControl, type BOMControlConfig } from '@/components/BOM';
import BOMItemMobileForm from '@/components/BOM/BOMItemMobileForm';
import { useIsMobile } from '@/hooks/useIsMobile';

const { Text, Title } = Typography;

const BOM_STATUS_CONFIG: Record<string, { color: string; text: string }> = {
  draft: { color: 'default', text: '草稿' },
  pending_review: { color: 'processing', text: '审批中' },
  published: { color: 'green', text: '已发布' },
  released: { color: 'green', text: '已发布' },
  rejected: { color: 'red', text: '已驳回' },
  frozen: { color: 'purple', text: '已冻结' },
  obsolete: { color: 'default', text: '已废弃' },
};

const BOMManagementDetail: React.FC = () => {
  const { projectId } = useParams<{ projectId: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();

  const [activeTab, setActiveTab] = useState<string>(searchParams.get('type') || 'EBOM');
  const [selectedBomId, setSelectedBomId] = useState<string | null>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [releaseModalOpen, setReleaseModalOpen] = useState(false);
  const [releaseNote, setReleaseNote] = useState('');
  const [rejectModalOpen, setRejectModalOpen] = useState(false);
  const [rejectComment, setRejectComment] = useState('');
  const [form] = Form.useForm();

  // Mobile form state
  const [editingItem, setEditingItem] = useState<Record<string, any> | null>(null);
  const [addingContext, setAddingContext] = useState<{ category: string; subCategory: string } | null>(null);

  // Debounce timer for auto-save
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Mobile swipe gesture for tab switching
  const tabOrder = useMemo(() => ['EBOM', 'PBOM', 'MBOM'], []);
  const touchStartRef = useRef<{ x: number; y: number } | null>(null);

  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartRef.current = { x: e.touches[0].clientX, y: e.touches[0].clientY };
  }, []);

  const handleTouchEnd = useCallback((e: React.TouchEvent) => {
    if (!touchStartRef.current) return;
    const dx = e.changedTouches[0].clientX - touchStartRef.current.x;
    const dy = e.changedTouches[0].clientY - touchStartRef.current.y;
    if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy)) {
      const idx = tabOrder.indexOf(activeTab);
      if (dx < 0 && idx < tabOrder.length - 1) setActiveTab(tabOrder[idx + 1]);
      if (dx > 0 && idx > 0) setActiveTab(tabOrder[idx - 1]);
    }
    touchStartRef.current = null;
  }, [activeTab, tabOrder]);

  // Fetch project info
  const { data: project } = useQuery({
    queryKey: ['project', projectId],
    queryFn: () => projectApi.get(projectId!),
    enabled: !!projectId,
  });

  // Fetch BOM permissions
  const { data: permissions } = useQuery({
    queryKey: ['bom-permissions', projectId],
    queryFn: () => projectBomApi.getBOMPermissions(projectId!),
    enabled: !!projectId,
  });

  // Fetch BOM list
  const { data: bomList = [], isLoading: listLoading } = useQuery({
    queryKey: ['project-boms', projectId],
    queryFn: () => projectBomApi.list(projectId!),
    enabled: !!projectId,
  });

  // Filter by active tab
  const filteredBomList = useMemo(() =>
    bomList.filter(b => b.bom_type === activeTab && b.status !== 'obsolete'),
    [bomList, activeTab]
  );

  // Auto-select first BOM when list changes
  useEffect(() => {
    if (filteredBomList.length > 0 && !filteredBomList.find(b => b.id === selectedBomId)) {
      setSelectedBomId(filteredBomList[0].id);
    } else if (filteredBomList.length === 0) {
      setSelectedBomId(null);
    }
  }, [filteredBomList, selectedBomId]);

  // Fetch BOM detail
  const { data: bomDetail, isLoading: detailLoading } = useQuery({
    queryKey: ['project-bom-detail', projectId, selectedBomId],
    queryFn: () => projectBomApi.get(projectId!, selectedBomId!),
    enabled: !!projectId && !!selectedBomId,
  });

  // Is editable? Only draft/rejected BOMs
  const isEditable = bomDetail && (bomDetail.status === 'draft' || bomDetail.status === 'rejected');

  // Flatten items
  const items = useMemo(() =>
    (bomDetail?.items || []).map(({ material, children, ...rest }: any) => ({
      ...rest,
      ...(rest.extended_attrs || {}),
      material_code: material?.code || '',
    })),
    [bomDetail]
  );

  // Create BOM
  const createMutation = useMutation({
    mutationFn: (data: { bom_type: string; name: string; description?: string }) =>
      projectBomApi.create(projectId!, data as any),
    onSuccess: (newBom) => {
      message.success('BOM创建成功');
      setCreateModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
      setSelectedBomId(newBom.id);
    },
    onError: () => message.error('创建失败'),
  });


  // Reject
  const rejectMutation = useMutation({
    mutationFn: (comment: string) => projectBomApi.reject(projectId!, selectedBomId!, comment),
    onSuccess: () => {
      message.success('已驳回');
      setRejectModalOpen(false);
      setRejectComment('');
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
    },
    onError: () => message.error('驳回失败'),
  });

  // Release
  const releaseMutation = useMutation({
    mutationFn: (note: string) => projectBomApi.release(projectId!, selectedBomId!, note),
    onSuccess: () => {
      message.success('发布成功');
      setReleaseModalOpen(false);
      setReleaseNote('');
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('发布失败'),
  });

  // Auto-save: debounced item save
  const handleItemSave = useCallback((itemId: string, field: string, value: any) => {
    if (!selectedBomId || !projectId) return;

    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);

    saveTimerRef.current = setTimeout(async () => {
      try {
        const updateData: any = {};
        // Check if this is an extended attr
        const commonFields = ['name', 'quantity', 'unit', 'supplier', 'unit_price', 'notes', 'item_number'];
        if (commonFields.includes(field)) {
          updateData[field] = value;
        } else {
          updateData.extended_attrs = { [field]: value };
        }
        await projectBomApi.updateItem(projectId, selectedBomId, itemId, updateData);
        queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      } catch {
        message.error('保存失败');
      }
    }, 1000);
  }, [selectedBomId, projectId, queryClient]);

  // Handle BOM items change (for new items, the onChange from BOM controls)
  const handleBOMChange = useCallback((_newItems: Record<string, any>[]) => {
    // Individual item saves are handled via onItemSave, so this is a no-op for now
    // Bulk changes (like from add row) are handled through the control's add row mechanism
  }, []);

  // Mobile: item card click → open edit form
  const handleMobileItemClick = useCallback((item: Record<string, any>) => {
    if (!isMobile || !isEditable) return;
    setEditingItem(item);
  }, [isMobile, isEditable]);

  // Mobile: add button click → open add form
  const handleMobileAddRow = useCallback((category: string, subCategory: string) => {
    if (!isMobile) return;
    setAddingContext({ category, subCategory });
  }, [isMobile]);

  // Mobile form save
  const handleMobileFormSave = useCallback(async (formData: Record<string, any>) => {
    if (!projectId || !selectedBomId) return;
    try {
      if (formData.id) {
        // Edit existing item
        const { id, category, sub_category, ...updateData } = formData;
        await projectBomApi.updateItem(projectId, selectedBomId, id, updateData as any);
        message.success('保存成功');
      } else {
        // Add new item
        await projectBomApi.addItem(projectId, selectedBomId, formData as any);
        message.success('添加成功');
      }
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
    } catch {
      message.error('保存失败');
    }
    setEditingItem(null);
    setAddingContext(null);
  }, [projectId, selectedBomId, queryClient]);

  // Full config for display
  const fullConfig: BOMControlConfig = useMemo(() => ({
    bom_type: activeTab as 'EBOM' | 'PBOM' | 'MBOM',
    visible_categories: [],
    category_config: {},
  }), [activeTab]);

  if (!projectId) return null;

  return (
    <div style={{ padding: isMobile ? 12 : 24 }}>
      {/* Header */}
      <div style={{ marginBottom: isMobile ? 12 : 16 }}>
        <Button
          type="link"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate('/bom-management')}
          style={{ padding: 0, marginBottom: 8 }}
        >
          返回项目列表
        </Button>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>
            {project?.name || '加载中...'}
          </Title>
          {project?.code && <Text code style={{ fontSize: 12 }}>{project.code}</Text>}
        </div>
      </div>

      {/* BOM Type Tabs */}
      <Tabs
        activeKey={activeTab}
        onChange={key => setActiveTab(key)}
        size={isMobile ? 'small' : undefined}
        items={[
          { key: 'EBOM', label: isMobile ? 'EBOM' : 'EBOM 工程BOM' },
          { key: 'PBOM', label: isMobile ? 'PBOM' : 'PBOM 生产BOM' },
          { key: 'MBOM', label: isMobile ? 'MBOM' : 'MBOM 制造BOM' },
        ]}
        style={{ marginBottom: 8 }}
      />

      {/* Version selector */}
      {filteredBomList.length > 1 && (
        <div style={{ marginBottom: 12 }}>
          <Select
            value={selectedBomId || undefined}
            onChange={setSelectedBomId}
            style={{ width: isMobile ? '100%' : 300 }}
            placeholder="选择BOM版本"
            loading={listLoading}
            options={filteredBomList.map(b => ({
              label: `${b.bom_type} ${b.version || '草稿'} - ${BOM_STATUS_CONFIG[b.status]?.text || b.status}`,
              value: b.id,
            }))}
          />
        </div>
      )}

      {/* Loading */}
      {(listLoading || detailLoading) && !bomDetail && (
        <div style={{ textAlign: 'center', padding: 60 }}><Spin /></div>
      )}

      {/* Empty state */}
      {!listLoading && filteredBomList.length === 0 && (
        <Empty description={`暂无${activeTab}`} style={{ padding: 60 }} />
      )}

      {/* BOM Content */}
      {bomDetail && (
        <div
          onTouchStart={isMobile ? handleTouchStart : undefined}
          onTouchEnd={isMobile ? handleTouchEnd : undefined}
        >
          {activeTab === 'EBOM' && (
            <EBOMControl
              config={fullConfig}
              value={items}
              onChange={handleBOMChange}
              readonly={!isEditable}
              onItemSave={isEditable ? handleItemSave : undefined}
              showMaterialCode
              editableCategories={isEditable ? permissions?.can_edit_categories : undefined}
              onItemClick={isMobile && isEditable ? handleMobileItemClick : undefined}
              onMobileAddRow={isMobile && isEditable ? handleMobileAddRow : undefined}
            />
          )}
          {activeTab === 'PBOM' && (
            <PBOMControl
              config={fullConfig}
              value={items}
              onChange={handleBOMChange}
              readonly={!isEditable}
              onItemSave={isEditable ? handleItemSave : undefined}
              showMaterialCode
              editableCategories={isEditable ? permissions?.can_edit_categories : undefined}
              onItemClick={isMobile && isEditable ? handleMobileItemClick : undefined}
              onMobileAddRow={isMobile && isEditable ? handleMobileAddRow : undefined}
            />
          )}
          {activeTab === 'MBOM' && (
            <MBOMControl
              config={fullConfig}
              value={items}
              onChange={handleBOMChange}
              readonly={!isEditable}
              onItemSave={isEditable ? handleItemSave : undefined}
              showMaterialCode
              editableCategories={isEditable ? permissions?.can_edit_categories : undefined}
              onItemClick={isMobile ? handleMobileItemClick : undefined}
            />
          )}
        </div>
      )}

      {/* Create BOM Modal */}
      <Modal
        title="新建BOM"
        open={createModalOpen}
        onCancel={() => { setCreateModalOpen(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(values) => createMutation.mutate({ ...values, name: values.bom_type })}>
          <Form.Item name="bom_type" label="BOM类型" initialValue={activeTab} rules={[{ required: true }]}>
            <Select options={[
              { label: 'EBOM - 工程BOM', value: 'EBOM' },
              { label: 'PBOM - 生产BOM', value: 'PBOM' },
              { label: 'MBOM - 制造BOM', value: 'MBOM' },
            ]} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="BOM描述（可选）" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Release Modal */}
      <Modal
        title={`发布 ${bomDetail?.bom_type || ''}`}
        open={releaseModalOpen}
        onCancel={() => { setReleaseModalOpen(false); setReleaseNote(''); }}
        onOk={() => releaseMutation.mutate(releaseNote)}
        confirmLoading={releaseMutation.isPending}
        okText="确认发布"
      >
        <Text type="secondary">发布后BOM将不可编辑，系统会自动分配版本号。</Text>
        <Input.TextArea
          rows={3}
          placeholder="请输入发布说明..."
          value={releaseNote}
          onChange={e => setReleaseNote(e.target.value)}
          style={{ marginTop: 12 }}
        />
      </Modal>

      {/* Reject Modal */}
      <Modal
        title="驳回BOM"
        open={rejectModalOpen}
        onCancel={() => { setRejectModalOpen(false); setRejectComment(''); }}
        onOk={() => rejectMutation.mutate(rejectComment)}
        confirmLoading={rejectMutation.isPending}
        okText="确认驳回"
        okButtonProps={{ danger: true }}
      >
        <Input.TextArea
          rows={4}
          placeholder="请输入驳回原因..."
          value={rejectComment}
          onChange={e => setRejectComment(e.target.value)}
        />
      </Modal>

      {/* Mobile: Edit item form */}
      {isMobile && editingItem && (
        <BOMItemMobileForm
          item={editingItem}
          category={editingItem.category || ''}
          subCategory={editingItem.sub_category || ''}
          config={fullConfig}
          bomType={activeTab}
          onSave={handleMobileFormSave}
          onClose={() => setEditingItem(null)}
        />
      )}

      {/* Mobile: Add item form */}
      {isMobile && addingContext && (
        <BOMItemMobileForm
          category={addingContext.category}
          subCategory={addingContext.subCategory}
          config={fullConfig}
          bomType={activeTab}
          onSave={handleMobileFormSave}
          onClose={() => setAddingContext(null)}
        />
      )}
    </div>
  );
};

export default BOMManagementDetail;
