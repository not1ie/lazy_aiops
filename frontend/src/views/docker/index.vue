<template>
  <el-card>
    <template #header>
      <div class="flex justify-between items-center">
        <span class="font-bold">Docker 环境列表</span>
        <div>
          <el-button type="primary" icon="Plus" @click="handleAdd">添加环境</el-button>
          <el-button icon="Refresh" @click="syncAll">刷新</el-button>
        </div>
      </div>
    </template>

    <el-table :data="tableData" v-loading="loading" style="width: 100%">
      <el-table-column prop="name" label="名称" width="180">
        <template #default="{ row }">
          <div class="flex items-center gap-2">
            <el-icon class="text-blue-500 text-xl"><Platform /></el-icon>
            <span class="font-bold">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="row.status === 'online' ? 'success' : 'danger'">
            {{ row.status || 'unknown' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="container_count" label="容器数" width="120" align="center" />
      <el-table-column prop="image_count" label="镜像数" width="120" align="center" />
      <el-table-column prop="version" label="版本" />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-space size="8">
            <el-button size="small" type="primary" plain icon="Monitor" @click="handleManage(row)">管理</el-button>
            <el-button size="small" type="warning" plain icon="FirstAidKit" @click="handleDiagnose(row)">诊断</el-button>
            <el-button size="small" type="danger" plain icon="Delete" @click="handleDelete(row)">删除</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <!-- 添加主机弹窗 -->
    <el-dialog v-model="dialogVisible" title="添加 Docker 环境" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如: Local Docker" />
        </el-form-item>
        <el-form-item label="关联主机">
          <el-select v-model="form.host_id" placeholder="请选择" class="w-100">
            <el-option label="本机 (Local Socket)" value="local" />
            <el-option v-for="h in hosts" :key="h.id" :label="h.name + ' (' + h.ip + ')'" :value="h.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- 管理抽屉 -->
    <el-drawer v-model="manageVisible" size="70%" :with-header="false">
      <div class="drawer-header">
        <div>
          <div class="drawer-title">{{ activeHost?.name || 'Docker 环境' }}</div>
          <div class="drawer-sub">
            状态：<el-tag size="small" :type="activeHost?.status === 'online' ? 'success' : 'danger'">{{ activeHost?.status || 'unknown' }}</el-tag>
            <span class="drawer-meta">容器：{{ activeHost?.container_count ?? '-' }}</span>
            <span class="drawer-meta">镜像：{{ activeHost?.image_count ?? '-' }}</span>
            <span class="drawer-meta">版本：{{ activeHost?.version || '-' }}</span>
          </div>
        </div>
        <div>
          <el-button size="small" icon="Refresh" @click="refreshManage">刷新</el-button>
        </div>
      </div>

      <el-tabs v-model="manageTab" class="manage-tabs">
        <el-tab-pane label="概览" name="overview">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="名称">{{ activeHost?.name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ activeHost?.status || 'unknown' }}</el-descriptions-item>
            <el-descriptions-item label="容器数">{{ activeHost?.container_count ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="镜像数">{{ activeHost?.image_count ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ activeHost?.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="主机ID">{{ activeHost?.host_id || '-' }}</el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>

        <el-tab-pane label="容器" name="containers">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="info" plain @click="loadContainerStats" :loading="statsLoading">刷新资源</el-button>
              <el-switch v-model="autoRefreshStats" active-text="自动刷新" />
              <el-button type="success" plain :disabled="selectedContainers.length === 0" @click="startSelectedContainers">批量启动</el-button>
              <el-button type="warning" plain :disabled="selectedContainers.length === 0" @click="stopSelectedContainers">批量停止</el-button>
              <el-button type="primary" plain :disabled="selectedContainers.length === 0" @click="restartSelectedContainers">批量重启</el-button>
              <el-button type="danger" plain :disabled="selectedContainers.length === 0" @click="removeSelectedContainers">批量删除</el-button>
            </div>
            <div class="toolbar-right">
              <el-button type="primary" icon="Plus" @click="openCreateContainer">创建容器</el-button>
              <el-button icon="Refresh" @click="loadContainers">刷新</el-button>
            </div>
          </div>
          <el-table
            ref="containerTableRef"
            :data="containers"
            v-loading="containersLoading"
            style="width: 100%"
            :row-key="row => row.id"
            @selection-change="onContainerSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="names" label="名称" min-width="200" />
            <el-table-column prop="image" label="镜像" min-width="180" />
            <el-table-column prop="state" label="状态" width="120" />
            <el-table-column prop="status" label="详情" min-width="180" />
            <el-table-column label="资源" min-width="200">
              <template #default="{ row }">
                <div class="text-xs">
                  <div>CPU {{ getContainerStats(row).cpu }}</div>
                  <div>MEM {{ getContainerStats(row).mem }}</div>
                  <div>NET {{ getContainerStats(row).net }}</div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="created" label="创建时间" width="170">
              <template #default="{ row }">
                {{ formatTime(row.created) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="470" fixed="right">
              <template #default="{ row }">
                <el-space size="8">
                  <el-button size="small" @click="openLogs(row)">日志</el-button>
                  <el-button size="small" type="primary" plain @click="openInspect(row)">详情</el-button>
                  <el-button size="small" type="info" plain @click="openExec(row)">执行命令</el-button>
                  <el-button size="small" type="success" plain @click="openTerminal(row)">终端</el-button>
                  <el-button size="small" type="success" plain @click="containerAction(row, 'start')">启动</el-button>
                  <el-button size="small" type="warning" plain @click="containerAction(row, 'stop')">停止</el-button>
                  <el-button size="small" type="primary" plain @click="containerAction(row, 'restart')">重启</el-button>
                  <el-button size="small" type="danger" plain @click="containerAction(row, 'remove')">删除</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="镜像" name="images">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-input v-model="imageFilters.keyword" class="w-48" placeholder="仓库/Tag/ID" clearable />
              <el-input v-model="imageFilters.minSize" class="w-28" placeholder="最小MB" clearable />
              <el-input v-model="imageFilters.maxSize" class="w-28" placeholder="最大MB" clearable />
              <el-checkbox v-model="imageFilters.danglingOnly">仅悬挂</el-checkbox>
              <el-button plain @click="clearImageFilters">重置</el-button>
            </div>
            <div class="toolbar-right">
              <el-button type="danger" plain :disabled="selectedImages.length === 0" @click="removeSelectedImages">
                批量删除
              </el-button>
              <el-button type="warning" plain @click="pruneImages">清理悬挂</el-button>
              <el-button type="primary" icon="Download" @click="openPullImage">拉取镜像</el-button>
              <el-button icon="Refresh" @click="loadImages">刷新</el-button>
            </div>
          </div>
          <el-table
            ref="imageTableRef"
            :data="filteredImages"
            v-loading="imagesLoading"
            style="width: 100%"
            :row-key="row => row.id"
            @selection-change="onImageSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="repository" label="仓库" min-width="200" />
            <el-table-column prop="tag" label="Tag" width="120" />
            <el-table-column prop="id" label="ID" min-width="180" />
            <el-table-column prop="size" label="大小" width="120" />
            <el-table-column prop="created" label="创建时间" min-width="180">
              <template #default="{ row }">
                {{ formatTime(row.created_at || row.created) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button size="small" type="danger" plain @click="removeImage(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="网络" name="networks">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="danger" plain :disabled="selectedNetworks.length === 0" @click="removeSelectedNetworks">批量删除</el-button>
            </div>
            <div class="toolbar-right">
              <el-button icon="Refresh" @click="loadNetworks">刷新</el-button>
            </div>
          </div>
          <el-table
            ref="networkTableRef"
            :data="networks"
            v-loading="networksLoading"
            style="width: 100%"
            :row-key="row => row.id"
            @selection-change="onNetworkSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="name" label="名称" min-width="180" />
            <el-table-column prop="id" label="ID" min-width="200" />
            <el-table-column prop="driver" label="驱动" width="120" />
            <el-table-column prop="scope" label="范围" width="120" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Volumes" name="volumes">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="danger" plain :disabled="selectedVolumes.length === 0" @click="removeSelectedVolumes">批量删除</el-button>
            </div>
            <div class="toolbar-right">
              <el-button type="primary" icon="Plus" @click="openCreateVolume">创建卷</el-button>
              <el-button icon="Refresh" @click="loadVolumes">刷新</el-button>
            </div>
          </div>
          <el-table
            ref="volumeTableRef"
            :data="volumes"
            v-loading="volumesLoading"
            style="width: 100%"
            :row-key="row => row.Name || row.name"
            @selection-change="onVolumeSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="Name" label="名称" min-width="200" />
            <el-table-column prop="Driver" label="驱动" width="140" />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openVolumeInspect(row)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Secrets" name="secrets">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="danger" plain :disabled="selectedSecrets.length === 0" @click="removeSelectedSecrets">批量删除</el-button>
            </div>
            <div class="toolbar-right">
              <el-button type="primary" icon="Plus" @click="openCreateSecret">创建 Secret</el-button>
              <el-button icon="Refresh" @click="loadSecrets">刷新</el-button>
            </div>
          </div>
          <el-table
            ref="secretTableRef"
            :data="secrets"
            v-loading="secretsLoading"
            style="width: 100%"
            :row-key="row => row.ID || row.Id || row.id || row.Name"
            @selection-change="onSecretSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="Name" label="名称" min-width="200" />
            <el-table-column prop="ID" label="ID" min-width="200" />
            <el-table-column prop="CreatedAt" label="创建时间" min-width="180" />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openSecretInspect(row)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Configs" name="configs">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="danger" plain :disabled="selectedConfigs.length === 0" @click="removeSelectedConfigs">批量删除</el-button>
            </div>
            <div class="toolbar-right">
              <el-button type="primary" icon="Plus" @click="openCreateConfig">创建 Config</el-button>
              <el-button icon="Refresh" @click="loadConfigs">刷新</el-button>
            </div>
          </div>
          <el-table
            ref="configTableRef"
            :data="configs"
            v-loading="configsLoading"
            style="width: 100%"
            :row-key="row => row.ID || row.Id || row.id || row.Name"
            @selection-change="onConfigSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="Name" label="名称" min-width="200" />
            <el-table-column prop="ID" label="ID" min-width="200" />
            <el-table-column prop="CreatedAt" label="创建时间" min-width="180" />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openConfigInspect(row)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Registries" name="registries">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="primary" icon="Plus" @click="openCreateRegistry">添加仓库</el-button>
            </div>
            <div class="toolbar-right">
              <el-button icon="Refresh" @click="loadRegistries">刷新</el-button>
            </div>
          </div>
          <el-table :data="registries" v-loading="registriesLoading" style="width: 100%">
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="url" label="地址" min-width="220" />
            <el-table-column prop="username" label="用户名" min-width="140" />
            <el-table-column prop="login_status" label="登录状态" width="120">
              <template #default="{ row }">
                <el-tag v-if="row.login_status === 'success'" type="success">成功</el-tag>
                <el-tag v-else-if="row.login_status === 'failed'" type="danger">失败</el-tag>
                <el-tag v-else type="info">未知</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="last_login_at" label="上次登录" min-width="180">
              <template #default="{ row }">
                {{ formatTime(row.last_login_at) }}
              </template>
            </el-table-column>
            <el-table-column prop="insecure" label="Insecure" width="120">
              <template #default="{ row }">
                <el-tag :type="row.insecure ? 'warning' : 'success'">{{ row.insecure ? '是' : '否' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="240" fixed="right">
              <template #default="{ row }">
                <el-space size="8">
                  <el-button size="small" type="primary" plain @click="loginRegistry(row)">登录当前主机</el-button>
                  <el-button size="small" type="danger" plain @click="removeRegistry(row)">删除</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Nodes" name="nodes">
          <div class="tab-toolbar">
            <div class="toolbar-right">
              <el-button icon="Refresh" @click="loadNodes">刷新</el-button>
            </div>
          </div>
          <el-table :data="nodes" v-loading="nodesLoading" style="width: 100%">
            <el-table-column prop="ID" label="ID" min-width="180" />
            <el-table-column prop="Hostname" label="主机名" min-width="180" />
            <el-table-column prop="Status" label="状态" width="120" />
            <el-table-column prop="Availability" label="可用性" width="120" />
            <el-table-column prop="ManagerStatus" label="管理状态" min-width="180" />
            <el-table-column prop="EngineVersion" label="Engine" width="140" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="拓扑" name="topology">
          <div class="tab-toolbar">
            <div class="toolbar-right">
              <el-button icon="Refresh" @click="loadServices">刷新</el-button>
            </div>
          </div>
          <el-tree
            :data="topologyTree"
            node-key="id"
            default-expand-all
            :expand-on-click-node="false"
          />
        </el-tab-pane>

        <el-tab-pane label="Services" name="services">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="primary" plain :disabled="selectedServices.length === 0" @click="restartSelectedServices">批量重启</el-button>
              <el-input-number v-model="batchServiceScale" :min="0" class="w-28" controls-position="right" />
              <el-button type="success" plain :disabled="selectedServices.length === 0" @click="scaleSelectedServices">
                批量设置副本
              </el-button>
              <el-select v-model="serviceStackFilter" placeholder="Stack" class="w-40" clearable>
                <el-option v-for="s in serviceStacks" :key="s" :label="s" :value="s" />
              </el-select>
            </div>
            <div class="toolbar-right">
              <el-button icon="Refresh" @click="loadServices">刷新</el-button>
            </div>
          </div>
          <el-table
            ref="serviceTableRef"
            :data="filteredServices"
            v-loading="servicesLoading"
            style="width: 100%"
            :row-key="row => row.ID || row.Id || row.id || row.Name"
            @selection-change="onServiceSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="Name" label="名称" min-width="200" />
            <el-table-column prop="Stack" label="Stack" width="140" />
            <el-table-column prop="Mode" label="模式" width="120" />
            <el-table-column prop="Replicas" label="副本" width="120" />
            <el-table-column prop="Image" label="镜像" min-width="180" />
            <el-table-column prop="Ports" label="端口" min-width="160" />
            <el-table-column label="操作" width="360" fixed="right">
              <template #default="{ row }">
                <el-space size="8">
                  <el-button size="small" @click="openServiceDetail(row)">详情</el-button>
                  <el-button size="small" type="info" plain @click="openServiceTasks(row)">任务</el-button>
                  <el-button size="small" @click="openServiceLogs(row)">日志</el-button>
                  <el-input-number v-model="serviceScaleMap[row.ID || row.Id || row.id]" :min="0" size="small" class="w-28" controls-position="right" />
                  <el-button size="small" type="success" plain @click="applyServiceScale(row)">应用</el-button>
                  <el-button size="small" type="primary" plain @click="scaleService(row)">扩缩容</el-button>
                  <el-button size="small" type="warning" plain @click="updateServiceImage(row)">更新镜像</el-button>
                  <el-button size="small" type="danger" plain @click="restartService(row)">重启</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Stacks" name="stacks">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="danger" plain :disabled="selectedStacks.length === 0" @click="removeSelectedStacks">批量删除</el-button>
            </div>
            <div class="toolbar-right">
              <el-button type="primary" icon="Plus" @click="openDeployStack">部署 Stack</el-button>
              <el-button icon="Refresh" @click="loadStacks">刷新</el-button>
            </div>
          </div>
          <el-table
            ref="stackTableRef"
            :data="stacks"
            v-loading="stacksLoading"
            style="width: 100%"
            :row-key="row => row.Name || row.name"
            @selection-change="onStackSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="Name" label="名称" min-width="200" />
            <el-table-column prop="Services" label="服务数" width="120" />
            <el-table-column prop="Orchestrator" label="编排" width="160" />
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openStackServices(row)">查看服务</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-drawer>

    <!-- 创建容器弹窗 -->
    <el-dialog v-model="createVisible" title="创建容器" width="640px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="镜像" required>
          <el-input v-model="createForm.image" placeholder="例如 nginx:latest" />
        </el-form-item>
        <el-form-item label="容器名">
          <el-input v-model="createForm.name" placeholder="可选" />
        </el-form-item>
        <el-form-item label="端口映射">
          <el-input v-model="createForm.ports" placeholder="如 8080:80, 8443:443" />
        </el-form-item>
        <el-form-item label="环境变量">
          <el-input v-model="createForm.env" type="textarea" :rows="4" placeholder="KEY=VALUE，每行一个" />
        </el-form-item>
        <el-form-item label="重启策略">
          <el-select v-model="createForm.restart_policy" placeholder="不设置">
            <el-option label="不设置" value="" />
            <el-option label="always" value="always" />
            <el-option label="unless-stopped" value="unless-stopped" />
            <el-option label="on-failure" value="on-failure" />
          </el-select>
        </el-form-item>
        <el-form-item label="自动删除">
          <el-switch v-model="createForm.auto_remove" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createLoading" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- 拉取镜像弹窗 -->
    <el-dialog v-model="pullVisible" title="拉取镜像" width="640px">
      <el-form label-width="100px">
        <el-form-item label="镜像" required>
          <el-input v-model="pullImage" placeholder="例如 redis:7" />
        </el-form-item>
      </el-form>
      <el-input v-model="pullOutput" type="textarea" :rows="8" readonly placeholder="输出" />
      <template #footer>
        <el-button @click="pullVisible = false">关闭</el-button>
        <el-button type="primary" :loading="pullLoading" @click="submitPull">拉取</el-button>
      </template>
    </el-dialog>

    <!-- 批量删除结果 -->
    <el-dialog v-model="batchResultVisible" title="批量删除结果" width="720px">
      <el-table :data="batchResultRows" style="width: 100%">
        <el-table-column prop="label" label="镜像" min-width="220" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="row.status === '成功' ? 'success' : 'danger'">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="信息" min-width="240" />
      </el-table>
      <template #footer>
        <el-button type="primary" @click="batchResultVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 清理悬挂镜像结果 -->
    <el-dialog v-model="pruneVisible" title="清理结果" width="720px">
      <el-input v-model="pruneOutput" type="textarea" :rows="10" readonly placeholder="输出" />
      <template #footer>
        <el-button type="primary" @click="pruneVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 诊断弹窗 -->
    <el-dialog v-model="diagnoseVisible" title="Docker 诊断" width="720px">
      <el-alert v-if="diagnoseError" type="error" :closable="false" show-icon>{{ diagnoseError }}</el-alert>
      <el-skeleton v-if="diagnoseLoading" :rows="6" animated />
      <div v-else class="diagnose-block">
        <div class="diagnose-title">Step1: docker info</div>
        <pre class="diagnose-pre">{{ diagnoseResult?.step1_info?.out || '-' }}</pre>
        <div class="diagnose-title">Step2: docker system info (json)</div>
        <pre class="diagnose-pre">{{ diagnoseResult?.step2_sync?.out || '-' }}</pre>
        <div class="diagnose-title">Step3: docker ps -a</div>
        <pre class="diagnose-pre">{{ diagnoseResult?.step3_list?.out || '-' }}</pre>
      </div>
      <template #footer>
        <el-button @click="diagnoseVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 容器日志弹窗 -->
    <el-dialog v-model="logVisible" title="容器日志" width="720px">
      <div class="log-controls">
        <el-input v-model="logTail" placeholder="tail" style="width: 120px" />
        <el-button icon="Refresh" @click="loadLogs" :loading="logLoading">刷新</el-button>
        <el-switch v-model="logFollow" active-text="实时" />
        <el-button @click="clearLogs">清空</el-button>
      </div>
      <el-input v-model="logText" type="textarea" :rows="16" readonly />
    </el-dialog>

    <!-- Service 日志弹窗 -->
    <el-dialog v-model="serviceLogVisible" title="Service 日志" width="760px">
      <div class="log-controls">
        <el-input v-model="serviceLogTail" placeholder="tail" style="width: 120px" />
        <el-button icon="Refresh" @click="loadServiceLogs" :loading="serviceLogLoading">刷新</el-button>
        <el-switch v-model="serviceLogFollow" active-text="实时" />
        <el-button @click="clearServiceLogs">清空</el-button>
      </div>
      <el-input v-model="serviceLogText" type="textarea" :rows="16" readonly />
    </el-dialog>

    <!-- 容器详情弹窗 -->
    <el-dialog v-model="inspectVisible" title="容器详情" width="880px">
      <el-skeleton v-if="inspectLoading" :rows="6" animated />
      <div v-else>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="ID">{{ inspectData?.Id || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Name">{{ inspectData?.Name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Image">{{ inspectData?.Config?.Image || '-' }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ inspectData?.State?.Status || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">Ports</el-divider>
        <el-table :data="inspectPorts" style="width: 100%">
          <el-table-column prop="container" label="容器端口" width="160" />
          <el-table-column prop="host" label="主机端口" width="160" />
          <el-table-column prop="ip" label="Host IP" width="160" />
          <el-table-column label="复制" width="120">
            <template #default="scope">
              <el-button size="small" @click="copyText(`${scope.row.ip}:${scope.row.host}`)">复制</el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-divider content-position="left">Networks</el-divider>
        <el-table :data="inspectNetworks" style="width: 100%">
          <el-table-column prop="name" label="名称" width="200" />
          <el-table-column prop="ip" label="IP" width="180" />
          <el-table-column prop="gateway" label="网关" width="180" />
        </el-table>

        <el-divider content-position="left">Mounts</el-divider>
        <el-table :data="inspectMounts" style="width: 100%">
          <el-table-column prop="type" label="类型" width="120" />
          <el-table-column prop="source" label="Source" min-width="220" />
          <el-table-column prop="destination" label="Destination" min-width="220" />
          <el-table-column prop="mode" label="Mode" width="120" />
          <el-table-column prop="rw" label="RW" width="80" />
        </el-table>

        <el-divider content-position="left">Env</el-divider>
        <el-input v-model="inspectEnvText" type="textarea" :rows="8" readonly />
      </div>
    </el-dialog>

    <!-- 容器执行命令弹窗 -->
    <el-dialog v-model="execVisible" title="执行容器命令" width="720px">
      <el-alert type="info" :closable="false" show-icon>该功能为非交互命令执行（需要容器内存在 /bin/sh）。</el-alert>
      <div class="log-controls">
        <el-input v-model="execCommand" placeholder="例如: ls / 或 ps aux" />
        <el-button type="primary" @click="runExec" :loading="execLoading">执行</el-button>
      </div>
      <el-input v-model="execOutput" type="textarea" :rows="16" readonly placeholder="输出" />
    </el-dialog>

    <!-- Service 详情弹窗 -->
    <el-dialog v-model="serviceVisible" title="Service 详情" width="880px">
      <el-skeleton v-if="serviceLoading" :rows="6" animated />
      <el-input v-else v-model="serviceJson" type="textarea" :rows="16" readonly />
    </el-dialog>

    <!-- Service 任务弹窗 -->
    <el-dialog v-model="tasksVisible" title="Service 任务" width="880px">
      <el-table :data="serviceTasks" v-loading="tasksLoading" style="width: 100%">
        <el-table-column prop="ID" label="ID" min-width="180" />
        <el-table-column prop="Name" label="名称" min-width="200" />
        <el-table-column prop="Node" label="节点" width="160" />
        <el-table-column prop="DesiredState" label="期望状态" width="120" />
        <el-table-column prop="CurrentState" label="当前状态" min-width="200" />
        <el-table-column prop="Error" label="错误" min-width="200" />
      </el-table>
    </el-dialog>

    <!-- Stack 服务弹窗 -->
    <el-dialog v-model="stackVisible" title="Stack 服务" width="880px">
      <el-table :data="stackServices" v-loading="stackLoading" style="width: 100%">
        <el-table-column prop="Name" label="名称" min-width="220" />
        <el-table-column prop="Mode" label="模式" width="120" />
        <el-table-column prop="Replicas" label="副本" width="120" />
        <el-table-column prop="Image" label="镜像" min-width="180" />
        <el-table-column prop="Ports" label="端口" min-width="180" />
      </el-table>
    </el-dialog>

    <!-- 创建卷 -->
    <el-dialog v-model="createVolumeVisible" title="创建卷" width="520px">
      <el-form :model="volumeForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="volumeForm.name" />
        </el-form-item>
        <el-form-item label="驱动">
          <el-input v-model="volumeForm.driver" placeholder="默认 local" />
        </el-form-item>
        <el-form-item label="Labels">
          <el-input v-model="volumeForm.labels" type="textarea" :rows="4" placeholder="KEY=VALUE 每行一个" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVolumeVisible = false">取消</el-button>
        <el-button type="primary" :loading="volumeLoading" @click="submitCreateVolume">创建</el-button>
      </template>
    </el-dialog>

    <!-- 卷详情 -->
    <el-dialog v-model="volumeInspectVisible" title="卷详情" width="760px">
      <el-input v-model="volumeInspectJson" type="textarea" :rows="14" readonly />
      <template #footer>
        <el-button @click="copyText(volumeInspectJson)">复制</el-button>
        <el-button type="primary" @click="downloadJson('volume.json', volumeInspectJson)">导出</el-button>
      </template>
    </el-dialog>

    <!-- 创建 Secret -->
    <el-dialog v-model="createSecretVisible" title="创建 Secret" width="520px">
      <el-form :model="secretForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="secretForm.name" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="secretForm.data" type="textarea" :rows="6" placeholder="内容" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createSecretVisible = false">取消</el-button>
        <el-button type="primary" :loading="secretLoading" @click="submitCreateSecret">创建</el-button>
      </template>
    </el-dialog>

    <!-- Secret 详情 -->
    <el-dialog v-model="secretInspectVisible" title="Secret 详情" width="760px">
      <el-input v-model="secretInspectJson" type="textarea" :rows="12" readonly />
      <template #footer>
        <el-button @click="copyText(secretInspectJson)">复制</el-button>
        <el-button type="primary" @click="downloadJson('secret.json', secretInspectJson)">导出</el-button>
      </template>
    </el-dialog>

    <!-- 创建 Config -->
    <el-dialog v-model="createConfigVisible" title="创建 Config" width="520px">
      <el-form :model="configForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="configForm.name" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="configForm.data" type="textarea" :rows="6" placeholder="内容" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createConfigVisible = false">取消</el-button>
        <el-button type="primary" :loading="configLoading" @click="submitCreateConfig">创建</el-button>
      </template>
    </el-dialog>

    <!-- Config 详情 -->
    <el-dialog v-model="configInspectVisible" title="Config 详情" width="760px">
      <el-input v-model="configInspectJson" type="textarea" :rows="12" readonly />
      <template #footer>
        <el-button @click="copyText(configInspectJson)">复制</el-button>
        <el-button type="primary" @click="downloadJson('config.json', configInspectJson)">导出</el-button>
      </template>
    </el-dialog>

    <!-- 部署 Stack -->
    <el-dialog v-model="deployVisible" title="部署 Stack" width="760px">
      <el-form :model="deployForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="deployForm.name" placeholder="stack 名称" />
        </el-form-item>
        <el-form-item label="Compose" required>
          <el-input v-model="deployForm.compose" type="textarea" :rows="10" placeholder="docker-compose.yml 内容" />
        </el-form-item>
      </el-form>
      <el-input v-model="deployOutput" type="textarea" :rows="6" readonly placeholder="输出" />
      <template #footer>
        <el-button @click="deployVisible = false">关闭</el-button>
        <el-button type="primary" :loading="deployLoading" @click="submitDeployStack">部署</el-button>
      </template>
    </el-dialog>

    <!-- 容器 WebShell -->
    <el-dialog v-model="terminalVisible" title="容器终端" width="920px" @closed="closeTerminal">
      <div class="terminal-toolbar">
        <span class="terminal-title">{{ terminalContainerName || terminalContainerId }}</span>
        <el-select v-model="terminalShell" class="w-28">
          <el-option label="/bin/sh" value="/bin/sh" />
          <el-option label="/bin/bash" value="/bin/bash" />
          <el-option label="/bin/ash" value="/bin/ash" />
        </el-select>
        <el-button type="primary" :disabled="!terminalContainerId" @click="toggleTerminal">
          {{ terminalConnected ? '断开' : '连接' }}
        </el-button>
      </div>
      <div ref="terminalRef" class="terminal-container"></div>
    </el-dialog>

    <!-- 添加仓库 -->
    <el-dialog v-model="createRegistryVisible" title="添加仓库" width="520px">
      <el-form :model="registryForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="registryForm.name" />
        </el-form-item>
        <el-form-item label="地址" required>
          <el-input v-model="registryForm.url" placeholder="例如 registry.example.com" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="registryForm.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="registryForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="Insecure">
          <el-switch v-model="registryForm.insecure" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createRegistryVisible = false">取消</el-button>
        <el-button type="primary" :loading="registryLoading" @click="submitCreateRegistry">保存</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, reactive, onMounted, watch, computed, onUnmounted, nextTick } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'

const loading = ref(false)
const tableData = ref([])
const dialogVisible = ref(false)
const submitting = ref(false)
const hosts = ref([])

const manageVisible = ref(false)
const manageTab = ref('overview')
const activeHost = ref(null)

const containers = ref([])
const containersLoading = ref(false)
const containerTableRef = ref(null)
const selectedContainers = ref([])
const statsLoading = ref(false)
const containerStats = ref({})
const images = ref([])
const imagesLoading = ref(false)
const imageTableRef = ref(null)
const selectedImages = ref([])
const imageFilters = reactive({
  keyword: '',
  minSize: '',
  maxSize: '',
  danglingOnly: false
})
const networks = ref([])
const networksLoading = ref(false)
const networkTableRef = ref(null)
const selectedNetworks = ref([])

const volumes = ref([])
const volumesLoading = ref(false)
const volumeTableRef = ref(null)
const selectedVolumes = ref([])

const secrets = ref([])
const secretsLoading = ref(false)
const secretTableRef = ref(null)
const selectedSecrets = ref([])

const configs = ref([])
const configsLoading = ref(false)
const configTableRef = ref(null)
const selectedConfigs = ref([])

const nodes = ref([])
const nodesLoading = ref(false)

const registries = ref([])
const registriesLoading = ref(false)
const createRegistryVisible = ref(false)
const registryLoading = ref(false)
const registryForm = reactive({
  name: '',
  url: '',
  username: '',
  password: '',
  insecure: false
})

const terminalVisible = ref(false)
const terminalContainerId = ref('')
const terminalContainerName = ref('')
const terminalShell = ref('/bin/sh')
const terminalConnected = ref(false)
const terminalRef = ref(null)
let terminal = null
let terminalFit = null
let terminalWs = null

const services = ref([])
const servicesLoading = ref(false)
const serviceStackFilter = ref('')
const serviceTableRef = ref(null)
const selectedServices = ref([])
const serviceScaleMap = reactive({})
const batchServiceScale = ref(1)
const stacks = ref([])
const stacksLoading = ref(false)
const stackTableRef = ref(null)
const selectedStacks = ref([])

const diagnoseVisible = ref(false)
const diagnoseLoading = ref(false)
const diagnoseResult = ref(null)
const diagnoseError = ref('')

const logVisible = ref(false)
const logLoading = ref(false)
const logText = ref('')
const logTail = ref('100')
const logContainerId = ref('')
const logFollow = ref(false)
let logTimer = null
let logSince = 0

const inspectVisible = ref(false)
const inspectLoading = ref(false)
const inspectData = ref(null)
const inspectPorts = ref([])
const inspectNetworks = ref([])
const inspectMounts = ref([])
const inspectEnvText = ref('')

const execVisible = ref(false)
const execLoading = ref(false)
const execCommand = ref('ls /')
const execOutput = ref('')
const execContainerId = ref('')

const serviceVisible = ref(false)
const serviceLoading = ref(false)
const serviceJson = ref('')
const tasksVisible = ref(false)
const tasksLoading = ref(false)
const serviceTasks = ref([])
const serviceLogVisible = ref(false)
const serviceLogLoading = ref(false)
const serviceLogText = ref('')
const serviceLogTail = ref('200')
const serviceLogFollow = ref(false)
let serviceLogTimer = null
let serviceLogSince = 0
const serviceLogTargetId = ref('')

const stackVisible = ref(false)
const stackLoading = ref(false)
const stackServices = ref([])

const serviceStacks = computed(() => {
  const set = new Set()
  services.value.forEach((s) => {
    const name = s.Name || ''
    const parts = name.split('_')
    if (parts.length > 1) set.add(parts[0])
  })
  return Array.from(set)
})

const filteredServices = computed(() => {
  if (!serviceStackFilter.value) return services.value
  return services.value.filter(s => (s.Name || '').startsWith(`${serviceStackFilter.value}_`))
})

const topologyTree = computed(() => {
  const map = new Map()
  services.value.forEach((s) => {
    const name = s.Name || s.name || ''
    const stack = name.includes('_') ? name.split('_')[0] : 'default'
    const arr = map.get(stack) || []
    arr.push({
      id: name,
      label: `${name} (${s.Replicas || '-'})`
    })
    map.set(stack, arr)
  })
  const tree = []
  map.forEach((children, stack) => {
    tree.push({ id: stack, label: stack, children })
  })
  return tree
})

const createVisible = ref(false)
const createLoading = ref(false)
const createForm = reactive({
  image: '',
  name: '',
  ports: '',
  env: '',
  restart_policy: '',
  auto_remove: false
})

const pullVisible = ref(false)
const pullLoading = ref(false)
const pullImage = ref('')
const pullOutput = ref('')
const pruneVisible = ref(false)
const pruneOutput = ref('')
const batchResultVisible = ref(false)
const batchResultRows = ref([])

const createVolumeVisible = ref(false)
const volumeLoading = ref(false)
const volumeInspectVisible = ref(false)
const volumeInspectJson = ref('')
const secretInspectVisible = ref(false)
const secretInspectJson = ref('')
const configInspectVisible = ref(false)
const configInspectJson = ref('')
const volumeForm = reactive({
  name: '',
  driver: '',
  labels: ''
})

const createSecretVisible = ref(false)
const secretLoading = ref(false)
const secretForm = reactive({
  name: '',
  data: ''
})

const createConfigVisible = ref(false)
const configLoading = ref(false)
const configForm = reactive({
  name: '',
  data: ''
})

const deployVisible = ref(false)
const deployLoading = ref(false)
const deployOutput = ref('')
const deployForm = reactive({
  name: '',
  compose: ''
})

const form = reactive({
  name: '',
  host_id: ''
})

const authHeaders = () => ({ Authorization: 'Bearer ' + localStorage.getItem('token') })

const normalizeContainers = (items) => items.map((row) => {
  const id = row.ID || row.Id || row.id
  const namesRaw = row.Names || row.names || row.Name || row.name
  const names = Array.isArray(namesRaw) ? namesRaw.join(',') : (namesRaw || '-')
  const createdAt = row.CreatedAt || row.created_at || row.Created || row.created || ''
  return {
    id,
    names,
    image: row.Image || row.image || '-',
    imageId: row.ImageID || row.ImageId || row.image_id || row.imageID || '',
    state: row.State || row.state || '-',
    status: row.Status || row.status || '-',
    created: createdAt || '-'
  }
})

const normalizeImages = (items) => items.map((row) => {
  const createdAt = row.CreatedAt || row.created_at || row.createdAt || ''
  const createdSince = row.CreatedSince || row.created_since || row.createdSince || row.created || ''
  return {
    id: row.ID || row.Id || row.id || '-',
    repository: row.Repository || row.repository || '-',
    tag: row.Tag || row.tag || '-',
    size: row.Size || row.size || '-',
    created_at: createdAt,
    created_since: createdSince,
    created: createdAt || createdSince || '-'
  }
})

const normalizeNetworks = (items) => items.map((row) => ({
  id: row.ID || row.Id || row.id || '-',
  name: row.Name || row.name || '-',
  driver: row.Driver || row.driver || '-',
  scope: row.Scope || row.scope || '-'
}))

const parseSizeToMB = (value) => {
  if (!value) return NaN
  const text = String(value).trim()
  const match = text.match(/^([0-9.]+)\s*([KMGTP]?B)$/i)
  if (!match) return NaN
  const num = Number(match[1])
  if (Number.isNaN(num)) return NaN
  const unit = match[2].toUpperCase()
  switch (unit) {
    case 'KB': return num / 1024
    case 'MB': return num
    case 'GB': return num * 1024
    case 'TB': return num * 1024 * 1024
    case 'PB': return num * 1024 * 1024 * 1024
    default: return num / (1024 * 1024)
  }
}

const formatTime = (value) => {
  if (!value) return '-'
  if (typeof value === 'number') {
    const ts = value > 1e12 ? value : value * 1000
    const d = new Date(ts)
    if (!Number.isNaN(d.getTime())) return formatDate(d)
  }
  const text = String(value).trim()
  if (!text) return '-'
  if (/^\d{10,13}$/.test(text)) {
    const num = Number(text)
    const ts = text.length === 10 ? num * 1000 : num
    const d = new Date(ts)
    if (!Number.isNaN(d.getTime())) return formatDate(d)
  }
  const d = new Date(text)
  if (!Number.isNaN(d.getTime())) return formatDate(d)
  return text
}

const formatDate = (d) => {
  const pad = (v) => String(v).padStart(2, '0')
  const y = d.getFullYear()
  const m = pad(d.getMonth() + 1)
  const day = pad(d.getDate())
  const h = pad(d.getHours())
  const mi = pad(d.getMinutes())
  const s = pad(d.getSeconds())
  return `${y}-${m}-${day} ${h}:${mi}:${s}`
}

const isDanglingImage = (row) => {
  const repo = (row.repository || '').toLowerCase()
  const tag = (row.tag || '').toLowerCase()
  return repo === '<none>' || tag === '<none>'
}

const filteredImages = computed(() => {
  let list = images.value.slice()
  const kw = imageFilters.keyword.trim().toLowerCase()
  if (kw) {
    list = list.filter((row) => {
      const repo = String(row.repository || '').toLowerCase()
      const tag = String(row.tag || '').toLowerCase()
      const id = String(row.id || '').toLowerCase()
      return repo.includes(kw) || tag.includes(kw) || id.includes(kw)
    })
  }
  if (imageFilters.danglingOnly) {
    list = list.filter(isDanglingImage)
  }
  const min = parseSizeToMB(imageFilters.minSize)
  if (!Number.isNaN(min)) {
    list = list.filter(row => parseSizeToMB(row.size) >= min)
  }
  const max = parseSizeToMB(imageFilters.maxSize)
  if (!Number.isNaN(max)) {
    list = list.filter(row => parseSizeToMB(row.size) <= max)
  }
  return list
})

const fetchData = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/docker/hosts', { headers: authHeaders() })
    if (res.data.code === 0) {
      tableData.value = res.data.data
    }
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    loading.value = false
  }
}

const syncAll = async () => {
  loading.value = true
  try {
    await axios.post('/api/v1/docker/hosts/sync', {}, { headers: authHeaders() })
  } catch (e) {
    ElMessage.error('同步失败')
  } finally {
    await fetchData()
  }
}

const fetchCMDBHosts = async () => {
  try {
    const res = await axios.get('/api/v1/cmdb/hosts', { headers: authHeaders() })
    if (res.data.code === 0) {
      hosts.value = res.data.data
    }
  } catch (e) {}
}

const handleAdd = () => {
  fetchCMDBHosts()
  form.name = ''
  form.host_id = ''
  dialogVisible.value = true
}

const submitForm = async () => {
  submitting.value = true
  try {
    const res = await axios.post('/api/v1/docker/hosts', form, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('添加成功')
      dialogVisible.value = false
      try {
        const id = res.data.data?.id
        if (id) {
          await axios.get(`/api/v1/docker/hosts/${id}/info`, { headers: authHeaders() })
        }
      } catch (e) {}
      fetchData()
    } else {
      ElMessage.error(res.data.message)
    }
  } finally {
    submitting.value = false
  }
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确定删除该 Docker 环境吗?', '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    await axios.delete(`/api/v1/docker/hosts/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    fetchData()
  })
}

const handleManage = async (row) => {
  activeHost.value = row
  manageVisible.value = true
  manageTab.value = 'overview'
  await refreshManage()
}

const refreshManage = async () => {
  if (!activeHost.value) return
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/info`, { headers: authHeaders() })
    if (res.data.code === 0) {
      const idx = tableData.value.findIndex(h => h.id === activeHost.value.id)
      if (idx >= 0) tableData.value[idx] = res.data.data
      activeHost.value = res.data.data
    }
  } catch (e) {}
  if (manageTab.value === 'containers') {
    await loadContainers()
    await loadContainerStats()
  }
  if (manageTab.value === 'images') await loadImages()
  if (manageTab.value === 'networks') await loadNetworks()
  if (manageTab.value === 'volumes') await loadVolumes()
  if (manageTab.value === 'secrets') await loadSecrets()
  if (manageTab.value === 'configs') await loadConfigs()
  if (manageTab.value === 'registries') await loadRegistries()
  if (manageTab.value === 'nodes') await loadNodes()
  if (manageTab.value === 'topology') await loadServices()
  if (manageTab.value === 'services') await loadServices()
  if (manageTab.value === 'stacks') await loadStacks()
}

const loadContainers = async () => {
  if (!activeHost.value) return
  containersLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers`, { headers: authHeaders() })
    if (res.data.code === 0) {
      containers.value = normalizeContainers(res.data.data || [])
      selectedContainers.value = []
      containerTableRef.value?.clearSelection?.()
    }
  } finally {
    containersLoading.value = false
  }
}

const loadContainerStats = async () => {
  if (!activeHost.value) return
  statsLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers/stats`, { headers: authHeaders() })
    if (res.data.code === 0) {
      const map = {}
      const list = res.data.data || []
      list.forEach((row) => {
        const id = row.Container || row.ID || row.Id || row.id
        const name = row.Name || row.name
        const item = {
          cpu: row.CPUPerc || row.CPU || '-',
          mem: row.MemUsage || row.Memory || '-',
          net: row.NetIO || row.Network || '-'
        }
        if (id) map[id] = item
        if (name) map[name] = item
      })
      containerStats.value = map
    }
  } finally {
    statsLoading.value = false
  }
}

const getContainerStats = (row) => {
  const id = row.id
  const names = (row.names || '').split(',').map(v => v.trim()).filter(Boolean)
  if (id && containerStats.value[id]) return containerStats.value[id]
  for (const n of names) {
    if (containerStats.value[n]) return containerStats.value[n]
  }
  return { cpu: '-', mem: '-', net: '-' }
}

const autoRefreshStats = ref(false)
let statsTimer = null

watch(autoRefreshStats, (val) => {
  if (val) {
    loadContainerStats()
    if (statsTimer) clearInterval(statsTimer)
    statsTimer = setInterval(() => {
      if (manageTab.value === 'containers') {
        loadContainerStats()
      }
    }, 5000)
  } else if (statsTimer) {
    clearInterval(statsTimer)
    statsTimer = null
  }
})

const loadImages = async () => {
  if (!activeHost.value) return
  imagesLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/images`, { headers: authHeaders() })
    if (res.data.code === 0) {
      images.value = normalizeImages(res.data.data || [])
      selectedImages.value = []
      imageTableRef.value?.clearSelection?.()
    }
  } finally {
    imagesLoading.value = false
  }
}

const loadNetworks = async () => {
  if (!activeHost.value) return
  networksLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/networks`, { headers: authHeaders() })
    if (res.data.code === 0) {
      networks.value = normalizeNetworks(res.data.data || [])
      selectedNetworks.value = []
      networkTableRef.value?.clearSelection?.()
    }
  } finally {
    networksLoading.value = false
  }
}

const loadVolumes = async () => {
  if (!activeHost.value) return
  volumesLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/volumes`, { headers: authHeaders() })
    if (res.data.code === 0) {
      volumes.value = res.data.data || []
      selectedVolumes.value = []
      volumeTableRef.value?.clearSelection?.()
    }
  } finally {
    volumesLoading.value = false
  }
}

const loadSecrets = async () => {
  if (!activeHost.value) return
  secretsLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/secrets`, { headers: authHeaders() })
    if (res.data.code === 0) {
      secrets.value = res.data.data || []
      selectedSecrets.value = []
      secretTableRef.value?.clearSelection?.()
    }
  } finally {
    secretsLoading.value = false
  }
}

const loadConfigs = async () => {
  if (!activeHost.value) return
  configsLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/configs`, { headers: authHeaders() })
    if (res.data.code === 0) {
      configs.value = res.data.data || []
      selectedConfigs.value = []
      configTableRef.value?.clearSelection?.()
    }
  } finally {
    configsLoading.value = false
  }
}

const loadNodes = async () => {
  if (!activeHost.value) return
  nodesLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/nodes`, { headers: authHeaders() })
    if (res.data.code === 0) {
      nodes.value = res.data.data || []
    }
  } finally {
    nodesLoading.value = false
  }
}

const loadRegistries = async () => {
  registriesLoading.value = true
  try {
    const url = activeHost.value
      ? `/api/v1/docker/hosts/${activeHost.value.id}/registries`
      : '/api/v1/docker/registries'
    const res = await axios.get(url, { headers: authHeaders() })
    if (res.data.code === 0) {
      registries.value = res.data.data || []
    }
  } finally {
    registriesLoading.value = false
  }
}

const loadServices = async () => {
  if (!activeHost.value) return
  servicesLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services`, { headers: authHeaders() })
    if (res.data.code === 0) {
      const list = res.data.data || []
      services.value = list.map((row) => {
        const name = row.Name || row.name || ''
        const stack = name.includes('_') ? name.split('_')[0] : ''
        return { ...row, Stack: stack }
      })
      selectedServices.value = []
      serviceTableRef.value?.clearSelection?.()
      services.value.forEach((row) => {
        const id = row.ID || row.Id || row.id
        const rep = String(row.Replicas || '')
        const match = rep.match(/(\d+)\s*\/\s*(\d+)/)
        if (id) {
          serviceScaleMap[id] = match ? Number(match[2]) : (Number(rep) || 0)
        }
      })
    }
  } finally {
    servicesLoading.value = false
  }
}

const loadStacks = async () => {
  if (!activeHost.value) return
  stacksLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/stacks`, { headers: authHeaders() })
    if (res.data.code === 0) {
      stacks.value = res.data.data || []
      selectedStacks.value = []
      stackTableRef.value?.clearSelection?.()
    }
  } finally {
    stacksLoading.value = false
  }
}

const clearImageFilters = () => {
  imageFilters.keyword = ''
  imageFilters.minSize = ''
  imageFilters.maxSize = ''
  imageFilters.danglingOnly = false
}

const formatImageLabel = (row) => {
  const repo = row.repository || ''
  const tag = row.tag || ''
  if (repo && repo !== '<none>' && tag && tag !== '<none>') {
    return `${repo}:${tag}`
  }
  return row.id || repo || '-'
}

const extractErrorMessage = (e) => {
  const msg = e?.response?.data?.message || e?.message || '操作失败'
  return String(msg)
}

const onContainerSelectionChange = (rows) => {
  selectedContainers.value = rows || []
}

const onNetworkSelectionChange = (rows) => {
  selectedNetworks.value = rows || []
}

const onStackSelectionChange = (rows) => {
  selectedStacks.value = rows || []
}

const onServiceSelectionChange = (rows) => {
  selectedServices.value = rows || []
}

const onVolumeSelectionChange = (rows) => {
  selectedVolumes.value = rows || []
}

const onSecretSelectionChange = (rows) => {
  selectedSecrets.value = rows || []
}

const onConfigSelectionChange = (rows) => {
  selectedConfigs.value = rows || []
}

const openCreateVolume = () => {
  volumeForm.name = ''
  volumeForm.driver = ''
  volumeForm.labels = ''
  createVolumeVisible.value = true
}

const submitCreateVolume = async () => {
  if (!activeHost.value || !volumeForm.name.trim()) {
    ElMessage.warning('请填写名称')
    return
  }
  const labels = {}
  volumeForm.labels.split('\n').map(v => v.trim()).filter(Boolean).forEach((line) => {
    const idx = line.indexOf('=')
    if (idx > 0) {
      labels[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
    }
  })
  volumeLoading.value = true
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/volumes`, {
      name: volumeForm.name.trim(),
      driver: volumeForm.driver.trim(),
      labels
    }, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('创建成功')
      createVolumeVisible.value = false
      loadVolumes()
    } else {
      ElMessage.error(res.data.message || '创建失败')
    }
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  } finally {
    volumeLoading.value = false
  }
}

const openVolumeInspect = async (row) => {
  if (!activeHost.value || !(row?.Name || row?.name)) return
  const name = row.Name || row.name
  volumeInspectVisible.value = true
  volumeInspectJson.value = ''
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/volumes/${encodeURIComponent(name)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      volumeInspectJson.value = JSON.stringify(res.data.data, null, 2)
    }
  } catch (e) {
    volumeInspectJson.value = extractErrorMessage(e)
  }
}

const removeSelectedVolumes = async () => {
  if (!activeHost.value) return
  const rows = selectedVolumes.value.filter(r => r.Name || r.name)
  if (rows.length === 0) {
    ElMessage.warning('请选择卷')
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${rows.length} 个卷吗?`, '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    const name = row.Name || row.name
    try {
      await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/volumes/${encodeURIComponent(name)}`, { headers: authHeaders() })
      ok += 1
      results.push({ label: name, status: '成功', message: '已删除' })
    } catch (e) {
      fail += 1
      results.push({ label: name, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已删除 ${ok} 个卷`)
  } else {
    ElMessage.warning(`已删除 ${ok} 个，失败 ${fail} 个`)
  }
  loadVolumes()
}

const openCreateSecret = () => {
  secretForm.name = ''
  secretForm.data = ''
  createSecretVisible.value = true
}

const submitCreateSecret = async () => {
  if (!activeHost.value || !secretForm.name.trim()) {
    ElMessage.warning('请填写名称')
    return
  }
  secretLoading.value = true
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/secrets`, {
      name: secretForm.name.trim(),
      data: secretForm.data || ''
    }, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('创建成功')
      createSecretVisible.value = false
      loadSecrets()
    } else {
      ElMessage.error(res.data.message || '创建失败')
    }
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  } finally {
    secretLoading.value = false
  }
}

const removeSelectedSecrets = async () => {
  if (!activeHost.value) return
  const rows = selectedSecrets.value.filter(r => r.Name || r.name || r.ID)
  if (rows.length === 0) {
    ElMessage.warning('请选择Secret')
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${rows.length} 个 Secret 吗?`, '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    const name = row.Name || row.name || row.ID
    try {
      await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/secrets/${encodeURIComponent(name)}`, { headers: authHeaders() })
      ok += 1
      results.push({ label: name, status: '成功', message: '已删除' })
    } catch (e) {
      fail += 1
      results.push({ label: name, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已删除 ${ok} 个 Secret`)
  } else {
    ElMessage.warning(`已删除 ${ok} 个，失败 ${fail} 个`)
  }
  loadSecrets()
}

