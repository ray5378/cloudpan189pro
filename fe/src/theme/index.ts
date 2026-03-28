import type { GlobalTheme, GlobalThemeOverrides } from 'naive-ui'
import { darkTheme } from 'naive-ui'

// 自定义浅色主题配置
export const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    // 主色调
    primaryColor: '#18a058',
    primaryColorHover: '#36ad6a',
    primaryColorPressed: '#0c7a43',
    primaryColorSuppl: '#36ad6a',

    // 信息色
    infoColor: '#2080f0',
    infoColorHover: '#4098fc',
    infoColorPressed: '#1060c9',
    infoColorSuppl: '#4098fc',

    // 成功色
    successColor: '#18a058',
    successColorHover: '#36ad6a',
    successColorPressed: '#0c7a43',
    successColorSuppl: '#36ad6a',

    // 警告色
    warningColor: '#f0a020',
    warningColorHover: '#fcb040',
    warningColorPressed: '#c97c10',
    warningColorSuppl: '#fcb040',

    // 错误色
    errorColor: '#d03050',
    errorColorHover: '#de576d',
    errorColorPressed: '#ab1f3f',
    errorColorSuppl: '#de576d',

    // 文字颜色
    textColorBase: '#000000',
    textColor1: 'rgba(0, 0, 0, 0.82)',
    textColor2: 'rgba(0, 0, 0, 0.68)',
    textColor3: 'rgba(0, 0, 0, 0.38)',

    // 背景颜色
    bodyColor: '#ffffff',
    cardColor: '#ffffff',
    modalColor: '#ffffff',
    popoverColor: '#ffffff',
    tableHeaderColor: '#fafafa',

    // 边框颜色
    borderColor: 'rgba(0, 0, 0, 0.12)',
    dividerColor: 'rgba(0, 0, 0, 0.09)',

    // 圆角
    borderRadius: '6px',
    borderRadiusSmall: '4px',

    // 字体
    fontFamily:
      '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif',
    fontSize: '14px',
    fontSizeMini: '12px',
    fontSizeTiny: '12px',
    fontSizeSmall: '14px',
    fontSizeMedium: '14px',
    fontSizeLarge: '15px',
    fontSizeHuge: '16px',

    // 阴影
    boxShadow1:
      '0 1px 2px -2px rgba(0, 0, 0, 0.08), 0 3px 6px 0 rgba(0, 0, 0, 0.06), 0 5px 12px 4px rgba(0, 0, 0, 0.04)',
    boxShadow2:
      '0 3px 6px -4px rgba(0, 0, 0, 0.12), 0 6px 16px 0 rgba(0, 0, 0, 0.08), 0 9px 28px 8px rgba(0, 0, 0, 0.05)',
    boxShadow3:
      '0 6px 16px -9px rgba(0, 0, 0, 0.08), 0 9px 28px 0 rgba(0, 0, 0, 0.05), 0 12px 48px 16px rgba(0, 0, 0, 0.03)',
  },

  // 按钮组件自定义
  Button: {
    textColor: '#ffffff',
    textColorHover: '#ffffff',
    textColorPressed: '#ffffff',
    textColorFocus: '#ffffff',
    textColorDisabled: 'rgba(255, 255, 255, 0.5)',
    color: '#18a058',
    colorHover: '#36ad6a',
    colorPressed: '#0c7a43',
    colorFocus: '#36ad6a',
    colorDisabled: 'rgba(24, 160, 88, 0.5)',
    rippleColor: '#18a058',
    borderRadius: '6px',
    fontWeight: '500',
  },

  // 输入框组件自定义
  Input: {
    borderRadius: '6px',
    border: '1px solid rgba(0, 0, 0, 0.12)',
    borderHover: '1px solid #18a058',
    borderFocus: '1px solid #18a058',
    boxShadowFocus: '0 0 0 2px rgba(24, 160, 88, 0.2)',
  },

  // 卡片组件自定义
  Card: {
    borderRadius: '8px',
    paddingMedium: '20px',
    paddingLarge: '24px',
    paddingHuge: '28px',
    boxShadow: '0 2px 8px 0 rgba(99, 110, 123, 0.08), 0 1px 3px 0 rgba(99, 110, 123, 0.12)',
  },

  // 表格组件自定义
  DataTable: {
    borderRadius: '8px',
    thColor: '#fafafa',
    thColorHover: '#f0f0f0',
    tdColor: '#ffffff',
    tdColorHover: '#fafafa',
    tdColorStriped: '#fafafa',
    borderColor: 'rgba(0, 0, 0, 0.12)',
  },

  // 菜单组件自定义
  Menu: {
    borderRadius: '6px',
    itemColorHover: 'rgba(24, 160, 88, 0.1)',
    itemColorActive: 'rgba(24, 160, 88, 0.15)',
    itemTextColorActive: '#18a058',
    itemIconColorActive: '#18a058',
    arrowColorActive: '#18a058',
  },

  // 标签页组件自定义
  Tabs: {
    tabTextColorActiveLine: '#18a058',
    tabTextColorHoverLine: '#36ad6a',
    lineColor: '#18a058',
    barColor: '#18a058',
  },

  // 消息组件自定义
  Message: {
    borderRadius: '8px',
    padding: '12px 16px',
  },

  // 通知组件自定义
  Notification: {
    borderRadius: '8px',
    padding: '16px 20px',
  },

  // 对话框组件自定义
  Dialog: {
    borderRadius: '12px',
    padding: '24px',
  },

  // 抽屉组件自定义
  Drawer: {
    borderRadius: '0',
    padding: '24px',
  },

  // 模态框组件自定义
  Modal: {
    borderRadius: '12px',
    padding: '24px',
  },
}

