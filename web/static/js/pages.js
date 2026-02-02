// ==================== CMDB 主机管理 ====================
async function loadCMDB() {
    try {
        const response = await apiRequest('/cmdb/hosts');
        
        const html = `
            <div class="page-header">
                <h1 class="page-title">
                    <i class="fas fa-server"></i>
                    CMDB 主机管理
                </h1>
                <p class="page-description">管理和监控所有主机资源</p>
            </div>

            <div class="toolbar">
                <div class="toolbar-left">
                    <button class="btn btn-primary" onclick="showAddHostModal()">
                        <i class="fas fa-plus"></i> 添加主机
                    </button>
                    <button class="btn btn-secondary" onclick="refreshHosts()">
                        <i class="fas fa-sync-alt"></i> 刷新
                    </button>
                </div>
                <div class="toolbar-right">
                    <input type="text" class="form-control" placeholder="搜索主机..." style="width: 250px;">
                </div>
            </div>

            <div class="card">
                <div class="card-body">
                    <div class="table-container">
                        <table class="table">
                            <thead>
                                <tr>
                                    <th>主机名</th>
                                    <th>IP地址</th>
                                    <th>操作系统</th>
                                    <th>状态</th>
                                    <th>分组</th>
                                    <th>操作</th>
                                </tr>
                            </thead>
                            <tbody id="hostsTableBody">
                                    ${response.code === 0 && response.data && response.data.length > 0 ? 
                                    response.data.map(host => {
                                        let osIcon = 'linux';
                                        const osName = (host.os || '').toLowerCase();
                                        if (osName.includes('ubuntu')) osIcon = 'ubuntu';
                                        else if (osName.includes('centos') || osName.includes('redhat')) osIcon = 'redhat';
                                        else if (osName.includes('debian')) osIcon = 'debian';
                                        else if (osName.includes('fedora')) osIcon = 'fedora';
                                        else if (osName.includes('windows')) osIcon = 'windows';
                                        else if (osName.includes('apple') || osName.includes('macos')) osIcon = 'apple';
                                        
                                        return `
                                        <tr>
                                            <td>
                                                <div style="display: flex; align-items: center; gap: 10px;">
                                                    <i class="fab fa-${osIcon}" style="font-size: 20px; color: var(--text-secondary);"></i>
                                                    <strong>${host.name || '-'}</strong>
                                                </div>
                                            </td>
                                            <td>${host.ip || '-'}</td>
                                            <td>${host.os || 'Unknown'}</td>
                                            <td><span class="badge badge-success">在线</span></td>
                                            <td>${host.group || '默认'}</td>
                                            <td>
                                                <button class="btn btn-secondary btn-sm" onclick="editHost('${host.id}')">
                                                    <i class="fas fa-edit"></i>
                                                </button>
                                                <button class="btn btn-danger btn-sm" onclick="deleteHost('${host.id}')">
                                                    <i class="fas fa-trash"></i>
                                                </button>
                                            </td>
                                        </tr>
                                    `}).join('') :
                                    '<tr><td colspan="6" class="text-center">暂无主机数据</td></tr>'
                                }
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('cmdbPage').innerHTML = html;
    } catch (error) {
        console.error('加载CMDB失败:', error);
    }
}

function showAddHostModal() {
    const body = `
        <form id="addHostForm">
            <div class="form-group">
                <label class="form-label">主机名</label>
                <input type="text" class="form-control" name="name" required>
            </div>
            <div class="form-group">
                <label class="form-label">IP地址</label>
                <input type="text" class="form-control" name="ip" required>
            </div>
            <div class="form-group">
                <label class="form-label">SSH端口</label>
                <input type="number" class="form-control" name="port" value="22">
            </div>
            <div class="form-group">
                <label class="form-label">用户名</label>
                <input type="text" class="form-control" name="username" value="root">
            </div>
            <div class="form-group">
                <label class="form-label">密码</label>
                <div class="input-wrapper" style="position: relative;">
                    <input type="password" class="form-control" name="password" id="addHostPassword">
                    <i class="fas fa-eye" onclick="togglePasswordVisibility('addHostPassword', this)" style="position: absolute; right: 10px; top: 10px; cursor: pointer; color: var(--text-disabled);"></i>
                </div>
            </div>
            <div class="form-group">
                <label class="form-label">分组</label>
                <input type="text" class="form-control" name="group" placeholder="默认">
            </div>
        </form>
    `;

    showModal('添加主机', body, async () => {
        const form = document.getElementById('addHostForm');
        const formData = new FormData(form);
        const data = Object.fromEntries(formData);
        
        // 数据类型转换
        data.port = parseInt(data.port, 10);
        
        // 重命名 group -> group_name 以避免后端类型不匹配 (Group struct vs string)
        // 无论是否为空，都要处理，否则后端尝试将空字符串解析为 HostGroup 结构体会报错
        data.group_name = data.group || "";
        delete data.group;

        try {
            showLoading();
            const response = await apiRequest('/cmdb/hosts', {
                method: 'POST',
                body: JSON.stringify(data)
            });

            if (response.code === 0) {
                await loadCMDB();
            }
        } catch (error) {
            console.error('添加主机失败:', error);
        } finally {
            hideLoading();
        }
    });
}

async function editHost(id) {
    try {
        showLoading();
        const response = await apiRequest(`/cmdb/hosts/${id}`);
        hideLoading();
        
        if (response.code !== 0) {
            alert('获取主机详情失败: ' + response.message);
            return;
        }
        
        const host = response.data;
        const body = `
            <form id="editHostForm">
                <div class="form-group">
                    <label class="form-label">主机名</label>
                    <input type="text" class="form-control" name="name" value="${host.name}" required>
                </div>
                <div class="form-group">
                    <label class="form-label">IP地址</label>
                    <input type="text" class="form-control" name="ip" value="${host.ip}" required>
                </div>
                <div class="form-group">
                    <label class="form-label">SSH端口</label>
                    <input type="number" class="form-control" name="port" value="${host.port || 22}">
                </div>
                <div class="form-group">
                    <label class="form-label">操作系统</label>
                    <input type="text" class="form-control" name="os" value="${host.os || ''}">
                </div>
                <div class="form-group" style="border-top: 1px solid var(--border-color); padding-top: 15px; margin-top: 15px;">
                    <label class="form-label">SSH 用户名</label>
                    <input type="text" class="form-control" name="username" value="${host.credential ? host.credential.username : 'root'}">
                </div>
                                            <div class="form-group">
                                                <label class="form-label">SSH 密码 (留空则不修改)</label>
                                                <div class="input-wrapper" style="position: relative;">
                                                    <input type="password" class="form-control" name="password" value="${host.credential ? host.credential.password : ''}" placeholder="••••••" id="editHostPassword">
                                                    <i class="fas fa-eye" onclick="togglePasswordVisibility('editHostPassword', this)" style="position: absolute; right: 10px; top: 10px; cursor: pointer; color: var(--text-disabled);"></i>
                                                </div>
                                            </div>
                                            <div class="form-group">
                    <label class="form-label">分组</label>
                    <input type="text" class="form-control" name="group" value="${host.group ? host.group.name : ''}" disabled>
                    <small style="color: var(--text-disabled);">修改分组请使用"移动分组"功能(待开发)</small>
                </div>
            </form>
        `;

        showModal('编辑主机', body, async () => {
            const form = document.getElementById('editHostForm');
            const formData = new FormData(form);
            const data = Object.fromEntries(formData);
            data.port = parseInt(data.port, 10);
            
            try {
                showLoading();
                const updateResp = await apiRequest(`/cmdb/hosts/${id}`, {
                    method: 'PUT',
                    body: JSON.stringify(data)
                });
                if (updateResp.code === 0) {
                    await loadCMDB();
                } else {
                    alert('更新失败: ' + updateResp.message);
                }
            } catch (error) {
                console.error('更新主机失败:', error);
            } finally {
                hideLoading();
            }
        });
    } catch (error) {
        hideLoading();
        console.error('编辑主机失败:', error);
    }
}

async function deleteHost(id) {
    showConfirm('删除主机', '确定要删除该主机吗？此操作不可恢复，且会删除关联的凭据。', async () => {
        try {
            showLoading();
            const response = await apiRequest(`/cmdb/hosts/${id}`, { method: 'DELETE' });
            if (response.code === 0) {
                await loadCMDB();
            } else {
                alert('删除失败: ' + response.message);
            }
        } catch (error) {
            console.error('删除主机失败:', error);
            alert('删除失败，请查看控制台日志');
        } finally {
            hideLoading();
        }
    });
}

async function refreshHosts() {
    showLoading();
    await loadCMDB();
    hideLoading();
}

// ==================== 任务调度 ====================
async function loadTask() {
// ... (existing loadTask)
}

// ==================== 故障自愈 ====================
async function loadRemediation() {
    try {
        const response = await apiRequest('/remediation/logs');
        
        const html = `
            <div class="page-header">
                <h1 class="page-title">
                    <i class="fas fa-magic"></i>
                    故障自愈
                </h1>
                <p class="page-description">自动检测告警并执行预定义的恢复脚本</p>
            </div>

            <div class="stats-grid">
                <div class="stat-card success">
                    <div class="stat-header">
                        <div>
                            <div class="stat-value" id="remSuccessCount">0</div>
                            <div class="stat-label">自愈成功</div>
                        </div>
                        <div class="stat-icon">
                            <i class="fas fa-check-circle"></i>
                        </div>
                    </div>
                </div>
                <div class="stat-card warning">
                    <div class="stat-header">
                        <div>
                            <div class="stat-value" id="remRunningCount">0</div>
                            <div class="stat-label">正在处理</div>
                        </div>
                        <div class="stat-icon">
                            <i class="fas fa-spinner fa-spin"></i>
                        </div>
                    </div>
                </div>
            </div>

            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">自愈执行记录</h3>
                    <button class="btn btn-secondary btn-sm" onclick="loadRemediation()">
                        <i class="fas fa-sync-alt"></i> 刷新
                    </button>
                </div>
                <div class="card-body">
                    <div class="table-container">
                        <table class="table">
                            <thead>
                                <tr>
                                    <th>告警ID</th>
                                    <th>目标</th>
                                    <th>状态</th>
                                    <th>执行动作</th>
                                    <th>开始时间</th>
                                    <th>持续时间</th>
                                    <th>操作</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${response.code === 0 && response.data && response.data.length > 0 ?
                                    response.data.map(log => `
                                        <tr>
                                            <td><code style="font-size: 11px;">${log.alert_id.substring(0, 8)}...</code></td>
                                            <td>${log.target}</td>
                                            <td>
                                                <span class="badge badge-${log.status === 'success' ? 'success' : (log.status === 'running' ? 'warning' : 'danger')}">
                                                    ${log.status === 'success' ? '成功' : (log.status === 'running' ? '处理中' : '失败')}
                                                </span>
                                            </td>
                                            <td><code style="font-size: 11px;">${log.action.substring(0, 20)}${log.action.length > 20 ? '...' : ''}</code></td>
                                            <td>${formatTime(log.started_at)}</td>
                                            <td>${log.duration}s</td>
                                            <td>
                                                <button class="btn btn-secondary btn-sm" onclick="viewRemediationDetail('${log.id}')">
                                                    <i class="fas fa-info-circle"></i> 详情
                                                </button>
                                            </td>
                                        </tr>
                                    `).join('') :
                                    '<tr><td colspan="7" class="text-center">暂无自愈记录</td></tr>'
                                }
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('remediationPage').innerHTML = html;
        
        // 更新统计
        if (response.data) {
            const success = response.data.filter(l => l.status === 'success').length;
            const running = response.data.filter(l => l.status === 'running').length;
            document.getElementById('remSuccessCount').textContent = success;
            document.getElementById('remRunningCount').textContent = running;
        }
    } catch (error) {
        console.error('加载自愈数据失败:', error);
    }
}