const openSecretInspect = async (row) => {
  if (!activeHost.value) return
  const name = row.Name || row.name || row.ID
  if (!name) return
  secretInspectVisible.value = true
  secretInspectJson.value = ''
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/secrets/${encodeURIComponent(name)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      secretInspectJson.value = JSON.stringify(res.data.data, null, 2)
    } else {
      secretInspectJson.value = res.data.message || '加载失败'
    }
  } catch (e) {
    secretInspectJson.value = extractErrorMessage(e)
  }
}

const openCreateConfig = () => {
  configForm.name = ''
  configForm.data = ''
  createConfigVisible.value = true
}

const submitCreateConfig = async () => {
  if (!activeHost.value || !configForm.name.trim()) {
    ElMessage.warning('请填写名称')
    return
  }
  configLoading.value = true
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/configs`, {
      name: configForm.name.trim(),
      data: configForm.data || ''
    }, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('创建成功')
      createConfigVisible.value = false
      loadConfigs()
    } else {
      ElMessage.error(res.data.message || '创建失败')
    }
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  } finally {
    configLoading.value = false
  }
}

const removeSelectedConfigs = async () => {
  if (!activeHost.value) return
  const rows = selectedConfigs.value.filter(r => r.Name || r.name || r.ID)
  if (rows.length === 0) {
    ElMessage.warning('请选择Config')
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${rows.length} 个 Config 吗?`, '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    const name = row.Name || row.name || row.ID
    try {
      await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/configs/${encodeURIComponent(name)}`, { headers: authHeaders() })
      ok += 1
      results.push({ label: name, status: '成功', message: '已删除' })
    } catch (e) {
      fail += 1
      results.push({ label: name, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已删除 ${ok} 个 Config`)
  } else {
    ElMessage.warning(`已删除 ${ok} 个，失败 ${fail} 个`)
  }
  loadConfigs()
}

