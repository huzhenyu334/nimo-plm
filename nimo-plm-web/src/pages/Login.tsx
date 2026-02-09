import React, { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Button, Card, Typography, Space, message } from 'antd';
import { LoginOutlined } from '@ant-design/icons';
import { useAuth } from '@/contexts/AuthContext';

const { Title, Text } = Typography;

const Login: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { login, isAuthenticated, refreshUser } = useAuth();

  useEffect(() => {
    // 处理 OAuth 回调
    const accessToken = searchParams.get('access_token');
    const refreshToken = searchParams.get('refresh_token');

    if (accessToken && refreshToken) {
      localStorage.setItem('access_token', accessToken);
      localStorage.setItem('refresh_token', refreshToken);
      refreshUser().then(() => {
        message.success('登录成功');
        navigate('/dashboard', { replace: true });
      });
    }
  }, [searchParams, refreshUser, navigate]);

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard', { replace: true });
    }
  }, [isAuthenticated, navigate]);

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      }}
    >
      <Card
        style={{
          width: 400,
          textAlign: 'center',
          borderRadius: 8,
          boxShadow: '0 4px 20px rgba(0,0,0,0.15)',
        }}
      >
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div>
            <img
              src="/logo.svg"
              alt="nimo PLM"
              style={{ width: 80, height: 80, marginBottom: 16 }}
            />
            <Title level={2} style={{ margin: 0 }}>
              nimo PLM
            </Title>
            <Text type="secondary">产品生命周期管理系统</Text>
          </div>

          <Button
            type="primary"
            size="large"
            icon={<LoginOutlined />}
            onClick={login}
            style={{
              width: '100%',
              height: 48,
              fontSize: 16,
              background: '#3370ff',
              borderColor: '#3370ff',
            }}
          >
            使用飞书登录
          </Button>

          <Text type="secondary" style={{ fontSize: 12 }}>
            点击登录即表示您同意我们的服务条款和隐私政策
          </Text>
        </Space>
      </Card>
    </div>
  );
};

export default Login;
