import React from 'react';
import { Card, Row, Col, Statistic, Table, Progress, Tag, Badge, App } from 'antd';
import {
  TeamOutlined,
  ShoppingCartOutlined,
  SafetyCertificateOutlined,
  DollarOutlined,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { srmApi, Supplier, PurchaseOrder, Inspection, Settlement } from '@/api/srm';
import dayjs from 'dayjs';

const poStatusColors: Record<string, string> = {
  draft: 'default',
  submitted: 'processing',
  approved: 'blue',
  ordered: 'cyan',
  shipped: 'orange',
  received: 'green',
  completed: 'green',
  cancelled: 'red',
};

export default function SupplierDashboard() {
  const { message } = App.useApp();

  // Fetch suppliers
  const { data: supplierData, isLoading: loadingSuppliers } = useQuery({
    queryKey: ['supplier-dashboard-suppliers'],
    queryFn: () => srmApi.listSuppliers({ page_size: 100 }),
  });

  // Fetch POs
  const { data: poData, isLoading: loadingPOs } = useQuery({
    queryKey: ['supplier-dashboard-pos'],
    queryFn: () => srmApi.listPOs({ page_size: 50 }),
  });

  // Fetch inspections
  const { data: inspectionData, isLoading: loadingInspections } = useQuery({
    queryKey: ['supplier-dashboard-inspections'],
    queryFn: () => srmApi.listInspections({ page_size: 100 }),
  });

  // Fetch settlements
  const { data: settlementData, isLoading: loadingSettlements } = useQuery({
    queryKey: ['supplier-dashboard-settlements'],
    queryFn: () => srmApi.listSettlements({ page_size: 100 }),
  });

  const suppliers = supplierData?.items || [];
  const pos = poData?.items || [];
  const inspections = inspectionData?.items || [];
  const settlements = settlementData?.items || [];

  // Compute stats
  const stats = React.useMemo(() => {
    const totalSuppliers = suppliers.length;

    const activePOs = pos.filter(
      (po: PurchaseOrder) => !['completed', 'cancelled', 'draft'].includes(po.status)
    ).length;

    const pendingInspections = inspections.filter(
      (insp: Inspection) => insp.status === 'pending' || insp.status === 'inspecting'
    ).length;

    const totalSettlement = settlements.reduce(
      (sum: number, s: Settlement) => sum + (s.final_amount || s.po_amount || 0),
      0
    );

    return { totalSuppliers, activePOs, pendingInspections, totalSettlement };
  }, [suppliers, pos, inspections, settlements]);

  // Inspection pass rate
  const inspectionPassRate = React.useMemo(() => {
    const completed = inspections.filter(
      (i: Inspection) => i.status === 'completed'
    );
    if (completed.length === 0) return 0;
    const passed = completed.filter(
      (i: Inspection) => i.overall_result === 'pass' || i.result === 'pass'
    ).length;
    return Math.round((passed / completed.length) * 100);
  }, [inspections]);

  // Top suppliers by order count
  const topSuppliers = React.useMemo(() => {
    const countMap: Record<string, { supplier: Supplier; count: number; totalAmount: number }> = {};
    pos.forEach((po: PurchaseOrder) => {
      if (!po.supplier_id) return;
      if (!countMap[po.supplier_id]) {
        const supplier = suppliers.find((s: Supplier) => s.id === po.supplier_id);
        countMap[po.supplier_id] = {
          supplier: supplier || { id: po.supplier_id, name: po.supplier?.name || po.supplier_id.slice(0, 8) } as Supplier,
          count: 0,
          totalAmount: 0,
        };
      }
      countMap[po.supplier_id].count++;
      countMap[po.supplier_id].totalAmount += po.total_amount || 0;
    });
    return Object.values(countMap)
      .sort((a, b) => b.count - a.count)
      .slice(0, 10);
  }, [pos, suppliers]);

  // Recent POs
  const recentPOs = React.useMemo(() => {
    return [...pos]
      .sort((a, b) => dayjs(b.created_at).valueOf() - dayjs(a.created_at).valueOf())
      .slice(0, 8);
  }, [pos]);

  const isLoading = loadingSuppliers || loadingPOs || loadingInspections || loadingSettlements;

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>供应商看板</h2>

      {/* Stat cards */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={12} sm={6}>
          <Card loading={isLoading}>
            <Statistic
              title="供应商总数"
              value={stats.totalSuppliers}
              prefix={<TeamOutlined style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card loading={isLoading}>
            <Statistic
              title="进行中订单"
              value={stats.activePOs}
              prefix={<ShoppingCartOutlined style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card loading={isLoading}>
            <Statistic
              title="待检验"
              value={stats.pendingInspections}
              valueStyle={stats.pendingInspections > 0 ? { color: '#faad14' } : undefined}
              prefix={<SafetyCertificateOutlined style={{ color: stats.pendingInspections > 0 ? '#faad14' : '#999' }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card loading={isLoading}>
            <Statistic
              title="结算总额"
              value={stats.totalSettlement}
              precision={2}
              prefix={<DollarOutlined style={{ color: '#722ed1' }} />}
              suffix="元"
            />
          </Card>
        </Col>
      </Row>

      {/* Inspection pass rate */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={8}>
          <Card title="检验通过率" loading={isLoading}>
            <div style={{ textAlign: 'center', padding: '16px 0' }}>
              <Progress
                type="circle"
                percent={inspectionPassRate}
                strokeColor={inspectionPassRate >= 80 ? '#52c41a' : inspectionPassRate >= 60 ? '#faad14' : '#ff4d4f'}
                size={120}
              />
              <div style={{ marginTop: 12, color: '#999', fontSize: 13 }}>
                共 {inspections.filter((i: Inspection) => i.status === 'completed').length} 次检验
              </div>
            </div>
          </Card>
        </Col>

        {/* Top suppliers */}
        <Col xs={24} sm={16}>
          <Card title="供应商订单排名" loading={isLoading}>
            <Table
              dataSource={topSuppliers}
              rowKey={(row) => row.supplier.id}
              pagination={false}
              size="small"
              columns={[
                {
                  title: '供应商',
                  dataIndex: ['supplier', 'name'],
                  key: 'name',
                  ellipsis: true,
                },
                {
                  title: '订单数',
                  dataIndex: 'count',
                  key: 'count',
                  width: 80,
                  sorter: (a, b) => a.count - b.count,
                },
                {
                  title: '订单金额',
                  dataIndex: 'totalAmount',
                  key: 'totalAmount',
                  width: 120,
                  render: (val: number) => val > 0 ? `¥${val.toLocaleString()}` : '-',
                },
              ]}
            />
          </Card>
        </Col>
      </Row>

      {/* Recent POs */}
      <Card title="最近采购订单" loading={isLoading}>
        <Table
          dataSource={recentPOs}
          rowKey="id"
          pagination={false}
          size="small"
          columns={[
            {
              title: '订单编号',
              dataIndex: 'po_code',
              key: 'po_code',
              width: 140,
            },
            {
              title: '供应商',
              key: 'supplier',
              ellipsis: true,
              render: (_: unknown, record: PurchaseOrder) => record.supplier?.name || '-',
            },
            {
              title: '金额',
              dataIndex: 'total_amount',
              key: 'total_amount',
              width: 120,
              render: (val: number | undefined) => val != null ? `¥${val.toLocaleString()}` : '-',
            },
            {
              title: '状态',
              dataIndex: 'status',
              key: 'status',
              width: 90,
              render: (status: string) => (
                <Badge status={poStatusColors[status] as any || 'default'} text={status} />
              ),
            },
            {
              title: '创建时间',
              dataIndex: 'created_at',
              key: 'created_at',
              width: 110,
              render: (val: string) => dayjs(val).format('MM-DD HH:mm'),
            },
          ]}
        />
      </Card>
    </div>
  );
}