const openConfigInspect = async (row) => {
  if (!activeHost.value) return
  const name = row.Name || row.name || row.ID
  if (!name) return
  configInspectVisible.value = true
  configInspectJson.value = ''
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/configs/${encodeURIComponent(name)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      configInspectJson.value = JSON.stringify(res.data.data, null, 2)
    } else {
      configInspectJson.value = res.data.message || '加载失败'
    }
  } catch (e) {
    configInspectJson.value = extractErrorMessage(e)
  }
}

const openDeployStack = () => {
  deployForm.name = ''
  deployForm.compose = ''
  deployOutput.value = ''
  deployVisible.value = true
}

const submitDeployStack = async () => {
  if (!activeHost.value || !deployForm.name.trim() || !deployForm.compose.trim()) {
    ElMessage.warning('请填写名称和Compose')
    return
  }
  deployLoading.value = true
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/stacks/deploy`, {
      name: deployForm.name.trim(),
      compose: deployForm.compose
    }, { headers: authHeaders() })
    if (res.data.code === 0) {
      deployOutput.value = res.data.data?.output || '部署完成'
      loadStacks()
      loadServices()
    } else {
      deployOutput.value = res.data.message || '部署失败'
    }
  } catch (e) {
    deployOutput.value = extractErrorMessage(e)
  } finally {
    deployLoading.value = false
  }
}

const openCreateRegistry = () => {
  registryForm.name = ''
  registryForm.url = ''
  registryForm.username = ''
  registryForm.password = ''
  registryForm.insecure = false
  createRegistryVisible.value = true
}

const submitCreateRegistry = async () => {
  if (!registryForm.name.trim() || !registryForm.url.trim()) {
    ElMessage.warning('请填写名称和地址')
    return
  }
  registryLoading.value = true
  try {
    const res = await axios.post('/api/v1/docker/registries', {
      name: registryForm.name.trim(),
      url: registryForm.url.trim(),
      username: registryForm.username,
      password: registryForm.password,
      insecure: registryForm.insecure
    }, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('保存成功')
      createRegistryVisible.value = false
      loadRegistries()
    } else {
      ElMessage.error(res.data.message || '保存失败')
    }
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  } finally {
    registryLoading.value = false
  }
}

const removeRegistry = async (row) => {
  try {
    await ElMessageBox.confirm('确定删除该仓库吗?', '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  try {
    await axios.delete(`/api/v1/docker/registries/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    loadRegistries()
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  }
}

