import { useState, useEffect } from 'react';
import { Card, Steps, Form, Input, Select, Button, Space, App, Breadcrumb, Checkbox, Radio } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { ecnApi, ECN, CreateECNRequest } from '@/api/ecn';
import { projectApi } from '@/api/projects';
import apiClient from '@/api/client';

const changeTypeOptions = [
  { label: '设计变更', value: 'design' },
  { label: '物料变更', value: 'material' },
  { label: '工艺变更', value: 'process' },
  { label: '规格变更', value: 'spec' },
  { label: '文档变更', value: 'document' },
];

const urgencyOptions = [
  { label: '常规', value: 'medium' },
  { label: '紧急', value: 'high' },
  { label: '特急', value: 'critical' },
];

interface ProductOption {
  id: string;
  name: string;
}

interface UserOption {
  id: string;
  name: string;
}

const ECNForm = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [currentStep, setCurrentStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [products, setProducts] = useState<ProductOption[]>([]);
  const [users, setUsers] = useState<UserOption[]>([]);
  const [_ecn, setEcn] = useState<ECN | null>(null);

  const isEdit = !!id;

  useEffect(() => {
    fetchProducts();
    fetchUsers();
    if (isEdit) {
      fetchECN();
    }
  }, [id]);

  const fetchProducts = async () => {
    try {
      const response = await apiClient.get('/products', { params: { page_size: 100 } });
      const items = (response.data?.data as any)?.materials || (response.data?.data as any)?.items || [];
      // fallback: try projects
      if (items.length === 0) {
        const data = await projectApi.list();
        const pItems = (data as any)?.items || [];
        setProducts(pItems.map((p: any) => ({ id: p.product_id || p.id, name: p.product_name || p.name })));
        return;
      }
      setProducts(items.map((p: any) => ({ id: p.id, name: p.name })));
    } catch {
      try {
        const data = await projectApi.list();
        const pItems = (data as any)?.items || [];
        setProducts(pItems.map((p: any) => ({ id: p.product_id || p.id, name: p.product_name || p.name })));
      } catch { /* ignore */ }
    }
  };

  const fetchUsers = async () => {
    try {
      const response = await apiClient.get('/users');
      const items = (response.data?.data as any)?.items || [];
      setUsers(items.map((u: any) => ({ id: u.id, name: u.name })));
    } catch { /* ignore */ }
  };

  const fetchECN = async () => {
    if (!id) return;
    try {
      const data = await ecnApi.get(id);
      setEcn(data);
      form.setFieldsValue({
        title: data.title,
        product_id: data.product_id,
        change_type: data.change_type,
        urgency: data.urgency,
        reason: data.reason,
        description: data.description,
        technical_plan: data.technical_plan,
        impact_analysis: data.impact_analysis,
        approval_mode: data.approval_mode || 'serial',
        sop_assembly: data.sop_impact?.assembly,
        sop_testing: data.sop_impact?.testing,
        sop_packaging: data.sop_impact?.packaging,
      });
    } catch {
      message.error('加载ECN详情失败');
    }
  };

  const handleSaveDraft = async () => {
    setLoading(true);
    try {
      const values = form.getFieldsValue(true);
      const sopImpact: Record<string, any> = {};
      if (values.sop_assembly) sopImpact.assembly = true;
      if (values.sop_assembly_desc) sopImpact.assembly_desc = values.sop_assembly_desc;
      if (values.sop_testing) sopImpact.testing = true;
      if (values.sop_testing_desc) sopImpact.testing_desc = values.sop_testing_desc;
      if (values.sop_packaging) sopImpact.packaging = true;
      if (values.sop_packaging_desc) sopImpact.packaging_desc = values.sop_packaging_desc;

      if (isEdit && id) {
        await ecnApi.update(id, {
          title: values.title,
          change_type: values.change_type,
          urgency: values.urgency,
          reason: values.reason,
          description: values.description,
          technical_plan: values.technical_plan,
          impact_analysis: values.impact_analysis,
          approval_mode: values.approval_mode,
          sop_impact: Object.keys(sopImpact).length > 0 ? sopImpact : undefined,
        });
        message.success('保存成功');
        navigate(`/ecn/${id}`);
      } else {
        const req: CreateECNRequest = {
          title: values.title,
          product_id: values.product_id,
          change_type: values.change_type,
          urgency: values.urgency || 'medium',
          reason: values.reason,
          description: values.description,
          technical_plan: values.technical_plan,
          impact_analysis: values.impact_analysis,
          approval_mode: values.approval_mode || 'serial',
          sop_impact: Object.keys(sopImpact).length > 0 ? sopImpact : undefined,
          approver_ids: values.approver_ids,
        };
        const created = await ecnApi.create(req);
        message.success('ECN已创建');
        navigate(`/ecn/${created.id}`);
      }
    } catch (e: any) {
      message.error('保存失败：' + (e?.response?.data?.message || e.message));
    } finally {
      setLoading(false);
    }
  };

  const handleSubmitForApproval = async () => {
    setLoading(true);
    try {
      const values = form.getFieldsValue(true);
      let ecnId = id;

      // If new, create first
      if (!isEdit) {
        const sopImpact: Record<string, any> = {};
        if (values.sop_assembly) sopImpact.assembly = true;
        if (values.sop_testing) sopImpact.testing = true;
        if (values.sop_packaging) sopImpact.packaging = true;

        const req: CreateECNRequest = {
          title: values.title,
          product_id: values.product_id,
          change_type: values.change_type,
          urgency: values.urgency || 'medium',
          reason: values.reason,
          description: values.description,
          technical_plan: values.technical_plan,
          impact_analysis: values.impact_analysis,
          approval_mode: values.approval_mode || 'serial',
          sop_impact: Object.keys(sopImpact).length > 0 ? sopImpact : undefined,
          approver_ids: values.approver_ids,
        };
        const created = await ecnApi.create(req);
        ecnId = created.id;
      }

      if (ecnId) {
        await ecnApi.submit(ecnId);
        message.success('已提交审批');
        navigate(`/ecn/${ecnId}`);
      }
    } catch (e: any) {
      message.error('操作失败：' + (e?.response?.data?.message || e.message));
    } finally {
      setLoading(false);
    }
  };

  const stepItems = [
    { title: '基本信息' },
    { title: '影响范围' },
    { title: '技术方案' },
    { title: '审批配置' },
  ];

  const renderStep0 = () => (
    <>
      <Form.Item label="标题" name="title" rules={[{ required: true, message: '请输入标题' }]}>
        <Input placeholder="简要描述变更内容，限120字" maxLength={120} showCount />
      </Form.Item>
      <Form.Item label="关联产品" name="product_id" rules={[{ required: true, message: '请选择产品' }]}>
        <Select
          placeholder="选择关联产品"
          showSearch
          optionFilterProp="label"
          options={products.map(p => ({ label: p.name, value: p.id }))}
        />
      </Form.Item>
      <Form.Item label="变更类型" name="change_type" rules={[{ required: true, message: '请选择变更类型' }]}>
        <Radio.Group options={changeTypeOptions} />
      </Form.Item>
      <Form.Item label="紧急程度" name="urgency" initialValue="medium">
        <Radio.Group options={urgencyOptions} optionType="button" buttonStyle="solid" />
      </Form.Item>
      <Form.Item label="变更原因" name="reason" rules={[{ required: true, message: '请输入变更原因' }]}>
        <Input.TextArea rows={4} placeholder="说明发起变更的原因" />
      </Form.Item>
    </>
  );

  const renderStep1 = () => (
    <>
      <Form.Item label="变更描述" name="description">
        <Input.TextArea rows={4} placeholder="详细描述变更内容" />
      </Form.Item>
      <Form.Item label="影响分析" name="impact_analysis">
        <Input.TextArea rows={3} placeholder="对成本、交期、质量等方面的影响评估" />
      </Form.Item>
      <div style={{ marginBottom: 24 }}>
        <h4>SOP影响评估</h4>
        <Card size="small">
          <Form.Item name="sop_assembly" valuePropName="checked" style={{ marginBottom: 8 }}>
            <Checkbox>影响装配SOP</Checkbox>
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, curr) => prev.sop_assembly !== curr.sop_assembly}>
            {({ getFieldValue }) => getFieldValue('sop_assembly') ? (
              <Form.Item name="sop_assembly_desc" style={{ marginLeft: 24 }}>
                <Input.TextArea rows={2} placeholder="请描述对装配SOP的影响" />
              </Form.Item>
            ) : null}
          </Form.Item>
          <Form.Item name="sop_testing" valuePropName="checked" style={{ marginBottom: 8 }}>
            <Checkbox>影响测试SOP</Checkbox>
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, curr) => prev.sop_testing !== curr.sop_testing}>
            {({ getFieldValue }) => getFieldValue('sop_testing') ? (
              <Form.Item name="sop_testing_desc" style={{ marginLeft: 24 }}>
                <Input.TextArea rows={2} placeholder="请描述对测试SOP的影响" />
              </Form.Item>
            ) : null}
          </Form.Item>
          <Form.Item name="sop_packaging" valuePropName="checked" style={{ marginBottom: 8 }}>
            <Checkbox>影响包装SOP</Checkbox>
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, curr) => prev.sop_packaging !== curr.sop_packaging}>
            {({ getFieldValue }) => getFieldValue('sop_packaging') ? (
              <Form.Item name="sop_packaging_desc" style={{ marginLeft: 24 }}>
                <Input.TextArea rows={2} placeholder="请描述对包装SOP的影响" />
              </Form.Item>
            ) : null}
          </Form.Item>
        </Card>
      </div>
    </>
  );

  const renderStep2 = () => (
    <>
      <Form.Item label="技术方案" name="technical_plan">
        <Input.TextArea rows={6} placeholder="详细描述如何实施变更（技术方案、实施步骤等）" />
      </Form.Item>
    </>
  );

  const renderStep3 = () => (
    <>
      <Form.Item label="审批模式" name="approval_mode" initialValue="serial">
        <Radio.Group>
          <Radio value="serial">串行审批（依次审批）</Radio>
          <Radio value="parallel">并行会签（同时审批）</Radio>
        </Radio.Group>
      </Form.Item>
      <Form.Item label="审批人" name="approver_ids">
        <Select
          mode="multiple"
          placeholder="选择审批人"
          showSearch
          optionFilterProp="label"
          options={users.map(u => ({ label: u.name, value: u.id }))}
        />
      </Form.Item>
    </>
  );

  const stepContents = [renderStep0, renderStep1, renderStep2, renderStep3];

  return (
    <div>
      <Breadcrumb items={[
        { title: <a onClick={() => navigate('/ecn')}>ECN管理</a> },
        { title: isEdit ? '编辑变更请求' : '新建变更请求' },
      ]} style={{ marginBottom: 16 }} />

      <Card>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 24 }}>
          <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/ecn')} />
          <h2 style={{ margin: '0 0 0 8px' }}>{isEdit ? '编辑变更请求' : '新建变更请求'}</h2>
        </div>

        <Steps current={currentStep} items={stepItems} style={{ marginBottom: 32 }} />

        <Form form={form} layout="vertical" style={{ maxWidth: 700 }}>
          {stepContents[currentStep]()}
        </Form>

        {/* 底部操作栏 */}
        <div style={{ borderTop: '1px solid #f0f0f0', paddingTop: 16, marginTop: 24, display: 'flex', justifyContent: 'space-between' }}>
          <Space>
            {currentStep > 0 && (
              <Button onClick={() => setCurrentStep(s => s - 1)}>上一步</Button>
            )}
          </Space>
          <Space>
            <Button onClick={handleSaveDraft} loading={loading}>保存草稿</Button>
            {currentStep < stepItems.length - 1 ? (
              <Button type="primary" onClick={() => {
                if (currentStep === 0) {
                  form.validateFields(['title', 'product_id', 'change_type', 'reason']).then(() => {
                    setCurrentStep(s => s + 1);
                  }).catch(() => { /* validation failed */ });
                } else {
                  setCurrentStep(s => s + 1);
                }
              }}>
                下一步
              </Button>
            ) : (
              <Button type="primary" onClick={handleSubmitForApproval} loading={loading}>
                提交审批
              </Button>
            )}
          </Space>
        </div>
      </Card>
    </div>
  );
};

export default ECNForm;
