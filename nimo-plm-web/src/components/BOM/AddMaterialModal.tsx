import React, { useState, useCallback } from 'react';
import { Modal, Input, List, Tag, Typography, Button, Space, Empty, Spin, Form, Select, InputNumber } from 'antd';
import { SearchOutlined, PlusOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { projectBomApi } from '@/api/projectBom';
import { CATEGORY_LABELS, SUB_CATEGORY_LABELS } from './bomConstants';

const { Text } = Typography;

export interface AddMaterialModalProps {
  open: boolean;
  onClose: () => void;
  onSelect: (item: Record<string, any>) => void;  // select existing item
  onCreate: (item: Record<string, any>) => void;   // create new item
  onSkip: () => void;                               // add empty row
  category: string;
  subCategory: string;
}

const AddMaterialModal: React.FC<AddMaterialModalProps> = ({
  open,
  onClose,
  onSelect,
  onCreate,
  onSkip,
  category,
  subCategory,
}) => {
  const [search, setSearch] = useState('');
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [form] = Form.useForm();

  // Debounced search — query fires when search changes
  const { data: results = [], isFetching } = useQuery({
    queryKey: ['bom-item-search', search, category],
    queryFn: () => projectBomApi.searchItems({ q: search, category, limit: 20 }),
    enabled: open && search.length > 0,
    staleTime: 15_000,
  });

  const handleSearch = useCallback((value: string) => {
    setSearch(value);
    setShowCreateForm(false);
  }, []);

  const handleSelect = useCallback((item: Record<string, any>) => {
    onSelect({
      ...item,
      id: 'new-' + Date.now(), // new ID for the copy
      bom_id: undefined,       // clear original bom_id
    });
    setSearch('');
    setShowCreateForm(false);
  }, [onSelect]);

  const handleCreate = useCallback(() => {
    form.validateFields().then(values => {
      onCreate({
        id: 'new-' + Date.now(),
        category,
        sub_category: subCategory,
        quantity: 1,
        unit: 'pcs',
        extended_attrs: {},
        ...values,
      });
      form.resetFields();
      setSearch('');
      setShowCreateForm(false);
    });
  }, [form, onCreate, category, subCategory]);

  const handleSkip = useCallback(() => {
    onSkip();
    setSearch('');
    setShowCreateForm(false);
  }, [onSkip]);

  const handleClose = useCallback(() => {
    setSearch('');
    setShowCreateForm(false);
    onClose();
  }, [onClose]);

  const categoryLabel = CATEGORY_LABELS[category] || category;
  const subCategoryLabel = SUB_CATEGORY_LABELS[subCategory] || subCategory;

  return (
    <Modal
      title={`添加物料 - ${categoryLabel} / ${subCategoryLabel}`}
      open={open}
      onCancel={handleClose}
      footer={null}
      width={600}
      destroyOnClose
    >
      {/* Search bar */}
      <Input
        prefix={<SearchOutlined />}
        placeholder="输入名称、料号、MPN搜索已有物料..."
        value={search}
        onChange={e => handleSearch(e.target.value)}
        allowClear
        size="large"
        style={{ marginBottom: 16 }}
        autoFocus
      />

      {/* Search results */}
      {search.length > 0 && !showCreateForm && (
        <div style={{ marginBottom: 16 }}>
          {isFetching ? (
            <div style={{ textAlign: 'center', padding: 24 }}><Spin /></div>
          ) : results.length > 0 ? (
            <List
              dataSource={results}
              size="small"
              style={{ maxHeight: 300, overflow: 'auto' }}
              renderItem={(item: any) => (
                <List.Item
                  style={{ cursor: 'pointer', padding: '8px 12px' }}
                  onClick={() => handleSelect(item)}
                  actions={[
                    <Button type="link" size="small" key="add">
                      添加
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    title={
                      <Space size={4}>
                        <Text strong style={{ fontSize: 13 }}>{item.name}</Text>
                        {item.mpn && <Tag color="blue" style={{ fontSize: 11 }}>{item.mpn}</Tag>}
                      </Space>
                    }
                    description={
                      <Space size={8} style={{ fontSize: 11 }}>
                        {item.category && <Tag style={{ fontSize: 10 }}>{CATEGORY_LABELS[item.category] || item.category}</Tag>}
                        {item.supplier && <Text type="secondary">供应商: {item.supplier}</Text>}
                        {item.extended_attrs?.specification && (
                          <Text type="secondary" ellipsis style={{ maxWidth: 200 }}>
                            {item.extended_attrs.specification}
                          </Text>
                        )}
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          ) : (
            <Empty
              description="未找到匹配物料"
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              style={{ padding: '16px 0' }}
            />
          )}

          {/* Create new button */}
          <Button
            type="dashed"
            icon={<PlusOutlined />}
            onClick={() => setShowCreateForm(true)}
            block
            style={{ marginTop: 8 }}
          >
            创建新物料
          </Button>
        </div>
      )}

      {/* Create form */}
      {showCreateForm && (
        <div style={{ marginBottom: 16 }}>
          <Text strong style={{ display: 'block', marginBottom: 8 }}>新建物料</Text>
          <Form form={form} layout="vertical" size="small">
            <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
              <Input placeholder="物料名称" />
            </Form.Item>
            <Space style={{ width: '100%' }} size={8}>
              <Form.Item name="quantity" label="数量" initialValue={1} style={{ flex: 1 }}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item name="unit" label="单位" initialValue="pcs" style={{ flex: 1 }}>
                <Select options={[
                  { label: 'pcs', value: 'pcs' },
                  { label: 'set', value: 'set' },
                  { label: 'kg', value: 'kg' },
                  { label: 'm', value: 'm' },
                ]} />
              </Form.Item>
            </Space>
            {category === 'electronic' && (
              <Form.Item name="mpn" label="MPN (制造商料号)">
                <Input placeholder="如 GRM155R71C104KA88D" />
              </Form.Item>
            )}
            <Form.Item name="notes" label="备注">
              <Input.TextArea rows={2} />
            </Form.Item>
            <Space>
              <Button type="primary" onClick={handleCreate}>创建并添加</Button>
              <Button onClick={() => setShowCreateForm(false)}>返回搜索</Button>
            </Space>
          </Form>
        </div>
      )}

      {/* Bottom: skip link */}
      {!showCreateForm && search.length === 0 && (
        <div style={{ textAlign: 'center', padding: '24px 0' }}>
          <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
            输入关键字搜索已有物料，或直接创建新物料
          </Text>
          <Button type="dashed" icon={<PlusOutlined />} onClick={() => setShowCreateForm(true)}>
            创建新物料
          </Button>
        </div>
      )}

      <div style={{ textAlign: 'center', marginTop: 12, borderTop: '1px solid #f0f0f0', paddingTop: 12 }}>
        <Button type="link" size="small" onClick={handleSkip}>
          跳过，直接添加空行
        </Button>
      </div>
    </Modal>
  );
};

export default AddMaterialModal;
