import React from 'react';
import { Select, Avatar, Space } from 'antd';
import { UserOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { userApi, User } from '@/api/users';

interface UserSelectProps {
  value?: string | string[];
  onChange?: (value: string | string[]) => void;
  mode?: 'single' | 'multiple';
  placeholder?: string;
  style?: React.CSSProperties;
}

const UserSelect: React.FC<UserSelectProps> = ({
  value,
  onChange,
  mode = 'single',
  placeholder = '选择用户',
  style,
}) => {
  const { data: users = [], isLoading } = useQuery({
    queryKey: ['users'],
    queryFn: userApi.list,
    staleTime: 5 * 60 * 1000,
  });

  const options = users.map((user: User) => ({
    label: (
      <Space size={8}>
        <Avatar size="small" src={user.avatar_url} icon={<UserOutlined />}>
          {user.name?.[0]}
        </Avatar>
        <span>{user.name}</span>
        {user.email && (
          <span style={{ color: '#999', fontSize: 12 }}>{user.email}</span>
        )}
      </Space>
    ),
    value: user.id,
    searchLabel: `${user.name} ${user.email || ''}`,
  }));

  return (
    <Select
      showSearch
      allowClear
      loading={isLoading}
      placeholder={placeholder}
      style={style}
      value={value || undefined}
      onChange={(val) => {
        if (onChange) {
          onChange(val);
        }
      }}
      mode={mode === 'multiple' ? 'multiple' : undefined}
      filterOption={(input, option) => {
        const searchLabel = (option as any)?.searchLabel || '';
        return searchLabel.toLowerCase().includes(input.toLowerCase());
      }}
      options={options}
      notFoundContent={isLoading ? '加载中...' : '暂无用户'}
      optionLabelProp="label"
    />
  );
};

export default UserSelect;
