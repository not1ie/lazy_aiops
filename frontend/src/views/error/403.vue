<template>
  <el-card class="page-card">
    <el-result
      icon="warning"
      title="无权限访问"
      sub-title="请联系管理员开通相应权限。"
    >
      <template #extra>
        <el-button type="primary" @click="goHome">返回首页</el-button>
      </template>
    </el-result>
  </el-card>
</template>

<script setup>
import { useRouter } from 'vue-router'

const router = useRouter()
const goHome = () => router.push('/')
</script>

<style scoped>
.page-card {
  max-width: 680px;
  margin: 0 auto;
}
</style>