const loginRegistry = async (row) => {
  if (!activeHost.value) return
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/registries/${row.id}/login`, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('登录成功')
    } else {
      ElMessage.error(res.data.message || '登录失败')
    }
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  }
}

const applyServiceScale = async (row) => {
  if (!activeHost.value) return
  const id = row.ID || row.Id || row.id
  if (!id) return
  const replicas = Number(serviceScaleMap[id])
  if (Number.isNaN(replicas) || replicas < 0) {
    ElMessage.warning('副本数无效')
    return
  }
  try {
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(id)}/scale`, {
      replicas
    }, { headers: authHeaders() })
    ElMessage.success('已提交扩缩容')
    loadServices()
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  }
}

const scaleSelectedServices = async () => {
  if (!activeHost.value) return
  const rows = selectedServices.value.filter(r => r.ID || r.Id || r.id)
  if (rows.length === 0) {
    ElMessage.warning('请选择服务')
    return
  }
  const replicas = Number(batchServiceScale.value)
  if (Number.isNaN(replicas) || replicas < 0) {
    ElMessage.warning('副本数无效')
    return
  }
  try {
    await ElMessageBox.confirm(`确定将选中的 ${rows.length} 个服务副本调整为 ${replicas} 吗?`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    const id = row.ID || row.Id || row.id
    const name = row.Name || row.name || id
    try {
      await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(id)}/scale`, {
        replicas
      }, { headers: authHeaders() })
      ok += 1
      results.push({ label: name, status: '成功', message: `副本=${replicas}` })
    } catch (e) {
      fail += 1
      results.push({ label: name, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已调整 ${ok} 个服务`)
  } else {
    ElMessage.warning(`已调整 ${ok} 个，失败 ${fail} 个`)
  }
  loadServices()
}