async function viewRemediationDetail(id) {
    try {
        showLoading();
        const response = await apiRequest(`/remediation/logs/${id}`);
        hideLoading();
        
        if (response.code === 0) {
            const log = response.data;
            const body = `
                <div style="font-family: monospace; background: #1e1e1e; color: #d4d4d4; padding: 15px; border-radius: 4px; max-height: 400px; overflow-y: auto;">
                    <div style="color: #569cd6; margin-bottom: 10px;"># 自愈详情 [ID: ${log.id}]</div>
                    <div style="margin-bottom: 5px;"><span style="color: #9cdcfe;">状态:</span> ${log.status}</div>
                    <div style="margin-bottom: 5px;"><span style="color: #9cdcfe;">目标:</span> ${log.target}</div>
                    <div style="margin-bottom: 10px;"><span style="color: #9cdcfe;">动作:</span><br>${log.action}</div>
                    
                    <div style="border-top: 1px solid #333; margin: 10px 0; padding-top: 10px;">
                        <span style="color: #ce9178;">STDOUT:</span><br>
                        ${log.stdout || '(无输出)'}
                    </div>
                    ${log.stderr ? `
                    <div style="color: #f44747; margin-top: 10px;">
                        <span>STDERR:</span><br>
                        ${log.stderr}
                    </div>` : ''}
                </div>
            `;
            showModal('自愈执行结果', body);
        }
    } catch (error) {
        hideLoading();
        console.error('获取自愈详情失败:', error);
    }
}

