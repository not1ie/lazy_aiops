<template>
  <el-card class="page-card">
    <div class="page-header">
      <h2>{ title }</h2>
      <p class="page-desc">{ desc }</p>
    </div>
    <el-empty description="页面骨架已建立，等待对接接口与交互。" />
  </el-card>
</template>

<script setup>
const title = '域名与证书'
const desc = '域名与 SSL 证书到期监控。'
</script>

<style scoped>
.page-card { max-width: 980px; margin: 0 auto; }
.page-header { margin-bottom: 12px; }
.page-desc { color: #606266; margin: 0; }
</style>
