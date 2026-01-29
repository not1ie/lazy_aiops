// ==================== 全局配置 ====================
const API_BASE = '/api/v1';
let authToken = localStorage.getItem('authToken');
let currentUser = null;
let currentPage = 'dashboard';

// ==================== 工具函数 ====================

// API 请求封装
async function apiRequest(endpoint, options = {}) {
    const defaultOptions = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${authToken}`
        }
    };

    try {
        const response = await fetch(`${API_BASE}${endpoint}`, {
            ...defaultOptions,
            ...options,
            headers: {
                ...defaultOptions.headers,
                ...options.headers
            }
        });

        const data = await response.json();

        if (response.status === 401) {
            handleLogout();
            throw new Error('未授权，请重新登录');
        }

        return data;
    } catch (error) {
        console.error('API请求失败:', error);
        throw error;
    }
}

// 显示加载状态
function showLoading() {
    document.getElementById('loadingOverlay').classList.add('show');
}

// 隐藏加载状态
function hideLoading() {
    document.getElementById('loadingOverlay').classList.remove('show');
}

// 显示模态框
function showModal(title, body, onConfirm) {
    const modal = document.getElementById('modal');
    document.getElementById('modalTitle').textContent = title;
    document.getElementById('modalBody').innerHTML = body;
    modal.classList.add('show');

    const confirmBtn = document.getElementById('modalConfirm');
    const newConfirmBtn = confirmBtn.cloneNode(true);
    confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

    if (onConfirm) {
        newConfirmBtn.addEventListener('click', () => {
            onConfirm();
            hideModal();
        });
    }
}

// 隐藏模态框
function hideModal() {
    document.getElementById('modal').classList.remove('show');
}

// 格式化时间
function formatTime(timestamp) {
    if (!timestamp) return '-';
    const date = new Date(timestamp);
    return date.toLocaleString('zh-CN');
}

// 格式化文件大小
function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

// ==================== 页面初始化 ====================
document.addEventListener('DOMContentLoaded', () => {
    // 检查登录状态
    if (authToken) {
        showMainPage();
        loadCurrentPage();
    } else {
        showLoginPage();
        
        // 自动填充记住的用户名
        const rememberedUsername = localStorage.getItem('rememberedUsername');
        if (rememberedUsername) {
            document.getElementById('username').value = rememberedUsername;
            document.getElementById('rememberMe').checked = true;
            // 自动聚焦到密码框
            document.getElementById('password').focus();
        } else {
            // 聚焦到用户名框
            document.getElementById('username').focus();
        }
    }

    // 登录表单
    document.getElementById('loginForm').addEventListener('submit', handleLogin);

    // 退出登录
    document.getElementById('logoutBtn').addEventListener('click', handleLogout);

    // 侧边栏切换
    document.getElementById('sidebarToggle').addEventListener('click', toggleSidebar);

    // 导航菜单
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const page = e.currentTarget.dataset.page;
            navigateTo(page);
        });
    });

    // 模态框关闭
    document.getElementById('modalClose').addEventListener('click', hideModal);
    document.getElementById('modalCancel').addEventListener('click', hideModal);
    document.getElementById('modal').addEventListener('click', (e) => {
        if (e.target.id === 'modal') {
            hideModal();
        }
    });
    
    // 回车键快捷登录
    document.getElementById('password').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            document.getElementById('loginForm').dispatchEvent(new Event('submit'));
        }
    });
});

// ==================== 登录相关 ====================

function showLoginPage() {
    document.getElementById('loginPage').classList.add('active');
    document.getElementById('mainPage').classList.remove('active');
}

function showMainPage() {
    document.getElementById('loginPage').classList.remove('active');
    document.getElementById('mainPage').classList.add('active');
}

async function handleLogin(e) {
    e.preventDefault();

    const username = document.getElementById('username').value.trim();
    const password = document.getElementById('password').value;
    const errorEl = document.getElementById('loginError');
    const errorText = document.getElementById('loginErrorText');
    const loginBtn = document.getElementById('loginBtn');

    // 验证输入
    if (!username || !password) {
        errorText.textContent = '请输入用户名和密码';
        errorEl.classList.add('show');
        return;
    }

    // 禁用按钮，显示加载状态
    loginBtn.disabled = true;
    loginBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i><span>登录中...</span>';
    errorEl.classList.remove('show');

    try {
        console.log('开始登录请求...', { username });
        
        const response = await fetch(`${API_BASE}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ 
                username: username, 
                password: password 
            })
        });

        console.log('登录响应状态:', response.status);
        
        const data = await response.json();
        console.log('登录响应数据:', data);

        if (response.ok && data.code === 0 && data.data && data.data.token) {
            authToken = data.data.token;
            localStorage.setItem('authToken', authToken);
            
            // 记住我功能
            if (document.getElementById('rememberMe').checked) {
                localStorage.setItem('rememberedUsername', username);
            } else {
                localStorage.removeItem('rememberedUsername');
            }
            
            console.log('登录成功，Token已保存');
            
            // 显示成功状态
            loginBtn.innerHTML = '<i class="fas fa-check-circle"></i><span>登录成功</span>';
            
            // 延迟跳转，让用户看到成功提示
            setTimeout(() => {
                showMainPage();
                loadCurrentPage();
            }, 500);
        } else {
            // 登录失败
            const errorMessage = data.message || '登录失败，请检查用户名和密码';
            console.error('登录失败:', errorMessage);
            errorText.textContent = errorMessage;
            errorEl.classList.add('show');
            
            // 恢复按钮
            loginBtn.disabled = false;
            loginBtn.innerHTML = '<i class="fas fa-sign-in-alt"></i><span>登录</span>';
        }
    } catch (error) {
        console.error('登录请求异常:', error);
        errorText.textContent = '网络错误，请检查服务器连接';
        errorEl.classList.add('show');
        
        // 恢复按钮
        loginBtn.disabled = false;
        loginBtn.innerHTML = '<i class="fas fa-sign-in-alt"></i><span>登录</span>';
    }
}

