/**
 * Agent Control Panel — Design Tokens (Dark Theme)
 *
 * 基于 Ant Design 5.x ConfigProvider theme 体系。
 * 所有页面通过 ConfigProvider 注入，禁止硬编码色值。
 */

/* ───────── 语义色 ───────── */
export const semanticColors = {
  /** agent 状态色 */
  status: {
    online:  '#52c41a', // 绿 — 在线
    offline: '#595959', // 灰 — 离线
    error:   '#ff4d4f', // 红 — 错误
    working: '#1677ff', // 蓝 — 工作中
  },

  /** 角色标签色 */
  role: {
    coo:       '#722ed1', // 紫
    pm:        '#1677ff', // 蓝
    ux:        '#13c2c2', // 青
    dev:       '#52c41a', // 绿
    assistant: '#faad14', // 黄
  },
} as const;

/* ───────── Ant Design ConfigProvider.theme ───────── */
export const acpTheme = {
  algorithm: 'darkAlgorithm' as const, // theme.darkAlgorithm

  token: {
    /* 色彩 */
    colorPrimary:   '#1677ff',
    colorSuccess:   '#52c41a',
    colorWarning:   '#faad14',
    colorError:     '#ff4d4f',
    colorInfo:      '#1677ff',

    colorBgBase:    '#141414',   // 页面底色
    colorBgLayout:  '#141414',
    colorBgContainer: '#1f1f1f', // 卡片/容器背景
    colorBgElevated:  '#262626', // 弹窗/下拉背景

    colorBorder:       '#303030',
    colorBorderSecondary: '#262626',

    colorText:           'rgba(255,255,255,0.88)',
    colorTextSecondary:  'rgba(255,255,255,0.65)',
    colorTextTertiary:   'rgba(255,255,255,0.45)',
    colorTextQuaternary: 'rgba(255,255,255,0.25)',

    /* 字体 */
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif",
    fontSize:   14,
    fontSizeSM: 12,
    fontSizeLG: 16,
    fontSizeXL: 20,

    /* 间距 */
    marginXS:  4,
    marginSM:  8,
    margin:    16,
    marginMD:  20,
    marginLG:  24,
    marginXL:  32,
    paddingXS: 4,
    paddingSM: 8,
    padding:   16,
    paddingMD: 20,
    paddingLG: 24,
    paddingXL: 32,

    /* 圆角 */
    borderRadius:   8,
    borderRadiusSM: 4,
    borderRadiusLG: 12,

    /* 阴影 — 深色主题弱化阴影，依靠边框分隔 */
    boxShadow:          '0 2px 8px rgba(0,0,0,0.32)',
    boxShadowSecondary: '0 1px 4px rgba(0,0,0,0.24)',

    /* 动效 */
    motionDurationFast: '0.1s',
    motionDurationMid:  '0.2s',
    motionDurationSlow: '0.3s',
  },

  components: {
    Layout: {
      headerBg:  '#1f1f1f',
      headerHeight: 56,
      siderBg:   '#1f1f1f',
      bodyBg:    '#141414',
    },
    Card: {
      colorBgContainer: '#1f1f1f',
      paddingLG: 20,
    },
    Table: {
      headerBg:    '#1f1f1f',
      rowHoverBg:  '#262626',
      borderColor: '#303030',
    },
    Tag: {
      borderRadiusSM: 4,
    },
    Button: {
      primaryShadow: 'none',
    },
    Input: {
      colorBgContainer: '#262626',
    },
    Select: {
      colorBgContainer: '#262626',
    },
  },
} as const;

/* ───────── 布局常量 ───────── */
export const layout = {
  headerHeight: 56,
  contentMaxWidth: 1440,
  contentPadding: 24,
  cardGap: 16,
  sidebarWidth: '40%',   // Agent详情页右侧栏
  mainWidth: '60%',       // Agent详情页左侧栏
} as const;

/* ───────── 刷新间隔 ───────── */
export const polling = {
  dashboardMs: 10_000, // Dashboard 10s
  detailMs:     5_000, // Agent详情 5s
} as const;
