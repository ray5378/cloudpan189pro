module.exports = {
  extends: ['stylelint-config-standard'],
  overrides: [
    {
      files: ['**/*.{vue,html}'],
      customSyntax: 'postcss-html',
    },
  ],
  rules: {
    // 允许 Vue SFC 中的 :deep 伪类
    'selector-pseudo-class-no-unknown': [
      true,
      {
        ignorePseudoClasses: ['deep', 'global', 'slotted'],
      },
    ],

    // 允许第三方组件库的类名模式
    'selector-class-pattern': null,

    // 允许空的 <style> 标签
    'no-empty-source': null,

    // Vue 特定伪元素
    'selector-pseudo-element-no-unknown': [
      true,
      {
        ignorePseudoElements: ['v-deep', 'v-global', 'v-slotted'],
      },
    ],

    // 允许在同一文件中基础样式后写更高特异性的覆盖（如 .container 与 .container.dark）
    // 以便暗色模式/状态样式能放在底部覆盖，而不被 no-descending-specificity 阻拦
    'no-descending-specificity': null,
  },
}