function handleLogout() {
    authToken = null;
    currentUser = null;
    localStorage.removeItem('authToken');
    showLoginPage();
    document.getElementById('username').value = '';
    document.getElementById('password').value = '';
}

// ==================== 导航相关 ====================

function toggleSidebar() {
    document.getElementById('sidebar').classList.toggle('show');
}

function navigateTo(page) {
    currentPage = page;

    // 更新导航激活状态
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
        if (item.dataset.page === page) {
            item.classList.add('active');
        }
    });

    // 更新页面内容
    document.querySelectorAll('.page-content').forEach(content => {
        content.classList.remove('active');
    });

    const pageElement = document.getElementById(`${page}Page`);
    if (pageElement) {
        pageElement.classList.add('active');
        loadCurrentPage();
    }

    // 移动端关闭侧边栏
    if (window.innerWidth <= 1024) {
        document.getElementById('sidebar').classList.remove('show');
    }
}

async function loadCurrentPage() {
    showLoading();

    try {
        // 加载用户信息
        if (!currentUser) {
            const userResponse = await apiRequest('/user/info');
            if (userResponse.code === 0) {
                currentUser = userResponse.data;
                document.getElementById('userName').textContent = currentUser.username || 'Admin';
            }
        }

        // 根据当前页面加载对应内容
        switch (currentPage) {
            case 'dashboard':
                await loadDashboard();
                break;
            case 'cmdb':
                await loadCMDB();
                break;
            case 'k8s':
                await loadK8s();
                break;
            case 'docker':
                await loadDocker();
                break;
            case 'monitor':
                await loadMonitor();
                break;
            case 'alert':
                await loadAlert();
                break;
            case 'domain':
                await loadDomain();
                break;
            case 'task':
                await loadTask();
                break;
            case 'remediation':
                await loadRemediation();
                break;
            case 'workflow':
                await loadWorkflow();
                break;
            case 'ansible':
                await loadAnsible();
                break;
            case 'executor':
                await loadExecutor();
                break;
            case 'cicd':
                await loadCICD();
                break;
            case 'gitops':
                await loadGitOps();
                break;
            case 'nacos':
                await loadNacos();
                break;
            case 'ai':
                await loadAI();
                break;
            case 'knowledge':
                await loadKnowledge();
                break;
            case 'workorder':
                await loadWorkorder();
                break;
            case 'oncall':
                await loadOncall();
                break;
            case 'cost':
                await loadCost();
                break;
            case 'topology':
                await loadTopology();
                break;
            default:
                console.warn('Unknown page:', currentPage);
        }
    } catch (error) {
        console.error('加载页面失败:', error);
    } finally {
        hideLoading();
    }
}