function showAddTaskModal() {
    const body = `
        <form id="addTaskForm">
            <div class="form-group">
                <label class="form-label">任务名称</label>
                <input type="text" class="form-control" name="name" required>
            </div>
            <div class="form-group">
                <label class="form-label">Cron表达式</label>
                <input type="text" class="form-control" name="cron" placeholder="0 0 * * * *" required>
                <small style="color: var(--text-disabled);">格式: 秒 分 时 日 月 周</small>
            </div>
            <div class="form-group">
                <label class="form-label">执行脚本</label>
                <textarea class="form-control" name="script" rows="6" required></textarea>
            </div>
            <div class="form-group">
                <label class="form-label">
                    <input type="checkbox" name="enabled" checked> 启用任务
                </label>
            </div>
        </form>
    `;

    showModal('创建任务', body, async () => {
        const form = document.getElementById('addTaskForm');
        const formData = new FormData(form);
        const data = {
            name: formData.get('name'),
            cron: formData.get('cron'),
            content: formData.get('script'), // 后端字段名为 content
            type: 'shell', // 默认为 shell
            enabled: formData.get('enabled') === 'on'
        };

        try {
            showLoading();
            const response = await apiRequest('/task/tasks', {
                method: 'POST',
                body: JSON.stringify(data)
            });

            if (response.code === 0) {
                await loadTask();
            }
        } catch (error) {
            console.error('创建任务失败:', error);
        } finally {
            hideLoading();
        }
    });
}

// ==================== Docker 管理 ====================
async function loadDocker() {
    try {
        const response = await apiRequest('/docker/hosts');
        
        const html = `
            <div class="page-header">
                <h1 class="page-title">
                    <i class="fab fa-docker"></i>
                    Docker 管理
                </h1>
                <p class="page-description">远程管理 Docker 容器主机</p>
            </div>

            <div class="toolbar">
                <div class="toolbar-left">
                    <button class="btn btn-primary" onclick="showAddDockerHostModal()">
                        <i class="fas fa-plus"></i> 添加主机
                    </button>
                    <button class="btn btn-secondary" onclick="refreshDocker()">
                        <i class="fas fa-sync-alt"></i> 刷新
                    </button>
                </div>
            </div>

            <div class="docker-hosts-grid" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 20px; margin-top: 20px;">
                ${response.code === 0 && response.data ? response.data.map(host => `
                    <div class="card docker-host-card" style="cursor: pointer; transition: transform 0.2s;" onclick="openDockerHost('${host.id}', '${host.name}')" onmouseover="this.style.transform='translateY(-5px)'" onmouseout="this.style.transform='translateY(0)'">
                        <div class="card-body">
                            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                                <div style="display: flex; align-items: center; gap: 15px;">
                                    <div style="background: #e6f7ff; color: #1890ff; width: 50px; height: 50px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 24px;">
                                        <i class="fab fa-docker"></i>
                                    </div>
                                    <div>
                                        <h3 style="margin: 0; font-size: 18px;">${host.name}</h3>
                                        <div style="color: var(--text-secondary); font-size: 12px; margin-top: 5px;">
                                            <span class="badge badge-${host.status === 'online' ? 'success' : 'secondary'}">${host.status || 'Unknown'}</span>
                                        </div>
                                    </div>
                                </div>
                                <button class="btn btn-icon" onclick="event.stopPropagation(); deleteDockerHost('${host.id}')" title="删除环境">
                                    <i class="fas fa-trash"></i>
                                </button>
                            </div>
                            
                            <div style="margin-bottom: 15px;">
                                <button class="btn btn-outline-primary btn-sm btn-block" onclick="event.stopPropagation(); testDockerConnection('${host.id}', '${host.name}')">
                                    <i class="fas fa-stethoscope"></i> 诊断连接
                                </button>
                            </div>
                            
                            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px; background: var(--bg-main); padding: 10px; border-radius: 6px;">
                                <div style="text-align: center;">
                                    <div style="font-size: 20px; font-weight: bold;">${host.container_count || '-'}</div>
                                    <div style="font-size: 12px; color: var(--text-secondary);">容器</div>
                                </div>
                                <div style="text-align: center;">
                                    <div style="font-size: 20px; font-weight: bold;">${host.image_count || '-'}</div>
                                    <div style="font-size: 12px; color: var(--text-secondary);">镜像</div>
                                </div>
                            </div>
                        </div>
                    </div>
                `).join('') : '<div class="empty-state"><i class="fab fa-docker"></i><p>暂无 Docker 主机</p></div>'}
            </div>
        `;

        document.getElementById('dockerPage').innerHTML = html;
    } catch (error) {
        console.error('加载 Docker 失败:', error);
    }
}