const startSelectedContainers = async () => {
  if (!activeHost.value) return
  const rows = selectedContainers.value.filter(r => r.id)
  if (rows.length === 0) {
    ElMessage.warning('请选择容器')
    return
  }
  try {
    await ElMessageBox.confirm(`确定启动选中的 ${rows.length} 个容器吗?`, '提示', {
      confirmButtonText: '启动',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    try {
      await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(row.id)}/start`, {}, { headers: authHeaders() })
      ok += 1
      results.push({ label: row.names || row.id, status: '成功', message: '已启动' })
    } catch (e) {
      fail += 1
      results.push({ label: row.names || row.id, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已启动 ${ok} 个容器`)
  } else {
    ElMessage.warning(`已启动 ${ok} 个，失败 ${fail} 个`)
  }
  loadContainers()
}

const stopSelectedContainers = async () => {
  if (!activeHost.value) return
  const rows = selectedContainers.value.filter(r => r.id)
  if (rows.length === 0) {
    ElMessage.warning('请选择容器')
    return
  }
  try {
    await ElMessageBox.confirm(`确定停止选中的 ${rows.length} 个容器吗?`, '提示', {
      confirmButtonText: '停止',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    try {
      await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(row.id)}/stop`, {}, { headers: authHeaders() })
      ok += 1
      results.push({ label: row.names || row.id, status: '成功', message: '已停止' })
    } catch (e) {
      fail += 1
      results.push({ label: row.names || row.id, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已停止 ${ok} 个容器`)
  } else {
    ElMessage.warning(`已停止 ${ok} 个，失败 ${fail} 个`)
  }
  loadContainers()
}

const restartSelectedContainers = async () => {
  if (!activeHost.value) return
  const rows = selectedContainers.value.filter(r => r.id)
  if (rows.length === 0) {
    ElMessage.warning('请选择容器')
    return
  }
  try {
    await ElMessageBox.confirm(`确定重启选中的 ${rows.length} 个容器吗?`, '提示', {
      confirmButtonText: '重启',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    try {
      await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(row.id)}/restart`, {}, { headers: authHeaders() })
      ok += 1
      results.push({ label: row.names || row.id, status: '成功', message: '已重启' })
    } catch (e) {
      fail += 1
      results.push({ label: row.names || row.id, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已重启 ${ok} 个容器`)
  } else {
    ElMessage.warning(`已重启 ${ok} 个，失败 ${fail} 个`)
  }
  loadContainers()
}

const removeSelectedContainers = async () => {
  if (!activeHost.value) return
  const rows = selectedContainers.value.filter(r => r.id)
  if (rows.length === 0) {
    ElMessage.warning('请选择容器')
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${rows.length} 个容器吗?`, '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    try {
      await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(row.id)}/remove`, {}, { headers: authHeaders() })
      ok += 1
      results.push({ label: row.names || row.id, status: '成功', message: '已删除' })
    } catch (e) {
      fail += 1
      results.push({ label: row.names || row.id, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已删除 ${ok} 个容器`)
  } else {
    ElMessage.warning(`已删除 ${ok} 个，失败 ${fail} 个`)
  }
  loadContainers()
  refreshManage()
}

const removeSelectedNetworks = async () => {
  if (!activeHost.value) return
  const rows = selectedNetworks.value.filter(r => r.id)
  if (rows.length === 0) {
    ElMessage.warning('请选择网络')
    return
  }
  try {
    await confirmDeleteNetworks(rows)
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    try {
      await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/networks/${encodeURIComponent(row.id)}`, { headers: authHeaders() })
      ok += 1
      results.push({ label: row.name || row.id, status: '成功', message: '已删除' })
    } catch (e) {
      fail += 1
      results.push({ label: row.name || row.id, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已删除 ${ok} 个网络`)
  } else {
    ElMessage.warning(`已删除 ${ok} 个，失败 ${fail} 个`)
  }
  loadNetworks()
}

const restartSelectedServices = async () => {
  if (!activeHost.value) return
  const rows = selectedServices.value.filter(r => r.ID || r.Id || r.id)
  if (rows.length === 0) {
    ElMessage.warning('请选择服务')
    return
  }
  try {
    await ElMessageBox.confirm(`确定重启选中的 ${rows.length} 个服务吗?`, '提示', {
      confirmButtonText: '重启',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    const id = row.ID || row.Id || row.id
    const name = row.Name || row.name || id
    try {
      await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(id)}/restart`, {}, { headers: authHeaders() })
      ok += 1
      results.push({ label: name, status: '成功', message: '已重启' })
    } catch (e) {
      fail += 1
      results.push({ label: name, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已重启 ${ok} 个服务`)
  } else {
    ElMessage.warning(`已重启 ${ok} 个，失败 ${fail} 个`)
  }
  loadServices()
}

const removeSelectedStacks = async () => {
  if (!activeHost.value) return
  const rows = selectedStacks.value.filter(r => r.Name || r.name)
  if (rows.length === 0) {
    ElMessage.warning('请选择Stack')
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${rows.length} 个Stack吗?`, '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  const results = []
  let ok = 0
  let fail = 0
  for (const row of rows) {
    const stackName = row.Name || row.name
    try {
      await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/stacks/${encodeURIComponent(stackName)}`, { headers: authHeaders() })
      ok += 1
      results.push({ label: stackName, status: '成功', message: '已删除' })
    } catch (e) {
      fail += 1
      results.push({ label: stackName, status: '失败', message: extractErrorMessage(e) })
    }
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  if (fail === 0) {
    ElMessage.success(`已删除 ${ok} 个Stack`)
  } else {
    ElMessage.warning(`已删除 ${ok} 个，失败 ${fail} 个`)
  }
  loadStacks()
  loadServices()
}

const fetchContainersForUsage = async () => {
  if (!activeHost.value) return []
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers`, { headers: authHeaders() })
    if (res.data.code === 0) {
      return normalizeContainers(res.data.data || [])
    }
  } catch (e) {}
  return []
}

const buildUsageMap = (list) => {
  const idMap = new Map()
  const nameMap = new Map()
  list.forEach((c) => {
    const name = c.names || c.id || '-'
    const imageId = (c.imageId || '').replace(/^sha256:/, '')
    if (imageId) {
      const arr = idMap.get(imageId) || []
      arr.push(name)
      idMap.set(imageId, arr)
    }
    const imageName = c.image || ''
    if (imageName) {
      const arr = nameMap.get(imageName) || []
      arr.push(name)
      nameMap.set(imageName, arr)
    }
  })
  return { idMap, nameMap }
}

const getImageUsage = (rows, usage) => {
  const result = []
  rows.forEach((row) => {
    const label = formatImageLabel(row)
    const imageId = String(row.id || '').replace(/^sha256:/, '')
    let containers = []
    if (imageId) {
      usage.idMap.forEach((names, key) => {
        if (key.startsWith(imageId)) {
          containers = containers.concat(names)
        }
      })
    }
    if (containers.length === 0) {
      const repo = row.repository || ''
      const tag = row.tag || ''
      if (repo && tag && repo !== '<none>' && tag !== '<none>') {
        const imageName = `${repo}:${tag}`
        containers = (usage.nameMap.get(imageName) || []).slice()
      }
    }
    if (containers.length > 0) {
      result.push({ label, containers })
    }
  })
  return result
}

const confirmDeleteNetworks = async (rows) => {
  if (!activeHost.value) return
  const used = []
  for (const row of rows) {
    try {
      const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/networks/${encodeURIComponent(row.id)}`, { headers: authHeaders() })
      if (res.data.code === 0) {
        const info = res.data.data || {}
        const containers = info.Containers || {}
        const names = Object.values(containers).map(c => c?.Name).filter(Boolean)
        if (names.length > 0) {
          used.push({ label: row.name || row.id, containers: names })
        }
      }
    } catch (e) {}
  }
  let message = `确定删除选中的 ${rows.length} 个网络吗?`
  if (used.length > 0) {
    const lines = used.map(u => `${u.label} -> ${u.containers.join(', ')}`).join('\n')
    message = `以下网络仍有容器连接，删除可能失败：\n${lines}\n\n是否继续?`
  }
  await ElMessageBox.confirm(message, '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  })
}

const confirmDeleteImages = async (rows) => {
  const containers = await fetchContainersForUsage()
  const usageMap = buildUsageMap(containers)
  const used = getImageUsage(rows, usageMap)
  let message = `确定删除选中的 ${rows.length} 个镜像吗?`
  if (used.length > 0) {
    const lines = used.map((u) => `${u.label} -> ${u.containers.join(', ')}`).join('\n')
    message = `以下镜像正在被容器使用，删除可能失败：\n${lines}\n\n是否继续?`
  }
  await ElMessageBox.confirm(message, '警告', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  })
}

const openCreateContainer = () => {
  createForm.image = ''
  createForm.name = ''
  createForm.ports = ''
  createForm.env = ''
  createForm.restart_policy = ''
  createForm.auto_remove = false
  createVisible.value = true
}

const submitCreate = async () => {
  if (!activeHost.value || !createForm.image) {
    ElMessage.warning('请填写镜像')
    return
  }
  createLoading.value = true
  try {
    const ports = createForm.ports
      .split(',')
      .map(v => v.trim())
      .filter(Boolean)
    const env = {}
    createForm.env.split('\n').map(v => v.trim()).filter(Boolean).forEach((line) => {
      const idx = line.indexOf('=')
      if (idx > 0) {
        const k = line.slice(0, idx).trim()
        const v = line.slice(idx + 1).trim()
        env[k] = v
      }
    })
    const payload = {
      name: createForm.name,
      image: createForm.image,
      ports,
      env,
      restart_policy: createForm.restart_policy,
      auto_remove: createForm.auto_remove
    }
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers`, payload, { headers: authHeaders() })
    if (res.data.code === 0) {
      ElMessage.success('容器创建成功')
      createVisible.value = false
      loadContainers()
      refreshManage()
    } else {
      ElMessage.error(res.data.message || '创建失败')
    }
  } catch (e) {
    ElMessage.error('创建失败')
  } finally {
    createLoading.value = false
  }
}

const containerAction = async (row, action) => {
  if (!activeHost.value) return
  const id = row.id
  if (!id) return
  try {
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(id)}/${action}`, {}, { headers: authHeaders() })
    ElMessage.success('操作成功')
    loadContainers()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

const openLogs = (row) => {
  const id = row.id
  if (!id) return
  logContainerId.value = id
  logText.value = ''
  logVisible.value = true
  loadLogs()
}

const openInspect = async (row) => {
  const id = row.id
  if (!activeHost.value || !id) return
  inspectVisible.value = true
  inspectLoading.value = true
  inspectData.value = null
  inspectPorts.value = []
  inspectNetworks.value = []
  inspectMounts.value = []
  inspectEnvText.value = ''
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(id)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      inspectData.value = res.data.data || null
      const ports = []
      const portMap = inspectData.value?.NetworkSettings?.Ports || {}
      Object.entries(portMap).forEach(([containerPort, hostBindings]) => {
        if (!hostBindings || hostBindings.length === 0) {
          ports.push({ container: containerPort, host: '-', ip: '-' })
          return
        }
        hostBindings.forEach((b) => {
          ports.push({ container: containerPort, host: b.HostPort || '-', ip: b.HostIp || '-' })
        })
      })
      inspectPorts.value = ports

      const networks = []
      const nets = inspectData.value?.NetworkSettings?.Networks || {}
      Object.entries(nets).forEach(([name, info]) => {
        networks.push({ name, ip: info.IPAddress || '-', gateway: info.Gateway || '-' })
      })
      inspectNetworks.value = networks

      const mounts = (inspectData.value?.Mounts || []).map(m => ({
        type: m.Type,
        source: m.Source,
        destination: m.Destination,
        mode: m.Mode,
        rw: m.RW ? 'true' : 'false'
      }))
      inspectMounts.value = mounts

      const env = inspectData.value?.Config?.Env || []
      inspectEnvText.value = env.join('\n')
    }
  } finally {
    inspectLoading.value = false
  }
}

const openExec = (row) => {
  const id = row.id
  if (!id) return
  execContainerId.value = id
  execCommand.value = 'ls /'
  execOutput.value = ''
  execVisible.value = true
}

const openTerminal = async (row) => {
  const id = row.id
  if (!id) return
  terminalContainerId.value = id
  terminalContainerName.value = row.names || id
  terminalShell.value = '/bin/sh'
  terminalVisible.value = true
  await nextTick()
  initTerminal()
  terminalFit?.fit()
  sendTerminalResize()
}

const initTerminal = () => {
  if (terminal) return
  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    theme: {
      background: '#0f172a',
      foreground: '#e2e8f0'
    }
  })
  terminalFit = new FitAddon()
  terminal.loadAddon(terminalFit)
  if (terminalRef.value) {
    terminal.open(terminalRef.value)
    terminalFit.fit()
  }
  terminal.onData((data) => {
    if (terminalWs && terminalWs.readyState === WebSocket.OPEN) {
      terminalWs.send(data)
    }
  })
}

const sendTerminalResize = () => {
  if (!terminalWs || terminalWs.readyState !== WebSocket.OPEN || !terminal) return
  terminalWs.send(JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }))
}

const connectTerminal = () => {
  if (!terminalContainerId.value || !activeHost.value) return
  const token = localStorage.getItem('token') || ''
  const wsProto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const wsUrl = `${wsProto}://${window.location.host}/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(terminalContainerId.value)}/exec/ws?token=${encodeURIComponent(token)}&shell=${encodeURIComponent(terminalShell.value)}`
  terminalWs = new WebSocket(wsUrl)
  terminalWs.binaryType = 'arraybuffer'
  terminalWs.onopen = () => {
    terminalConnected.value = true
    terminal?.writeln('连接成功。')
    sendTerminalResize()
  }
  terminalWs.onmessage = (evt) => {
    if (evt.data instanceof ArrayBuffer) {
      const text = new TextDecoder().decode(new Uint8Array(evt.data))
      terminal?.write(text)
      return
    }
    terminal?.write(evt.data)
  }
  terminalWs.onclose = () => {
    terminalConnected.value = false
    terminal?.writeln('\r\n连接已关闭。')
  }
  terminalWs.onerror = () => {
    ElMessage.error('连接失败')
  }
}

const closeTerminal = () => {
  if (terminalWs) {
    terminalWs.close()
    terminalWs = null
  }
  terminalConnected.value = false
}

const toggleTerminal = () => {
  if (terminalConnected.value) closeTerminal()
  else connectTerminal()
}

const copyText = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch (e) {
    ElMessage.warning('复制失败')
  }
}

const downloadJson = (filename, content) => {
  try {
    const blob = new Blob([content || ''], { type: 'application/json;charset=utf-8' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(link.href)
  } catch (e) {
    ElMessage.error('导出失败')
  }
}

const runExec = async () => {
  if (!activeHost.value || !execContainerId.value || !execCommand.value) {
    ElMessage.warning('请输入命令')
    return
  }
  execLoading.value = true
  try {
    const res = await axios.post(
      `/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(execContainerId.value)}/exec`,
      { command: execCommand.value },
      { headers: authHeaders() }
    )
    if (res.data.code === 0) {
      execOutput.value = res.data.data || ''
    } else {
      execOutput.value = res.data.message || '执行失败'
    }
  } catch (e) {
    execOutput.value = '执行失败'
  } finally {
    execLoading.value = false
  }
}

const loadLogs = async () => {
  if (!activeHost.value || !logContainerId.value) return
  logLoading.value = true
  try {
    const params = { tail: logTail.value || '100' }
    if (logFollow.value && logSince > 0) {
      params.since = logSince
      params.timestamps = 1
    }
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(logContainerId.value)}/logs`, {
      params,
      headers: authHeaders()
    })
    if (res.data.code === 0) {
      if (logFollow.value) {
        logText.value += (res.data.data || '')
      } else {
        logText.value = res.data.data || ''
      }
    }
  } finally {
    if (logFollow.value) {
      logSince = Math.floor(Date.now() / 1000)
    }
    logLoading.value = false
  }
}

const clearLogs = () => {
  logText.value = ''
}

watch(logFollow, (val) => {
  if (val) {
    logSince = Math.floor(Date.now() / 1000)
    loadLogs()
    if (logTimer) clearInterval(logTimer)
    logTimer = setInterval(() => {
      if (logVisible.value) loadLogs()
    }, 2000)
  } else if (logTimer) {
    clearInterval(logTimer)
    logTimer = null
  }
})

const openServiceDetail = async (row) => {
  if (!activeHost.value || !row?.ID) return
  serviceVisible.value = true
  serviceLoading.value = true
  serviceJson.value = ''
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      serviceJson.value = JSON.stringify(res.data.data, null, 2)
    }
  } finally {
    serviceLoading.value = false
  }
}

const openServiceTasks = async (row) => {
  if (!activeHost.value || !row?.ID) return
  tasksVisible.value = true
  tasksLoading.value = true
  serviceTasks.value = []
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/tasks`, { headers: authHeaders() })
    if (res.data.code === 0) {
      serviceTasks.value = res.data.data || []
    }
  } finally {
    tasksLoading.value = false
  }
}

const openServiceLogs = (row) => {
  if (!activeHost.value) return
  const id = row.ID || row.Id || row.id
  if (!id) return
  serviceLogText.value = ''
  serviceLogVisible.value = true
  serviceLogSince = 0
  serviceLogTargetId.value = id
  loadServiceLogs()
}

const loadServiceLogs = async () => {
  if (!activeHost.value || !serviceLogTargetId.value) return
  serviceLogLoading.value = true
  try {
    const params = { tail: serviceLogTail.value || '200' }
    if (serviceLogFollow.value && serviceLogSince > 0) {
      params.since = serviceLogSince
      params.timestamps = 1
    }
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(serviceLogTargetId.value)}/logs`, {
      params,
      headers: authHeaders()
    })
    if (res.data.code === 0) {
      if (serviceLogFollow.value) {
        serviceLogText.value += (res.data.data || '')
      } else {
        serviceLogText.value = res.data.data || ''
      }
    }
  } finally {
    if (serviceLogFollow.value) {
      serviceLogSince = Math.floor(Date.now() / 1000)
    }
    serviceLogLoading.value = false
  }
}

const clearServiceLogs = () => {
  serviceLogText.value = ''
}

watch(serviceLogFollow, (val) => {
  if (val) {
    serviceLogSince = Math.floor(Date.now() / 1000)
    loadServiceLogs()
    if (serviceLogTimer) clearInterval(serviceLogTimer)
    serviceLogTimer = setInterval(() => {
      if (serviceLogVisible.value) loadServiceLogs()
    }, 2000)
  } else if (serviceLogTimer) {
    clearInterval(serviceLogTimer)
    serviceLogTimer = null
  }
})

const openStackServices = async (row) => {
  if (!activeHost.value || !row?.Name) return
  stackVisible.value = true
  stackLoading.value = true
  stackServices.value = []
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/stacks/${encodeURIComponent(row.Name)}/services`, { headers: authHeaders() })
    if (res.data.code === 0) {
      stackServices.value = res.data.data || []
    }
  } finally {
    stackLoading.value = false
  }
}

const scaleService = async (row) => {
  if (!activeHost.value || !row?.ID) return
  try {
    const { value } = await ElMessageBox.prompt('输入副本数', '扩缩容', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputPattern: /^[0-9]+$/,
      inputErrorMessage: '请输入数字'
    })
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/scale`, {
      replicas: Number(value)
    }, { headers: authHeaders() })
    ElMessage.success('已提交扩缩容')
    loadServices()
  } catch (e) {}
}

const updateServiceImage = async (row) => {
  if (!activeHost.value || !row?.ID) return
  try {
    const { value } = await ElMessageBox.prompt('输入镜像 (如 nginx:latest)', '更新镜像', {
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    })
    if (!value) return
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/update_image`, {
      image: value
    }, { headers: authHeaders() })
    ElMessage.success('已提交镜像更新')
    loadServices()
  } catch (e) {}
}

const restartService = async (row) => {
  if (!activeHost.value || !row?.ID) return
  try {
    await ElMessageBox.confirm('确认滚动重启该服务吗？', '提示', { type: 'warning' })
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/restart`, {}, { headers: authHeaders() })
    ElMessage.success('已触发重启')
    loadServices()
  } catch (e) {}
}

const openPullImage = () => {
  pullImage.value = ''
  pullOutput.value = ''
  pullVisible.value = true
}

const submitPull = async () => {
  if (!activeHost.value || !pullImage.value) {
    ElMessage.warning('请填写镜像')
    return
  }
  pullLoading.value = true
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/images/pull`, { image: pullImage.value }, { headers: authHeaders() })
    if (res.data.code === 0) {
      pullOutput.value = res.data.output || res.data.message || '拉取完成'
      loadImages()
      refreshManage()
    } else {
      ElMessage.error(res.data.message || '拉取失败')
    }
  } catch (e) {
    ElMessage.error('拉取失败')
  } finally {
    pullLoading.value = false
  }
}

const pruneImages = async () => {
  if (!activeHost.value) return
  try {
    await ElMessageBox.confirm('确定清理悬挂镜像吗?', '提示', {
      confirmButtonText: '清理',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/images/prune`, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      pruneOutput.value = res.data.data?.output || '清理完成'
      pruneVisible.value = true
      loadImages()
      refreshManage()
    } else {
      ElMessage.error(res.data.message || '清理失败')
    }
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  }
}

const removeImage = (row) => {
  if (!activeHost.value) return
  const id = row.id
  if (!id) return
  confirmDeleteImages([row]).then(async () => {
    try {
      await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/images/${encodeURIComponent(id)}`, { headers: authHeaders() })
      ElMessage.success('删除成功')
      loadImages()
    } catch (e) {
      ElMessage.error(extractErrorMessage(e))
    }
  }).catch(() => {})
}

