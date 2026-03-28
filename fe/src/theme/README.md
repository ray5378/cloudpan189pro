# Naive UI 主题自定义指南

本项目已经配置了完整的 Naive UI 主题自定义系统，支持浅色和深色主题的完全自定义。

## 文件结构

```
fe/src/theme/
├── index.ts           # 主要的主题配置文件
├── custom-themes.ts   # 额外的主题配置示例
└── README.md         # 使用说明文档
```

## 基本使用

### 1. 当前配置

项目已经在 `App.vue` 中配置了主题系统：

```vue
<template>
  <n-config-provider :theme="theme" :theme-overrides="themeOverrides">
    <!-- 应用内容 -->
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useThemeStore } from '@/stores'
import { createTheme, createThemeOverrides } from '@/theme'

const themeStore = useThemeStore()
const theme = computed(() => createTheme(themeStore.isDark))
const themeOverrides = computed(() => createThemeOverrides(themeStore.isDark))
</script>
```

### 2. 主题切换

使用 `useThemeStore` 来控制主题切换：

```typescript
import { useThemeStore } from '@/stores'

const themeStore = useThemeStore()

// 切换主题
themeStore.toggleTheme()

// 检查当前是否为深色主题
console.log(themeStore.isDark)
```

## 自定义主题

### 1. 修改现有主题

编辑 `fe/src/theme/index.ts` 文件中的 `lightThemeOverrides` 或 `darkThemeOverrides`：

```typescript
export const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    // 修改主色调
    primaryColor: '#your-color',
    primaryColorHover: '#your-hover-color',
    primaryColorPressed: '#your-pressed-color',

    // 修改其他颜色...
  },

  // 修改特定组件样式
  Button: {
    borderRadius: '8px', // 修改按钮圆角
    fontWeight: '600', // 修改按钮字体粗细
  },
}
```

### 2. 创建新的主题配置

参考 `fe/src/theme/custom-themes.ts` 中的示例，创建新的主题配置：

```typescript
export const myCustomTheme: GlobalThemeOverrides = {
  common: {
    primaryColor: '#your-primary-color',
    // ... 其他配置
  },

  Button: {
    // 按钮样式配置
  },

  Card: {
    // 卡片样式配置
  },
}
```

### 3. 应用自定义主题

有两种方式应用自定义主题：

#### 方式一：替换默认主题

直接修改 `fe/src/theme/index.ts` 中的配置。

#### 方式二：动态切换主题

1. 扩展主题 store：

```typescript
// fe/src/stores/modules/theme/index.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ThemeType } from '@/theme/custom-themes'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(false)
  const themeType = ref<ThemeType>('default')

  const setThemeType = (type: ThemeType) => {
    themeType.value = type
    localStorage.setItem('themeType', type)
  }

  // ... 其他代码

  return {
    isDark,
    themeType,
    setThemeType,
    // ... 其他返回值
  }
})
```

2. 更新 App.vue：

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useThemeStore } from '@/stores'
import { createTheme, createThemeOverrides } from '@/theme'
import { getCustomThemeOverrides } from '@/theme/custom-themes'

const themeStore = useThemeStore()

const theme = computed(() => createTheme(themeStore.isDark))
const themeOverrides = computed(() => {
  const baseOverrides = createThemeOverrides(themeStore.isDark)
  const customOverrides = getCustomThemeOverrides(themeStore.themeType)

  // 合并主题配置
  return customOverrides ? { ...baseOverrides, ...customOverrides } : baseOverrides
})
</script>
```

## 可自定义的属性

### Common 通用属性

```typescript
common: {
  // 主色调
  primaryColor: string,
  primaryColorHover: string,
  primaryColorPressed: string,
  primaryColorSuppl: string,

  // 功能色
  infoColor: string,
  successColor: string,
  warningColor: string,
  errorColor: string,

  // 文字颜色
  textColorBase: string,
  textColor1: string,
  textColor2: string,
  textColor3: string,

  // 背景颜色
  bodyColor: string,
  cardColor: string,
  modalColor: string,
  popoverColor: string,

  // 边框和分割线
  borderColor: string,
  dividerColor: string,

  // 圆角
  borderRadius: string,
  borderRadiusSmall: string,

  // 字体
  fontFamily: string,
  fontSize: string,
  fontSizeMini: string,
  fontSizeSmall: string,
  fontSizeMedium: string,
  fontSizeLarge: string,
  fontSizeHuge: string,

  // 阴影
  boxShadow1: string,
  boxShadow2: string,
  boxShadow3: string,
}
```

### 组件特定属性

每个组件都有自己的可自定义属性，例如：

```typescript
// 按钮组件
Button: {
  textColor: string,
  textColorHover: string,
  color: string,
  colorHover: string,
  borderRadius: string,
  fontWeight: string,
}