async function refreshDocker() {
    showLoading();
    await loadDocker();
    hideLoading();
}

function showAddDockerHostModal() {
    // 获取 CMDB 主机列表
    apiRequest('/cmdb/hosts').then(resp => {
        const hosts = resp.data || [];
        const options = hosts.map(h => `<option value="${h.id}">${h.name} (${h.ip})</option>`).join('');
        
        const body = `
            <form id="addDockerHostForm">
                <div class="form-group">
                    <label class="form-label">名称</label>
                    <input type="text" class="form-control" name="name" required placeholder="例如: Production Docker">
                </div>
                <div class="form-group">
                    <label class="form-label">关联主机 (需在CMDB中配置凭据)</label>
                    <select class="form-control" name="host_id" required>
                        ${options}
                    </select>
                </div>
            </form>
        `;

        showModal('添加 Docker 主机', body, async () => {
            const form = document.getElementById('addDockerHostForm');
            const formData = new FormData(form);
            
            try {
                showLoading();
                const response = await apiRequest('/docker/hosts', {
                    method: 'POST',
                    body: JSON.stringify(Object.fromEntries(formData))
                });
                if (response.code === 0) await loadDocker();
            } catch (error) {
                console.error('添加失败', error);
            } finally {
                hideLoading();
            }
        });
    });
}

