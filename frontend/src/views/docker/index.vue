<template>
  <el-card class="docker-page-card">
    <template #header>
      <div class="header-row">
        <div class="header-left">
          <span class="font-bold header-title">Docker 环境列表</span>
          <div class="header-stats">
            <el-tag effect="plain">总环境 {{ tableData.length }}</el-tag>
            <el-tag type="success" effect="plain">在线 {{ onlineHosts }}</el-tag>
            <el-tag type="danger" effect="plain">离线 {{ offlineHosts }}</el-tag>
          </div>
        </div>
        <div class="header-actions">
          <el-button type="primary" icon="Plus" @click="handleAdd">添加环境</el-button>
          <el-button icon="Refresh" @click="syncAll">刷新</el-button>
        </div>
      </div>
    </template>

    <div class="table-scroll">
      <el-table :fit="true" :data="tableData" v-loading="loading" style="width: 100%; min-width: 980px">
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
    </div>

    <!-- 添加主机弹窗 -->
    <el-dialog append-to-body v-model="dialogVisible" title="添加 Docker 环境" width="500px">
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
    <el-drawer v-model="manageVisible" size="100%" :with-header="false" :append-to-body="true" class="docker-manage-drawer">
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
        <div class="drawer-actions">
          <el-button size="small" icon="Refresh" @click="refreshManage">刷新</el-button>
          <el-button size="small" plain icon="Close" @click="manageVisible = false">退出管理</el-button>
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
              <el-input v-model="containerFilters.keyword" class="w-48" placeholder="名称/镜像/ID" clearable />
              <el-select v-model="containerFilters.state" class="w-28" clearable placeholder="状态">
                <el-option label="running" value="running" />
                <el-option label="exited" value="exited" />
                <el-option label="paused" value="paused" />
                <el-option label="restarting" value="restarting" />
                <el-option label="created" value="created" />
              </el-select>
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
          <div class="table-scroll">
            <el-table :fit="true"
            ref="containerTableRef"
            :data="filteredContainers"
            v-loading="containersLoading"
            style="width: 100%; min-width: 1880px"
            :row-key="row => row.id"
            @selection-change="onContainerSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="names" label="名称" min-width="200" />
            <el-table-column prop="image" label="镜像" min-width="180" />
            <el-table-column prop="ports" label="端口" min-width="160" />
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
                  <el-button size="small" type="info" plain :disabled="!isContainerRunning(row)" @click="openExec(row)">执行命令</el-button>
                  <el-button size="small" type="success" plain :disabled="!isContainerRunning(row)" @click="openTerminal(row)">终端</el-button>
                  <el-button size="small" type="primary" plain :disabled="!isContainerRunning(row)" @click="openStatsChart(row)">趋势</el-button>
                  <el-button size="small" type="success" plain @click="containerAction(row, 'start')">启动</el-button>
                  <el-button size="small" type="warning" plain @click="containerAction(row, 'stop')">停止</el-button>
                  <el-button size="small" type="primary" plain @click="containerAction(row, 'restart')">重启</el-button>
                  <el-button size="small" type="danger" plain @click="containerAction(row, 'remove')">删除</el-button>
                </el-space>
              </template>
            </el-table-column>
            </el-table>
          </div>
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
              <el-button type="success" plain @click="openBuildImage">构建镜像</el-button>
              <el-button type="info" plain @click="openLoadImage">导入镜像</el-button>
              <el-button type="primary" icon="Download" @click="openPullImage">拉取镜像</el-button>
              <el-button icon="Refresh" @click="loadImages">刷新</el-button>
            </div>
          </div>
          <div class="table-scroll">
            <el-table :fit="true"
            ref="imageTableRef"
            :data="filteredImages"
            v-loading="imagesLoading"
            style="width: 100%; min-width: 1120px"
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
          </div>
        </el-tab-pane>

        <el-tab-pane label="网络" name="networks">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="danger" plain :disabled="selectedNetworks.length === 0" @click="removeSelectedNetworks">批量删除</el-button>
              <el-button plain @click="loadNetworkUsage" :loading="networkUsageLoading">使用情况</el-button>
            </div>
            <div class="toolbar-right">
              <el-button icon="Refresh" @click="loadNetworks">刷新</el-button>
            </div>
          </div>
          <div class="table-scroll">
            <el-table :fit="true"
            ref="networkTableRef"
            :data="networks"
            v-loading="networksLoading"
            style="width: 100%; min-width: 980px"
            :row-key="row => row.id"
            @selection-change="onNetworkSelectionChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="name" label="名称" min-width="180" />
            <el-table-column prop="id" label="ID" min-width="200" />
            <el-table-column prop="driver" label="驱动" width="120" />
            <el-table-column prop="scope" label="范围" width="120" />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="openNetworkInspect(row)">详情</el-button>
              </template>
            </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane label="Events" name="events">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-select v-model="eventFilters.type" placeholder="类型" class="w-28" clearable>
                <el-option label="container" value="container" />
                <el-option label="image" value="image" />
                <el-option label="volume" value="volume" />
                <el-option label="network" value="network" />
                <el-option label="node" value="node" />
                <el-option label="service" value="service" />
                <el-option label="secret" value="secret" />
                <el-option label="config" value="config" />
              </el-select>
              <el-input v-model="eventFilters.action" class="w-24" placeholder="动作" clearable />
              <el-input v-model="eventFilters.container" class="w-28" placeholder="容器" clearable />
              <el-input v-model="eventFilters.image" class="w-28" placeholder="镜像" clearable />
              <el-input v-model="eventFilters.volume" class="w-28" placeholder="卷" clearable />
              <el-input v-model="eventFilters.network" class="w-28" placeholder="网络" clearable />
              <el-input v-model="eventFilters.service" class="w-28" placeholder="服务" clearable />
              <el-input-number v-model="eventFilters.sinceMinutes" :min="1" class="w-28" controls-position="right" />
              <el-input-number v-model="eventFilters.limit" :min="1" class="w-24" controls-position="right" />
            </div>
            <div class="toolbar-right">
              <el-button icon="Refresh" @click="loadEvents">刷新</el-button>
            </div>
          </div>
          <el-table :fit="true" :data="events" v-loading="eventsLoading" style="width: 100%">
            <el-table-column prop="time" label="时间" width="180" />
            <el-table-column prop="type" label="类型" width="120" />
            <el-table-column prop="action" label="动作" width="140" />
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="id" label="ID" min-width="180" />
            <el-table-column prop="detail" label="详情" min-width="260" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Volumes" name="volumes">
          <div class="tab-toolbar">
            <div class="toolbar-left">
              <el-button type="danger" plain :disabled="selectedVolumes.length === 0" @click="removeSelectedVolumes">批量删除</el-button>
              <el-button plain @click="loadVolumeUsage" :loading="volumeUsageLoading">使用情况</el-button>
            </div>
            <div class="toolbar-right">
              <el-button type="primary" icon="Plus" @click="openCreateVolume">创建卷</el-button>
              <el-button icon="Refresh" @click="loadVolumes">刷新</el-button>
            </div>
          </div>
          <el-table :fit="true"
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
          <el-table :fit="true"
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
          <el-table :fit="true"
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
          <el-table :fit="true" :data="registries" v-loading="registriesLoading" style="width: 100%">
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
          <el-table :fit="true" :data="nodes" v-loading="nodesLoading" style="width: 100%">
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
              <el-input v-model="serviceFilters.keyword" class="w-48" placeholder="名称/镜像/端口" clearable />
              <el-input-number v-model="batchServiceScale" :min="0" class="w-28" controls-position="right" />
              <el-button type="success" plain :disabled="selectedServices.length === 0" @click="scaleSelectedServices">
                批量设置副本
              </el-button>
              <el-button type="danger" plain :disabled="selectedServices.length === 0" @click="removeSelectedServices">
                批量删除
              </el-button>
              <el-select v-model="serviceStackFilter" placeholder="Stack" class="w-40" clearable>
                <el-option v-for="s in serviceStacks" :key="s" :label="s" :value="s" />
              </el-select>
            </div>
            <div class="toolbar-right">
              <el-button type="primary" icon="Plus" @click="openCreateService">创建服务</el-button>
              <el-button icon="Refresh" @click="loadServices">刷新</el-button>
            </div>
          </div>
          <el-table :fit="true"
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
            <el-table-column label="更新状态" min-width="140">
              <template #default="{ row }">
                <el-tooltip v-if="row.UpdateStatus?.Message" :content="row.UpdateStatus.Message" placement="top">
                  <el-tag :type="formatServiceStatusTag(row.UpdateStatus).type">
                    {{ formatServiceStatusTag(row.UpdateStatus).text }}
                  </el-tag>
                </el-tooltip>
                <el-tag v-else :type="formatServiceStatusTag(row.UpdateStatus).type">
                  {{ formatServiceStatusTag(row.UpdateStatus).text }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="回滚状态" min-width="140">
              <template #default="{ row }">
                <el-tooltip v-if="row.RollbackStatus?.Message" :content="row.RollbackStatus.Message" placement="top">
                  <el-tag :type="formatServiceStatusTag(row.RollbackStatus).type">
                    {{ formatServiceStatusTag(row.RollbackStatus).text }}
                  </el-tag>
                </el-tooltip>
                <el-tag v-else :type="formatServiceStatusTag(row.RollbackStatus).type">
                  {{ formatServiceStatusTag(row.RollbackStatus).text }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="440" fixed="right">
              <template #default="{ row }">
                <el-space size="8">
                  <el-button size="small" @click="openServiceDetail(row)">详情</el-button>
                  <el-button size="small" type="primary" plain @click="openEditService(row)">编辑</el-button>
                  <el-button size="small" type="info" plain @click="openServiceTasks(row)">任务</el-button>
                  <el-button size="small" type="danger" plain @click="openServiceTasks(row, true)">错误</el-button>
                  <el-button size="small" @click="openServiceLogs(row)">日志</el-button>
                  <el-input-number v-model="serviceScaleMap[row.ID || row.Id || row.id]" :min="0" size="small" class="w-28" controls-position="right" />
                  <el-button size="small" type="success" plain @click="applyServiceScale(row)">应用</el-button>
                  <el-button size="small" type="primary" plain @click="scaleService(row)">扩缩容</el-button>
                  <el-button size="small" type="warning" plain @click="updateServiceImage(row)">更新镜像</el-button>
                  <el-button size="small" type="danger" plain @click="restartService(row)">重启</el-button>
                  <el-button size="small" type="warning" plain @click="rollbackService(row)">回滚</el-button>
                  <el-button size="small" type="danger" @click="removeService(row)">删除</el-button>
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
              <el-button type="success" plain @click="openGitDeploy">Git 部署</el-button>
              <el-button type="primary" icon="Plus" @click="openDeployStack">部署 Stack</el-button>
              <el-button icon="Refresh" @click="loadStacks">刷新</el-button>
            </div>
          </div>
          <el-table :fit="true"
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
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <el-space size="8">
                  <el-button size="small" @click="openStackServices(row)">查看服务</el-button>
                  <el-button size="small" type="danger" plain @click="removeStack(row)">删除</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-drawer>

    <!-- 创建容器弹窗 -->
    <el-dialog append-to-body v-model="createVisible" title="创建容器" width="640px">
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
        <el-form-item label="标签">
          <el-input v-model="createForm.labels" type="textarea" :rows="3" placeholder="KEY=VALUE，每行一个" />
        </el-form-item>
        <el-form-item label="网络">
          <el-input v-model="createForm.networks" type="textarea" :rows="2" placeholder="每行一个网络名称或ID" />
        </el-form-item>
        <el-form-item label="挂载">
          <el-input v-model="createForm.mounts" type="textarea" :rows="3" placeholder="每行一个挂载，如 /host:/container 或 type=bind,src=/host,dst=/container" />
        </el-form-item>
        <el-form-item label="Entrypoint">
          <el-input v-model="createForm.entrypoint" placeholder="可选，如 /bin/sh" />
        </el-form-item>
        <el-form-item label="命令">
          <el-input v-model="createForm.command" type="textarea" :rows="2" placeholder="可选，按空格分隔" />
        </el-form-item>
        <el-divider content-position="left">高级</el-divider>
        <el-form-item label="特权模式">
          <el-switch v-model="createForm.privileged" />
        </el-form-item>
        <el-form-item label="Cap Add">
          <el-input v-model="createForm.cap_add" type="textarea" :rows="2" placeholder="每行一个，如 NET_ADMIN" />
        </el-form-item>
        <el-form-item label="Network Mode">
          <el-select v-model="createForm.network_mode" filterable allow-create clearable placeholder="bridge/host/none/自定义">
            <el-option label="bridge" value="bridge" />
            <el-option label="host" value="host" />
            <el-option label="none" value="none" />
          </el-select>
          <div class="text-xs text-gray-400 mt-1">填写网络列表时，此项将被忽略</div>
        </el-form-item>
        <el-form-item label="DNS">
          <el-input v-model="createForm.dns" type="textarea" :rows="2" placeholder="每行一个 DNS，如 8.8.8.8" />
        </el-form-item>
        <el-form-item label="Extra Hosts">
          <el-input v-model="createForm.extra_hosts" type="textarea" :rows="2" placeholder="每行一个，如 example.com:1.2.3.4" />
        </el-form-item>
        <el-divider content-position="left">健康检查</el-divider>
        <el-form-item label="禁用检查">
          <el-switch v-model="createForm.health_disable" />
        </el-form-item>
        <el-form-item label="Health Cmd">
          <el-input v-model="createForm.health_cmd" :disabled="createForm.health_disable" placeholder="如 curl -f http://localhost/health || exit 1" />
        </el-form-item>
        <el-form-item label="Interval">
          <el-input v-model="createForm.health_interval" :disabled="createForm.health_disable" placeholder="如 30s" />
        </el-form-item>
        <el-form-item label="Timeout">
          <el-input v-model="createForm.health_timeout" :disabled="createForm.health_disable" placeholder="如 5s" />
        </el-form-item>
        <el-form-item label="Retries">
          <el-input v-model="createForm.health_retries" :disabled="createForm.health_disable" placeholder="如 3" />
        </el-form-item>
        <el-form-item label="Start Period">
          <el-input v-model="createForm.health_start_period" :disabled="createForm.health_disable" placeholder="如 10s" />
        </el-form-item>
        <el-divider content-position="left">资源限制</el-divider>
        <el-form-item label="CPUs">
          <el-input v-model="createForm.cpus" placeholder="如 0.5" />
        </el-form-item>
        <el-form-item label="Memory">
          <el-input v-model="createForm.memory" placeholder="如 512M" />
        </el-form-item>
        <el-form-item label="Memory Reservation">
          <el-input v-model="createForm.memory_reservation" placeholder="如 256M" />
        </el-form-item>
        <el-form-item label="Pids Limit">
          <el-input v-model="createForm.pids_limit" placeholder="如 200" />
        </el-form-item>
        <el-divider content-position="left">安全与系统</el-divider>
        <el-form-item label="User">
          <el-input v-model="createForm.user" placeholder="如 1000:1000" />
        </el-form-item>
        <el-form-item label="Workdir">
          <el-input v-model="createForm.workdir" placeholder="如 /app" />
        </el-form-item>
        <el-form-item label="Hostname">
          <el-input v-model="createForm.hostname" placeholder="可选" />
        </el-form-item>
        <el-form-item label="Runtime">
          <el-input v-model="createForm.runtime" placeholder="如 runc" />
        </el-form-item>
        <el-form-item label="Read-only">
          <el-switch v-model="createForm.read_only" />
        </el-form-item>
        <el-form-item label="Ulimits">
          <el-input v-model="createForm.ulimits" type="textarea" :rows="2" placeholder="每行一条，如 nofile=1024:2048" />
        </el-form-item>
        <el-form-item label="Sysctls">
          <el-input v-model="createForm.sysctls" type="textarea" :rows="2" placeholder="KEY=VALUE，每行一个" />
        </el-form-item>
        <el-form-item label="Security Opt">
          <el-input v-model="createForm.security_opt" type="textarea" :rows="2" placeholder="每行一个，如 no-new-privileges" />
        </el-form-item>
        <el-form-item label="Devices">
          <el-input v-model="createForm.devices" type="textarea" :rows="2" placeholder="每行一条，如 /dev/sda:/dev/xvda:rwm" />
        </el-form-item>
        <el-form-item label="Tmpfs">
          <el-input v-model="createForm.tmpfs" type="textarea" :rows="2" placeholder="每行一条，如 /tmp:rw,noexec,nosuid,size=64m" />
        </el-form-item>
        <el-divider content-position="left">日志</el-divider>
        <el-form-item label="Log Driver">
          <el-input v-model="createForm.log_driver" placeholder="如 json-file 或 syslog" />
        </el-form-item>
        <el-form-item label="Log Opts">
          <el-input v-model="createForm.log_opts" type="textarea" :rows="2" placeholder="KEY=VALUE，每行一个" />
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

    <!-- 创建 Service 弹窗 -->
    <el-dialog v-model="createServiceVisible" title="创建 Service" width="760px" append-to-body>
      <el-form :model="serviceCreateForm" label-width="110px">
        <el-form-item label="名称" required>
          <el-input v-model="serviceCreateForm.name" placeholder="例如 web-service" />
        </el-form-item>
        <el-form-item label="镜像" required>
          <el-input v-model="serviceCreateForm.image" placeholder="例如 nginx:latest" />
        </el-form-item>
        <el-form-item label="模式">
          <el-select v-model="serviceCreateForm.mode">
            <el-option label="replicated" value="replicated" />
            <el-option label="global" value="global" />
          </el-select>
        </el-form-item>
        <el-form-item label="Endpoint">
          <el-select v-model="serviceCreateForm.endpoint_mode">
            <el-option label="vip" value="vip" />
            <el-option label="dnsrr" value="dnsrr" />
          </el-select>
        </el-form-item>
        <el-form-item label="副本数">
          <el-input-number v-model="serviceCreateForm.replicas" :min="0" :disabled="serviceCreateIsGlobal" />
        </el-form-item>
        <el-form-item label="端口发布">
          <el-input v-model="serviceCreateForm.ports" type="textarea" :rows="3" placeholder="每行一条，如 8080:80 或 published=8080,target=80" />
        </el-form-item>
        <el-form-item label="环境变量">
          <el-input v-model="serviceCreateForm.env" type="textarea" :rows="4" placeholder="KEY=VALUE，每行一条" />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="serviceCreateForm.labels" type="textarea" :rows="3" placeholder="KEY=VALUE，每行一条" />
        </el-form-item>
        <el-form-item label="网络">
          <el-input v-model="serviceCreateForm.networks" type="textarea" :rows="2" placeholder="每行一个网络名称或ID" />
        </el-form-item>
        <el-divider content-position="left">调度</el-divider>
        <el-form-item label="约束">
          <el-input v-model="serviceCreateForm.constraints" type="textarea" :rows="2" placeholder="如 node.role==manager" />
        </el-form-item>
        <el-form-item label="Placement Pref">
          <el-input v-model="serviceCreateForm.placement_prefs" type="textarea" :rows="2" placeholder="每行一条，如 spread=node.labels.zone" />
        </el-form-item>
        <el-form-item label="每节点上限">
          <el-input v-model="serviceCreateForm.max_replicas_per_node" :disabled="serviceCreateIsGlobal" placeholder="如 1" />
          <div class="text-xs text-gray-400 mt-1">仅 replicated 模式生效</div>
        </el-form-item>
        <el-form-item label="挂载">
          <el-input v-model="serviceCreateForm.mounts" type="textarea" :rows="3" placeholder="每行一条 --mount 参数，如 type=bind,src=/data,dst=/data" />
        </el-form-item>
        <el-form-item label="命令">
          <el-input v-model="serviceCreateForm.command" type="textarea" :rows="2" placeholder="可选，按空格分隔" />
        </el-form-item>
        <el-divider content-position="left">更新策略</el-divider>
        <el-form-item label="重启条件">
          <el-select v-model="serviceCreateForm.restart_condition" placeholder="不设置">
            <el-option label="none" value="none" />
            <el-option label="on-failure" value="on-failure" />
            <el-option label="any" value="any" />
          </el-select>
        </el-form-item>
        <el-form-item label="并行度">
          <el-input v-model="serviceCreateForm.update_parallelism" placeholder="如 2" />
        </el-form-item>
        <el-form-item label="更新延迟">
          <el-input v-model="serviceCreateForm.update_delay" placeholder="如 10s" />
        </el-form-item>
        <el-form-item label="更新失败策略">
          <el-select v-model="serviceCreateForm.update_failure_action" placeholder="不设置">
            <el-option label="pause" value="pause" />
            <el-option label="continue" value="continue" />
            <el-option label="rollback" value="rollback" />
          </el-select>
        </el-form-item>
        <el-form-item label="更新顺序">
          <el-select v-model="serviceCreateForm.update_order" placeholder="不设置">
            <el-option label="stop-first" value="stop-first" />
            <el-option label="start-first" value="start-first" />
          </el-select>
        </el-form-item>
        <el-form-item label="回滚并行度">
          <el-input v-model="serviceCreateForm.rollback_parallelism" placeholder="如 2" />
        </el-form-item>
        <el-form-item label="回滚延迟">
          <el-input v-model="serviceCreateForm.rollback_delay" placeholder="如 10s" />
        </el-form-item>
        <el-form-item label="回滚失败策略">
          <el-select v-model="serviceCreateForm.rollback_failure_action" placeholder="不设置">
            <el-option label="pause" value="pause" />
            <el-option label="continue" value="continue" />
            <el-option label="rollback" value="rollback" />
          </el-select>
        </el-form-item>
        <el-form-item label="回滚顺序">
          <el-select v-model="serviceCreateForm.rollback_order" placeholder="不设置">
            <el-option label="stop-first" value="stop-first" />
            <el-option label="start-first" value="start-first" />
          </el-select>
        </el-form-item>
        <el-divider content-position="left">资源限制</el-divider>
        <el-form-item label="Limit CPU">
          <el-input v-model="serviceCreateForm.limit_cpu" placeholder="如 0.5" />
        </el-form-item>
        <el-form-item label="Limit Memory">
          <el-input v-model="serviceCreateForm.limit_memory" placeholder="如 512M" />
        </el-form-item>
        <el-form-item label="Reserve CPU">
          <el-input v-model="serviceCreateForm.reserve_cpu" placeholder="如 0.25" />
        </el-form-item>
        <el-form-item label="Reserve Memory">
          <el-input v-model="serviceCreateForm.reserve_memory" placeholder="如 256M" />
        </el-form-item>
        <el-divider content-position="left">配置预览</el-divider>
        <el-input :model-value="serviceCreatePreview" type="textarea" :rows="6" readonly />
      </el-form>
      <template #footer>
        <el-button @click="createServiceVisible = false">取消</el-button>
        <el-button type="primary" :loading="createServiceLoading" @click="submitCreateService">创建</el-button>
      </template>
    </el-dialog>

    <!-- 编辑 Service 弹窗 -->
    <el-dialog v-model="editServiceVisible" title="编辑 Service" width="760px" append-to-body>
      <el-form :model="serviceEditForm" label-width="110px" v-loading="editServiceLoading">
        <el-form-item label="名称">
          <el-input v-model="serviceEditForm.name" disabled />
        </el-form-item>
        <el-form-item label="镜像">
          <el-input v-model="serviceEditForm.image" placeholder="例如 nginx:latest" />
        </el-form-item>
        <el-form-item label="模式">
          <el-select v-model="serviceEditForm.mode" placeholder="不调整">
            <el-option label="replicated" value="replicated" />
            <el-option label="global" value="global" />
          </el-select>
        </el-form-item>
        <el-form-item label="Endpoint">
          <el-select v-model="serviceEditForm.endpoint_mode" placeholder="不调整">
            <el-option label="vip" value="vip" />
            <el-option label="dnsrr" value="dnsrr" />
          </el-select>
        </el-form-item>
        <el-form-item label="副本数">
          <el-input-number v-model="serviceEditForm.replicas" :min="0" :disabled="serviceEditIsGlobal" />
        </el-form-item>
        <el-form-item label="端口发布">
          <el-input v-model="serviceEditForm.ports" type="textarea" :rows="3" placeholder="每行一条，如 8080:80 或 published=8080,target=80" />
          <el-checkbox v-model="serviceEditForm.reset_ports">覆盖现有端口</el-checkbox>
        </el-form-item>
        <el-form-item label="环境变量">
          <el-input v-model="serviceEditForm.env" type="textarea" :rows="4" placeholder="KEY=VALUE，每行一条" />
          <el-checkbox v-model="serviceEditForm.reset_env">覆盖现有环境变量</el-checkbox>
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="serviceEditForm.labels" type="textarea" :rows="3" placeholder="KEY=VALUE，每行一条" />
          <el-checkbox v-model="serviceEditForm.reset_labels">覆盖现有标签</el-checkbox>
        </el-form-item>
        <el-form-item label="网络">
          <el-input v-model="serviceEditForm.networks" type="textarea" :rows="2" placeholder="每行一个网络名称或ID" />
          <el-checkbox v-model="serviceEditForm.reset_networks">覆盖现有网络</el-checkbox>
        </el-form-item>
        <el-form-item label="约束">
          <el-input v-model="serviceEditForm.constraints" type="textarea" :rows="2" placeholder="如 node.role==manager" />
          <el-checkbox v-model="serviceEditForm.reset_constraints">覆盖现有约束</el-checkbox>
        </el-form-item>
        <el-form-item label="Placement Pref">
          <el-input v-model="serviceEditForm.placement_prefs" type="textarea" :rows="2" placeholder="每行一条，如 spread=node.labels.zone" />
          <el-checkbox v-model="serviceEditForm.reset_placement_prefs">覆盖现有 Placement</el-checkbox>
        </el-form-item>
        <el-form-item label="挂载">
          <el-input v-model="serviceEditForm.mounts" type="textarea" :rows="3" placeholder="每行一条 --mount 参数" />
          <el-checkbox v-model="serviceEditForm.reset_mounts">覆盖现有挂载</el-checkbox>
        </el-form-item>
        <el-form-item label="命令">
          <el-input v-model="serviceEditForm.command" type="textarea" :rows="2" placeholder="可选，按空格分隔" />
          <el-checkbox v-model="serviceEditForm.reset_command">覆盖现有命令</el-checkbox>
        </el-form-item>
        <el-form-item label="每节点上限">
          <el-input v-model="serviceEditForm.max_replicas_per_node" :disabled="serviceEditIsGlobal" placeholder="如 1" />
          <div class="text-xs text-gray-400 mt-1">仅 replicated 模式生效</div>
        </el-form-item>
        <el-divider content-position="left">更新策略</el-divider>
        <el-form-item label="重启条件">
          <el-select v-model="serviceEditForm.restart_condition" placeholder="不设置">
            <el-option label="none" value="none" />
            <el-option label="on-failure" value="on-failure" />
            <el-option label="any" value="any" />
          </el-select>
        </el-form-item>
        <el-form-item label="并行度">
          <el-input v-model="serviceEditForm.update_parallelism" placeholder="如 2" />
        </el-form-item>
        <el-form-item label="更新延迟">
          <el-input v-model="serviceEditForm.update_delay" placeholder="如 10s" />
        </el-form-item>
        <el-form-item label="更新失败策略">
          <el-select v-model="serviceEditForm.update_failure_action" placeholder="不设置">
            <el-option label="pause" value="pause" />
            <el-option label="continue" value="continue" />
            <el-option label="rollback" value="rollback" />
          </el-select>
        </el-form-item>
        <el-form-item label="更新顺序">
          <el-select v-model="serviceEditForm.update_order" placeholder="不设置">
            <el-option label="stop-first" value="stop-first" />
            <el-option label="start-first" value="start-first" />
          </el-select>
        </el-form-item>
        <el-form-item label="回滚并行度">
          <el-input v-model="serviceEditForm.rollback_parallelism" placeholder="如 2" />
        </el-form-item>
        <el-form-item label="回滚延迟">
          <el-input v-model="serviceEditForm.rollback_delay" placeholder="如 10s" />
        </el-form-item>
        <el-form-item label="回滚失败策略">
          <el-select v-model="serviceEditForm.rollback_failure_action" placeholder="不设置">
            <el-option label="pause" value="pause" />
            <el-option label="continue" value="continue" />
            <el-option label="rollback" value="rollback" />
          </el-select>
        </el-form-item>
        <el-form-item label="回滚顺序">
          <el-select v-model="serviceEditForm.rollback_order" placeholder="不设置">
            <el-option label="stop-first" value="stop-first" />
            <el-option label="start-first" value="start-first" />
          </el-select>
        </el-form-item>
        <el-divider content-position="left">资源限制</el-divider>
        <el-form-item label="Limit CPU">
          <el-input v-model="serviceEditForm.limit_cpu" placeholder="如 0.5" />
        </el-form-item>
        <el-form-item label="Limit Memory">
          <el-input v-model="serviceEditForm.limit_memory" placeholder="如 512M" />
        </el-form-item>
        <el-form-item label="Reserve CPU">
          <el-input v-model="serviceEditForm.reserve_cpu" placeholder="如 0.25" />
        </el-form-item>
        <el-form-item label="Reserve Memory">
          <el-input v-model="serviceEditForm.reserve_memory" placeholder="如 256M" />
        </el-form-item>
        <el-divider content-position="left">配置预览</el-divider>
        <el-input :model-value="serviceEditPreview" type="textarea" :rows="6" readonly />
      </el-form>
      <template #footer>
        <el-button @click="editServiceVisible = false">取消</el-button>
        <el-button type="primary" :loading="editServiceSaving" @click="submitEditService">保存</el-button>
      </template>
    </el-dialog>

    <!-- 拉取镜像弹窗 -->
    <el-dialog append-to-body v-model="pullVisible" title="拉取镜像" width="640px">
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

    <!-- 构建镜像弹窗 -->
    <el-dialog append-to-body v-model="buildVisible" title="构建镜像" width="760px">
      <el-form :model="buildForm" label-width="120px">
        <el-form-item label="镜像标签" required>
          <el-input v-model="buildForm.tag" placeholder="例如 myapp:latest" />
        </el-form-item>
        <el-form-item label="Dockerfile" required>
          <el-input v-model="buildForm.dockerfile" type="textarea" :rows="10" placeholder="粘贴 Dockerfile 内容" />
        </el-form-item>
        <el-form-item label="构建上下文">
          <el-upload :auto-upload="false" :limit="1" :on-change="handleBuildContextChange">
            <el-button>选择 tar 包</el-button>
          </el-upload>
          <div class="text-xs text-gray-400 mt-1">可选，未选择时仅 Dockerfile 生效</div>
        </el-form-item>
        <el-form-item label="Build Args">
          <el-input v-model="buildArgsText" type="textarea" :rows="3" placeholder="KEY=VALUE，每行一个" />
        </el-form-item>
      </el-form>
      <el-input v-model="buildOutput" type="textarea" :rows="6" readonly placeholder="输出" />
      <template #footer>
        <el-button @click="buildVisible = false">关闭</el-button>
        <el-button type="primary" :loading="buildLoading" @click="submitBuildImage">开始构建</el-button>
      </template>
    </el-dialog>

    <!-- 导入镜像弹窗 -->
    <el-dialog append-to-body v-model="loadVisible" title="导入镜像" width="640px">
      <el-upload :auto-upload="false" :limit="1" :on-change="handleLoadTarChange">
        <el-button>选择 docker save 的 tar 包</el-button>
      </el-upload>
      <el-input v-model="loadOutput" type="textarea" :rows="6" readonly placeholder="输出" class="mt-3" />
      <template #footer>
        <el-button @click="loadVisible = false">关闭</el-button>
        <el-button type="primary" :loading="loadLoading" @click="submitLoadImage">导入</el-button>
      </template>
    </el-dialog>

    <!-- 批量删除结果 -->
    <el-dialog append-to-body v-model="batchResultVisible" title="批量删除结果" width="720px">
      <el-table :fit="true" :data="batchResultRows" style="width: 100%">
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
    <el-dialog append-to-body v-model="pruneVisible" title="清理结果" width="720px">
      <el-input v-model="pruneOutput" type="textarea" :rows="10" readonly placeholder="输出" />
      <template #footer>
        <el-button type="primary" @click="pruneVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 诊断弹窗 -->
    <el-dialog append-to-body v-model="diagnoseVisible" title="Docker 诊断" width="720px">
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
    <el-dialog v-model="logVisible" title="容器日志" width="720px" append-to-body>
      <div class="log-controls">
        <el-input v-model="logTail" placeholder="tail" style="width: 120px" />
        <el-button icon="Refresh" @click="loadLogs" :loading="logLoading">刷新</el-button>
        <el-switch v-model="logFollow" active-text="实时" />
        <el-button @click="clearLogs">清空</el-button>
      </div>
      <el-input v-model="logText" type="textarea" :rows="16" readonly />
    </el-dialog>

    <!-- Service 日志弹窗 -->
    <el-dialog v-model="serviceLogVisible" title="Service 日志" width="760px" append-to-body>
      <div class="log-controls">
        <el-input v-model="serviceLogTail" placeholder="tail" style="width: 120px" />
        <el-button icon="Refresh" @click="loadServiceLogs" :loading="serviceLogLoading">刷新</el-button>
        <el-switch v-model="serviceLogFollow" active-text="实时" />
        <el-button @click="clearServiceLogs">清空</el-button>
      </div>
      <el-input v-model="serviceLogText" type="textarea" :rows="16" readonly />
    </el-dialog>

    <!-- 容器资源趋势 -->
    <el-dialog v-model="statsChartVisible" title="容器资源趋势" width="880px" append-to-body>
      <div ref="statsChartRef" style="height: 360px"></div>
    </el-dialog>

    <!-- 容器详情弹窗 -->
    <el-dialog v-model="inspectVisible" title="容器详情" width="90%" append-to-body>
      <el-skeleton v-if="inspectLoading" :rows="6" animated />
      <div v-else>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="ID">{{ inspectData?.Id || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Name">{{ inspectData?.Name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Image">{{ inspectData?.Config?.Image || '-' }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ inspectData?.State?.Status || '-' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(inspectData?.Created) }}</el-descriptions-item>
          <el-descriptions-item label="重启策略">{{ inspectRestartPolicy || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Network Mode">{{ inspectNetworkMode || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Privileged">{{ inspectPrivileged ? 'true' : 'false' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">命令</el-divider>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Entrypoint">{{ inspectEntrypoint || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Command">{{ inspectCommand || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">运行参数</el-divider>
        <el-descriptions :column="3" border>
          <el-descriptions-item label="User">{{ inspectUser || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Workdir">{{ inspectWorkdir || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Hostname">{{ inspectHostname || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Runtime">{{ inspectRuntime || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Read-only">{{ inspectReadOnly ? 'true' : 'false' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">Labels</el-divider>
        <el-table :fit="true" :data="inspectLabels" style="width: 100%">
          <el-table-column prop="key" label="Key" min-width="200" />
          <el-table-column prop="value" label="Value" min-width="220" />
        </el-table>

        <el-divider content-position="left">Host 配置</el-divider>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Cap Add">{{ inspectCapAdd?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="DNS">{{ inspectDns?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Extra Hosts">{{ inspectExtraHosts?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Security Opt">{{ inspectSecurityOpt?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Sysctls">{{ inspectSysctls?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Ulimits">{{ inspectUlimits?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Devices">{{ inspectDevices?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Tmpfs">{{ inspectTmpfs?.join(', ') || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">日志配置</el-divider>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Log Driver">{{ inspectLogDriver || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Log Opts">{{ inspectLogOpts?.join(', ') || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">资源限制</el-divider>
        <el-descriptions :column="3" border>
          <el-descriptions-item label="CPU Limit">{{ inspectResources?.cpu || '-' }}</el-descriptions-item>
          <el-descriptions-item label="CPU Shares">{{ inspectResources?.cpuShares ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="CPU Quota">{{ inspectResources?.cpuQuota ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="CPU Period">{{ inspectResources?.cpuPeriod ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="Cpuset">{{ inspectResources?.cpuset || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Memory Limit">{{ inspectResources?.memory || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Memory Reservation">{{ inspectResources?.memoryReservation || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Memory Swap">{{ inspectResources?.memorySwap || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Pids Limit">{{ inspectResources?.pidsLimit ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="OOM Kill Disable">{{ inspectResources?.oomKillDisable ? 'true' : 'false' }}</el-descriptions-item>
          <el-descriptions-item label="BlkIO Weight">{{ inspectResources?.blkioWeight ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="Log Driver">{{ inspectResources?.logDriver || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider v-if="inspectHealth" content-position="left">Health</el-divider>
        <el-descriptions v-if="inspectHealth" :column="4" border>
          <el-descriptions-item label="状态">{{ inspectHealth?.Status || '-' }}</el-descriptions-item>
          <el-descriptions-item label="失败次数">{{ inspectHealth?.FailingStreak ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="日志数">{{ inspectHealth?.Log?.length ?? 0 }}</el-descriptions-item>
          <el-descriptions-item label="失败率">{{ healthStats ? `${Math.round(healthStats.failureRate * 100)}%` : '-' }}</el-descriptions-item>
          <el-descriptions-item label="最近检查">{{ healthStats?.lastEnd || '-' }}</el-descriptions-item>
          <el-descriptions-item label="最近退出码">{{ healthStats?.lastExit ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="平均耗时">{{ healthStats?.avgDuration || '-' }}</el-descriptions-item>
          <el-descriptions-item label="最长耗时">{{ healthStats?.maxDuration || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="healthExitStats.length" class="mt-2">
          <span class="text-xs text-gray-500 mr-2">Exit Code 分布</span>
          <el-tag v-for="item in healthExitStats" :key="item.code" class="mr-2" :type="item.code === 0 ? 'success' : 'danger'">
            {{ item.code }}: {{ item.count }}
          </el-tag>
        </div>
        <el-table :fit="true" v-if="healthLogRows.length" :data="healthLogRows.slice(-10)" style="width: 100%">
          <el-table-column prop="startText" label="Start" min-width="180" />
          <el-table-column prop="endText" label="End" min-width="180" />
          <el-table-column prop="duration" label="Duration" width="120" />
          <el-table-column prop="ExitCode" label="Exit" width="80" />
          <el-table-column prop="Output" label="Output" min-width="200" />
        </el-table>

        <el-divider content-position="left">Ports</el-divider>
        <el-table :fit="true" :data="inspectPorts" style="width: 100%">
          <el-table-column prop="container" label="容器端口" width="160" />
          <el-table-column prop="proto" label="协议" width="100" />
          <el-table-column prop="host" label="主机端口" width="160" />
          <el-table-column prop="ip" label="Host IP" width="160" />
          <el-table-column label="复制" width="120">
            <template #default="scope">
              <el-button size="small" @click="copyText(`${scope.row.ip}:${scope.row.host}`)">复制</el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-divider content-position="left">Exposed Ports</el-divider>
        <div class="text-xs text-gray-400" v-if="inspectExposedPorts.length === 0">无</div>
        <el-tag v-else v-for="p in inspectExposedPorts" :key="p" class="mr-2 mb-2">{{ p }}</el-tag>

        <el-divider content-position="left">Networks</el-divider>
        <el-table :fit="true" :data="inspectNetworks" style="width: 100%">
          <el-table-column prop="name" label="名称" width="200" />
          <el-table-column prop="ip" label="IP" width="180" />
          <el-table-column prop="gateway" label="网关" width="180" />
          <el-table-column prop="mac" label="MAC" width="180" />
          <el-table-column prop="aliases" label="Aliases" min-width="180" />
        </el-table>

        <el-divider content-position="left">Mounts</el-divider>
        <el-table :fit="true" :data="inspectMounts" style="width: 100%">
          <el-table-column prop="type" label="类型" width="120" />
          <el-table-column prop="name" label="名称" width="180" />
          <el-table-column prop="source" label="Source" min-width="220" />
          <el-table-column prop="destination" label="Destination" min-width="220" />
          <el-table-column prop="mode" label="Mode" width="120" />
          <el-table-column prop="rw" label="RW" width="80" />
          <el-table-column prop="propagation" label="Propagation" width="140" />
        </el-table>

        <el-divider content-position="left">Binds</el-divider>
        <div class="text-xs text-gray-400" v-if="inspectBinds.length === 0">无</div>
        <el-tag v-else v-for="b in inspectBinds" :key="b" class="mr-2 mb-2">{{ b }}</el-tag>

        <el-divider content-position="left">Env</el-divider>
        <el-input v-model="inspectEnvText" type="textarea" :rows="8" readonly />
      </div>
    </el-dialog>

    <!-- 容器执行命令弹窗 -->
    <el-dialog v-model="execVisible" title="执行容器命令" width="720px" append-to-body>
      <el-alert type="info" :closable="false" show-icon>该功能为非交互命令执行（需要容器内存在 /bin/sh）。</el-alert>
      <div class="log-controls">
        <el-input v-model="execCommand" placeholder="例如: ls / 或 ps aux" />
        <el-button type="primary" @click="runExec" :loading="execLoading">执行</el-button>
      </div>
      <el-input v-model="execOutput" type="textarea" :rows="16" readonly placeholder="输出" />
    </el-dialog>

    <!-- Service 详情弹窗 -->
    <el-dialog v-model="serviceVisible" title="Service 详情" width="90%" append-to-body>
      <el-skeleton v-if="serviceLoading" :rows="6" animated />
      <div v-else>
        <el-descriptions :column="4" border>
          <el-descriptions-item label="任务总数">{{ serviceSummary?.tasks?.total ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="运行中">
            <el-tag type="success">{{ serviceSummary?.tasks?.running ?? 0 }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="失败">
            <el-tag type="danger">{{ serviceSummary?.tasks?.failed ?? 0 }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="关闭">
            <el-tag type="info">{{ serviceSummary?.tasks?.shutdown ?? 0 }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Rejected">{{ serviceSummary?.tasks?.rejected ?? 0 }}</el-descriptions-item>
          <el-descriptions-item label="Pending">{{ serviceSummary?.tasks?.pending ?? 0 }}</el-descriptions-item>
          <el-descriptions-item label="Starting">{{ serviceSummary?.tasks?.starting ?? 0 }}</el-descriptions-item>
          <el-descriptions-item label="Preparing">{{ serviceSummary?.tasks?.preparing ?? 0 }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">Spec</el-divider>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="名称">{{ serviceInspectSummary?.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="镜像">{{ serviceInspectSummary?.image || '-' }}</el-descriptions-item>
          <el-descriptions-item label="镜像引用">{{ serviceInspectImage.ref }}</el-descriptions-item>
          <el-descriptions-item label="镜像 Digest">{{ serviceInspectImage.digest }}</el-descriptions-item>
          <el-descriptions-item label="模式">{{ serviceInspectSummary?.mode || '-' }}</el-descriptions-item>
          <el-descriptions-item label="副本">{{ serviceInspectSummary?.replicas ?? '-' }}</el-descriptions-item>
          <el-descriptions-item label="Endpoint">{{ serviceInspectSummary?.endpoint || '-' }}</el-descriptions-item>
          <el-descriptions-item label="环境变量">{{ serviceInspectSummary?.envCount ?? 0 }}</el-descriptions-item>
          <el-descriptions-item label="标签数量">{{ serviceInspectSummary?.labelCount ?? 0 }}</el-descriptions-item>
          <el-descriptions-item label="命令">{{ serviceInspectSummary?.command || '-' }}</el-descriptions-item>
          <el-descriptions-item label="网络">{{ serviceInspectSummary?.networks?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="约束">{{ serviceInspectSummary?.constraints?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Placement">{{ serviceInspectSummary?.placement?.join(', ') || '-' }}</el-descriptions-item>
          <el-descriptions-item label="挂载">{{ serviceInspectSummary?.mounts?.join(', ') || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">更新/回滚配置</el-divider>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="更新并行">{{ serviceInspectSummary?.update?.parallelism || '-' }}</el-descriptions-item>
          <el-descriptions-item label="更新延迟">{{ serviceInspectSummary?.update?.delay || '-' }}</el-descriptions-item>
          <el-descriptions-item label="更新失败">{{ serviceInspectSummary?.update?.failureAction || '-' }}</el-descriptions-item>
          <el-descriptions-item label="更新顺序">{{ serviceInspectSummary?.update?.order || '-' }}</el-descriptions-item>
          <el-descriptions-item label="回滚并行">{{ serviceInspectSummary?.rollback?.parallelism || '-' }}</el-descriptions-item>
          <el-descriptions-item label="回滚延迟">{{ serviceInspectSummary?.rollback?.delay || '-' }}</el-descriptions-item>
          <el-descriptions-item label="回滚失败">{{ serviceInspectSummary?.rollback?.failureAction || '-' }}</el-descriptions-item>
          <el-descriptions-item label="回滚顺序">{{ serviceInspectSummary?.rollback?.order || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">更新/回滚</el-divider>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="更新状态">{{ serviceSummary?.update?.state || '-' }}</el-descriptions-item>
          <el-descriptions-item label="更新信息">{{ serviceSummary?.update?.message || '-' }}</el-descriptions-item>
          <el-descriptions-item label="开始时间">{{ serviceSummary?.update?.startedAt || '-' }}</el-descriptions-item>
          <el-descriptions-item label="完成时间">{{ serviceSummary?.update?.completedAt || '-' }}</el-descriptions-item>
          <el-descriptions-item label="回滚状态">{{ serviceSummary?.rollback?.state || '-' }}</el-descriptions-item>
          <el-descriptions-item label="回滚信息">{{ serviceSummary?.rollback?.message || '-' }}</el-descriptions-item>
          <el-descriptions-item label="回滚开始">{{ serviceSummary?.rollback?.startedAt || '-' }}</el-descriptions-item>
          <el-descriptions-item label="回滚完成">{{ serviceSummary?.rollback?.completedAt || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-divider content-position="left">任务错误</el-divider>
        <el-table :fit="true" v-if="serviceSummary?.tasks?.errors?.length" :data="serviceSummary?.tasks?.errors?.slice(0, 10) || []" style="width: 100%">
          <el-table-column prop="id" label="ID" min-width="180" />
          <el-table-column prop="name" label="名称" min-width="200" />
          <el-table-column prop="error" label="错误" min-width="240" />
        </el-table>
        <div v-else class="text-xs text-gray-400">暂无错误</div>

        <el-divider content-position="left">JSON</el-divider>
        <el-input v-model="serviceJson" type="textarea" :rows="16" readonly />
      </div>
    </el-dialog>

    <!-- Service 任务弹窗 -->
    <el-dialog v-model="tasksVisible" title="Service 任务" width="90%" append-to-body>
      <div class="tab-toolbar">
        <div class="toolbar-left">
          <el-switch v-model="serviceTaskFilterError" active-text="仅错误" />
          <span class="text-xs text-gray-500 ml-2">显示 {{ filteredServiceTasks.length }}/{{ serviceTasks.length }}</span>
        </div>
        <div class="toolbar-right">
          <el-button icon="Refresh" @click="reloadServiceTasks">刷新</el-button>
        </div>
      </div>
      <el-table :fit="true" :data="filteredServiceTasks" v-loading="tasksLoading" style="width: 100%">
        <el-table-column prop="ID" label="ID" min-width="180" />
        <el-table-column prop="Name" label="名称" min-width="200" />
        <el-table-column prop="Node" label="节点" width="160" />
        <el-table-column prop="DesiredState" label="期望状态" width="120" />
        <el-table-column prop="CurrentState" label="当前状态" min-width="200" />
        <el-table-column prop="Error" label="错误" min-width="200" />
      </el-table>
    </el-dialog>

    <!-- Stack 服务弹窗 -->
    <el-dialog v-model="stackVisible" title="Stack 服务" width="90%" append-to-body>
      <el-descriptions v-if="stackSummary" :column="4" border>
        <el-descriptions-item label="服务数">{{ stackSummary.total }}</el-descriptions-item>
        <el-descriptions-item label="运行中">{{ stackSummary.running }}</el-descriptions-item>
        <el-descriptions-item label="期望副本">{{ stackSummary.desired }}</el-descriptions-item>
        <el-descriptions-item label="副本异常">
          <el-tag :type="stackSummary.warning > 0 ? 'danger' : 'success'">{{ stackSummary.warning }}</el-tag>
        </el-descriptions-item>
      </el-descriptions>
      <el-descriptions v-if="stackMeta" :column="2" border class="mt-2">
        <el-descriptions-item label="Networks">{{ stackMeta.networks?.join(', ') || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Images">{{ stackMeta.images?.join(', ') || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Labels">{{ stackMeta.labels?.join(', ') || '-' }}</el-descriptions-item>
      </el-descriptions>
      <el-table :fit="true" :data="stackServiceRows" v-loading="stackLoading" style="width: 100%">
        <el-table-column prop="Name" label="名称" min-width="220" />
        <el-table-column prop="Mode" label="模式" width="120" />
        <el-table-column prop="Replicas" label="副本" width="120" />
        <el-table-column prop="running" label="运行中" width="100" />
        <el-table-column prop="desired" label="期望" width="100" />
        <el-table-column label="差异" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.delta < 0" type="danger">{{ row.delta }}</el-tag>
            <el-tag v-else-if="row.delta > 0" type="warning">+{{ row.delta }}</el-tag>
            <el-tag v-else type="success">0</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="Image" label="镜像" min-width="180" />
        <el-table-column prop="Ports" label="端口" min-width="180" />
      </el-table>
    </el-dialog>

    <!-- Git 部署 Stack -->
    <el-dialog append-to-body v-model="gitDeployVisible" title="Git 部署 Stack" width="720px">
      <el-form :model="gitDeployForm" label-width="120px">
        <el-form-item label="Stack 名称" required>
          <el-input v-model="gitDeployForm.name" placeholder="例如 my-stack" />
        </el-form-item>
        <el-form-item label="仓库地址" required>
          <el-input v-model="gitDeployForm.repo" placeholder="https://github.com/org/repo.git" />
        </el-form-item>
        <el-form-item label="分支">
          <el-input v-model="gitDeployForm.branch" placeholder="main" />
        </el-form-item>
        <el-form-item label="Compose 路径">
          <el-input v-model="gitDeployForm.compose_path" placeholder="docker-compose.yml" />
        </el-form-item>
        <el-form-item label="认证方式">
          <el-select v-model="gitDeployForm.authType" class="w-40">
            <el-option label="无" value="none" />
            <el-option label="Token" value="token" />
            <el-option label="用户名/密码" value="basic" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="gitDeployForm.authType === 'token'" label="Token">
          <el-input v-model="gitDeployForm.token" show-password />
        </el-form-item>
        <el-form-item v-if="gitDeployForm.authType === 'basic'" label="用户名">
          <el-input v-model="gitDeployForm.username" />
        </el-form-item>
        <el-form-item v-if="gitDeployForm.authType === 'basic'" label="密码">
          <el-input v-model="gitDeployForm.password" type="password" show-password />
        </el-form-item>
      </el-form>
      <el-input v-model="gitDeployOutput" type="textarea" :rows="6" readonly placeholder="输出" />
      <template #footer>
        <el-button @click="gitDeployVisible = false">关闭</el-button>
        <el-button type="primary" :loading="gitDeployLoading" @click="submitGitDeploy">部署</el-button>
      </template>
    </el-dialog>

    <!-- 创建卷 -->
    <el-dialog append-to-body v-model="createVolumeVisible" title="创建卷" width="520px">
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
    <el-dialog append-to-body v-model="volumeInspectVisible" title="卷详情" width="760px">
      <el-input v-model="volumeInspectJson" type="textarea" :rows="14" readonly />
      <template #footer>
        <el-button @click="copyText(volumeInspectJson)">复制</el-button>
        <el-button type="primary" @click="downloadJson('volume.json', volumeInspectJson)">导出</el-button>
      </template>
    </el-dialog>

    <!-- 卷使用情况 -->
    <el-dialog append-to-body v-model="volumeUsageVisible" title="卷使用情况" width="720px">
      <el-table :fit="true" :data="volumeUsage" v-loading="volumeUsageLoading" style="width: 100%">
        <el-table-column prop="name" label="卷" min-width="200" />
        <el-table-column label="容器" min-width="240">
          <template #default="{ row }">
            <el-tag v-for="c in row.containers" :key="c" class="mr-1 mb-1">{{ c }}</el-tag>
            <span v-if="!row.containers || row.containers.length === 0" class="text-xs text-gray-400">无</span>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 网络详情 -->
    <el-dialog append-to-body v-model="networkInspectVisible" title="网络详情" width="760px">
      <div class="mb-3">
        <div class="text-sm mb-2">连接的容器：</div>
        <el-tag v-for="item in networkInspectContainers" :key="item" class="mr-2 mb-2">{{ item }}</el-tag>
        <div v-if="networkInspectContainers.length === 0" class="text-xs text-gray-400">无容器连接</div>
      </div>
      <el-input v-model="networkInspectJson" type="textarea" :rows="12" readonly />
      <template #footer>
        <el-button @click="copyText(networkInspectJson)">复制</el-button>
        <el-button type="primary" @click="downloadJson('network.json', networkInspectJson)">导出</el-button>
      </template>
    </el-dialog>

    <!-- 网络使用情况 -->
    <el-dialog append-to-body v-model="networkUsageVisible" title="网络使用情况" width="720px">
      <el-table :fit="true" :data="networkUsage" v-loading="networkUsageLoading" style="width: 100%">
        <el-table-column prop="name" label="网络" min-width="200" />
        <el-table-column prop="driver" label="驱动" width="120" />
        <el-table-column prop="scope" label="范围" width="120" />
        <el-table-column label="容器" min-width="240">
          <template #default="{ row }">
            <el-tag v-for="c in row.containers" :key="c" class="mr-1 mb-1">{{ c }}</el-tag>
            <span v-if="!row.containers || row.containers.length === 0" class="text-xs text-gray-400">无</span>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 创建 Secret -->
    <el-dialog append-to-body v-model="createSecretVisible" title="创建 Secret" width="520px">
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
    <el-dialog append-to-body v-model="secretInspectVisible" title="Secret 详情" width="760px">
      <el-input v-model="secretInspectJson" type="textarea" :rows="12" readonly />
      <template #footer>
        <el-button @click="copyText(secretInspectJson)">复制</el-button>
        <el-button type="primary" @click="downloadJson('secret.json', secretInspectJson)">导出</el-button>
      </template>
    </el-dialog>

    <!-- 创建 Config -->
    <el-dialog append-to-body v-model="createConfigVisible" title="创建 Config" width="520px">
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
    <el-dialog append-to-body v-model="configInspectVisible" title="Config 详情" width="760px">
      <el-input v-model="configInspectJson" type="textarea" :rows="12" readonly />
      <template #footer>
        <el-button @click="copyText(configInspectJson)">复制</el-button>
        <el-button type="primary" @click="downloadJson('config.json', configInspectJson)">导出</el-button>
      </template>
    </el-dialog>

    <!-- 部署 Stack -->
    <el-dialog append-to-body v-model="deployVisible" title="部署 Stack" width="760px">
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
    <el-dialog v-model="terminalVisible" title="容器终端" width="920px" append-to-body @closed="closeTerminal">
      <div class="terminal-toolbar">
        <span class="terminal-title">{{ terminalContainerName || terminalContainerId }}</span>
        <el-select v-model="terminalShell" class="w-28">
          <el-option label="/bin/sh" value="/bin/sh" />
          <el-option label="/bin/bash" value="/bin/bash" />
          <el-option label="/bin/ash" value="/bin/ash" />
        </el-select>
        <el-button
          type="primary"
          :disabled="!terminalContainerId"
          :loading="terminalConnecting"
          @click="toggleTerminal"
        >
          {{ terminalConnected ? '断开' : '连接' }}
        </el-button>
      </div>
      <div ref="terminalRef" class="terminal-container"></div>
    </el-dialog>

    <!-- 添加仓库 -->
    <el-dialog append-to-body v-model="createRegistryVisible" title="添加仓库" width="520px">
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
import * as echarts from 'echarts'
import 'xterm/css/xterm.css'

const loading = ref(false)
const tableData = ref([])
const dialogVisible = ref(false)
const submitting = ref(false)
const hosts = ref([])
const onlineHosts = computed(() => (tableData.value || []).filter((item) => item.status === 'online').length)
const offlineHosts = computed(() => Math.max(0, (tableData.value || []).length - onlineHosts.value))

const manageVisible = ref(false)
const manageTab = ref('overview')
const activeHost = ref(null)

const containers = ref([])
const containersLoading = ref(false)
const containerTableRef = ref(null)
const selectedContainers = ref([])
const containerFilters = reactive({
  keyword: '',
  state: ''
})
const statsLoading = ref(false)
const containerStats = ref({})
const containerStatsHistory = ref({})
const statsChartVisible = ref(false)
const statsChartRef = ref(null)
const statsChartInstance = ref(null)
const statsChartContainerId = ref('')
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
const networkInspectVisible = ref(false)
const networkInspectJson = ref('')
const networkInspectContainers = ref([])
const networkUsageVisible = ref(false)
const networkUsageLoading = ref(false)
const networkUsage = ref([])

const volumes = ref([])
const volumesLoading = ref(false)
const volumeTableRef = ref(null)
const selectedVolumes = ref([])
const volumeUsageVisible = ref(false)
const volumeUsageLoading = ref(false)
const volumeUsage = ref([])

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

const events = ref([])
const eventsLoading = ref(false)
const eventFilters = reactive({
  type: '',
  action: '',
  container: '',
  image: '',
  volume: '',
  network: '',
  node: '',
  service: '',
  sinceMinutes: 60,
  limit: 200
})

const buildVisible = ref(false)
const buildLoading = ref(false)
const buildOutput = ref('')
const buildArgsText = ref('')
const buildForm = reactive({
  tag: '',
  dockerfile: '',
  contextTar: ''
})

const loadVisible = ref(false)
const loadLoading = ref(false)
const loadOutput = ref('')
const loadForm = reactive({
  tar: ''
})

const gitDeployVisible = ref(false)
const gitDeployLoading = ref(false)
const gitDeployOutput = ref('')
const gitDeployForm = reactive({
  name: '',
  repo: '',
  branch: 'main',
  compose_path: 'docker-compose.yml',
  authType: 'none',
  username: '',
  password: '',
  token: ''
})

const terminalVisible = ref(false)
const terminalContainerId = ref('')
const terminalContainerName = ref('')
const terminalShell = ref('/bin/sh')
const terminalConnected = ref(false)
const terminalConnecting = ref(false)
const terminalRef = ref(null)
let terminal = null
let terminalFit = null
let terminalWs = null

const services = ref([])
const servicesLoading = ref(false)
const serviceStackFilter = ref('')
const serviceTableRef = ref(null)
const selectedServices = ref([])
const serviceFilters = reactive({
  keyword: ''
})
const serviceScaleMap = reactive({})
const batchServiceScale = ref(1)
const createServiceVisible = ref(false)
const createServiceLoading = ref(false)
const serviceCreateForm = reactive({
  name: '',
  image: '',
  mode: 'replicated',
  endpoint_mode: 'vip',
  replicas: 1,
  ports: '',
  env: '',
  labels: '',
  networks: '',
  constraints: '',
  placement_prefs: '',
  max_replicas_per_node: '',
  mounts: '',
  command: '',
  restart_condition: '',
  update_parallelism: '',
  update_delay: '',
  update_failure_action: '',
  update_order: '',
  rollback_parallelism: '',
  rollback_delay: '',
  rollback_failure_action: '',
  rollback_order: '',
  limit_cpu: '',
  limit_memory: '',
  reserve_cpu: '',
  reserve_memory: ''
})
const editServiceVisible = ref(false)
const editServiceLoading = ref(false)
const editServiceSaving = ref(false)
const serviceEditForm = reactive({
  id: '',
  name: '',
  image: '',
  mode: '',
  endpoint_mode: '',
  replicas: 1,
  ports: '',
  env: '',
  labels: '',
  networks: '',
  constraints: '',
  reset_constraints: true,
  placement_prefs: '',
  reset_placement_prefs: true,
  mounts: '',
  reset_mounts: true,
  command: '',
  reset_command: false,
  max_replicas_per_node: '',
  reset_env: true,
  reset_labels: true,
  reset_ports: true,
  reset_networks: true,
  restart_condition: '',
  update_parallelism: '',
  update_delay: '',
  update_failure_action: '',
  update_order: '',
  rollback_parallelism: '',
  rollback_delay: '',
  rollback_failure_action: '',
  rollback_order: '',
  limit_cpu: '',
  limit_memory: '',
  reserve_cpu: '',
  reserve_memory: ''
})
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
const inspectLabels = ref([])
const inspectCommand = ref('')
const inspectEntrypoint = ref('')
const inspectRestartPolicy = ref('')
const inspectNetworkMode = ref('')
const inspectPrivileged = ref(false)
const inspectCapAdd = ref([])
const inspectDns = ref([])
const inspectExtraHosts = ref([])
const inspectHealth = ref(null)
const inspectResources = ref(null)
const inspectUser = ref('')
const inspectWorkdir = ref('')
const inspectHostname = ref('')
const inspectRuntime = ref('')
const inspectReadOnly = ref(false)
const inspectSecurityOpt = ref([])
const inspectSysctls = ref([])
const inspectUlimits = ref([])
const inspectDevices = ref([])
const inspectTmpfs = ref([])
const inspectLogDriver = ref('')
const inspectLogOpts = ref([])
const inspectExposedPorts = ref([])
const inspectBinds = ref([])
const serviceTaskFilterError = ref(false)

const execVisible = ref(false)
const execLoading = ref(false)
const execCommand = ref('ls /')
const execOutput = ref('')
const execContainerId = ref('')

const serviceVisible = ref(false)
const serviceLoading = ref(false)
const serviceJson = ref('')
const serviceSummary = ref(null)
const serviceInspectData = ref(null)
const tasksVisible = ref(false)
const tasksLoading = ref(false)
const serviceTasks = ref([])
const serviceTasksTargetId = ref('')
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
const stackSummary = ref(null)
const stackMeta = ref(null)

const serviceStacks = computed(() => {
  const set = new Set()
  services.value.forEach((s) => {
    const name = s.Name || ''
    const parts = name.split('_')
    if (parts.length > 1) set.add(parts[0])
  })
  return Array.from(set)
})

const filteredContainers = computed(() => {
  let rows = containers.value
  const keyword = String(containerFilters.keyword || '').trim().toLowerCase()
  if (keyword) {
    rows = rows.filter((c) => {
      const names = Array.isArray(c.names) ? c.names.join(',') : String(c.names || '')
      const hay = `${names} ${c.image || ''} ${c.id || ''} ${c.status || ''}`.toLowerCase()
      return hay.includes(keyword)
    })
  }
  const state = String(containerFilters.state || '').trim().toLowerCase()
  if (state) {
    rows = rows.filter(c => String(c.state || '').toLowerCase() === state)
  }
  return rows
})

const filteredServices = computed(() => {
  let rows = services.value
  if (serviceStackFilter.value) {
    rows = rows.filter(s => (s.Name || '').startsWith(`${serviceStackFilter.value}_`))
  }
  const keyword = String(serviceFilters.keyword || '').trim().toLowerCase()
  if (keyword) {
    rows = rows.filter((s) => {
      const hay = `${s.Name || ''} ${s.Image || ''} ${s.Ports || ''}`.toLowerCase()
      return hay.includes(keyword)
    })
  }
  return rows
})

const filteredServiceTasks = computed(() => {
  if (!serviceTaskFilterError.value) return serviceTasks.value
  return (serviceTasks.value || []).filter(task => isTaskError(task))
})

const stackServiceRows = computed(() => {
  return (stackServices.value || []).map((row) => {
    const stats = parseReplicaStats(row.Replicas)
    return {
      ...row,
      running: stats.running,
      desired: stats.desired,
      delta: stats.delta
    }
  })
})

const healthLogRows = computed(() => {
  const logs = inspectHealth.value?.Log || []
  return logs.map((row) => {
    const durationMs = calcDurationMs(row.Start, row.End)
    return {
      ...row,
      startText: formatTime(row.Start),
      endText: formatTime(row.End),
      durationMs,
      duration: formatMs(durationMs)
    }
  })
})

const healthExitStats = computed(() => {
  const logs = inspectHealth.value?.Log || []
  const map = new Map()
  logs.forEach((row) => {
    const code = Number.isFinite(row.ExitCode) ? row.ExitCode : Number(row.ExitCode || 0)
    const key = Number.isNaN(code) ? 0 : code
    map.set(key, (map.get(key) || 0) + 1)
  })
  return Array.from(map.entries())
    .map(([code, count]) => ({ code, count }))
    .sort((a, b) => a.code - b.code)
})

const healthStats = computed(() => {
  const logs = inspectHealth.value?.Log || []
  if (!logs.length) return null
  let failures = 0
  const durations = []
  logs.forEach((row) => {
    if (Number(row.ExitCode) !== 0) failures += 1
    const durationMs = calcDurationMs(row.Start, row.End)
    if (durationMs !== null) durations.push(durationMs)
  })
  const total = logs.length
  const avg = durations.length ? durations.reduce((a, b) => a + b, 0) / durations.length : null
  const max = durations.length ? Math.max(...durations) : null
  const last = logs[logs.length - 1]
  return {
    total,
    failures,
    failureRate: total ? failures / total : 0,
    avgDuration: formatMs(avg),
    maxDuration: formatMs(max),
    lastExit: last?.ExitCode ?? '-',
    lastEnd: formatTime(last?.End),
    lastStart: formatTime(last?.Start)
  }
})

const topologyTree = computed(() => {
  const tree = []

  const stackMap = new Map()
  services.value.forEach((s) => {
    const name = s.Name || s.name || ''
    const stack = name.includes('_') ? name.split('_')[0] : 'default'
    const arr = stackMap.get(stack) || []
    arr.push({
      id: name,
      label: `${name} (${s.Replicas || '-'})`
    })
    stackMap.set(stack, arr)
  })
  if (stackMap.size > 0) {
    const stacks = []
    stackMap.forEach((children, stack) => {
      stacks.push({ id: `stack-${stack}`, label: stack, children })
    })
    tree.push({ id: 'stacks-root', label: 'Stacks', children: stacks })
  }

  if (networkUsage.value.length > 0) {
    const networks = networkUsage.value.map((n) => ({
      id: `net-${n.id || n.name}`,
      label: `${n.name} (${(n.containers || []).length})`,
      children: (n.containers || []).map(c => ({ id: `net-${n.name}-${c}`, label: c }))
    }))
    tree.push({ id: 'networks-root', label: 'Networks', children: networks })
  }

  if (volumeUsage.value.length > 0) {
    const volumes = volumeUsage.value.map((v) => ({
      id: `vol-${v.name}`,
      label: `${v.name} (${(v.containers || []).length})`,
      children: (v.containers || []).map(c => ({ id: `vol-${v.name}-${c}`, label: c }))
    }))
    tree.push({ id: 'volumes-root', label: 'Volumes', children: volumes })
  }

  return tree
})

const createVisible = ref(false)
const createLoading = ref(false)
const createForm = reactive({
  image: '',
  name: '',
  ports: '',
  env: '',
  labels: '',
  networks: '',
  mounts: '',
  entrypoint: '',
  command: '',
  privileged: false,
  cap_add: '',
  network_mode: '',
  dns: '',
  extra_hosts: '',
  health_disable: false,
  health_cmd: '',
  health_interval: '',
  health_timeout: '',
  health_retries: '',
  health_start_period: '',
  user: '',
  workdir: '',
  hostname: '',
  read_only: false,
  runtime: '',
  cpus: '',
  memory: '',
  memory_reservation: '',
  pids_limit: '',
  ulimits: '',
  sysctls: '',
  security_opt: '',
  devices: '',
  tmpfs: '',
  log_driver: '',
  log_opts: '',
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
  const match = text.match(/^([0-9.]+)\s*([KMGTP]?i?B)$/i)
  if (!match) return NaN
  const num = Number(match[1])
  if (Number.isNaN(num)) return NaN
  const unit = match[2].toUpperCase()
  switch (unit) {
    case 'KIB': return num / 1024
    case 'MIB': return num
    case 'GIB': return num * 1024
    case 'TIB': return num * 1024 * 1024
    case 'PIB': return num * 1024 * 1024 * 1024
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
    ElMessage.error(extractErrorMessage(e, '加载失败'))
  } finally {
    loading.value = false
  }
}

const syncAll = async () => {
  loading.value = true
  try {
    await axios.post('/api/v1/docker/hosts/sync', {}, { headers: authHeaders() })
  } catch (e) {
    ElMessage.error(extractErrorMessage(e, '同步失败'))
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
  if (manageTab.value === 'events') await loadEvents()
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
  } catch (e) {
    containers.value = []
    ElMessage.error(extractErrorMessage(e))
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
        const id = String(row.Container || row.ID || row.Id || row.id || '').trim()
        const shortID = id.length > 12 ? id.slice(0, 12) : id
        const name = String(row.Name || row.name || '').trim()
        const item = {
          cpu: row.CPUPerc || row.CPU || '-',
          mem: row.MemUsage || row.Memory || '-',
          net: row.NetIO || row.Network || '-'
        }
        if (id) map[id] = item
        if (shortID) map[shortID] = item
        if (name) map[name] = item
        updateStatsHistory([id, shortID, name], row)
      })
      containerStats.value = map
      if (statsChartVisible.value && statsChartContainerId.value) {
        renderStatsChart()
      }
    }
  } catch (e) {
    containerStats.value = {}
    ElMessage.error(extractErrorMessage(e, '获取容器资源失败'))
  } finally {
    statsLoading.value = false
  }
}

const parsePercent = (value) => {
  if (!value) return 0
  const num = Number(String(value).replace('%', '').trim())
  return Number.isNaN(num) ? 0 : num
}

const parseUsagePair = (value) => {
  if (!value) return { used: 0, total: 0 }
  const parts = String(value).split('/')
  const used = parseSizeToMB(parts[0] || '')
  const total = parseSizeToMB(parts[1] || '')
  return {
    used: Number.isNaN(used) ? 0 : used,
    total: Number.isNaN(total) ? 0 : total
  }
}

const parseNetPair = (value) => {
  if (!value) return { rx: 0, tx: 0 }
  const parts = String(value).split('/')
  const rx = parseSizeToMB(parts[0] || '')
  const tx = parseSizeToMB(parts[1] || '')
  return {
    rx: Number.isNaN(rx) ? 0 : rx,
    tx: Number.isNaN(tx) ? 0 : tx
  }
}

const updateStatsHistory = (keys, row) => {
  const aliases = (Array.isArray(keys) ? keys : [keys]).map(v => String(v || '').trim()).filter(Boolean)
  if (aliases.length === 0) return

  let history = null
  for (const key of aliases) {
    if (containerStatsHistory.value[key]?.length) {
      history = containerStatsHistory.value[key]
      break
    }
  }
  if (!history) history = []

  const mem = parseUsagePair(row.MemUsage || row.Memory || '')
  const net = parseNetPair(row.NetIO || row.Network || '')
  history.push({
    t: Date.now(),
    cpu: parsePercent(row.CPUPerc || row.CPU || ''),
    mem: mem.used,
    netRx: net.rx,
    netTx: net.tx
  })
  if (history.length > 60) {
    history.splice(0, history.length - 60)
  }
  aliases.forEach((key) => {
    containerStatsHistory.value[key] = history
  })
}

const getContainerAliases = (row) => {
  const aliases = new Set()
  const id = String(row?.id || '').trim()
  if (id) {
    aliases.add(id)
    if (id.length > 12) aliases.add(id.slice(0, 12))
  }
  String(row?.names || '')
    .split(',')
    .map(v => v.trim())
    .filter(Boolean)
    .forEach((name) => aliases.add(name))
  return Array.from(aliases)
}

const isContainerRunning = (row) => {
  const state = String(row?.state || '').toLowerCase()
  const status = String(row?.status || '').toLowerCase()
  return state === 'running' || status.includes('up')
}

const getContainerStats = (row) => {
  const aliases = getContainerAliases(row)
  for (const key of aliases) {
    if (containerStats.value[key]) return containerStats.value[key]
  }
  return { cpu: '-', mem: '-', net: '-' }
}

const getContainerRef = (row) => getContainerAliases(row)[0] || ''

const getContainerHistory = (row) => {
  const aliases = getContainerAliases(row)
  for (const key of aliases) {
    if (containerStatsHistory.value[key]?.length) return containerStatsHistory.value[key]
  }
  return []
}

const openStatsChart = async (row) => {
  if (!isContainerRunning(row)) {
    ElMessage.warning('容器未运行，暂无实时趋势')
    return
  }
  const id = getContainerRef(row)
  if (!id) return
  await loadContainerStats()
  statsChartContainerId.value = id
  statsChartVisible.value = true
  await nextTick()
  initStatsChart()
  renderStatsChart()
}

const initStatsChart = () => {
  if (statsChartInstance.value || !statsChartRef.value) return
  statsChartInstance.value = echarts.init(statsChartRef.value)
}

const renderStatsChart = () => {
  if (!statsChartInstance.value || !statsChartContainerId.value) return
  const history = getContainerHistory({ id: statsChartContainerId.value }) || []
  const times = history.map(h => formatTime(h.t))
  const cpu = history.map(h => h.cpu)
  const mem = history.map(h => h.mem)
  const netRx = history.map(h => h.netRx)
  const netTx = history.map(h => h.netTx)
  statsChartInstance.value.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['CPU %', 'MEM MB', 'NET RX MB', 'NET TX MB'] },
    grid: { left: 50, right: 20, top: 40, bottom: 40 },
    xAxis: { type: 'category', data: times, axisLabel: { rotate: 45 } },
    yAxis: { type: 'value' },
    series: [
      { name: 'CPU %', type: 'line', data: cpu, smooth: true },
      { name: 'MEM MB', type: 'line', data: mem, smooth: true },
      { name: 'NET RX MB', type: 'line', data: netRx, smooth: true },
      { name: 'NET TX MB', type: 'line', data: netTx, smooth: true }
    ]
  })
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

watch(statsChartVisible, (val) => {
  if (!val) {
    if (statsChartInstance.value) {
      statsChartInstance.value.dispose()
      statsChartInstance.value = null
    }
  } else {
    nextTick(() => {
      initStatsChart()
      renderStatsChart()
    })
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

const openNetworkInspect = async (row) => {
  if (!activeHost.value || !row?.id) return
  networkInspectVisible.value = true
  networkInspectJson.value = ''
  networkInspectContainers.value = []
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/networks/${encodeURIComponent(row.id)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      const info = res.data.data || {}
      const containers = info.Containers || {}
      const names = Object.values(containers).map(c => c?.Name).filter(Boolean)
      networkInspectContainers.value = names
      networkInspectJson.value = JSON.stringify(info, null, 2)
    }
  } catch (e) {
    networkInspectJson.value = extractErrorMessage(e)
  }
}

const loadNetworkUsage = async () => {
  if (!activeHost.value) return
  networkUsageLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/networks/usage`, { headers: authHeaders() })
    if (res.data.code === 0) {
      networkUsage.value = res.data.data || []
      networkUsageVisible.value = true
    }
  } finally {
    networkUsageLoading.value = false
  }
}

const normalizeEventRow = (row) => {
  const actor = row.Actor || {}
  const attrs = actor.Attributes || {}
  const name = attrs.name || attrs.container || attrs.image || attrs.volume || attrs.network || attrs.service || ''
  const detail = Object.entries(attrs)
    .slice(0, 6)
    .map(([k, v]) => `${k}=${v}`)
    .join(', ')
  return {
    time: formatTime(row.time || row.Time || ''),
    type: row.Type || '-',
    action: row.Action || '-',
    name,
    id: actor.ID || '',
    detail
  }
}

const loadEvents = async () => {
  if (!activeHost.value) return
  eventsLoading.value = true
  try {
    const params = {
      since: `${eventFilters.sinceMinutes}m`,
      type: eventFilters.type || undefined,
      event: eventFilters.action || undefined,
      container: eventFilters.container || undefined,
      image: eventFilters.image || undefined,
      volume: eventFilters.volume || undefined,
      network: eventFilters.network || undefined,
      node: eventFilters.node || undefined,
      service: eventFilters.service || undefined
    }
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/events`, { params, headers: authHeaders() })
    if (res.data.code === 0) {
      const list = res.data.data || []
      const normalized = list.map(normalizeEventRow).reverse()
      const limit = eventFilters.limit || 200
      events.value = normalized.slice(0, limit)
    }
  } finally {
    eventsLoading.value = false
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

const loadVolumeUsage = async () => {
  if (!activeHost.value) return
  volumeUsageLoading.value = true
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/volumes/usage`, { headers: authHeaders() })
    if (res.data.code === 0) {
      volumeUsage.value = res.data.data || []
      volumeUsageVisible.value = true
    }
  } finally {
    volumeUsageLoading.value = false
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
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services`, {
      params: { detail: 1 },
      headers: authHeaders()
    })
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

const extractErrorMessage = (e, fallback = '操作失败') => {
  const msg = e?.response?.data?.message || e?.message || fallback
  return String(msg)
}

const parseKeyValueText = (text) => {
  const result = {}
  String(text || '')
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean)
    .forEach((line) => {
      const idx = line.indexOf('=')
      if (idx > 0) {
        const key = line.slice(0, idx).trim()
        const val = line.slice(idx + 1).trim()
        if (key) result[key] = val
      }
    })
  return result
}

const parseListText = (text) => {
  return String(text || '')
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean)
}

const parseCommandText = (text) => {
  return String(text || '')
    .trim()
    .split(/\s+/)
    .filter(Boolean)
}

const isValidIPv4 = (value) => {
  const parts = String(value || '').trim().split('.')
  if (parts.length !== 4) return false
  return parts.every((p) => {
    if (!/^\d+$/.test(p)) return false
    const n = Number(p)
    return n >= 0 && n <= 255
  })
}

const isValidIPv6 = (value) => {
  const text = String(value || '').trim()
  if (!text.includes(':')) return false
  return /^[0-9a-fA-F:]+$/.test(text)
}

const isValidIp = (value) => isValidIPv4(value) || isValidIPv6(value)

const isValidHostname = (value) => {
  const text = String(value || '').trim()
  if (!text) return false
  return /^[a-zA-Z0-9.-]+$/.test(text)
}

const formatServiceMount = (mount) => {
  if (!mount) return ''
  const parts = []
  if (mount.Type) parts.push(`type=${mount.Type}`)
  if (mount.Source) parts.push(`src=${mount.Source}`)
  if (mount.Target) parts.push(`dst=${mount.Target}`)
  if (mount.ReadOnly) parts.push('readonly')
  return parts.join(',')
}

const extractTaskState = (state) => {
  const text = String(state || '').trim()
  if (!text) return 'unknown'
  return text.split(' ')[0].toLowerCase()
}

const isTaskError = (task) => {
  if (!task) return false
  if (task.Error || task.error) return true
  const state = extractTaskState(task.CurrentState || task.currentstate)
  return state === 'failed' || state === 'rejected'
}

const summarizeServiceTasks = (tasks) => {
  const summary = {
    total: tasks.length,
    running: 0,
    failed: 0,
    shutdown: 0,
    rejected: 0,
    pending: 0,
    starting: 0,
    preparing: 0,
    assigned: 0,
    accepted: 0,
    ready: 0,
    unknown: 0,
    errors: []
  }
  tasks.forEach((t) => {
    const state = extractTaskState(t.CurrentState || t.currentstate)
    if (summary[state] === undefined) {
      summary.unknown += 1
    } else {
      summary[state] += 1
    }
    const err = t.Error || t.error
    if (err) {
      summary.errors.push({
        id: t.ID || t.Id || t.id || '',
        name: t.Name || t.name || '',
        error: err
      })
    }
  })
  return summary
}

const buildServiceInspectSummary = (data) => {
  if (!data) return null
  const spec = data.Spec || {}
  const task = spec.TaskTemplate || {}
  const containerSpec = task.ContainerSpec || {}
  const mode = spec.Mode?.Global ? 'global' : 'replicated'
  const replicas = spec.Mode?.Replicated?.Replicas
  const endpointMode = spec.EndpointSpec?.Mode || ''
  const networks = (task.Networks || []).map(n => n?.Target || n?.Name).filter(Boolean)
  const constraints = task.Placement?.Constraints || []
  const prefs = (task.Placement?.Preferences || []).map(p => p?.Spread?.SpreadDescriptor).filter(Boolean).map(v => `spread=${v}`)
  const mounts = (containerSpec.Mounts || []).map(formatServiceMount).filter(Boolean)
  const command = containerSpec.Command || containerSpec.Args || []
  const updateCfg = spec.UpdateConfig || {}
  const rollbackCfg = spec.RollbackConfig || {}
  return {
    name: spec.Name || spec?.Annotations?.Name || '-',
    image: containerSpec.Image || '-',
    mode,
    replicas: replicas === undefined ? '-' : replicas,
    endpoint: endpointMode || '-',
    networks,
    constraints,
    placement: prefs,
    mounts,
    command: command.join(' ') || '-',
    envCount: (containerSpec.Env || []).length,
    labelCount: Object.keys(spec.Labels || {}).length,
    update: {
      parallelism: updateCfg.Parallelism ?? '-',
      delay: formatDuration(updateCfg.Delay) || '-',
      failureAction: updateCfg.FailureAction || '-',
      order: updateCfg.Order || '-'
    },
    rollback: {
      parallelism: rollbackCfg.Parallelism ?? '-',
      delay: formatDuration(rollbackCfg.Delay) || '-',
      failureAction: rollbackCfg.FailureAction || '-',
      order: rollbackCfg.Order || '-'
    }
  }
}

const serviceInspectSummary = computed(() => buildServiceInspectSummary(serviceInspectData.value))

const serviceInspectImage = computed(() => {
  const image = serviceInspectData.value?.Spec?.TaskTemplate?.ContainerSpec?.Image || ''
  if (!image) return { ref: '-', digest: '-' }
  const parts = String(image).split('@')
  return {
    ref: parts[0] || image,
    digest: parts[1] ? `@${parts[1]}` : '-'
  }
})

const formatStatusInfo = (status) => {
  if (!status) return null
  return {
    state: status.State || status.state || '-',
    message: status.Message || status.message || '-',
    startedAt: formatTime(status.StartedAt || status.startedAt || ''),
    completedAt: formatTime(status.CompletedAt || status.completedAt || '')
  }
}

const formatServiceStatusTag = (status) => {
  if (!status) return { type: 'info', text: '-' }
  const state = String(status.State || status.state || '').toLowerCase()
  if (!state) return { type: 'info', text: '-' }
  if (state.includes('update') || state.includes('updat')) return { type: 'warning', text: status.State }
  if (state.includes('rollback')) return { type: 'warning', text: status.State }
  if (state.includes('pause') || state.includes('paused')) return { type: 'danger', text: status.State }
  if (state.includes('complete') || state.includes('finished')) return { type: 'success', text: status.State }
  if (state.includes('fail')) return { type: 'danger', text: status.State }
  return { type: 'info', text: status.State }
}

const listToCommandText = (list) => {
  return (list || []).filter(Boolean).join(' ')
}

const serviceCreateIsGlobal = computed(() => String(serviceCreateForm.mode || '').toLowerCase() === 'global')
const serviceEditIsGlobal = computed(() => String(serviceEditForm.mode || '').toLowerCase() === 'global')

const buildServicePreview = (form, options = {}) => {
  const lines = []
  const name = form.name || '-'
  const image = form.image || '-'
  lines.push(`名称: ${name}`)
  lines.push(`镜像: ${image}`)
  const mode = form.mode ? String(form.mode) : 'replicated'
  lines.push(`模式: ${mode}`)
  if (mode === 'replicated') {
    const replicas = form.replicas !== undefined && form.replicas !== null ? form.replicas : '-'
    lines.push(`副本: ${replicas}`)
    if (form.max_replicas_per_node) {
      lines.push(`每节点上限: ${form.max_replicas_per_node}`)
    }
  }
  if (form.endpoint_mode) {
    lines.push(`Endpoint: ${form.endpoint_mode}`)
  }
  const ports = parseListText(form.ports || '')
  if (ports.length) lines.push(`端口发布: ${ports.join(', ')}`)
  const env = parseKeyValueText(form.env || '')
  const labels = parseKeyValueText(form.labels || '')
  if (Object.keys(env).length) lines.push(`环境变量: ${Object.keys(env).length} 项`)
  if (Object.keys(labels).length) lines.push(`标签: ${Object.keys(labels).length} 项`)
  const networks = parseListText(form.networks || '')
  if (networks.length) lines.push(`网络: ${networks.join(', ')}`)
  if (options.includeConstraints) {
    const constraints = parseListText(form.constraints || '')
    if (constraints.length) lines.push(`约束: ${constraints.join(', ')}`)
    const prefs = parseListText(form.placement_prefs || '')
    if (prefs.length) lines.push(`Placement: ${prefs.join(', ')}`)
  }
  const mounts = parseListText(form.mounts || '')
  if (mounts.length) lines.push(`挂载: ${mounts.length} 项`)
  const command = parseCommandText(form.command || '')
  if (command.length) lines.push(`命令: ${command.join(' ')}`)
  const update = []
  if (form.update_parallelism) update.push(`并行=${form.update_parallelism}`)
  if (form.update_delay) update.push(`延迟=${form.update_delay}`)
  if (form.update_failure_action) update.push(`失败策略=${form.update_failure_action}`)
  if (form.update_order) update.push(`顺序=${form.update_order}`)
  if (form.restart_condition) update.push(`重启=${form.restart_condition}`)
  if (update.length) lines.push(`更新策略: ${update.join(', ')}`)
  const rollback = []
  if (form.rollback_parallelism) rollback.push(`并行=${form.rollback_parallelism}`)
  if (form.rollback_delay) rollback.push(`延迟=${form.rollback_delay}`)
  if (form.rollback_failure_action) rollback.push(`失败策略=${form.rollback_failure_action}`)
  if (form.rollback_order) rollback.push(`顺序=${form.rollback_order}`)
  if (rollback.length) lines.push(`回滚策略: ${rollback.join(', ')}`)
  const limits = []
  if (form.limit_cpu) limits.push(`CPU=${form.limit_cpu}`)
  if (form.limit_memory) limits.push(`内存=${form.limit_memory}`)
  const reserves = []
  if (form.reserve_cpu) reserves.push(`CPU=${form.reserve_cpu}`)
  if (form.reserve_memory) reserves.push(`内存=${form.reserve_memory}`)
  if (limits.length) lines.push(`限制: ${limits.join(', ')}`)
  if (reserves.length) lines.push(`保留: ${reserves.join(', ')}`)
  return lines.join('\n')
}

const serviceCreatePreview = computed(() => buildServicePreview(serviceCreateForm, { includeConstraints: true }))
const serviceEditPreview = computed(() => buildServicePreview(serviceEditForm, { includeConstraints: true }))

const mapToText = (obj) => {
  return Object.entries(obj || {}).map(([k, v]) => `${k}=${v}`).join('\n')
}

const listToText = (list) => {
  return (list || []).filter(Boolean).join('\n')
}

const parseReplicaValue = (replicas) => {
  const rep = String(replicas || '')
  const match = rep.match(/(\d+)\s*\/\s*(\d+)/)
  if (match) return Number(match[2])
  const num = Number(rep)
  return Number.isNaN(num) ? 0 : num
}

const parseReplicaStats = (replicas) => {
  const rep = String(replicas || '')
  const match = rep.match(/(\d+)\s*\/\s*(\d+)/)
  if (match) {
    const running = Number(match[1])
    const desired = Number(match[2])
    const safeRunning = Number.isNaN(running) ? 0 : running
    const safeDesired = Number.isNaN(desired) ? 0 : desired
    return {
      running: safeRunning,
      desired: safeDesired,
      delta: safeRunning - safeDesired
    }
  }
  const desired = parseReplicaValue(rep)
  return { running: 0, desired, delta: 0 - desired }
}

const formatDuration = (value) => {
  if (value === undefined || value === null || value === '') return ''
  if (typeof value === 'string') return value
  if (typeof value === 'number') {
    const seconds = value / 1e9
    if (seconds >= 1) return `${Math.round(seconds)}s`
    return `${Math.round(value)}ns`
  }
  return String(value)
}

const calcDurationMs = (start, end) => {
  const s = new Date(start || '').getTime()
  const e = new Date(end || '').getTime()
  if (Number.isNaN(s) || Number.isNaN(e)) return null
  return Math.max(0, e - s)
}

const formatMs = (ms) => {
  if (ms === null || ms === undefined) return '-'
  if (ms < 1000) return `${Math.round(ms)}ms`
  const seconds = ms / 1000
  if (seconds < 60) return `${seconds.toFixed(2)}s`
  const min = Math.floor(seconds / 60)
  const sec = (seconds % 60).toFixed(1)
  return `${min}m${sec}s`
}

const nanoToCpu = (nano) => {
  if (!nano || typeof nano !== 'number') return ''
  return (nano / 1e9).toFixed(3).replace(/\.?0+$/, '')
}

const bytesToMem = (bytes) => {
  if (!bytes || typeof bytes !== 'number') return ''
  const mib = Math.round(bytes / 1024 / 1024)
  return `${mib}M`
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

const openGitDeploy = () => {
  gitDeployForm.name = ''
  gitDeployForm.repo = ''
  gitDeployForm.branch = 'main'
  gitDeployForm.compose_path = 'docker-compose.yml'
  gitDeployForm.authType = 'none'
  gitDeployForm.username = ''
  gitDeployForm.password = ''
  gitDeployForm.token = ''
  gitDeployOutput.value = ''
  gitDeployVisible.value = true
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

const submitGitDeploy = async () => {
  if (!activeHost.value || !gitDeployForm.name.trim() || !gitDeployForm.repo.trim()) {
    ElMessage.warning('请填写名称和仓库地址')
    return
  }
  gitDeployLoading.value = true
  try {
    const payload = {
      name: gitDeployForm.name.trim(),
      repo: gitDeployForm.repo.trim(),
      branch: gitDeployForm.branch.trim(),
      compose_path: gitDeployForm.compose_path.trim()
    }
    if (gitDeployForm.authType === 'token') {
      payload.token = gitDeployForm.token
    }
    if (gitDeployForm.authType === 'basic') {
      payload.username = gitDeployForm.username
      payload.password = gitDeployForm.password
    }
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/stacks/deploy/git`, payload, { headers: authHeaders() })
    if (res.data.code === 0) {
      gitDeployOutput.value = res.data.data?.output || '部署完成'
      loadStacks()
      loadServices()
    } else {
      gitDeployOutput.value = res.data.message || '部署失败'
    }
  } catch (e) {
    gitDeployOutput.value = extractErrorMessage(e)
  } finally {
    gitDeployLoading.value = false
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
  if (String(row.Mode || '').toLowerCase().includes('global')) {
    ElMessage.warning('Global 模式不支持设置副本')
    return
  }
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
  const globalServices = rows.filter(r => String(r.Mode || '').toLowerCase().includes('global'))
  if (globalServices.length > 0) {
    ElMessage.warning('包含 Global 模式服务，已跳过')
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
    if (String(row.Mode || '').toLowerCase().includes('global')) {
      results.push({ label: row.Name || row.name || row.ID, status: '跳过', message: 'Global 模式不支持' })
      continue
    }
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

const removeStack = async (row) => {
  if (!activeHost.value) return
  const stackName = row?.Name || row?.name
  if (!stackName) return
  try {
    await ElMessageBox.confirm(`确定删除 Stack ${stackName} 吗?`, '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch (e) {
    return
  }
  try {
    await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/stacks/${encodeURIComponent(stackName)}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    loadStacks()
    loadServices()
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  }
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
  createForm.labels = ''
  createForm.networks = ''
  createForm.mounts = ''
  createForm.entrypoint = ''
  createForm.command = ''
  createForm.privileged = false
  createForm.cap_add = ''
  createForm.network_mode = ''
  createForm.dns = ''
  createForm.extra_hosts = ''
  createForm.health_disable = false
  createForm.health_cmd = ''
  createForm.health_interval = ''
  createForm.health_timeout = ''
  createForm.health_retries = ''
  createForm.health_start_period = ''
  createForm.user = ''
  createForm.workdir = ''
  createForm.hostname = ''
  createForm.runtime = ''
  createForm.read_only = false
  createForm.cpus = ''
  createForm.memory = ''
  createForm.memory_reservation = ''
  createForm.pids_limit = ''
  createForm.ulimits = ''
  createForm.sysctls = ''
  createForm.security_opt = ''
  createForm.devices = ''
  createForm.tmpfs = ''
  createForm.log_driver = ''
  createForm.log_opts = ''
  createForm.restart_policy = ''
  createForm.auto_remove = false
  createVisible.value = true
}

const submitCreate = async () => {
  if (!activeHost.value || !createForm.image) {
    ElMessage.warning('请填写镜像')
    return
  }
  const dnsList = parseListText(createForm.dns)
  const invalidDns = dnsList.filter(item => !isValidIp(item))
  if (invalidDns.length > 0) {
    ElMessage.warning(`DNS 格式错误: ${invalidDns.join(', ')}`)
    return
  }
  const extraHosts = parseListText(createForm.extra_hosts)
  const invalidHosts = extraHosts.filter((item) => {
    const idx = item.indexOf(':')
    if (idx <= 0) return true
    const host = item.slice(0, idx).trim()
    const ip = item.slice(idx + 1).trim()
    return !isValidHostname(host) || !isValidIp(ip)
  })
  if (invalidHosts.length > 0) {
    ElMessage.warning(`Extra Hosts 格式错误: ${invalidHosts.join(', ')}`)
    return
  }
  const hasHealthConfig = !!createForm.health_cmd || !!createForm.health_interval || !!createForm.health_timeout ||
    !!createForm.health_retries || !!createForm.health_start_period
  if (!createForm.health_disable && hasHealthConfig && !createForm.health_cmd) {
    ElMessage.warning('配置健康检查时必须填写 Health Cmd')
    return
  }
  if (createForm.health_retries && Number.isNaN(Number(createForm.health_retries))) {
    ElMessage.warning('Health Retries 必须是数字')
    return
  }
  if (createForm.cpus && Number.isNaN(Number(createForm.cpus))) {
    ElMessage.warning('CPUs 必须是数字')
    return
  }
  if (createForm.pids_limit && Number.isNaN(Number(createForm.pids_limit))) {
    ElMessage.warning('Pids Limit 必须是数字')
    return
  }
  createLoading.value = true
  try {
    const ports = String(createForm.ports || '')
      .split(/[,\\n]/)
      .map(v => v.trim())
      .filter(Boolean)
    const payload = {
      name: createForm.name,
      image: createForm.image,
      ports,
      env: parseKeyValueText(createForm.env),
      labels: parseKeyValueText(createForm.labels),
      networks: parseListText(createForm.networks),
      mounts: parseListText(createForm.mounts),
      entrypoint: String(createForm.entrypoint || '').trim(),
      command: parseCommandText(createForm.command),
      privileged: !!createForm.privileged,
      cap_add: parseListText(createForm.cap_add),
      network_mode: String(createForm.network_mode || '').trim(),
      dns: dnsList,
      extra_hosts: extraHosts,
      health_disable: !!createForm.health_disable,
      health_cmd: String(createForm.health_cmd || '').trim(),
      health_interval: String(createForm.health_interval || '').trim(),
      health_timeout: String(createForm.health_timeout || '').trim(),
      health_retries: createForm.health_retries ? Number(createForm.health_retries) : undefined,
      health_start_period: String(createForm.health_start_period || '').trim(),
      user: String(createForm.user || '').trim(),
      workdir: String(createForm.workdir || '').trim(),
      hostname: String(createForm.hostname || '').trim(),
      runtime: String(createForm.runtime || '').trim(),
      read_only: !!createForm.read_only,
      cpus: String(createForm.cpus || '').trim(),
      memory: String(createForm.memory || '').trim(),
      memory_reservation: String(createForm.memory_reservation || '').trim(),
      pids_limit: createForm.pids_limit ? Number(createForm.pids_limit) : undefined,
      ulimits: parseListText(createForm.ulimits),
      sysctls: parseKeyValueText(createForm.sysctls),
      security_opt: parseListText(createForm.security_opt),
      devices: parseListText(createForm.devices),
      tmpfs: parseListText(createForm.tmpfs),
      log_driver: String(createForm.log_driver || '').trim(),
      log_opts: parseKeyValueText(createForm.log_opts),
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
    ElMessage.error(extractErrorMessage(e, '创建失败'))
  } finally {
    createLoading.value = false
  }
}

const containerAction = async (row, action) => {
  if (!activeHost.value) return
  const id = getContainerRef(row)
  if (!id) return
  try {
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(id)}/${action}`, {}, { headers: authHeaders() })
    ElMessage.success('操作成功')
    loadContainers()
  } catch (e) {
    ElMessage.error(extractErrorMessage(e, '操作失败'))
  }
}

const openLogs = (row) => {
  const id = getContainerRef(row)
  if (!id) return
  logContainerId.value = id
  logText.value = ''
  logVisible.value = true
  loadLogs()
}

const openInspect = async (row) => {
  const id = getContainerRef(row)
  if (!activeHost.value || !id) return
  inspectVisible.value = true
  inspectLoading.value = true
  inspectData.value = null
  inspectPorts.value = []
  inspectNetworks.value = []
  inspectMounts.value = []
  inspectEnvText.value = ''
  inspectLabels.value = []
  inspectCommand.value = ''
  inspectEntrypoint.value = ''
  inspectRestartPolicy.value = ''
  inspectNetworkMode.value = ''
  inspectPrivileged.value = false
  inspectCapAdd.value = []
  inspectDns.value = []
  inspectExtraHosts.value = []
  inspectHealth.value = null
  inspectResources.value = null
  inspectUser.value = ''
  inspectWorkdir.value = ''
  inspectHostname.value = ''
  inspectRuntime.value = ''
  inspectReadOnly.value = false
  inspectSecurityOpt.value = []
  inspectSysctls.value = []
  inspectUlimits.value = []
  inspectDevices.value = []
  inspectTmpfs.value = []
  inspectLogDriver.value = ''
  inspectLogOpts.value = []
  inspectExposedPorts.value = []
  inspectBinds.value = []
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(id)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      inspectData.value = res.data.data || null
      const config = inspectData.value?.Config || {}
      const hostConfig = inspectData.value?.HostConfig || {}
      const state = inspectData.value?.State || {}
      const ports = []
      const portMap = inspectData.value?.NetworkSettings?.Ports || {}
      Object.entries(portMap).forEach(([containerPort, hostBindings]) => {
        const parts = String(containerPort || '').split('/')
        const container = parts[0] || containerPort
        const proto = parts[1] || ''
        if (!hostBindings || hostBindings.length === 0) {
          ports.push({ container, proto, host: '-', ip: '-' })
          return
        }
        hostBindings.forEach((b) => {
          ports.push({ container, proto, host: b.HostPort || '-', ip: b.HostIp || '-' })
        })
      })
      inspectPorts.value = ports

      const networks = []
      const nets = inspectData.value?.NetworkSettings?.Networks || {}
      Object.entries(nets).forEach(([name, info]) => {
        networks.push({
          name,
          ip: info.IPAddress || '-',
          gateway: info.Gateway || '-',
          mac: info.MacAddress || '-',
          aliases: (info.Aliases || []).join(', ')
        })
      })
      inspectNetworks.value = networks

      const mounts = (inspectData.value?.Mounts || []).map(m => ({
        type: m.Type,
        name: m.Name || '-',
        source: m.Source,
        destination: m.Destination,
        mode: m.Mode,
        rw: m.RW ? 'true' : 'false',
        propagation: m.Propagation || '-'
      }))
      inspectMounts.value = mounts

      const env = inspectData.value?.Config?.Env || []
      inspectEnvText.value = env.join('\n')

      const labels = config.Labels || {}
      inspectLabels.value = Object.entries(labels).map(([key, value]) => ({ key, value }))
      inspectCommand.value = Array.isArray(config.Cmd) ? config.Cmd.join(' ') : (config.Cmd || '')
      inspectEntrypoint.value = Array.isArray(config.Entrypoint) ? config.Entrypoint.join(' ') : (config.Entrypoint || '')
      inspectRestartPolicy.value = hostConfig.RestartPolicy?.Name || '-'
      inspectNetworkMode.value = hostConfig.NetworkMode || '-'
      inspectPrivileged.value = !!hostConfig.Privileged
      inspectCapAdd.value = hostConfig.CapAdd || []
      inspectDns.value = hostConfig.Dns || []
      inspectExtraHosts.value = hostConfig.ExtraHosts || []
      inspectHealth.value = state.Health || null
      inspectResources.value = {
        cpu: hostConfig.NanoCpus ? nanoToCpu(hostConfig.NanoCpus) : '',
        cpuShares: hostConfig.CpuShares ?? '',
        cpuQuota: hostConfig.CpuQuota ?? '',
        cpuPeriod: hostConfig.CpuPeriod ?? '',
        cpuset: hostConfig.CpusetCpus || '',
        memory: hostConfig.Memory ? bytesToMem(hostConfig.Memory) : '',
        memoryReservation: hostConfig.MemoryReservation ? bytesToMem(hostConfig.MemoryReservation) : '',
        memorySwap: hostConfig.MemorySwap ? bytesToMem(hostConfig.MemorySwap) : '',
        pidsLimit: hostConfig.PidsLimit ?? '',
        oomKillDisable: !!hostConfig.OomKillDisable,
        blkioWeight: hostConfig.BlkioWeight ?? '',
        logDriver: hostConfig.LogConfig?.Type || ''
      }
      inspectUser.value = config.User || ''
      inspectWorkdir.value = config.WorkingDir || ''
      inspectHostname.value = config.Hostname || inspectData.value?.Config?.Hostname || ''
      inspectRuntime.value = hostConfig.Runtime || ''
      inspectReadOnly.value = !!hostConfig.ReadonlyRootfs
      inspectSecurityOpt.value = hostConfig.SecurityOpt || []
      inspectSysctls.value = Object.entries(hostConfig.Sysctls || {}).map(([k, v]) => `${k}=${v}`)
      inspectUlimits.value = (hostConfig.Ulimits || []).map(u => `${u.Name}=${u.Soft}:${u.Hard}`)
      inspectDevices.value = (hostConfig.Devices || []).map(d => `${d.PathOnHost}:${d.PathInContainer}:${d.CgroupPermissions}`)
      inspectTmpfs.value = Object.entries(hostConfig.Tmpfs || {}).map(([k, v]) => `${k}:${v}`)
      inspectLogDriver.value = hostConfig.LogConfig?.Type || ''
      inspectLogOpts.value = Object.entries(hostConfig.LogConfig?.Config || {}).map(([k, v]) => `${k}=${v}`)
      inspectExposedPorts.value = Object.keys(config.ExposedPorts || {})
      inspectBinds.value = hostConfig.Binds || []
    }
  } finally {
    inspectLoading.value = false
  }
}

const openExec = (row) => {
  if (!isContainerRunning(row)) {
    ElMessage.warning('容器未运行，无法执行命令')
    return
  }
  const id = getContainerRef(row)
  if (!id) return
  execContainerId.value = id
  execCommand.value = 'ls /'
  execOutput.value = ''
  execVisible.value = true
}

const openTerminal = async (row) => {
  if (!isContainerRunning(row)) {
    ElMessage.warning('容器未运行，无法打开终端')
    return
  }
  const id = getContainerRef(row)
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

const handleWindowResize = () => {
  sendTerminalResize()
  if (statsChartInstance.value) {
    statsChartInstance.value.resize()
  }
}

const connectTerminal = async () => {
  if (!terminalContainerId.value || !activeHost.value) return
  if (terminalConnected.value || terminalConnecting.value) return
  const token = localStorage.getItem('token') || ''
  if (!token) {
    ElMessage.error('登录状态失效，请重新登录')
    return
  }
  if (terminalWs) {
    terminalWs.close()
    terminalWs = null
  }
  terminalConnecting.value = true
  const basePath = `/api/v1/docker/hosts/${activeHost.value.id}/containers/${encodeURIComponent(terminalContainerId.value)}/exec/ws`
  try {
    await axios.get(basePath, {
      headers: authHeaders(),
      params: {
        shell: terminalShell.value,
        dry_run: 1
      }
    })
  } catch (e) {
    terminalConnecting.value = false
    ElMessage.error(extractErrorMessage(e, '终端预检查失败'))
    return
  }

  const wsProto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const wsUrl = `${wsProto}://${window.location.host}${basePath}?token=${encodeURIComponent(token)}&shell=${encodeURIComponent(terminalShell.value)}`
  terminalWs = new WebSocket(wsUrl)
  terminalWs.binaryType = 'arraybuffer'
  terminalWs.onopen = () => {
    terminalConnecting.value = false
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
  terminalWs.onclose = (evt) => {
    terminalConnecting.value = false
    terminalConnected.value = false
    const reason = evt?.reason ? ` (${evt.reason})` : ''
    terminal?.writeln(`\r\n连接已关闭 [${evt?.code ?? '-'}]${reason}。`)
    terminalWs = null
  }
  terminalWs.onerror = (evt) => {
    terminalConnecting.value = false
    console.error('[Docker Terminal] websocket error', evt)
    ElMessage.error('WebSocket连接失败，请检查 Docker 节点网络/权限')
  }
}

const closeTerminal = () => {
  terminalConnecting.value = false
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
    ElMessage.error(extractErrorMessage(e, '导出失败'))
  }
}

const readFileAsBase64 = (file) => new Promise((resolve, reject) => {
  const reader = new FileReader()
  reader.onload = () => {
    const result = String(reader.result || '')
    const base64 = result.includes(',') ? result.split(',')[1] : result
    resolve(base64)
  }
  reader.onerror = () => reject(new Error('读取失败'))
  reader.readAsDataURL(file)
})

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
    execOutput.value = extractErrorMessage(e)
    ElMessage.error(execOutput.value)
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
  } catch (e) {
    const msg = extractErrorMessage(e)
    logText.value = msg
    ElMessage.error(msg)
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
  serviceSummary.value = null
  serviceInspectData.value = null
  try {
    const [inspectRes, taskRes] = await Promise.allSettled([
      axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}`, { headers: authHeaders() }),
      axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/tasks`, { headers: authHeaders() })
    ])
    if (inspectRes.status === 'fulfilled' && inspectRes.value.data.code === 0) {
      const data = inspectRes.value.data.data || {}
      serviceJson.value = JSON.stringify(data, null, 2)
      serviceInspectData.value = data
      const updateInfo = formatStatusInfo(data.UpdateStatus)
      const rollbackInfo = formatStatusInfo(data.RollbackStatus)
      const tasks = taskRes.status === 'fulfilled' && taskRes.value.data.code === 0 ? (taskRes.value.data.data || []) : []
      const taskSummary = summarizeServiceTasks(tasks)
      serviceSummary.value = {
        update: updateInfo,
        rollback: rollbackInfo,
        tasks: taskSummary
      }
    }
  } finally {
    serviceLoading.value = false
  }
}

const loadServiceTasks = async () => {
  if (!activeHost.value || !serviceTasksTargetId.value) return
  tasksLoading.value = true
  serviceTasks.value = []
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(serviceTasksTargetId.value)}/tasks`, { headers: authHeaders() })
    if (res.data.code === 0) {
      serviceTasks.value = res.data.data || []
    }
  } finally {
    tasksLoading.value = false
  }
}

const reloadServiceTasks = async () => {
  await loadServiceTasks()
}

const openServiceTasks = async (row, onlyErrors = false) => {
  if (!activeHost.value || !row?.ID) return
  serviceTaskFilterError.value = !!onlyErrors
  serviceTasksTargetId.value = row.ID
  tasksVisible.value = true
  await loadServiceTasks()
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

const openCreateService = () => {
  serviceCreateForm.name = ''
  serviceCreateForm.image = ''
  serviceCreateForm.mode = 'replicated'
  serviceCreateForm.endpoint_mode = 'vip'
  serviceCreateForm.replicas = 1
  serviceCreateForm.ports = ''
  serviceCreateForm.env = ''
  serviceCreateForm.labels = ''
  serviceCreateForm.networks = ''
  serviceCreateForm.constraints = ''
  serviceCreateForm.placement_prefs = ''
  serviceCreateForm.max_replicas_per_node = ''
  serviceCreateForm.mounts = ''
  serviceCreateForm.command = ''
  serviceCreateForm.restart_condition = ''
  serviceCreateForm.update_parallelism = ''
  serviceCreateForm.update_delay = ''
  serviceCreateForm.update_failure_action = ''
  serviceCreateForm.update_order = ''
  serviceCreateForm.rollback_parallelism = ''
  serviceCreateForm.rollback_delay = ''
  serviceCreateForm.rollback_failure_action = ''
  serviceCreateForm.rollback_order = ''
  serviceCreateForm.limit_cpu = ''
  serviceCreateForm.limit_memory = ''
  serviceCreateForm.reserve_cpu = ''
  serviceCreateForm.reserve_memory = ''
  createServiceVisible.value = true
}

const submitCreateService = async () => {
  if (!activeHost.value) return
  if (!serviceCreateForm.name || !serviceCreateForm.image) {
    ElMessage.warning('请填写名称和镜像')
    return
  }
  createServiceLoading.value = true
  try {
    const isGlobal = String(serviceCreateForm.mode || '').toLowerCase() === 'global'
    const payload = {
      name: serviceCreateForm.name,
      image: serviceCreateForm.image,
      mode: serviceCreateForm.mode,
      endpoint_mode: serviceCreateForm.endpoint_mode,
      replicas: isGlobal ? undefined : Number(serviceCreateForm.replicas || 0),
      ports: parseListText(serviceCreateForm.ports),
      env: parseKeyValueText(serviceCreateForm.env),
      labels: parseKeyValueText(serviceCreateForm.labels),
      networks: parseListText(serviceCreateForm.networks),
      constraints: parseListText(serviceCreateForm.constraints),
      placement_prefs: parseListText(serviceCreateForm.placement_prefs),
      max_replicas_per_node: !isGlobal && serviceCreateForm.max_replicas_per_node ? Number(serviceCreateForm.max_replicas_per_node) : undefined,
      mounts: parseListText(serviceCreateForm.mounts),
      command: parseCommandText(serviceCreateForm.command),
      restart_condition: serviceCreateForm.restart_condition,
      update_parallelism: serviceCreateForm.update_parallelism ? Number(serviceCreateForm.update_parallelism) : undefined,
      update_delay: serviceCreateForm.update_delay,
      update_failure_action: serviceCreateForm.update_failure_action,
      update_order: serviceCreateForm.update_order,
      rollback_parallelism: serviceCreateForm.rollback_parallelism ? Number(serviceCreateForm.rollback_parallelism) : undefined,
      rollback_delay: serviceCreateForm.rollback_delay,
      rollback_failure_action: serviceCreateForm.rollback_failure_action,
      rollback_order: serviceCreateForm.rollback_order,
      limit_cpu: serviceCreateForm.limit_cpu,
      limit_memory: serviceCreateForm.limit_memory,
      reserve_cpu: serviceCreateForm.reserve_cpu,
      reserve_memory: serviceCreateForm.reserve_memory
    }
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services`, payload, { headers: authHeaders() })
    ElMessage.success('创建成功')
    createServiceVisible.value = false
    loadServices()
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  } finally {
    createServiceLoading.value = false
  }
}

const openEditService = async (row) => {
  if (!activeHost.value) return
  const id = row?.ID || row?.Id || row?.id
  if (!id) return
  editServiceVisible.value = true
  editServiceLoading.value = true
  serviceEditForm.id = id
  serviceEditForm.name = row?.Name || ''
  serviceEditForm.image = row?.Image || ''
  serviceEditForm.mode = ''
  serviceEditForm.endpoint_mode = ''
  serviceEditForm.replicas = parseReplicaValue(row?.Replicas)
  serviceEditForm.ports = ''
  serviceEditForm.env = ''
  serviceEditForm.labels = ''
  serviceEditForm.networks = ''
  serviceEditForm.constraints = ''
  serviceEditForm.placement_prefs = ''
  serviceEditForm.mounts = ''
  serviceEditForm.command = ''
  serviceEditForm.max_replicas_per_node = ''
  serviceEditForm.reset_env = true
  serviceEditForm.reset_labels = true
  serviceEditForm.reset_ports = true
  serviceEditForm.reset_networks = true
  serviceEditForm.reset_constraints = true
  serviceEditForm.reset_placement_prefs = true
  serviceEditForm.reset_mounts = true
  serviceEditForm.reset_command = false
  try {
    const res = await axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(id)}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      const data = res.data.data || {}
      const spec = data.Spec || {}
      const containerSpec = spec.TaskTemplate?.ContainerSpec || {}
      const envList = containerSpec.Env || []
      const mounts = (containerSpec.Mounts || []).map(formatServiceMount).filter(Boolean)
      const args = containerSpec.Args || containerSpec.Command || []
      const labels = spec.Labels || {}
      const networks = (spec.TaskTemplate?.Networks || []).map(n => n?.Target || n?.Name).filter(Boolean)
      const constraints = spec.TaskTemplate?.Placement?.Constraints || []
      const placementPrefs = (spec.TaskTemplate?.Placement?.Preferences || []).map(p => {
        const spread = p?.Spread?.SpreadDescriptor
        return spread ? `spread=${spread}` : ''
      }).filter(Boolean)
      const mode = spec.Mode?.Global ? 'global' : 'replicated'
      const endpointMode = spec.EndpointSpec?.Mode || ''
      const restartCondition = spec.TaskTemplate?.RestartPolicy?.Condition || ''
      const updateConfig = spec.UpdateConfig || {}
      const rollbackConfig = spec.RollbackConfig || {}
      const resources = spec.TaskTemplate?.Resources || {}
      const limits = resources.Limits || {}
      const reserves = resources.Reservations || {}
      const ports = (spec.EndpointSpec?.Ports || []).map((p) => {
        if (p?.PublishedPort) {
          const parts = [`published=${p.PublishedPort}`, `target=${p.TargetPort}`]
          if (p.Protocol) parts.push(`protocol=${p.Protocol}`)
          if (p.PublishMode) parts.push(`mode=${p.PublishMode}`)
          return parts.join(',')
        }
        return p?.TargetPort ? String(p.TargetPort) : ''
      }).filter(Boolean)
      serviceEditForm.image = containerSpec.Image || serviceEditForm.image
      serviceEditForm.replicas = Number(spec.Mode?.Replicated?.Replicas ?? serviceEditForm.replicas)
      const maxReplicas = spec.Mode?.Replicated?.MaxReplicasPerNode ?? spec.Mode?.Replicated?.MaxReplicas ?? ''
      serviceEditForm.max_replicas_per_node = maxReplicas ? String(maxReplicas) : ''
      serviceEditForm.mode = mode
      serviceEditForm.endpoint_mode = endpointMode
      serviceEditForm.env = listToText(envList)
      serviceEditForm.labels = mapToText(labels)
      serviceEditForm.networks = listToText(networks)
      serviceEditForm.constraints = listToText(constraints)
      serviceEditForm.placement_prefs = listToText(placementPrefs)
      serviceEditForm.mounts = listToText(mounts)
      serviceEditForm.command = listToCommandText(args)
      serviceEditForm.ports = listToText(ports)
      serviceEditForm.restart_condition = restartCondition
      serviceEditForm.update_parallelism = updateConfig.Parallelism ? String(updateConfig.Parallelism) : ''
      serviceEditForm.update_delay = formatDuration(updateConfig.Delay)
      serviceEditForm.update_failure_action = updateConfig.FailureAction || ''
      serviceEditForm.update_order = updateConfig.Order || ''
      serviceEditForm.rollback_parallelism = rollbackConfig.Parallelism ? String(rollbackConfig.Parallelism) : ''
      serviceEditForm.rollback_delay = formatDuration(rollbackConfig.Delay)
      serviceEditForm.rollback_failure_action = rollbackConfig.FailureAction || ''
      serviceEditForm.rollback_order = rollbackConfig.Order || ''
      serviceEditForm.limit_cpu = nanoToCpu(limits.NanoCPUs)
      serviceEditForm.limit_memory = bytesToMem(limits.MemoryBytes)
      serviceEditForm.reserve_cpu = nanoToCpu(reserves.NanoCPUs)
      serviceEditForm.reserve_memory = bytesToMem(reserves.MemoryBytes)
    }
  } finally {
    editServiceLoading.value = false
  }
}

const submitEditService = async () => {
  if (!activeHost.value || !serviceEditForm.id) return
  editServiceSaving.value = true
  try {
    const isGlobal = String(serviceEditForm.mode || '').toLowerCase() === 'global'
    const payload = {
      image: serviceEditForm.image,
      mode: serviceEditForm.mode,
      endpoint_mode: serviceEditForm.endpoint_mode,
      replicas: isGlobal ? undefined : Number(serviceEditForm.replicas || 0),
      ports: parseListText(serviceEditForm.ports),
      env: parseKeyValueText(serviceEditForm.env),
      labels: parseKeyValueText(serviceEditForm.labels),
      networks: parseListText(serviceEditForm.networks),
      constraints: serviceEditForm.reset_constraints ? parseListText(serviceEditForm.constraints) : [],
      placement_prefs: serviceEditForm.reset_placement_prefs ? parseListText(serviceEditForm.placement_prefs) : [],
      mounts: serviceEditForm.reset_mounts ? parseListText(serviceEditForm.mounts) : [],
      command: serviceEditForm.reset_command ? parseCommandText(serviceEditForm.command) : [],
      reset_env: serviceEditForm.reset_env,
      reset_labels: serviceEditForm.reset_labels,
      reset_ports: serviceEditForm.reset_ports,
      reset_networks: serviceEditForm.reset_networks,
      reset_constraints: serviceEditForm.reset_constraints,
      reset_placement_prefs: serviceEditForm.reset_placement_prefs,
      reset_mounts: serviceEditForm.reset_mounts,
      reset_command: serviceEditForm.reset_command,
      restart_condition: serviceEditForm.restart_condition,
      update_parallelism: serviceEditForm.update_parallelism ? Number(serviceEditForm.update_parallelism) : undefined,
      update_delay: serviceEditForm.update_delay,
      update_failure_action: serviceEditForm.update_failure_action,
      update_order: serviceEditForm.update_order,
      rollback_parallelism: serviceEditForm.rollback_parallelism ? Number(serviceEditForm.rollback_parallelism) : undefined,
      rollback_delay: serviceEditForm.rollback_delay,
      rollback_failure_action: serviceEditForm.rollback_failure_action,
      rollback_order: serviceEditForm.rollback_order,
      limit_cpu: serviceEditForm.limit_cpu,
      limit_memory: serviceEditForm.limit_memory,
      reserve_cpu: serviceEditForm.reserve_cpu,
      reserve_memory: serviceEditForm.reserve_memory,
      max_replicas_per_node: !isGlobal && serviceEditForm.max_replicas_per_node ? Number(serviceEditForm.max_replicas_per_node) : undefined
    }
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(serviceEditForm.id)}/update`, payload, { headers: authHeaders() })
    ElMessage.success('更新成功')
    editServiceVisible.value = false
    loadServices()
  } catch (e) {
    ElMessage.error(extractErrorMessage(e))
  } finally {
    editServiceSaving.value = false
  }
}

const openStackServices = async (row) => {
  if (!activeHost.value || !row?.Name) return
  stackVisible.value = true
  stackLoading.value = true
  stackServices.value = []
  stackSummary.value = null
  stackMeta.value = null
  try {
    const [stackRes, servicesRes] = await Promise.allSettled([
      axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/stacks/${encodeURIComponent(row.Name)}/services`, { headers: authHeaders() }),
      axios.get(`/api/v1/docker/hosts/${activeHost.value.id}/services`, { params: { detail: 1 }, headers: authHeaders() })
    ])
    if (stackRes.status === 'fulfilled' && stackRes.value.data.code === 0) {
      stackServices.value = stackRes.value.data.data || []
      const summary = { total: stackServices.value.length, desired: 0, running: 0, warning: 0 }
      stackServices.value.forEach((s) => {
        const rep = String(s.Replicas || '')
        const match = rep.match(/(\d+)\s*\/\s*(\d+)/)
        if (match) {
          const running = Number(match[1])
          const desired = Number(match[2])
          summary.running += Number.isNaN(running) ? 0 : running
          summary.desired += Number.isNaN(desired) ? 0 : desired
          if (running < desired) summary.warning += 1
        }
      })
      stackSummary.value = summary
    }
    if (servicesRes.status === 'fulfilled' && servicesRes.value.data.code === 0) {
      const list = servicesRes.value.data.data || []
      const stackName = row.Name
      const detailServices = list.filter(s => (s.Name || '').startsWith(`${stackName}_`))
      const networks = new Set()
      const labels = new Set()
      const images = new Set()
      detailServices.forEach((svc) => {
        const spec = svc.Spec || {}
        const task = spec.TaskTemplate || {}
        const container = task.ContainerSpec || {}
        const nets = task.Networks || []
        nets.forEach((n) => {
          const target = n?.Target || n?.Name
          if (target) networks.add(target)
        })
        const lbls = spec.Labels || {}
        Object.entries(lbls).forEach(([k, v]) => labels.add(`${k}=${v}`))
        if (container.Image) images.add(container.Image)
      })
      stackMeta.value = {
        networks: Array.from(networks),
        labels: Array.from(labels),
        images: Array.from(images)
      }
    }
  } finally {
    stackLoading.value = false
  }
}

const scaleService = async (row) => {
  if (!activeHost.value || !row?.ID) return
  if (String(row.Mode || '').toLowerCase().includes('global')) {
    ElMessage.warning('Global 模式不支持设置副本')
    return
  }
  try {
    const { value } = await ElMessageBox.prompt('输入副本数', '扩缩容', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputValue: String(parseReplicaValue(row?.Replicas)),
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
      cancelButtonText: '取消',
      inputValue: row?.Image || ''
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

const rollbackService = async (row) => {
  if (!activeHost.value || !row?.ID) return
  try {
    await ElMessageBox.confirm('确认回滚该服务到上一个版本吗？', '提示', { type: 'warning' })
    await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}/rollback`, {}, { headers: authHeaders() })
    ElMessage.success('已触发回滚')
    loadServices()
  } catch (e) {}
}

const removeService = async (row) => {
  if (!activeHost.value || !row?.ID) return
  try {
    await ElMessageBox.confirm(`确认删除服务 ${row.Name || row.ID} 吗？`, '提示', { type: 'warning' })
    await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(row.ID)}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    loadServices()
  } catch (e) {}
}

const removeSelectedServices = async () => {
  if (!activeHost.value) return
  const rows = selectedServices.value.filter(r => r.ID || r.Id || r.id)
  if (rows.length === 0) {
    ElMessage.warning('请选择服务')
    return
  }
  try {
    await ElMessageBox.confirm(`确认删除选中的 ${rows.length} 个服务吗？`, '提示', { type: 'warning' })
    let ok = 0
    for (const row of rows) {
      const id = row.ID || row.Id || row.id
      if (!id) continue
      try {
        await axios.delete(`/api/v1/docker/hosts/${activeHost.value.id}/services/${encodeURIComponent(id)}`, { headers: authHeaders() })
        ok += 1
      } catch (e) {}
    }
    ElMessage.success(`已删除 ${ok} 个服务`)
    loadServices()
  } catch (e) {}
}

const openPullImage = () => {
  pullImage.value = ''
  pullOutput.value = ''
  pullVisible.value = true
}

const openBuildImage = () => {
  buildForm.tag = ''
  buildForm.dockerfile = ''
  buildForm.contextTar = ''
  buildArgsText.value = ''
  buildOutput.value = ''
  buildVisible.value = true
}

const openLoadImage = () => {
  loadForm.tar = ''
  loadOutput.value = ''
  loadVisible.value = true
}

const handleBuildContextChange = async (file) => {
  if (!file?.raw) return
  try {
    buildForm.contextTar = await readFileAsBase64(file.raw)
  } catch (e) {
    ElMessage.error(extractErrorMessage(e, '读取文件失败'))
  }
}

const handleLoadTarChange = async (file) => {
  if (!file?.raw) return
  try {
    loadForm.tar = await readFileAsBase64(file.raw)
  } catch (e) {
    ElMessage.error(extractErrorMessage(e, '读取文件失败'))
  }
}

const submitBuildImage = async () => {
  if (!activeHost.value || !buildForm.tag || !buildForm.dockerfile) {
    ElMessage.warning('请填写镜像标签和 Dockerfile')
    return
  }
  buildLoading.value = true
  try {
    const buildArgs = {}
    buildArgsText.value.split('\n').map(v => v.trim()).filter(Boolean).forEach((line) => {
      const idx = line.indexOf('=')
      if (idx > 0) {
        const k = line.slice(0, idx).trim()
        const v = line.slice(idx + 1).trim()
        if (k) buildArgs[k] = v
      }
    })
    const payload = {
      tag: buildForm.tag,
      dockerfile: buildForm.dockerfile,
      context_tar: buildForm.contextTar,
      build_args: buildArgs
    }
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/images/build`, payload, { headers: authHeaders() })
    if (res.data.code === 0) {
      buildOutput.value = res.data.data?.output || '构建完成'
      loadImages()
      refreshManage()
    } else {
      buildOutput.value = res.data.message || '构建失败'
    }
  } catch (e) {
    buildOutput.value = extractErrorMessage(e)
  } finally {
    buildLoading.value = false
  }
}

const submitLoadImage = async () => {
  if (!activeHost.value || !loadForm.tar) {
    ElMessage.warning('请选择镜像 tar 包')
    return
  }
  loadLoading.value = true
  try {
    const res = await axios.post(`/api/v1/docker/hosts/${activeHost.value.id}/images/load`, { tar: loadForm.tar }, { headers: authHeaders() })
    if (res.data.code === 0) {
      loadOutput.value = res.data.data?.output || '导入完成'
      loadImages()
      refreshManage()
    } else {
      loadOutput.value = res.data.message || '导入失败'
    }
  } catch (e) {
    loadOutput.value = extractErrorMessage(e)
  } finally {
    loadLoading.value = false
  }
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
    ElMessage.error(extractErrorMessage(e, '拉取失败'))
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
  if (tab === 'events') loadEvents()
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
  window.addEventListener('resize', handleWindowResize)
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
  if (statsChartInstance.value) {
    statsChartInstance.value.dispose()
    statsChartInstance.value = null
  }
  window.removeEventListener('resize', handleWindowResize)
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
.docker-page-card {
  max-width: 100%;
  margin: 0 auto;
  border-radius: 16px;
  border: 1px solid #e8edf4;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
}
.header-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  flex-wrap: wrap;
}
.header-left {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.header-title {
  font-size: 18px;
  letter-spacing: -0.2px;
}
.header-stats {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.header-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.w-100 { width: 100%; }
.text-xs { font-size: 12px; line-height: 1.3; }
.text-blue-500 { color: #409eff; }
.text-xl { font-size: 18px; }
.drawer-header {
  position: sticky;
  top: 0;
  z-index: 5;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: -4px -4px 12px;
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid #e8edf4;
  backdrop-filter: blur(8px);
}
.drawer-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.drawer-title { font-size: 18px; font-weight: 600; letter-spacing: -0.2px; }
.drawer-sub { color: #606266; margin-top: 6px; display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
.drawer-meta { color: #909399; }
.w-40 { width: 140px; }
.w-48 { width: 180px; }
.w-28 { width: 110px; }
.manage-tabs { margin-top: 8px; }
.manage-tabs :deep(.el-tabs__header) { margin-bottom: 12px; }
.manage-tabs :deep(.el-tabs__nav-wrap::after) { display: none; }
.manage-tabs :deep(.el-tabs__content) {
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.65);
  border: 1px solid rgba(15, 23, 42, 0.07);
  padding: 12px;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8);
}
.manage-tabs :deep(.el-tab-pane) {
  animation: docker-pane-enter 0.22s cubic-bezier(0.22, 1, 0.36, 1) both;
}
.manage-tabs :deep(.el-tabs__nav) {
  padding: 4px;
  border-radius: 999px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.68);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.8), 0 6px 18px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(12px);
}
.manage-tabs :deep(.el-tabs__active-bar) { display: none; }
.manage-tabs :deep(.el-tabs__item) {
  height: 34px;
  margin: 0 2px;
  padding: 0 14px;
  border-radius: 999px;
  color: #5b6471;
  font-weight: 500;
  transition: all 0.22s ease;
}
.manage-tabs :deep(.el-tabs__item:hover) {
  color: #1f2937;
  background: rgba(255, 255, 255, 0.66);
}
.manage-tabs :deep(.el-tabs__item.is-active) {
  color: #0b1324;
  font-weight: 600;
  background: linear-gradient(180deg, #ffffff 0%, #f2f5fa 100%);
  box-shadow: 0 6px 14px rgba(15, 23, 42, 0.12);
}
.tab-toolbar { display: flex; justify-content: space-between; gap: 8px; margin-bottom: 10px; flex-wrap: wrap; }
.toolbar-left { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.toolbar-right { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.tab-toolbar :deep(.el-button) {
  border-radius: 999px;
  border-color: rgba(148, 163, 184, 0.32);
  background: rgba(255, 255, 255, 0.78);
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.06);
}
.tab-toolbar :deep(.el-button:hover) {
  border-color: rgba(59, 130, 246, 0.5);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.18);
}
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
:deep(.docker-manage-drawer .el-drawer__body) {
  padding: 16px;
  background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
}
:deep(.el-table th.el-table__cell) {
  background: #f8fafc;
}

@keyframes docker-pane-enter {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