const onImageSelectionChange = (rows) => {
  selectedImages.value = rows || []
}

const removeSelectedImages = async () => {
  if (!activeHost.value) return
  const rows = selectedImages.value.filter(r => r.id && r.id !== '-')
  if (rows.length === 0) {
    ElMessage.warning('请选择要删除的镜像')
    return
  }
  try {
    await confirmDeleteImages(rows)
  } catch (e) {
    return
  }
  let ok = 0
  let fail = 0
  const results = []
  for (const row of rows) {
    try {
      await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/images/${encodeURIComponent(row.id)}`, { headers: authHeaders() })
      ok += 1
      results.push({ label: formatImageLabel(row), status: '成功', message: '删除成功' })
    } catch (e) {
      fail += 1
      results.push({ label: formatImageLabel(row), status: '失败', message: extractErrorMessage(e) })
    }
  }
  if (fail === 0) {
    ElMessage.success(`已删除 ${ok} 个镜像`)
  } else {
    ElMessage.warning(`已删除 ${ok} 个，失败 ${fail} 个`)
  }
  batchResultRows.value = results
  batchResultVisible.value = true
  loadImages()
}

const handleDiagnose = async (row) => {
  diagnoseVisible.value = true
  diagnoseLoading.value = true
  diagnoseResult.value = null
  diagnoseError.value = ''
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${row.id}/test`, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      diagnoseResult.value = res.data.data
    } else {
      diagnoseError.value = res.data.message || '诊断失败'
    }
  } catch (e) {
    diagnoseError.value = '诊断失败'
  } finally {
    diagnoseLoading.value = false
  }
}