async function viewContainers(hostId, hostName) {
    showLoading();
    try {
        const response = await apiRequest(`/docker/hosts/${hostId}/containers`);
        hideLoading();
        
        if (response.code !== 0) {
            alert('获取容器失败: ' + response.message);
            return;
        }

        const containers = response.data;
        const body = `
            <div class="table-container" style="max-height: 500px; overflow-y: auto;">
                <table class="table">
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>名称</th>
                            <th>镜像</th>
                            <th>状态</th>
                            <th>操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${containers.map(c => `
                            <tr>
                                <td><code style="font-size: 11px;">${c.id.substring(0, 12)}</code></td>
                                <td>${c.names[0].replace('/', '')}</td>
                                <td>${c.image}</td>
                                <td>${c.status}</td>
                                <td>
                                    ${c.state === 'running' ? 
                                        `<button class="btn btn-warning btn-sm" onclick="dockerAction('${hostId}', '${c.id}', 'stop')"><i class="fas fa-stop"></i></button>` :
                                        `<button class="btn btn-success btn-sm" onclick="dockerAction('${hostId}', '${c.id}', 'start')"><i class="fas fa-play"></i></button>`
                                    }
                                    <button class="btn btn-secondary btn-sm" onclick="dockerAction('${hostId}', '${c.id}', 'restart')"><i class="fas fa-sync"></i></button>
                                </td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        `;
        
        showModal(`容器列表 - ${hostName}`, body);
    } catch (error) {
        hideLoading();
        console.error(error);
    }
}

async function dockerAction(hostId, containerId, action) {
    try {
        await apiRequest(`/docker/hosts/${hostId}/containers/${containerId}/${action}`, { method: 'POST' });
        // 刷新列表 (这有点hacky，因为我们在模态框里，这里简单处理)
        alert('操作成功');
        // 关闭模态框并重新打开以刷新? 或者直接不做任何事
    } catch (error) {
        alert('操作失败');
    }
}

async function deleteDockerHost(id) {
    showConfirm('删除 Docker 主ique', '确定删除该 Docker 主机吗？这不会停止运行中的容器。', async () => {
        await apiRequest(`/docker/hosts/${id}`, { method: 'DELETE' });
        await loadDocker();
    });
}

async function testDockerConnection(id, name) {
    showLoading();
    try {
        const response = await apiRequest(`/docker/hosts/${id}/test`, { method: 'POST' });
        hideLoading();
        
        const res = response.data;
        const isSuccess = !res.error && (!res.stderr || res.stderr.trim() === '');
        
        let statusHtml = '';
        if (res.error) {
            statusHtml = `
                <div style="background: #fff1f0; border: 1px solid #ffa39e; padding: 15px; border-radius: 6px; margin-bottom: 15px; display: flex; align-items: center; gap: 10px; color: #cf1322;">
                    <i class="fas fa-times-circle" style="font-size: 20px;"></i>
                    <div>
                        <div style="font-weight: bold;">连接失败</div>
                        <div style="font-size: 12px; opacity: 0.8;">请检查 SSH 凭据或网络连通性</div>
                    </div>
                </div>`;
        } else if (res.stderr && res.stderr.trim() !== '') {
            statusHtml = `
                <div style="background: #fffbe6; border: 1px solid #ffe58f; padding: 15px; border-radius: 6px; margin-bottom: 15px; display: flex; align-items: center; gap: 10px; color: #d48806;">
                    <i class="fas fa-exclamation-circle" style="font-size: 20px;"></i>
                    <div>
                        <div style="font-weight: bold;">命令执行有警告</div>
                        <div style="font-size: 12px; opacity: 0.8;">连接成功，但 Docker 返回了错误信息</div>
                    </div>
                </div>`;
        } else {
            statusHtml = `
                <div style="background: #f6ffed; border: 1px solid #b7eb8f; padding: 15px; border-radius: 6px; margin-bottom: 15px; display: flex; align-items: center; gap: 10px; color: #389e0d;">
                    <i class="fas fa-check-circle" style="font-size: 20px;"></i>
                    <div>
                        <div style="font-weight: bold;">连接诊断通过</div>
                        <div style="font-size: 12px; opacity: 0.8;">SSH 连接正常，Docker 命令执行成功</div>
                    </div>
                </div>`;
        }

        const body = `
            ${statusHtml}
            <div style="font-family: monospace; font-size: 12px; background: #1e1e1e; color: #d4d4d4; padding: 10px; border-radius: 4px; max-height: 500px; overflow-y: auto;">
                <div style="color: #569cd6; font-weight: bold; margin-bottom: 5px;">>>> 检查 1: 标准输出 (docker info)</div>
                <div style="white-space: pre-wrap; margin-bottom: 15px; border-bottom: 1px solid #333; padding-bottom: 10px;">${res.stdout || '(无输出)'}</div>
                
                <div style="color: #ce9178;" class="${res.stderr ? '' : 'd-none'}">STDERR:</div>
                <div style="margin-bottom: 10px;" class="${res.stderr ? '' : 'd-none'}">${res.stderr}</div>

                <div style="color: #569cd6; font-weight: bold; margin-bottom: 5px; margin-top: 15px;">>>> 检查 2: JSON 格式 (API使用)</div>
                <div style="color: #888; font-style: italic;">$ ${res.command_json}</div>
                ${res.error_json ? `<div style="color: #f44747;">执行错误: ${res.error_json}</div>` : ''}
                ${res.stderr_json ? `<div style="color: #ce9178;">STDERR: ${res.stderr_json}</div>` : ''}
                <div style="color: #b5cea8; white-space: pre-wrap; margin-top: 5px;">${res.stdout_json || '(无输出 - 这可能是导致数据不显示的原因)'}</div>
            </div>
        `;
        
        showModal(`诊断报告 - ${name}`, body, isSuccess ? async () => {
            // 如果成功，尝试刷新列表
            await refreshDocker();
        } : null);
        
    } catch (error) {
        hideLoading();
        alert('诊断请求失败: ' + error.message);
    }
}

// ==================== 监控中心 ====================
async function loadMonitor() {
    const html = `
        <div class="page-header">
            <h1 class="page-title">
                <i class="fas fa-chart-line"></i>
                监控中心
            </h1>
            <p class="page-description">实时监控系统和服务状态</p>
        </div>

        <div class="stats-grid">
            <div class="stat-card success">
                <div class="stat-header">
                    <div>
                        <div class="stat-value">98.5%</div>
                        <div class="stat-label">系统可用性</div>
                    </div>
                    <div class="stat-icon">
                        <i class="fas fa-heartbeat"></i>
                    </div>
                </div>
            </div>

            <div class="stat-card">
                <div class="stat-header">
                    <div>
                        <div class="stat-value">45</div>
                        <div class="stat-label">监控项</div>
                    </div>
                    <div class="stat-icon">
                        <i class="fas fa-eye"></i>
                    </div>
                </div>
            </div>

            <div class="stat-card warning">
                <div class="stat-header">
                    <div>
                        <div class="stat-value">3</div>
                        <div class="stat-label">告警中</div>
                    </div>
                    <div class="stat-icon">
                        <i class="fas fa-exclamation-triangle"></i>
                    </div>
                </div>
            </div>

            <div class="stat-card danger">
                <div class="stat-header">
                    <div>
                        <div class="stat-value">1</div>
                        <div class="stat-label">故障</div>
                    </div>
                    <div class="stat-icon">
                        <i class="fas fa-times-circle"></i>
                    </div>
                </div>
            </div>
        </div>

        <div class="card">
            <div class="card-header">
                <h3 class="card-title">
                    <i class="fas fa-server"></i>
                    主机监控
                </h3>
                <button class="btn btn-primary" onclick="showAddMonitorModal()">
                    <i class="fas fa-plus"></i> 添加监控
                </button>
            </div>
            <div class="card-body">
                <div class="empty-state">
                    <i class="fas fa-chart-line"></i>
                    <p>暂无监控数据</p>
                    <button class="btn btn-primary" onclick="showAddMonitorModal()">添加监控项</button>
                </div>
            </div>
        </div>
    `;

    document.getElementById('monitorPage').innerHTML = html;
}

// ==================== 告警管理 ====================
async function loadAlert() {
    const html = `
        <div class="page-header">
            <h1 class="page-title">
                <i class="fas fa-bell"></i>
                告警管理
            </h1>
            <p class="page-description">管理和处理系统告警</p>
        </div>

        <div class="toolbar">
            <div class="toolbar-left">
                <button class="btn btn-primary">
                    <i class="fas fa-filter"></i> 全部
                </button>
                <button class="btn btn-secondary">
                    <i class="fas fa-exclamation-circle"></i> 未处理
                </button>
                <button class="btn btn-secondary">
                    <i class="fas fa-check-circle"></i> 已处理
                </button>
            </div>
            <div class="toolbar-right">
                <button class="btn btn-secondary">
                    <i class="fas fa-cog"></i> 告警规则
                </button>
            </div>
        </div>

        <div class="card">
            <div class="card-body">
                <div class="empty-state">
                    <i class="fas fa-bell-slash"></i>
                    <p>暂无告警信息</p>
                </div>
            </div>
        </div>
    `;

    document.getElementById('alertPage').innerHTML = html;
}

// ==================== AI 助手 ====================
let chatSessionId = null;

async function loadAI() {
    const html = `
        <div class="page-header">
            <h1 class="page-title">
                <i class="fas fa-robot"></i>
                AI 运维助手
            </h1>
            <p class="page-description">智能故障诊断和运维建议</p>
        </div>

        <div class="card">
            <div class="card-header">
                <h3 class="card-title">
                    <i class="fas fa-comments"></i>
                    智能对话
                </h3>
                <button class="btn btn-secondary btn-sm" onclick="newChat()">
                    <i class="fas fa-plus"></i> 新对话
                </button>
            </div>
            <div class="card-body">
                <div id="aiChatBox" style="min-height: 400px; max-height: 600px; overflow-y: auto; padding: 20px; background: var(--bg-darker); border-radius: var(--border-radius); margin-bottom: 16px;">
                    <div style="text-align: center; color: var(--text-disabled); padding: 40px;">
                        <i class="fas fa-robot" style="font-size: 48px; margin-bottom: 16px;"></i>
                        <p>你好！我是AI运维助手，有什么可以帮助你的吗？</p>
                        <p style="font-size: 14px; margin-top: 8px;">我可以帮你分析日志、写脚本、或者解答运维疑问。</p>
                    </div>
                </div>
                <div style="display: flex; gap: 12px;">
                    <input type="text" id="aiInput" class="form-control" placeholder="输入你的问题... (按下回车发送)" style="flex: 1;">
                    <button class="btn btn-primary" id="sendAiBtn" onclick="sendAIMessage()">
                        <i class="fas fa-paper-plane"></i> 发送
                    </button>
                </div>
            </div>
        </div>
    `;

    document.getElementById('aiPage').innerHTML = html;
    
    // 绑定回车事件
    document.getElementById('aiInput').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') sendAIMessage();
    });
}

function newChat() {
    chatSessionId = null;
    loadAI();
}

async function sendAIMessage() {
    const input = document.getElementById('aiInput');
    const sendBtn = document.getElementById('sendAiBtn');
    const message = input.value.trim();
    
    if (!message) return;

    const chatBox = document.getElementById('aiChatBox');
    
    // 如果是第一次对话，清空欢迎语
    if (chatBox.querySelector('.fas.fa-robot')) {
        chatBox.innerHTML = '';
    }

    // 添加用户消息
    chatBox.innerHTML += `
        <div style="margin-bottom: 16px; text-align: right;">
            <div style="display: inline-block; background: var(--primary-color); color: white; padding: 10px 16px; border-radius: 12px; max-width: 80%; text-align: left;">
                ${message.replace(/\n/g, '<br>')}
            </div>
        </div>
    `;

    input.value = '';
    input.disabled = true;
    sendBtn.disabled = true;

    // 添加等待状态
    const waitingId = 'waiting_' + Date.now();
    chatBox.innerHTML += `
        <div id="${waitingId}" style="margin-bottom: 16px;">
            <div style="display: inline-block; background: var(--bg-card); color: var(--text-primary); padding: 10px 16px; border-radius: 12px; max-width: 80%;">
                <i class="fas fa-robot"></i> <i class="fas fa-spinner fa-spin"></i> 思考中...
            </div>
        </div>
    `;
    chatBox.scrollTop = chatBox.scrollHeight;

    try {
        const response = await apiRequest('/ai/chat', {
            method: 'POST',
            body: JSON.stringify({
                message: message,
                session_id: chatSessionId
            })
        });

        document.getElementById(waitingId).remove();

        if (response.code === 0) {
            chatSessionId = response.data.session_id;
            const reply = response.data.reply;
            
            chatBox.innerHTML += `
                <div style="margin-bottom: 16px;">
                    <div style="display: inline-block; background: var(--bg-card); color: var(--text-primary); padding: 10px 16px; border-radius: 12px; max-width: 80%; line-height: 1.6;">
                        <div style="margin-bottom: 4px; font-weight: bold;"><i class="fas fa-robot"></i> AI 助手</div>
                        <div>${reply.replace(/\n/g, '<br>')}</div>
                    </div>
                </div>
            `;
        } else {
            chatBox.innerHTML += `
                <div style="margin-bottom: 16px;">
                    <div style="display: inline-block; background: #fee2e2; color: #991b1b; padding: 10px 16px; border-radius: 12px;">
                        出错了: ${response.message}
                    </div>
                </div>
            `;
        }
    } catch (error) {
        document.getElementById(waitingId).remove();
        chatBox.innerHTML += `<div style="color: var(--danger-color); margin-bottom: 16px;">网络错误，请重试。</div>`;
    } finally {
        input.disabled = false;
        sendBtn.disabled = false;
        input.focus();
        chatBox.scrollTop = chatBox.scrollHeight;
    }
}

// ==================== 知识库 ====================
async function loadKnowledge() {
    try {
        const response = await apiRequest('/knowledge/docs');
        
        const html = `
            <div class="page-header">
                <h1 class="page-title">
                    <i class="fas fa-book"></i>
                    AI 知识库
                </h1>
                <p class="page-description">沉淀运维经验，AI 智能辅助检索</p>
            </div>

            <div class="stats-grid">
                <div class="stat-card" style="cursor: pointer;" onclick="showAskModal()">
                    <div class="stat-header">
                        <div>
                            <div class="stat-value">智能提问</div>
                            <div class="stat-label">基于知识库回答</div>
                        </div>
                        <div class="stat-icon">
                            <i class="fas fa-question-circle"></i>
                        </div>
                    </div>
                </div>
                <div class="stat-card" onclick="showAddDocModal()">
                    <div class="stat-header">
                        <div>
                            <div class="stat-value">添加文档</div>
                            <div class="stat-label">上传 Runbook 或 经验</div>
                        </div>
                        <div class="stat-icon">
                            <i class="fas fa-plus-circle"></i>
                        </div>
                    </div>
                </div>
            </div>

            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">文档列表</h3>
                    <div class="toolbar-right">
                        <input type="text" class="form-control" placeholder="搜索文档..." onkeyup="searchDocs(this.value)">
                    </div>
                </div>
                <div class="card-body">
                    <div class="table-container">
                        <table class="table">
                            <thead>
                                <tr>
                                    <th>标题</th>
                                    <th>分类</th>
                                    <th>标签</th>
                                    <th>作者</th>
                                    <th>更新时间</th>
                                    <th>操作</th>
                                </tr>
                            </thead>
                            <tbody id="docsTableBody">
                                ${response.code === 0 && response.data && response.data.length > 0 ?
                                    response.data.map(doc => `
                                        <tr>
                                            <td><strong>${doc.title}</strong></td>
                                            <td><span class="badge badge-secondary">${doc.category || '未分类'}</span></td>
                                            <td>${(doc.tags || '').split(',').map(t => t ? `<span class="badge badge-outline">${t}</span>` : '').join(' ')}</td>
                                            <td>${doc.created_by}</td>
                                            <td>${formatTime(doc.updated_at)}</td>
                                            <td>
                                                <button class="btn btn-secondary btn-sm" onclick="viewDoc('${doc.id}')">
                                                    <i class="fas fa-eye"></i>
                                                </button>
                                                <button class="btn btn-danger btn-sm" onclick="deleteDoc('${doc.id}')">
                                                    <i class="fas fa-trash"></i>
                                                </button>
                                            </td>
                                        </tr>
                                    `).join('') :
                                    '<tr><td colspan="6" class="text-center">暂无文档数据</td></tr>'
                                }
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('knowledgePage').innerHTML = html;
    } catch (error) {
        console.error('加载知识库失败:', error);
    }
}