// 自定义深色主题配置
export const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    // 主色调
    primaryColor: '#63e2b7',
    primaryColorHover: '#7fe7c4',
    primaryColorPressed: '#5acea7',
    primaryColorSuppl: '#7fe7c4',

    // 信息色
    infoColor: '#70c0e8',
    infoColorHover: '#8acbec',
    infoColorPressed: '#66afd3',
    infoColorSuppl: '#8acbec',

    // 成功色
    successColor: '#63e2b7',
    successColorHover: '#7fe7c4',
    successColorPressed: '#5acea7',
    successColorSuppl: '#7fe7c4',

    // 警告色
    warningColor: '#f2c97d',
    warningColorHover: '#f5d599',
    warningColorPressed: '#e6ba64',
    warningColorSuppl: '#f5d599',

    // 错误色
    errorColor: '#e88080',
    errorColorHover: '#e98b8b',
    errorColorPressed: '#e57272',
    errorColorSuppl: '#e98b8b',

    // 文字颜色
    textColorBase: '#ffffff',
    textColor1: 'rgba(255, 255, 255, 0.9)',
    textColor2: 'rgba(255, 255, 255, 0.82)',
    textColor3: 'rgba(255, 255, 255, 0.52)',

    // 背景颜色
    bodyColor: '#101014',
    cardColor: '#18181c',
    modalColor: '#18181c',
    popoverColor: '#18181c',
    tableHeaderColor: '#101014',

    // 边框颜色
    borderColor: 'rgba(255, 255, 255, 0.24)',
    dividerColor: 'rgba(255, 255, 255, 0.09)',

    // 圆角
    borderRadius: '6px',
    borderRadiusSmall: '4px',

    // 字体
    fontFamily:
      '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif',
    fontSize: '14px',
    fontSizeMini: '12px',
    fontSizeTiny: '12px',
    fontSizeSmall: '14px',
    fontSizeMedium: '14px',
    fontSizeLarge: '15px',
    fontSizeHuge: '16px',

    // 阴影
    boxShadow1:
      '0 1px 2px -2px rgba(0, 0, 0, 0.24), 0 3px 6px 0 rgba(0, 0, 0, 0.18), 0 5px 12px 4px rgba(0, 0, 0, 0.12)',
    boxShadow2:
      '0 3px 6px -4px rgba(0, 0, 0, 0.48), 0 6px 16px 0 rgba(0, 0, 0, 0.32), 0 9px 28px 8px rgba(0, 0, 0, 0.2)',
    boxShadow3:
      '0 6px 16px -9px rgba(0, 0, 0, 0.32), 0 9px 28px 0 rgba(0, 0, 0, 0.2), 0 12px 48px 16px rgba(0, 0, 0, 0.12)',
  },

  // 按钮组件自定义
  Button: {
    textColor: '#101014',
    textColorHover: '#101014',
    textColorPressed: '#101014',
    textColorFocus: '#101014',
    textColorDisabled: 'rgba(16, 16, 20, 0.5)',
    color: '#63e2b7',
    colorHover: '#7fe7c4',
    colorPressed: '#5acea7',
    colorFocus: '#7fe7c4',
    colorDisabled: 'rgba(99, 226, 183, 0.5)',
    rippleColor: '#63e2b7',
    borderRadius: '6px',
    fontWeight: '500',
  },

  // 输入框组件自定义
  Input: {
    borderRadius: '6px',
    border: '1px solid rgba(255, 255, 255, 0.24)',
    borderHover: '1px solid #63e2b7',
    borderFocus: '1px solid #63e2b7',
    boxShadowFocus: '0 0 0 2px rgba(99, 226, 183, 0.2)',
  },

  // 卡片组件自定义
  Card: {
    borderRadius: '8px',
    paddingMedium: '20px',
    paddingLarge: '24px',
    paddingHuge: '28px',
    boxShadow: '0 4px 12px 0 rgba(0, 0, 0, 0.15), 0 2px 4px 0 rgba(0, 0, 0, 0.12)',
  },

  // 表格组件自定义
  DataTable: {
    borderRadius: '8px',
    thColor: '#101014',
    thColorHover: '#0f0f13',
    tdColor: '#18181c',
    tdColorHover: '#1a1a1e',
    tdColorStriped: '#1a1a1e',
    borderColor: 'rgba(255, 255, 255, 0.24)',
  },

  // 菜单组件自定义
  Menu: {
    borderRadius: '6px',
    itemColorHover: 'rgba(99, 226, 183, 0.1)',
    itemColorActive: 'rgba(99, 226, 183, 0.15)',
    itemTextColorActive: '#63e2b7',
    itemIconColorActive: '#63e2b7',
    arrowColorActive: '#63e2b7',
  },

  // 标签页组件自定义
  Tabs: {
    tabTextColorActiveLine: '#63e2b7',
    tabTextColorHoverLine: '#7fe7c4',
    lineColor: '#63e2b7',
    barColor: '#63e2b7',
  },

  // 消息组件自定义
  Message: {
    borderRadius: '8px',
    padding: '12px 16px',
  },

  // 通知组件自定义
  Notification: {
    borderRadius: '8px',
    padding: '16px 20px',
  },

  // 对话框组件自定义
  Dialog: {
    borderRadius: '12px',
    padding: '24px',
  },

  // 抽屉组件自定义
  Drawer: {
    borderRadius: '0',
    padding: '24px',
  },

  // 模态框组件自定义
  Modal: {
    borderRadius: '12px',
    padding: '24px',
  },
}

// 创建完整的主题配置
export const createTheme = (isDark: boolean): GlobalTheme | null => {
  return isDark ? darkTheme : null
}

export const createThemeOverrides = (isDark: boolean): GlobalThemeOverrides => {
  return isDark ? darkThemeOverrides : lightThemeOverrides
}
