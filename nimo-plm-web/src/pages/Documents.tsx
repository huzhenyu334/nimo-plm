import React, { useState, useEffect } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Tag,
  Input,
  Select,
  Modal,
  Form,
  Upload,
  message,
  Descriptions,
  Drawer,
  List,
  Tooltip,
  Popconfirm,
} from 'antd';
import {
  ReloadOutlined,
  EyeOutlined,
  DeleteOutlined,
  DownloadOutlined,
  UploadOutlined,
  FileOutlined,
  FilePdfOutlined,
  FileWordOutlined,
  FileExcelOutlined,
  FileImageOutlined,
  FileZipOutlined,
  HistoryOutlined,
} from '@ant-design/icons';
import { documentsApi, Document, DocumentCategory, DocumentVersion } from '@/api';

const { Search } = Input;
const { Option } = Select;

const Documents: React.FC = () => {
  const [documents, setDocuments] = useState<Document[]>([]);
  const [categories, setCategories] = useState<DocumentCategory[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [versionsVisible, setVersionsVisible] = useState(false);
  const [currentDocument, setCurrentDocument] = useState<Document | null>(null);
  const [versions, setVersions] = useState<DocumentVersion[]>([]);
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [form] = Form.useForm();
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 });

  const fetchDocuments = async (page = 1) => {
    setLoading(true);
    try {
      const res = await documentsApi.list({
        category: selectedCategory || undefined,
        search: searchText || undefined,
        page,
        page_size: pagination.pageSize,
      });
      setDocuments(res.items || []);
      setPagination({ ...pagination, current: page, total: res.total });
    } catch (error) {
      console.error('获取文档列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchCategories = async () => {
    try {
      const res = await documentsApi.listCategories();
      setCategories(res.categories || []);
    } catch (error) {
      console.error('获取分类失败:', error);
    }
  };

  useEffect(() => {
    fetchDocuments();
    fetchCategories();
  }, [selectedCategory]);

  const handleSearch = () => {
    fetchDocuments(1);
  };

  const handleUpload = () => {
    form.resetFields();
    setUploadFile(null);
    setCurrentDocument(null);
    setModalVisible(true);
  };

  const handleView = (record: Document) => {
    setCurrentDocument(record);
    setDetailVisible(true);
  };

  const handleViewVersions = async (record: Document) => {
    try {
      const res = await documentsApi.listVersions(record.id);
      setVersions(res.versions || []);
      setCurrentDocument(record);
      setVersionsVisible(true);
    } catch (error) {
      message.error('获取版本列表失败');
    }
  };

  const handleDownload = async (id: string, filename: string) => {
    try {
      const blob = await documentsApi.download(id);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      message.error('下载失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await documentsApi.delete(id);
      message.success('删除成功');
      fetchDocuments();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleRelease = async (id: string) => {
    try {
      await documentsApi.release(id);
      message.success('发布成功');
      fetchDocuments();
    } catch (error) {
      message.error('发布失败');
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (!uploadFile) {
        message.warning('请选择文件');
        return;
      }
      await documentsApi.upload(uploadFile, {
        title: values.title,
        category_id: values.category_id,
        description: values.description,
      });
      message.success('上传成功');
      setModalVisible(false);
      fetchDocuments();
    } catch (error) {
      console.error('上传失败:', error);
      message.error('上传失败');
    }
  };

  const getFileIcon = (fileType: string) => {
    if (fileType?.includes('pdf')) return <FilePdfOutlined style={{ color: '#ff4d4f', fontSize: 20 }} />;
    if (fileType?.includes('word') || fileType?.includes('doc')) return <FileWordOutlined style={{ color: '#1890ff', fontSize: 20 }} />;
    if (fileType?.includes('excel') || fileType?.includes('sheet')) return <FileExcelOutlined style={{ color: '#52c41a', fontSize: 20 }} />;
    if (fileType?.includes('image')) return <FileImageOutlined style={{ color: '#faad14', fontSize: 20 }} />;
    if (fileType?.includes('zip') || fileType?.includes('rar')) return <FileZipOutlined style={{ color: '#722ed1', fontSize: 20 }} />;
    return <FileOutlined style={{ color: '#666', fontSize: 20 }} />;
  };

  const formatFileSize = (bytes: number) => {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getStatusTag = (status: string) => {
    const config: Record<string, { color: string; text: string }> = {
      draft: { color: 'default', text: '草稿' },
      released: { color: 'green', text: '已发布' },
      obsolete: { color: 'red', text: '已废弃' },
    };
    const { color, text } = config[status] || { color: 'default', text: status };
    return <Tag color={color}>{text}</Tag>;
  };

  const columns = [
    {
      title: '文件',
      key: 'file',
      width: 50,
      render: (_: any, record: Document) => getFileIcon(record.file_type),
    },
    {
      title: '文档编号',
      dataIndex: 'code',
      key: 'code',
      width: 130,
      render: (text: string) => <span style={{ fontFamily: 'monospace' }}>{text}</span>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 100,
      render: (cat: DocumentCategory) => cat?.name || '-',
    },
    {
      title: '版本',
      dataIndex: 'version',
      key: 'version',
      width: 70,
    },
    {
      title: '大小',
      dataIndex: 'file_size',
      key: 'file_size',
      width: 90,
      render: (size: number) => formatFileSize(size),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => getStatusTag(status),
    },
    {
      title: '上传人',
      dataIndex: 'uploader',
      key: 'uploader',
      width: 100,
      render: (user: any) => user?.name || '-',
    },
    {
      title: '上传时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (t: string) => new Date(t).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_: any, record: Document) => (
        <Space>
          <Tooltip title="查看">
            <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => handleView(record)} />
          </Tooltip>
          <Tooltip title="下载">
            <Button
              type="link"
              size="small"
              icon={<DownloadOutlined />}
              onClick={() => handleDownload(record.id, record.file_name)}
            />
          </Tooltip>
          <Tooltip title="版本">
            <Button type="link" size="small" icon={<HistoryOutlined />} onClick={() => handleViewVersions(record)} />
          </Tooltip>
          {record.status === 'draft' && (
            <Tooltip title="发布">
              <Popconfirm title="确认发布此文档？" onConfirm={() => handleRelease(record.id)}>
                <Button type="link" size="small" style={{ color: 'green' }}>
                  发布
                </Button>
              </Popconfirm>
            </Tooltip>
          )}
          <Tooltip title="删除">
            <Popconfirm title="确认删除此文档？" onConfirm={() => handleDelete(record.id)}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="文档管理"
        extra={
          <Space>
            <Select
              placeholder="选择分类"
              allowClear
              style={{ width: 150 }}
              value={selectedCategory || undefined}
              onChange={(v) => setSelectedCategory(v || '')}
            >
              {categories.map((cat) => (
                <Option key={cat.id} value={cat.id}>
                  {cat.name}
                </Option>
              ))}
            </Select>
            <Search
              placeholder="搜索文档"
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              onSearch={handleSearch}
              style={{ width: 200 }}
            />
            <Button icon={<ReloadOutlined />} onClick={() => fetchDocuments()}>
              刷新
            </Button>
            <Button type="primary" icon={<UploadOutlined />} onClick={handleUpload}>
              上传文档
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={documents}
          rowKey="id"
          loading={loading}
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => {
              setPagination({ ...pagination, current: page, pageSize });
              fetchDocuments(page);
            },
          }}
        />
      </Card>

      {/* 上传弹窗 */}
      <Modal
        title="上传文档"
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={500}
      >
        <Form form={form} layout="vertical">
          <Form.Item label="选择文件" required>
            <Upload
              beforeUpload={(file) => {
                setUploadFile(file);
                return false;
              }}
              maxCount={1}
              onRemove={() => setUploadFile(null)}
            >
              <Button icon={<UploadOutlined />}>选择文件</Button>
            </Upload>
            {uploadFile && <div style={{ marginTop: 8 }}>{uploadFile.name}</div>}
          </Form.Item>
          <Form.Item name="title" label="文档标题" rules={[{ required: true, message: '请输入标题' }]}>
            <Input placeholder="请输入文档标题" />
          </Form.Item>
          <Form.Item name="category_id" label="分类">
            <Select placeholder="选择分类" allowClear>
              {categories.map((cat) => (
                <Option key={cat.id} value={cat.id}>
                  {cat.name}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入描述" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title="文档详情"
        open={detailVisible}
        onClose={() => setDetailVisible(false)}
        width={500}
        extra={
          <Button
            type="primary"
            icon={<DownloadOutlined />}
            onClick={() => currentDocument && handleDownload(currentDocument.id, currentDocument.file_name)}
          >
            下载
          </Button>
        }
      >
        {currentDocument && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="文档编号">{currentDocument.code}</Descriptions.Item>
            <Descriptions.Item label="标题">{currentDocument.title}</Descriptions.Item>
            <Descriptions.Item label="分类">{currentDocument.category?.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="文件名">{currentDocument.file_name}</Descriptions.Item>
            <Descriptions.Item label="文件大小">{formatFileSize(currentDocument.file_size)}</Descriptions.Item>
            <Descriptions.Item label="版本">{currentDocument.version}</Descriptions.Item>
            <Descriptions.Item label="状态">{getStatusTag(currentDocument.status)}</Descriptions.Item>
            <Descriptions.Item label="描述">{currentDocument.description || '-'}</Descriptions.Item>
            <Descriptions.Item label="上传人">{currentDocument.uploader?.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="上传时间">
              {new Date(currentDocument.created_at).toLocaleString()}
            </Descriptions.Item>
            {currentDocument.released_at && (
              <>
                <Descriptions.Item label="发布人">{currentDocument.releaser?.name || '-'}</Descriptions.Item>
                <Descriptions.Item label="发布时间">
                  {new Date(currentDocument.released_at).toLocaleString()}
                </Descriptions.Item>
              </>
            )}
          </Descriptions>
        )}
      </Drawer>

      {/* 版本历史抽屉 */}
      <Drawer
        title={`版本历史 - ${currentDocument?.title || ''}`}
        open={versionsVisible}
        onClose={() => setVersionsVisible(false)}
        width={600}
      >
        <List
          dataSource={versions}
          renderItem={(item) => (
            <List.Item
              actions={[
                <Button
                  type="link"
                  icon={<DownloadOutlined />}
                  onClick={() => documentsApi.downloadVersion(currentDocument!.id, item.id).then((blob) => {
                    const url = window.URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = item.file_name;
                    a.click();
                    window.URL.revokeObjectURL(url);
                  })}
                >
                  下载
                </Button>,
              ]}
            >
              <List.Item.Meta
                avatar={getFileIcon(item.file_name)}
                title={`版本 ${item.version}`}
                description={
                  <div>
                    <div>{item.file_name} ({formatFileSize(item.file_size)})</div>
                    <div style={{ color: '#999' }}>
                      {item.creator?.name || '未知'} · {new Date(item.created_at).toLocaleString()}
                    </div>
                    {item.change_notes && <div style={{ marginTop: 4 }}>{item.change_notes}</div>}
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Drawer>
    </div>
  );
};

export default Documents;