function showAskModal() {
    const body = `
        <div class="form-group">
            <label class="form-label">请输入您的问题</label>
            <input type="text" id="askQuestion" class="form-control" placeholder="例如：Nginx 502 怎么排查？">
        </div>
        <div id="askResult" style="margin-top: 20px; padding: 15px; background: var(--bg-darker); border-radius: 8px; display: none; line-height: 1.6;">
            <div id="askAnswer"></div>
            <div id="askRefs" style="margin-top: 15px; font-size: 12px; color: var(--text-disabled); border-top: 1px solid var(--border-color); pt: 10px;"></div>
        </div>
    `;

    showModal('智能问答', body, async () => {
        const question = document.getElementById('askQuestion').value;
        if (!question) return;

        const resultEl = document.getElementById('askResult');
        const answerEl = document.getElementById('askAnswer');
        
        resultEl.style.display = 'block';
        answerEl.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 正在检索知识库...';

        try {
            const response = await apiRequest('/knowledge/ask', {
                method: 'POST',
                body: JSON.stringify({ question })
            });

            if (response.code === 0) {
                answerEl.innerHTML = `<strong>AI 回答：</strong><br>${response.data.answer.replace(/\n/g, '<br>')}`;
                if (response.data.references && response.data.references.length > 0) {
                    const refs = response.data.references.map(r => `<li>${r.title}</li>`).join('');
                    document.getElementById('askRefs').innerHTML = `参考文档：<ul>${refs}</ul>`;
                }
            }
        } catch (error) {
            answerEl.innerHTML = '问答请求失败。';
        }
    });
}