watch(manageTab, (tab) => {
  if (tab === 'containers') {
    loadContainers()
    loadContainerStats()
  }
  if (tab === 'images') loadImages()
  if (tab === 'networks') loadNetworks()
  if (tab === 'volumes') loadVolumes()
  if (tab === 'secrets') loadSecrets()
  if (tab === 'configs') loadConfigs()
  if (tab === 'registries') loadRegistries()
  if (tab === 'nodes') loadNodes()
  if (tab === 'topology') loadServices()
  if (tab === 'services') loadServices()
  if (tab === 'stacks') loadStacks()
})

onMounted(() => {
  fetchData()
  window.addEventListener('resize', sendTerminalResize)
})

onUnmounted(() => {
  if (statsTimer) {
    clearInterval(statsTimer)
    statsTimer = null
  }
  if (logTimer) {
    clearInterval(logTimer)
    logTimer = null
  }
  if (serviceLogTimer) {
    clearInterval(serviceLogTimer)
    serviceLogTimer = null
  }
  window.removeEventListener('resize', sendTerminalResize)
  closeTerminal()
  if (terminal) {
    terminal.dispose()
    terminal = null
    terminalFit = null
  }
})
</script>

<style scoped>
.flex { display: flex; }
.justify-between { justify-content: space-between; }
.items-center { align-items: center; }
.gap-2 { gap: 8px; }
.font-bold { font-weight: bold; }
.w-100 { width: 100%; }
.text-xs { font-size: 12px; line-height: 1.3; }
.text-blue-500 { color: #409eff; }
.text-xl { font-size: 18px; }
.drawer-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.drawer-title { font-size: 18px; font-weight: 600; }
.drawer-sub { color: #606266; margin-top: 6px; display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
.drawer-meta { color: #909399; }
.w-40 { width: 140px; }
.w-48 { width: 180px; }
.w-28 { width: 110px; }
.manage-tabs { margin-top: 8px; }
.tab-toolbar { display: flex; justify-content: space-between; gap: 8px; margin-bottom: 10px; flex-wrap: wrap; }
.toolbar-left { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.toolbar-right { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.diagnose-block { display: flex; flex-direction: column; gap: 12px; }
.diagnose-title { font-weight: 600; }
.diagnose-pre { background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 6px; overflow: auto; max-height: 200px; }
.log-controls { display: flex; gap: 8px; margin-bottom: 10px; }
.terminal-container {
  height: 360px;
  background: #0f172a;
  border-radius: 8px;
  overflow: hidden;
  padding: 8px;
  margin-top: 8px;
}
.terminal-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
.terminal-title { font-weight: 600; flex: 1; }
:deep(.el-drawer__body) { padding: 16px; }
</style>
