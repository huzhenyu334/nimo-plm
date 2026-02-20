/**
 * ACP — 登录页
 * 路由: /login
 *
 * 极简密码登录，深色主题，居中布局。
 * v1 仅 Admin 密码认证。
 */

import React, { useState } from 'react';
import {
  Layout, Card, Form, Input, Button, Typography, Space, Alert, theme,
} from 'antd';
import { LockOutlined, RobotOutlined, LoginOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

const { Content } = Layout;
const { Title, Text } = Typography;

const Login: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onFinish = async (values: { password: string }) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password: values.password }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.message ?? '认证失败');
      }
      const { token } = await res.json();
      localStorage.setItem('acp_token', token);
      navigate('/');
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Layout style={{ minHeight: '100vh', background: '#141414' }}>
      <Content
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
        }}
      >
        <Card
          bordered={false}
          style={{ width: 380, background: '#1f1f1f' }}
          styles={{ body: { padding: 32 } }}
        >
          {/* Logo + 标题 */}
          <div style={{ textAlign: 'center', marginBottom: 32 }}>
            <Space direction="vertical" size={8}>
              <RobotOutlined style={{ fontSize: 40, color: '#1677ff' }} />
              <Title level={3} style={{ margin: 0 }}>Agent Control Panel</Title>
              <Text type="secondary">BitFantasy AI 团队管理平台</Text>
            </Space>
          </div>

          {/* 错误提示 */}
          {error && (
            <Alert
              type="error"
              message={error}
              showIcon
              closable
              onClose={() => setError(null)}
              style={{ marginBottom: 16 }}
            />
          )}

          {/* 表单 */}
          <Form onFinish={onFinish} size="large" autoComplete="off">
            <Form.Item
              name="password"
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password
                prefix={<LockOutlined style={{ color: 'rgba(255,255,255,0.25)' }} />}
                placeholder="输入管理密码"
              />
            </Form.Item>

            <Form.Item style={{ marginBottom: 0 }}>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                icon={<LoginOutlined />}
                block
              >
                登录
              </Button>
            </Form.Item>
          </Form>

          <div style={{ textAlign: 'center', marginTop: 24 }}>
            <Text type="tertiary" style={{ fontSize: 12 }}>v1 · 单管理员模式</Text>
          </div>
        </Card>
      </Content>
    </Layout>
  );
};

export default Login;
