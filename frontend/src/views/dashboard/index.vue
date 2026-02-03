<template>
  <div>
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>总主机数</span>
              <el-tag type="success">实时</el-tag>
            </div>
          </template>
          <div class="card-value">12</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>Docker 环境</span>
              <el-tag type="primary">Docker</el-tag>
            </div>
          </template>
          <div class="card-value">3</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>告警总数</span>
              <el-tag type="danger">Alert</el-tag>
            </div>
          </template>
          <div class="card-value">0</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>今日任务</span>
              <el-tag type="warning">Task</el-tag>
            </div>
          </template>
          <div class="card-value">5</div>
        </el-card>
      </el-col>
    </el-row>
    
    <el-card class="mt-20" header="系统状态">
      <div style="height: 300px; display: flex; align-items: center; justify-content: center; color: #909399;">
        图表区域 (ECharts 待集成)
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.card-value { font-size: 28px; font-weight: bold; color: #303133; margin-top: 10px; }
.mt-20 { margin-top: 20px; }
</style>
