import type { GlobalThemeOverrides } from 'naive-ui'

// 蓝色主题配置示例
export const blueThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#1890ff',
    primaryColorHover: '#40a9ff',
    primaryColorPressed: '#096dd9',
    primaryColorSuppl: '#40a9ff',

    infoColor: '#1890ff',
    infoColorHover: '#40a9ff',
    infoColorPressed: '#096dd9',
    infoColorSuppl: '#40a9ff',

    successColor: '#52c41a',
    successColorHover: '#73d13d',
    successColorPressed: '#389e0d',
    successColorSuppl: '#73d13d',

    warningColor: '#faad14',
    warningColorHover: '#ffc53d',
    warningColorPressed: '#d48806',
    warningColorSuppl: '#ffc53d',

    errorColor: '#ff4d4f',
    errorColorHover: '#ff7875',
    errorColorPressed: '#d9363e',
    errorColorSuppl: '#ff7875',
  },

  Button: {
    color: '#1890ff',
    colorHover: '#40a9ff',
    colorPressed: '#096dd9',
    colorFocus: '#40a9ff',
  },

  Menu: {
    itemColorHover: 'rgba(24, 144, 255, 0.1)',
    itemColorActive: 'rgba(24, 144, 255, 0.15)',
    itemTextColorActive: '#1890ff',
    itemIconColorActive: '#1890ff',
    arrowColorActive: '#1890ff',
  },

  Tabs: {
    tabTextColorActiveLine: '#1890ff',
    tabTextColorHoverLine: '#40a9ff',
    lineColor: '#1890ff',
    barColor: '#1890ff',
  },
}

// 紫色主题配置示例
export const purpleThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#722ed1',
    primaryColorHover: '#9254de',
    primaryColorPressed: '#531dab',
    primaryColorSuppl: '#9254de',

    infoColor: '#722ed1',
    infoColorHover: '#9254de',
    infoColorPressed: '#531dab',
    infoColorSuppl: '#9254de',

    successColor: '#52c41a',
    successColorHover: '#73d13d',
    successColorPressed: '#389e0d',
    successColorSuppl: '#73d13d',

    warningColor: '#faad14',
    warningColorHover: '#ffc53d',
    warningColorPressed: '#d48806',
    warningColorSuppl: '#ffc53d',

    errorColor: '#ff4d4f',
    errorColorHover: '#ff7875',
    errorColorPressed: '#d9363e',
    errorColorSuppl: '#ff7875',
  },

  Button: {
    color: '#722ed1',
    colorHover: '#9254de',
    colorPressed: '#531dab',
    colorFocus: '#9254de',
  },

  Menu: {
    itemColorHover: 'rgba(114, 46, 209, 0.1)',
    itemColorActive: 'rgba(114, 46, 209, 0.15)',
    itemTextColorActive: '#722ed1',
    itemIconColorActive: '#722ed1',
    arrowColorActive: '#722ed1',
  },

  Tabs: {
    tabTextColorActiveLine: '#722ed1',
    tabTextColorHoverLine: '#9254de',
    lineColor: '#722ed1',
    barColor: '#722ed1',
  },
}

// 橙色主题配置示例
export const orangeThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#fa8c16',
    primaryColorHover: '#ffa940',
    primaryColorPressed: '#d46b08',
    primaryColorSuppl: '#ffa940',

    infoColor: '#fa8c16',
    infoColorHover: '#ffa940',
    infoColorPressed: '#d46b08',
    infoColorSuppl: '#ffa940',

    successColor: '#52c41a',
    successColorHover: '#73d13d',
    successColorPressed: '#389e0d',
    successColorSuppl: '#73d13d',

    warningColor: '#faad14',
    warningColorHover: '#ffc53d',
    warningColorPressed: '#d48806',
    warningColorSuppl: '#ffc53d',

    errorColor: '#ff4d4f',
    errorColorHover: '#ff7875',
    errorColorPressed: '#d9363e',
    errorColorSuppl: '#ff7875',
  },

  Button: {
    color: '#fa8c16',
    colorHover: '#ffa940',
    colorPressed: '#d46b08',
    colorFocus: '#ffa940',
  },

  Menu: {
    itemColorHover: 'rgba(250, 140, 22, 0.1)',
    itemColorActive: 'rgba(250, 140, 22, 0.15)',
    itemTextColorActive: '#fa8c16',
    itemIconColorActive: '#fa8c16',
    arrowColorActive: '#fa8c16',
  },

  Tabs: {
    tabTextColorActiveLine: '#fa8c16',
    tabTextColorHoverLine: '#ffa940',
    lineColor: '#fa8c16',
    barColor: '#fa8c16',
  },
}

// 主题类型定义
export type ThemeType = 'default' | 'blue' | 'purple' | 'orange'

// 获取主题配置的工具函数
export const getCustomThemeOverrides = (themeType: ThemeType): GlobalThemeOverrides | null => {
  switch (themeType) {
    case 'blue':
      return blueThemeOverrides
    case 'purple':
      return purpleThemeOverrides
    case 'orange':
      return orangeThemeOverrides
    default:
      return null
  }
}