// ==================== 仪表板 ====================
async function loadDashboard() {
    try {
        // 获取系统信息
        const sysResponse = await apiRequest('/system/info');
        const pluginsResponse = await apiRequest('/plugins');

        if (sysResponse.code === 0 && pluginsResponse.code === 0) {
            const systemInfo = sysResponse.data;
            const pluginsInfo = pluginsResponse.data;

            const html = `
                <div class="page-header">
                    <h1 class="page-title">
                        <i class="fas fa-home"></i>
                        仪表板
                    </h1>
                    <p class="page-description">系统概览和运行状态</p>
                </div>

                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-header">
                            <div>
                                <div class="stat-value">${pluginsInfo.loaded.length}</div>
                                <div class="stat-label">已加载插件</div>
                            </div>
                            <div class="stat-icon">
                                <i class="fas fa-plug"></i>
                            </div>
                        </div>
                        <div class="stat-trend up">
                            <i class="fas fa-arrow-up"></i> 运行正常
                        </div>
                    </div>

                    <div class="stat-card success">
                        <div class="stat-header">
                            <div>
                                <div class="stat-value">运行中</div>
                                <div class="stat-label">系统状态</div>
                            </div>
                            <div class="stat-icon">
                                <i class="fas fa-check-circle"></i>
                            </div>
                        </div>
                        <div class="stat-trend up">
                            <i class="fas fa-heartbeat"></i> 健康
                        </div>
                    </div>

                    <div class="stat-card warning">
                        <div class="stat-header">
                            <div>
                                <div class="stat-value">${systemInfo.version}</div>
                                <div class="stat-label">系统版本</div>
                            </div>
                            <div class="stat-icon">
                                <i class="fas fa-code-branch"></i>
                            </div>
                        </div>
                        <div class="stat-trend">
                            <i class="fas fa-info-circle"></i> 最新版本
                        </div>
                    </div>

                    <div class="stat-card danger">
                        <div class="stat-header">
                            <div>
                                <div class="stat-value">0</div>
                                <div class="stat-label">告警数量</div>
                            </div>
                            <div class="stat-icon">
                                <i class="fas fa-exclamation-triangle"></i>
                            </div>
                        </div>
                        <div class="stat-trend">
                            <i class="fas fa-shield-alt"></i> 安全
                        </div>
                    </div>
                </div>

                <div class="card">
                    <div class="card-header">
                        <h3 class="card-title">
                            <i class="fas fa-plug"></i>
                            已加载插件
                        </h3>
                    </div>
                    <div class="card-body">
                        <div class="table-container">
                            <table class="table">
                                <thead>
                                    <tr>
                                        <th>插件名称</th>
                                        <th>描述</th>
                                        <th>版本</th>
                                        <th>状态</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${pluginsInfo.loaded.map(plugin => `
                                        <tr>
                                            <td><strong>${plugin.name}</strong></td>
                                            <td>${plugin.description}</td>
                                            <td><span class="badge badge-primary">${plugin.version}</span></td>
                                            <td><span class="badge badge-success">运行中</span></td>
                                        </tr>
                                    `).join('')}
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
            `;

            document.getElementById('dashboardPage').innerHTML = html;
        }
    } catch (error) {
        console.error('加载仪表板失败:', error);
        document.getElementById('dashboardPage').innerHTML = `
            <div class="empty-state">
                <i class="fas fa-exclamation-circle"></i>
                <p>加载失败，请刷新重试</p>
            </div>
        `;
    }
}

