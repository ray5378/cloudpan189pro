<script setup lang="ts">
// ✅ 改用 import 方式
import logoSrc from '@/assets/logo.png'
</script>

<template>
  <img :src="logoSrc" :width="24" :height="24" class="cloudpan-logo" alt="CloudPan Logo" />
</template>

<style scoped>
.cloudpan-logo {
  color: var(--n-primary-color);
  flex-shrink: 0;
  transition: color 0.3s ease;
}

.cloudpan-logo:hover {
  opacity: 0.8;
}
</style>
