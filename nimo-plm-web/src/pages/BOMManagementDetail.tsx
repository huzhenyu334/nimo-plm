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
  editing: { color: 'blue', text: '编辑中' },
  ecn_pending: { color: 'orange', text: 'ECN审批中' },
};

// Common fields that live on the item itself (not in extended_attrs)
const COMMON_FIELDS = ['name', 'quantity', 'unit', 'supplier', 'unit_price', 'notes', 'item_number', 'category', 'sub_category', 'material_id', 'parent_item_id', 'level', 'is_alternative', 'thumbnail_url'];

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

  // ECN related state
  const [ecnModalOpen, setEcnModalOpen] = useState(false);
  const [ecnTitle, setEcnTitle] = useState('');
  const [hasChanges, setHasChanges] = useState(false);
  const [_draftData, setDraftData] = useState<any>(null);

  // Mobile form state
  const [editingItem, setEditingItem] = useState<Record<string, any> | null>(null);
  const [addingContext, setAddingContext] = useState<{ category: string; subCategory: string } | null>(null);

  // Local items state — the single source of truth for UI
  const [localItems, setLocalItems] = useState<Record<string, any>[]>([]);
  // Track server state for diffing
  const serverItemsRef = useRef<Record<string, any>[]>([]);
  // Debounce timer for auto-save
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Prevent sync during API operations
  const syncingRef = useRef(false);

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

  // Is editable? draft/rejected/editing BOMs
  const isEditable = bomDetail && (bomDetail.status === 'draft' || bomDetail.status === 'rejected' || bomDetail.status === 'editing');

  // Can start editing? released/frozen BOMs
  const canStartEditing = bomDetail && (bomDetail.status === 'released' || bomDetail.status === 'frozen');

  // Is in ECN flow?
  const isInECNFlow = bomDetail && bomDetail.status === 'editing';
  const isECNPending = bomDetail && bomDetail.status === 'ecn_pending';

  // Flatten server items (for initializing local state)
  const flattenItems = useCallback((items: any[]) =>
    (items || []).map(({ material, children, ...rest }: any) => ({
      ...rest,
      ...(rest.extended_attrs || {}),
      material_code: material?.code || '',
    })),
  []);

  // Sync server data → local state when bomDetail changes
  useEffect(() => {
    if (!bomDetail?.items) return;
    const flat = flattenItems(bomDetail.items);
    serverItemsRef.current = flat;
    setLocalItems(flat);
  }, [bomDetail, flattenItems]);

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

  // ECN: Start editing
  const startEditingMutation = useMutation({
    mutationFn: () => projectBomApi.startEditing(projectId!, selectedBomId!),
    onSuccess: () => {
      message.success('已进入编辑模式');
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      setHasChanges(false);
    },
    onError: () => message.error('操作失败'),
  });

  // ECN: Discard draft
  const discardDraftMutation = useMutation({
    mutationFn: () => projectBomApi.discardDraft(projectId!, selectedBomId!),
    onSuccess: () => {
      message.success('已撤销编辑');
      setHasChanges(false);
      setDraftData(null);
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
    },
    onError: () => message.error('操作失败'),
  });

  // ECN: Submit ECN
  const submitECNMutation = useMutation({
    mutationFn: (title: string) => projectBomApi.submitECN(projectId!, selectedBomId!, { title }),
    onSuccess: () => {
      message.success('ECN已提交，等待审批');
      setEcnModalOpen(false);
      setEcnTitle('');
      setHasChanges(false);
      queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      queryClient.invalidateQueries({ queryKey: ['project-boms', projectId] });
    },
    onError: () => message.error('提交失败'),
  });

  // ========== onChange-driven local state ==========
  // EBOMControl/PBOMControl/MBOMControl all call this with the full items array
  const handleBOMChange = useCallback((newItems: Record<string, any>[]) => {
    setLocalItems(newItems);
    if (isInECNFlow) setHasChanges(true);
  }, [isInECNFlow]);

  // ========== Auto-save: debounced diff & sync ==========
  useEffect(() => {
    if (!projectId || !selectedBomId || !isEditable) return;
    if (syncingRef.current) return;

    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);

    saveTimerRef.current = setTimeout(async () => {
      if (syncingRef.current) return;
      syncingRef.current = true;

      try {
        const server = serverItemsRef.current;
        const local = localItems;
        const serverIds = new Set(server.map(i => i.id));
        const localIds = new Set(local.map(i => i.id));

        // ECN flow: save entire draft
        if (isInECNFlow) {
          const changed = JSON.stringify(local) !== JSON.stringify(server);
          if (changed) {
            await projectBomApi.saveDraft(projectId, selectedBomId, { items: local as any });
            setDraftData({ items: local });
            serverItemsRef.current = local;
          }
          syncingRef.current = false;
          return;
        }

        // Normal flow: individual API calls

        // 1. Deleted items (in server but not in local)
        const deletedIds = [...serverIds].filter(id => !localIds.has(id));
        for (const id of deletedIds) {
          await projectBomApi.deleteItem(projectId, selectedBomId, id);
        }

        // 2. New items (id starts with 'new-' or not in server)
        const newItems = local.filter(i =>
          (typeof i.id === 'string' && i.id.startsWith('new-')) || !serverIds.has(i.id)
        );
        const idMapping: Record<string, string> = {};
        for (const item of newItems) {
          const created = await projectBomApi.addItem(projectId, selectedBomId, {
            category: item.category,
            sub_category: item.sub_category,
            name: item.name || '',
            quantity: item.quantity ?? 1,
            unit: item.unit || 'pcs',
            item_number: item.item_number,
          });
          idMapping[item.id] = created.id;
        }

        // 3. Updated items (exist in both, check for changes)
        const existingItems = local.filter(i =>
          !(typeof i.id === 'string' && i.id.startsWith('new-')) && serverIds.has(i.id)
        );
        for (const item of existingItems) {
          const serverItem = server.find(s => s.id === item.id);
          if (!serverItem) continue;

          // Build update payload with only changed fields
          const updateData: any = {};
          let hasUpdate = false;
          for (const f of COMMON_FIELDS) {
            if (JSON.stringify(item[f]) !== JSON.stringify(serverItem[f])) {
              updateData[f] = item[f] ?? null;
              hasUpdate = true;
            }
          }

          // Check extended_attrs fields (anything not in COMMON_FIELDS)
          const extChanged: Record<string, any> = {};
          let hasExtUpdate = false;
          const serverExt = serverItem.extended_attrs || {};
          const itemKeys = Object.keys(item).filter(k => !COMMON_FIELDS.includes(k) && !['id', 'bom_id', 'created_at', 'updated_at', 'extended_cost', 'material_code', 'material', 'children', 'extended_attrs'].includes(k));
          for (const k of itemKeys) {
            if (JSON.stringify(item[k]) !== JSON.stringify(serverItem[k]) &&
                JSON.stringify(item[k]) !== JSON.stringify(serverExt[k])) {
              extChanged[k] = item[k];
              hasExtUpdate = true;
            }
          }
          if (hasExtUpdate) {
            updateData.extended_attrs = extChanged;
            hasUpdate = true;
          }

          if (hasUpdate) {
            await projectBomApi.updateItem(projectId, selectedBomId, item.id, updateData);
          }
        }

        // Update server ref with new state (replace temp ids with real ids)
        const updatedLocal = local.map(i => {
          if (idMapping[i.id]) return { ...i, id: idMapping[i.id] };
          return i;
        });
        serverItemsRef.current = updatedLocal;
        // Also update localItems to replace temp ids
        if (Object.keys(idMapping).length > 0) {
          setLocalItems(updatedLocal);
        }

      } catch {
        message.error('保存失败');
        // Rollback: refetch from server
        queryClient.invalidateQueries({ queryKey: ['project-bom-detail', projectId, selectedBomId] });
      } finally {
        syncingRef.current = false;
      }
    }, 1500);

    return () => {
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    };
  }, [localItems, projectId, selectedBomId, isEditable, isInECNFlow, queryClient, message]);

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

      {/* ECN Action Buttons */}
      {bomDetail && (
        <div style={{ marginBottom: 16, display: 'flex', gap: 8, alignItems: 'center' }}>
          {canStartEditing && (
            <Button
              type="primary"
              onClick={() => startEditingMutation.mutate()}
              loading={startEditingMutation.isPending}
            >
              编辑
            </Button>
          )}

          {isInECNFlow && (
            <>
              <Button
                onClick={() => {
                  Modal.confirm({
                    title: '确认撤销编辑？',
                    content: '撤销后，所有未提交的修改将丢失',
                    onOk: () => discardDraftMutation.mutate(),
                  });
                }}
                loading={discardDraftMutation.isPending}
              >
                撤销编辑
              </Button>
              <Button
                type="primary"
                disabled={!hasChanges}
                onClick={() => setEcnModalOpen(true)}
              >
                提交ECN
              </Button>
            </>
          )}

          {isECNPending && (
            <Text type="warning" strong>⚠️ ECN变更审批中，BOM已锁定</Text>
          )}
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
              value={localItems}
              onChange={handleBOMChange}
              readonly={!isEditable}
              showMaterialCode
              editableCategories={isEditable ? permissions?.can_edit_categories : undefined}
              onItemClick={isMobile && isEditable ? handleMobileItemClick : undefined}
              onMobileAddRow={isMobile && isEditable ? handleMobileAddRow : undefined}
            />
          )}
          {activeTab === 'PBOM' && (
            <PBOMControl
              config={fullConfig}
              value={localItems}
              onChange={handleBOMChange}
              readonly={!isEditable}
              showMaterialCode
              editableCategories={isEditable ? permissions?.can_edit_categories : undefined}
              onItemClick={isMobile && isEditable ? handleMobileItemClick : undefined}
              onMobileAddRow={isMobile && isEditable ? handleMobileAddRow : undefined}
            />
          )}
          {activeTab === 'MBOM' && (
            <MBOMControl
              config={fullConfig}
              value={localItems}
              onChange={handleBOMChange}
              readonly={!isEditable}
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

      {/* ECN Submit Modal */}
      <Modal
        title="提交ECN变更申请"
        open={ecnModalOpen}
        onCancel={() => { setEcnModalOpen(false); setEcnTitle(''); }}
        onOk={() => submitECNMutation.mutate(ecnTitle)}
        confirmLoading={submitECNMutation.isPending}
        okText="确认提交"
      >
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary">
            提交后将创建ECN变更申请，BOM将进入审批状态，无法继续编辑。
          </Text>
        </div>
        <Input
          placeholder="请输入ECN标题（例如：更新物料规格）"
          value={ecnTitle}
          onChange={e => setEcnTitle(e.target.value)}
          maxLength={100}
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