// ==================== 页面加载函数占位符 ====================
// 这些函数将在 pages.js 中实现

async function loadCMDB() {
    document.getElementById('cmdbPage').innerHTML = '<div class="empty-state"><i class="fas fa-server"></i><p>CMDB 功能开发中...</p></div>';
}

async function loadK8s() {
    document.getElementById('k8sPage').innerHTML = '<div class="empty-state"><i class="fab fa-docker"></i><p>Kubernetes 功能开发中...</p></div>';
}

async function loadDocker() {
    document.getElementById('dockerPage').innerHTML = '<div class="empty-state"><i class="fab fa-docker"></i><p>Docker 功能加载中...</p></div>';
}

async function loadMonitor() {
    document.getElementById('monitorPage').innerHTML = '<div class="empty-state"><i class="fas fa-chart-line"></i><p>监控功能开发中...</p></div>';
}

async function loadAlert() {
    document.getElementById('alertPage').innerHTML = '<div class="empty-state"><i class="fas fa-bell"></i><p>告警功能开发中...</p></div>';
}

async function loadDomain() {
    document.getElementById('domainPage').innerHTML = '<div class="empty-state"><i class="fas fa-globe"></i><p>域名监控功能开发中...</p></div>';
}

async function loadTask() {
    document.getElementById('taskPage').innerHTML = '<div class="empty-state"><i class="fas fa-tasks"></i><p>任务调度功能开发中...</p></div>';
}

async function loadRemediation() {
    document.getElementById('remediationPage').innerHTML = '<div class="empty-state"><i class="fas fa-magic"></i><p>故障自愈功能开发中...</p></div>';
}

async function loadWorkflow() {
    document.getElementById('workflowPage').innerHTML = '<div class="empty-state"><i class="fas fa-project-diagram"></i><p>工作流功能开发中...</p></div>';
}

async function loadAnsible() {
    document.getElementById('ansiblePage').innerHTML = '<div class="empty-state"><i class="fas fa-cogs"></i><p>Ansible 功能开发中...</p></div>';
}

async function loadExecutor() {
    document.getElementById('executorPage').innerHTML = '<div class="empty-state"><i class="fas fa-terminal"></i><p>批量执行功能开发中...</p></div>';
}

async function loadCICD() {
    document.getElementById('cicdPage').innerHTML = '<div class="empty-state"><i class="fas fa-code-branch"></i><p>CI/CD 功能开发中...</p></div>';
}

async function loadGitOps() {
    document.getElementById('gitopsPage').innerHTML = '<div class="empty-state"><i class="fab fa-git-alt"></i><p>GitOps 功能开发中...</p></div>';
}

async function loadNacos() {
    document.getElementById('nacosPage').innerHTML = '<div class="empty-state"><i class="fas fa-file-code"></i><p>Nacos 功能开发中...</p></div>';
}

async function loadAI() {
    document.getElementById('aiPage').innerHTML = '<div class="empty-state"><i class="fas fa-robot"></i><p>AI 助手功能开发中...</p></div>';
}

async function loadKnowledge() {
    document.getElementById('knowledgePage').innerHTML = '<div class="empty-state"><i class="fas fa-book"></i><p>知识库功能开发中...</p></div>';
}

async function loadWorkorder() {
    document.getElementById('workorderPage').innerHTML = '<div class="empty-state"><i class="fas fa-clipboard-list"></i><p>工单管理功能开发中...</p></div>';
}

async function loadOncall() {
    document.getElementById('oncallPage').innerHTML = '<div class="empty-state"><i class="fas fa-user-clock"></i><p>值班管理功能开发中...</p></div>';
}

async function loadCost() {
    document.getElementById('costPage').innerHTML = '<div class="empty-state"><i class="fas fa-dollar-sign"></i><p>成本分析功能开发中...</p></div>';
}

async function loadTopology() {
    document.getElementById('topologyPage').innerHTML = '<div class="empty-state"><i class="fas fa-sitemap"></i><p>服务拓扑功能开发中...</p></div>';
}
