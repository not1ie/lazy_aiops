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
                                    response.data.map(host => `
                                        <tr>
                                            <td><strong>${host.hostname || '-'}</strong></td>
                                            <td>${host.ip || '-'}</td>
                                            <td>${host.os || '-'}</td>
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
                                    `).join('') :
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
                <input type="text" class="form-control" name="hostname" required>
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
                <input type="password" class="form-control" name="password">
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

// ==================== 任务调度 ====================
async function loadTask() {
    try {
        const response = await apiRequest('/task/tasks');
        
        const html = `
            <div class="page-header">
                <h1 class="page-title">
                    <i class="fas fa-tasks"></i>
                    任务调度
                </h1>
                <p class="page-description">管理定时任务和脚本执行</p>
            </div>

            <div class="toolbar">
                <div class="toolbar-left">
                    <button class="btn btn-primary" onclick="showAddTaskModal()">
                        <i class="fas fa-plus"></i> 创建任务
                    </button>
                    <button class="btn btn-secondary" onclick="loadTask()">
                        <i class="fas fa-sync-alt"></i> 刷新
                    </button>
                </div>
            </div>

            <div class="card">
                <div class="card-body">
                    <div class="table-container">
                        <table class="table">
                            <thead>
                                <tr>
                                    <th>任务名称</th>
                                    <th>Cron表达式</th>
                                    <th>状态</th>
                                    <th>下次执行</th>
                                    <th>操作</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${response.code === 0 && response.data && response.data.length > 0 ?
                                    response.data.map(task => `
                                        <tr>
                                            <td><strong>${task.name}</strong></td>
                                            <td><code>${task.cron || '-'}</code></td>
                                            <td><span class="badge badge-${task.enabled ? 'success' : 'secondary'}">${task.enabled ? '启用' : '禁用'}</span></td>
                                            <td>${formatTime(task.next_run)}</td>
                                            <td>
                                                <button class="btn btn-primary btn-sm" onclick="runTask('${task.id}')">
                                                    <i class="fas fa-play"></i>
                                                </button>
                                                <button class="btn btn-secondary btn-sm" onclick="editTask('${task.id}')">
                                                    <i class="fas fa-edit"></i>
                                                </button>
                                                <button class="btn btn-danger btn-sm" onclick="deleteTask('${task.id}')">
                                                    <i class="fas fa-trash"></i>
                                                </button>
                                            </td>
                                        </tr>
                                    `).join('') :
                                    '<tr><td colspan="5" class="text-center">暂无任务</td></tr>'
                                }
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        `;

        document.getElementById('taskPage').innerHTML = html;
    } catch (error) {
        console.error('加载任务失败:', error);
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
            script: formData.get('script'),
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
    if (!confirm('确定删除该文档吗？')) return;
    try {
        const response = await apiRequest(`/knowledge/docs/${id}`, { method: 'DELETE' });
        if (response.code === 0) await loadKnowledge();
    } catch (error) {
        console.error('删除文档失败:', error);
    }
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