// 输入框组件
Input: {
  borderRadius: string,
  border: string,
  borderHover: string,
  borderFocus: string,
  boxShadowFocus: string,
}

// 卡片组件
Card: {
  borderRadius: string,
  paddingMedium: string,
  paddingLarge: string,
  boxShadow: string,
}
```

## 最佳实践

### 1. 颜色系统

建议使用一致的颜色系统：

```typescript
// 定义颜色变量
const colors = {
  primary: {
    50: '#f0f9ff',
    100: '#e0f2fe',
    500: '#0ea5e9',
    600: '#0284c7',
    700: '#0369a1',
  },
}

// 在主题配置中使用
export const customTheme: GlobalThemeOverrides = {
  common: {
    primaryColor: colors.primary[500],
    primaryColorHover: colors.primary[400],
    primaryColorPressed: colors.primary[600],
  },
}
```

### 2. 响应式设计

考虑不同屏幕尺寸的适配：

```typescript
export const responsiveTheme: GlobalThemeOverrides = {
  common: {
    fontSize: '14px',
    fontSizeMobile: '16px', // 移动端使用更大的字体
  },

  Button: {
    paddingMedium: '0 16px',
    paddingMediumMobile: '0 20px', // 移动端使用更大的内边距
  },
}
```

### 3. 无障碍访问

确保颜色对比度符合无障碍标准：

```typescript
export const accessibleTheme: GlobalThemeOverrides = {
  common: {
    // 确保文字和背景有足够的对比度
    textColor1: '#1a1a1a', // 深色文字
    bodyColor: '#ffffff', // 白色背景

    // 使用无障碍友好的颜色
    errorColor: '#dc2626', // 高对比度的红色
    successColor: '#16a34a', // 高对比度的绿色
  },
}
```

## 调试技巧

### 1. 使用浏览器开发者工具

在浏览器中检查元素，查看 Naive UI 组件的 CSS 变量：

```css
/* 在开发者工具中可以看到这些 CSS 变量 */
:root {
  --n-color-primary: #18a058;
  --n-color-primary-hover: #36ad6a;
  --n-color-primary-pressed: #0c7a43;
}
```

### 2. 临时测试主题

在组件中临时应用主题进行测试：

```vue
<template>
  <n-config-provider :theme-overrides="testTheme">
    <n-button type="primary">测试按钮</n-button>
  </n-config-provider>
</template>

<script setup lang="ts">
const testTheme = {
  Button: {
    color: '#ff0000', // 临时测试红色按钮
  },
}
</script>
```

## 常见问题

### Q: 主题配置不生效？

A: 检查以下几点：

1. 确保在 `n-config-provider` 中正确传入了 `theme-overrides`
2. 检查属性名是否正确（参考 Naive UI 官方文档）
3. 确保颜色值格式正确（使用十六进制或 rgba）

### Q: 如何覆盖特定组件的样式？

A: 使用组件特定的主题配置：

```typescript
export const customTheme: GlobalThemeOverrides = {
  Button: {
    // 只影响按钮组件
    color: '#custom-color',
  },

  DataTable: {
    // 只影响数据表格组件
    thColor: '#custom-header-color',
  },
}
```

### Q: 深色主题和浅色主题如何保持一致性？

A: 使用相同的设计令牌（design tokens）：

```typescript
const designTokens = {
  spacing: {
    small: '8px',
    medium: '16px',
    large: '24px',
  },
  borderRadius: {
    small: '4px',
    medium: '6px',
    large: '8px',
  },
}

// 在两个主题中使用相同的令牌
export const lightTheme = {
  common: {
    borderRadius: designTokens.borderRadius.medium,
  },
}

export const darkTheme = {
  common: {
    borderRadius: designTokens.borderRadius.medium, // 保持一致
  },
}
```

## 参考资源

- [Naive UI 官方主题文档](https://www.naiveui.com/zh-CN/os-theme/docs/customize-theme)
- [Naive UI 组件主题配置](https://www.naiveui.com/zh-CN/os-theme/docs/theme)
- [CSS 颜色对比度检查工具](https://webaim.org/resources/contrastchecker/)
