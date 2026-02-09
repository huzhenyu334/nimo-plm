import React, { useState, useEffect, useMemo } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  DatePicker,
  App,
  Typography,
  Badge,
  Spin,
  Popconfirm,
} from 'antd';
import {
  PlusOutlined,
  EyeOutlined,
  EditOutlined,
  RocketOutlined,
  SendOutlined,
  DeleteOutlined,
  RollbackOutlined,
} from '@ant-design/icons';
import { templateApi, ProjectTemplate } from '@/api/templates';
import { codenameApi, Codename } from '@/api/codenames';
import UserSelect from '@/components/UserSelect';
import { useAuth } from '@/contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import type { ColumnsType } from 'antd/es/table';

const { Title, Text } = Typography;

const phaseColors: Record<string, string> = {
  CONCEPT: 'purple',
  EVT: 'blue',
  DVT: 'cyan',
  PVT: 'orange',
  MP: 'green',
};

const Templates: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message, modal } = App.useApp();
  const [selectedTemplate, setSelectedTemplate] = useState<ProjectTemplate | null>(null);
  const [createProjectModalOpen, setCreateProjectModalOpen] = useState(false);
  const [createTemplateModalOpen, setCreateTemplateModalOpen] = useState(false);
  const [codenames, setCodenames] = useState<Codename[]>([]);
  const [codenamesLoading, setCodenamesLoading] = useState(false);

  const [form] = Form.useForm();
  const [createTemplateForm] = Form.useForm();

  // 获取流程列表
  const { data: templates, isLoading } = useQuery({
    queryKey: ['templates'],
    queryFn: () => templateApi.list(),
  });



  // 只显示每个模板族的最新版本
  const displayedTemplates = useMemo(() => {
    if (!templates) return [];

    // Group templates by family: use base_code to group versions together
    const familyMap = new Map<string, ProjectTemplate[]>();
    for (const t of templates) {
      const familyKey = t.base_code || t.code || t.id;
      if (!familyMap.has(familyKey)) {
        familyMap.set(familyKey, []);
      }
      familyMap.get(familyKey)!.push(t);
    }

    // Pick the latest version (highest version number) from each family
    const result: ProjectTemplate[] = [];
    for (const [, family] of familyMap) {
      const latest = family.reduce((best, current) => {
        const cv = parseFloat(String(current.version)) || 0;
        const bv = parseFloat(String(best.version)) || 0;
        return cv > bv ? current : best;
      });
      result.push(latest);
    }

    return result;
  }, [templates]);

  // Determine codename type from template's product_type
  const getCodenameType = (template: ProjectTemplate | null): string => {
    if (!template) return 'platform';
    const pt = template.product_type?.toLowerCase() || '';
    if (pt === 'platform') return 'platform';
    return 'product';
  };

  // 平台代号拼音缩写映射
  const pinyinMap: Record<string, string> = {
    '微光': 'WG', '晨曦': 'CX', '朝霞': 'ZX', '旭日': 'XR', '明辉': 'MH',
    '皓月': 'HY', '星河': 'XH', '天枢': 'TS', '瑶光': 'YG', '紫微': 'ZW',
    '青龙': 'QL', '朱雀': 'ZQ', '玄武': 'XW', '白虎': 'BH', '麒麟': 'QiL',
    '凤凰': 'FH', '鲲鹏': 'KP', '九天': 'JT', '太极': 'TJ', '鸿蒙': 'HM',
  };
  const getCodeAbbr = (codename: string, type: string) => {
    if (type === 'platform') return pinyinMap[codename] || codename;
    return codename;
  };

  // Fetch codenames when create project modal opens
  useEffect(() => {
    if (createProjectModalOpen && selectedTemplate) {
      const codenameType = getCodenameType(selectedTemplate);
      setCodenamesLoading(true);
      codenameApi
        .list(codenameType, true)
        .then((data) => {
          const sorted = [...data].sort((a, b) => (a.generation || 0) - (b.generation || 0));
          setCodenames(sorted);
          if (sorted.length > 0) {
            const first = sorted[0];
            const year = new Date().getFullYear();
            const abbr = getCodeAbbr(first.codename, codenameType);
            const prefix = codenameType === 'platform' ? 'PLT' : 'PRD';
            form.setFieldsValue({
              codename_id: first.id,
              project_name: first.codename,
              project_code: `${prefix}-${abbr}-${year}`,
            });
          }
        })
        .catch(() => {
          message.warning('获取代号列表失败');
          setCodenames([]);
        })
        .finally(() => setCodenamesLoading(false));
    }
  }, [createProjectModalOpen, selectedTemplate]);

  const handleCodenameChange = (codenameId: string) => {
    const selected = codenames.find((c) => c.id === codenameId);
    if (selected) {
      const codenameType = getCodenameType(selectedTemplate);
      const year = new Date().getFullYear();
      const abbr = getCodeAbbr(selected.codename, codenameType);
      const prefix = codenameType === 'platform' ? 'PLT' : 'PRD';
      form.setFieldsValue({
        project_name: selected.codename,
        project_code: `${prefix}-${abbr}-${year}`,
      });
    }
  };

  const formatCodenameLabel = (c: Codename): string => {
    const codenameType = getCodenameType(selectedTemplate);
    if (codenameType === 'platform') {
      return `Gen ${c.generation}: ${c.codename} — ${c.description || c.theme}`;
    }
    return `${c.codename} — ${c.theme}`;
  };

  const formatVersion = (record: ProjectTemplate): string => {
    if (record.version) {
      return `v${record.version}`;
    }
    return '-';
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-';
    const d = new Date(dateStr);
    return d.toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // 新建模板
  const handleCreateTemplate = async () => {
    try {
      const values = await createTemplateForm.validateFields();
      await templateApi.create({
        name: values.name,
        description: values.description,
        product_type: values.product_type,
      });
      message.success('模板创建成功');
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      setCreateTemplateModalOpen(false);
      createTemplateForm.resetFields();
    } catch (error: any) {
      if (error?.errorFields) return; // form validation error
      message.error(error?.response?.data?.message || '创建模板失败');
    }
  };

  // 删除模板
  const handleDelete = (record: ProjectTemplate) => {
    const isPublished = record.status === 'published';

    if (isPublished) {
      // 已发布模板需要二次确认
      modal.confirm({
        title: '删除已发布模板',
        content: (
          <div>
            <p style={{ color: '#ff4d4f', fontWeight: 'bold' }}>
              ⚠️ 警告：该模板已发布，可能已被项目引用！
            </p>
            <p>确定要删除模板「{record.name} v{record.version}」吗？此操作不可恢复。</p>
          </div>
        ),
        okText: '确定删除',
        okType: 'danger',
        cancelText: '取消',
        onOk: async () => {
          try {
            await templateApi.delete(record.id);
            message.success('删除成功');
            queryClient.invalidateQueries({ queryKey: ['templates'] });
          } catch (error: any) {
            message.error(error?.response?.data?.message || '删除失败');
          }
        },
      });
    } else {
      // 草稿模板直接用 Popconfirm（在列中处理）—— 这里作为备选的 modal 确认
      modal.confirm({
        title: '删除模板',
        content: `确定要删除模板「${record.name}」吗？`,
        okText: '确定',
        okType: 'danger',
        cancelText: '取消',
        onOk: async () => {
          try {
            await templateApi.delete(record.id);
            message.success('删除成功');
            queryClient.invalidateQueries({ queryKey: ['templates'] });
          } catch (error: any) {
            message.error(error?.response?.data?.message || '删除失败');
          }
        },
      });
    }
  };

  // Publish handler for list page
  const handlePublishFromList = (record: ProjectTemplate) => {
    modal.confirm({
      title: '发布流程',
      content: `发布后流程将被锁定，无法直接修改。如需修改请升级版本创建新的草稿。确定要发布 v${record.version || 1} 吗？`,
      okText: '确定发布',
      okType: 'primary',
      cancelText: '取消',
      onOk: async () => {
        try {
          await templateApi.publish(record.id);
          message.success('发布成功');
          queryClient.invalidateQueries({ queryKey: ['templates'] });
        } catch (error: any) {
          message.error(error?.response?.data?.message || '发布失败');
        }
      },
    });
  };

  const columns: ColumnsType<ProjectTemplate> = [
    {
      title: '流程编码',
      dataIndex: 'code',
      key: 'code',
      width: 150,
      render: (text: string) => <Text code>{text}</Text>,
    },
    {
      title: '流程名称',
      dataIndex: 'name',
      key: 'name',
      width: 250,
      render: (name: string, record: ProjectTemplate) => (
        <a onClick={() => navigate(`/templates/${record.id}`)}>
          {record.version ? `${name} v${record.version}` : name}
        </a>
      ),
    },
    {
      title: '流程版本',
      dataIndex: 'version',
      key: 'version',
      width: 100,
      render: (_: any, record: ProjectTemplate) => formatVersion(record),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        if (status === 'published') {
          return <Tag color="green">已发布</Tag>;
        }
        return <Tag color="orange">草稿</Tag>;
      },
    },
    {
      title: '产品类型',
      dataIndex: 'product_type',
      key: 'product_type',
      width: 120,
      render: (type: string) => type || '-',
    },
    {
      title: '阶段',
      dataIndex: 'phases',
      key: 'phases',
      width: 200,
      render: (phases: string[]) => (
        <Space size={4} wrap>
          {phases?.map((phase) => (
            <Tag key={phase} color={phaseColors[phase]}>
              {phase}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: '预估工期',
      dataIndex: 'estimated_days',
      key: 'estimated_days',
      width: 100,
      render: (days: number) => `${days}天`,
    },
    {
      title: '发布时间',
      dataIndex: 'published_at',
      key: 'published_at',
      width: 160,
      render: (publishedAt: string) => formatDate(publishedAt),
    },
    {
      title: '启用',
      dataIndex: 'is_active',
      key: 'is_active',
      width: 80,
      render: (active: boolean) => (
        <Badge status={active ? 'success' : 'default'} text={active ? '启用' : '禁用'} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 380,
      render: (_, record) => {
        const isPublished = record.status === 'published';
        const isDraft = !isPublished;
        const canRevert = isDraft && !!record.parent_template_id;

        return (
          <Space size="small">
            {isDraft && (
              <>
                <Button
                  type="link"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => navigate(`/templates/${record.id}`)}
                >
                  编辑
                </Button>
                <Button
                  type="link"
                  size="small"
                  icon={<SendOutlined />}
                  style={{ color: '#52c41a' }}
                  onClick={() => handlePublishFromList(record)}
                >
                  发布
                </Button>
              </>
            )}
            {isPublished && (
              <Button
                type="link"
                size="small"
                icon={<EyeOutlined />}
                onClick={() => navigate(`/templates/${record.id}`)}
              >
                查看
              </Button>
            )}
            <Button
              type="link"
              size="small"
              icon={<RocketOutlined />}
              disabled={isDraft}
              onClick={() => {
                setSelectedTemplate(record);
                setCreateProjectModalOpen(true);
              }}
            >
              创建项目
            </Button>
            {canRevert && (
              <Popconfirm
                title="撤销草稿"
                description="确定要撤销此草稿版本吗？撤销后将恢复到上一个已发布版本"
                onConfirm={async () => {
                  try {
                    await templateApi.revert(record.id);
                    message.success('已撤销草稿，恢复到上一个已发布版本');
                    queryClient.invalidateQueries({ queryKey: ['templates'] });
                  } catch (error: any) {
                    message.error(error?.response?.data?.message || '撤销失败');
                  }
                }}
                okText="确定撤销"
                cancelText="取消"
              >
                <Button
                  type="link"
                  size="small"
                  icon={<RollbackOutlined />}
                  style={{ color: '#faad14' }}
                >
                  撤销草稿
                </Button>
              </Popconfirm>
            )}
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record)}
            >
              删除
            </Button>
          </Space>
        );
      },
    },
  ];

  const handleCreateProject = async () => {
    try {
      const values = await form.validateFields();
      const project = await templateApi.createProjectFromTemplate({
        template_id: selectedTemplate!.id,
        project_name: values.project_name,
        project_code: values.project_code,
        start_date: values.start_date.format('YYYY-MM-DD'),
        pm_user_id: values.pm_user_id || user!.id,
        skip_weekends: values.skip_weekends,
        codename_id: values.codename_id,
      }) as { id: string };
      message.success('项目创建成功！');
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      setCreateProjectModalOpen(false);
      form.resetFields();
      navigate(`/projects/${project.id}`);
    } catch (error) {
      message.error('项目创建失败');
    }
  };

  const isPlatformType = getCodenameType(selectedTemplate) === 'platform';

  return (
    <div style={{ padding: 24 }}>
      <Card>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <Space size="middle" align="center">
            <Title level={4} style={{ margin: 0 }}>
              研发流程管理
            </Title>
          </Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateTemplateModalOpen(true)}
          >
            新建模板
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={displayedTemplates}
          rowKey="id"
          loading={isLoading}
          pagination={false}
          scroll={{ x: 1500 }}
        />
      </Card>

      {/* 新建模板弹窗 */}
      <Modal
        title="新建流程模板"
        open={createTemplateModalOpen}
        onCancel={() => {
          setCreateTemplateModalOpen(false);
          createTemplateForm.resetFields();
        }}
        onOk={handleCreateTemplate}
        okText="创建"
        cancelText="取消"
        width={500}
      >
        <Form form={createTemplateForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="name"
            label="模板名称"
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input placeholder="如：平台项目研发流程" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="模板描述（选填）" />
          </Form.Item>
          <Form.Item name="product_type" label="产品类型">
            <Select
              placeholder="选择产品类型（选填）"
              allowClear
              options={[
                { value: 'platform', label: '平台 (Platform)' },
                { value: 'product', label: '产品 (Product)' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 基于研发流程创建项目弹窗 */}
      <Modal
        title={`基于研发流程创建项目 - ${selectedTemplate?.name}${selectedTemplate?.version ? ` v${selectedTemplate.version}` : ''}`}
        open={createProjectModalOpen}
        onCancel={() => {
          setCreateProjectModalOpen(false);
          form.resetFields();
          setCodenames([]);
        }}
        onOk={handleCreateProject}
        okText="创建项目"
        width={600}
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="codename_id"
            label="项目代号"
            rules={[{ required: true, message: '请选择项目代号' }]}
          >
            <Select
              placeholder="选择项目代号"
              loading={codenamesLoading}
              onChange={handleCodenameChange}
              notFoundContent={codenamesLoading ? <Spin size="small" /> : '暂无可用代号'}
              options={codenames.map((c) => ({
                value: c.id,
                label: formatCodenameLabel(c),
              }))}
            />
          </Form.Item>
          <Form.Item
            name="project_code"
            label="项目编码"
            rules={[{ required: true, message: '请输入项目编码' }]}
          >
            <Input placeholder={isPlatformType ? '如：PLT-WG-2026' : '如：PRD-Nova-2026'} disabled />
          </Form.Item>
          <Form.Item name="project_name" label="项目名称（自动生成）">
            <Input disabled placeholder="选择代号后自动生成" />
          </Form.Item>
          <Form.Item
            name="start_date"
            label="计划开始日期"
            rules={[{ required: true, message: '请选择开始日期' }]}
          >
            <DatePicker style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            name="pm_user_id"
            label="项目经理"
            rules={[{ required: true, message: '请选择项目经理' }]}
            initialValue={user?.id}
          >
            <UserSelect
              mode="single"
              placeholder="选择项目经理"
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item name="skip_weekends" label="跳过周末" initialValue={true}>
            <Select
              options={[
                { value: true, label: '是 - 工期计算跳过周末' },
                { value: false, label: '否 - 包含周末' },
              ]}
            />
          </Form.Item>
        </Form>
        <div style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
          <Text type="secondary">
            将从研发流程「{selectedTemplate?.name}{selectedTemplate?.version ? ` v${selectedTemplate.version}` : ''}」复制 {selectedTemplate?.estimated_days} 天的任务计划
          </Text>
        </div>
      </Modal>
    </div>
  );
};

export default Templates;