function showAddDocModal() {
    const body = `
        <form id="addDocForm">
            <div class="form-group">
                <label class="form-label">标题</label>
                <input type="text" class="form-control" name="title" required placeholder="例如：MySQL 主从切换步骤">
            </div>
            <div class="form-group">
                <label class="form-label">内容 (Markdown)</label>
                <textarea class="form-control" name="content" rows="10" required></textarea>
            </div>
            <div class="form-group">
                <label class="form-label">分类</label>
                <select class="form-control" name="category">
                    <option value="runbook">Runbook (操作手册)</option>
                    <option value="post-mortem">Post-mortem (事故回顾)</option>
                    <option value="guide">Guide (指南)</option>
                    <option value="other">其他</option>
                </select>
            </div>
            <div class="form-group">
                <label class="form-label">标签 (逗号分隔)</label>
                <input type="text" class="form-control" name="tags" placeholder="mysql,ops,ha">
            </div>
        </form>
    `;

    showModal('添加文档', body, async () => {
        const form = document.getElementById('addDocForm');
        const formData = new FormData(form);
        const data = Object.fromEntries(formData);

        try {
            showLoading();
            const response = await apiRequest('/knowledge/docs', {
                method: 'POST',
                body: JSON.stringify(data)
            });

            if (response.code === 0) {
                await loadKnowledge();
            }
        } catch (error) {
            console.error('添加文档失败:', error);
        } finally {
            hideLoading();
        }
    });
}

async function deleteDoc(id) {
    showConfirm('删除文档', '确定删除该文档吗？此操作不可恢复。', async () => {
        try {
            const response = await apiRequest(`/knowledge/docs/${id}`, { method: 'DELETE' });
            if (response.code === 0) await loadKnowledge();
        } catch (error) {
            console.error('删除文档失败:', error);
        }
    });
}

// ==================== 工作流编排 ====================
async function loadWorkflow() {
    const html = `
        <div class="page-header">
            <h1 class="page-title">
                <i class="fas fa-project-diagram"></i>
                工作流编排
            </h1>
            <p class="page-description">可视化自动化流程编排</p>
        </div>

        <div class="toolbar">
            <div class="toolbar-left">
                <button class="btn btn-primary">
                    <i class="fas fa-plus"></i> 创建工作流
                </button>
                <button class="btn btn-secondary">
                    <i class="fas fa-folder-open"></i> 模板库
                </button>
            </div>
        </div>

        <div class="card">
            <div class="card-body">
                <div class="empty-state">
                    <i class="fas fa-project-diagram"></i>
                    <p>暂无工作流</p>
                    <button class="btn btn-primary">创建第一个工作流</button>
                </div>
            </div>
        </div>
    `;

    document.getElementById('workflowPage').innerHTML = html;
}

// ==================== 成本分析 ====================
async function loadCost() {
    const html = `
        <div class="page-header">
            <h1 class="page-title">
                <i class="fas fa-dollar-sign"></i>
                成本分析
            </h1>
            <p class="page-description">云资源成本统计和优化建议</p>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-header">
                    <div>
                        <div class="stat-value">¥12,580</div>
                        <div class="stat-label">本月费用</div>
                    </div>
                    <div class="stat-icon">
                        <i class="fas fa-yen-sign"></i>
                    </div>
                </div>
                <div class="stat-trend up">
                    <i class="fas fa-arrow-up"></i> 较上月 +8.5%
                </div>
            </div>

            <div class="stat-card warning">
                <div class="stat-header">
                    <div>
                        <div class="stat-value">¥15,000</div>
                        <div class="stat-label">预算</div>
                    </div>
                    <div class="stat-icon">
                        <i class="fas fa-wallet"></i>
                    </div>
                </div>
                <div class="stat-trend">
                    <i class="fas fa-info-circle"></i> 已使用 83.9%
                </div>
            </div>

            <div class="stat-card success">
                <div class="stat-header">
                    <div>
                        <div class="stat-value">¥2,340</div>
                        <div class="stat-label">可优化</div>
                    </div>
                    <div class="stat-icon">
                        <i class="fas fa-chart-line"></i>
                    </div>
                </div>
                <div class="stat-trend">
                    <i class="fas fa-lightbulb"></i> 15.6% 节省空间
                </div>
            </div>
        </div>

        <div class="card">
            <div class="card-header">
                <h3 class="card-title">
                    <i class="fas fa-chart-pie"></i>
                    费用分布
                </h3>
            </div>
            <div class="card-body">
                <div class="empty-state">
                    <i class="fas fa-chart-pie"></i>
                    <p>暂无费用数据</p>
                </div>
            </div>
        </div>
    `;

    document.getElementById('costPage').innerHTML = html;
}

// 其他页面的占位符实现
async function loadK8s() {
    document.getElementById('k8sPage').innerHTML = createPlaceholder('Kubernetes', 'docker', 'Kubernetes集群管理功能开发中');
}

async function loadDomain() {
    document.getElementById('domainPage').innerHTML = createPlaceholder('域名监控', 'globe', '域名和SSL证书监控功能开发中');
}

async function loadAnsible() {
    document.getElementById('ansiblePage').innerHTML = createPlaceholder('Ansible', 'cogs', 'Ansible自动化管理功能开发中');
}

async function loadExecutor() {
    document.getElementById('executorPage').innerHTML = createPlaceholder('批量执行', 'terminal', '批量命令执行功能开发中');
}

async function loadCICD() {
    document.getElementById('cicdPage').innerHTML = createPlaceholder('CI/CD', 'code-branch', 'CI/CD流水线管理功能开发中');
}

async function loadGitOps() {
    document.getElementById('gitopsPage').innerHTML = createPlaceholder('GitOps', 'git-alt', 'GitOps配置管理功能开发中');
}

async function loadNacos() {
    document.getElementById('nacosPage').innerHTML = createPlaceholder('Nacos', 'file-code', 'Nacos配置中心功能开发中');
}

async function loadWorkorder() {
    document.getElementById('workorderPage').innerHTML = createPlaceholder('工单管理', 'clipboard-list', '工单管理功能开发中');
}

async function loadOncall() {
    document.getElementById('oncallPage').innerHTML = createPlaceholder('值班管理', 'user-clock', '值班排班功能开发中');
}

async function loadTopology() {
    document.getElementById('topologyPage').innerHTML = createPlaceholder('服务拓扑', 'sitemap', '服务拓扑可视化功能开发中');
}

function createPlaceholder(title, icon, message) {
    return `
        <div class="page-header">
            <h1 class="page-title">
                <i class="fas fa-${icon}"></i>
                ${title}
            </h1>
        </div>
        <div class="card">
            <div class="card-body">
                <div class="empty-state">
                    <i class="fas fa-${icon}"></i>
                    <p>${message}</p>
                </div>
            </div>
        </div>
    `;
}
