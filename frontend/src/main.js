/* ==========================================================================
   AnyZone - Universal DNS Manager
   Frontend Application Controller
   ========================================================================== */

// 状态管理
const state = {
  isUnlocked: false,
  isPinConfigured: false,
  pinMode: 'unlock', // 'unlock' 或 'setup'
  accounts: [],
  selectedAccount: null,
  isAllAccountsMode: true, // 默认开启全选所有厂商聚合模式
  zones: [],
  selectedZone: null,
  zoneFilter: 'all', // 'all', 'active', 'mismatch', 'hidden'
  expandAllZones: false, // 是否临时展开超出设置限制的域名
  nsCache: {}, // 域名公网权威 NS 缓存，严格以 account_id:domain 为唯一键隔离
  records: [],
  settings: {},
  idleTimer: null,
};

// 平台友好中文名称字典
const PROVIDER_NAMES = {
  cloudflare: 'Cloudflare',
  aliyun: '阿里云',
  aliyun_esa: '阿里云 ESA',
  tencent_cloud: '腾讯云',
  tencent_edgeone: '腾讯云 EdgeOne',
  dnspod: 'DNSPod',
  huawei: '华为云',
  volcengine: '火山引擎',
  baidu: '百度云',
  aws_route53: 'AWS Route 53',
  gcp_dns: 'Google Cloud DNS',
  azure_dns: 'Azure DNS',
  oci_dns: 'Oracle Cloud',
  porkbun: 'Porkbun',
  hetzner: 'Hetzner',
  digitalocean: 'DigitalOcean',
  godaddy: 'GoDaddy',
  namecheap: 'Namecheap',
  dnsimple: 'DNSimple',
  bunny: 'Bunny DNS',
  desec: 'deSEC',
  powerdns: 'PowerDNS',
  bind9_rfc2136: 'BIND 9',
  adguard_home: 'AdGuard Home',
};

// 平台凭据模板映射
const providerTemplates = {
  // Token 模式
  cloudflare: { template: 'template-token', label: 'Cloudflare API Token', help: '具备 Zone.DNS 读写权限的受限令牌。' },
  hetzner: { template: 'template-token', label: 'Hetzner Auth-API-Token', help: '可在 dns.hetzner.com 控制台生成。' },
  digitalocean: { template: 'template-token', label: 'DigitalOcean Personal Access Token', help: '具备读写权限的个人访问令牌。' },
  vultr: { template: 'template-token', label: 'Vultr API Key', help: '在 Vultr 账号设置的 API 页面获取。' },
  linode: { template: 'template-token', label: 'Linode (Akamai) Personal Access Token', help: '具备 Domains 读写权限的令牌。' },
  bunny: { template: 'template-token', label: 'Bunny.net API Key', help: '可在 Bunny.net 账户安全设置中获取。' },
  desec: { template: 'template-token', label: 'deSEC Access Token', help: 'deSEC 平台专属访问令牌。' },
  dnsimple: { template: 'template-token', label: 'DNSimple API Token', help: '用户或团队 User Access Token。' },
  huawei: { template: 'template-token', label: '华为云 IAM API Token (X-Auth-Token)', help: '具备 DNS 管理权限的 IAM Token。' },

  // Key + Secret 模式
  aliyun: { template: 'template-aksk', akLabel: 'AccessKey ID', skLabel: 'AccessKey Secret', help: '建议使用 RAM 子账号并授予 AliyunDNSFullAccess。' },
  aliyun_esa: { template: 'template-aksk', akLabel: 'ESA AccessKey ID', skLabel: 'ESA AccessKey Secret', help: '用于管理阿里云 ESA 边缘安全加速域名解析。' },
  tencent_cloud: { template: 'template-aksk', akLabel: 'SecretId', skLabel: 'SecretKey', help: '腾讯云 CAM 访问密钥。' },
  tencent_edgeone: { template: 'template-aksk', akLabel: 'EdgeOne SecretId', skLabel: 'EdgeOne SecretKey', help: '腾讯云 EdgeOne 边缘安全平台专属密钥。' },
  volcengine: { template: 'template-aksk', akLabel: '火山引擎 AccessKey ID', skLabel: 'AccessKey Secret', help: '火山引擎 TrafficRoute DNS 管理权限凭据。' },
  baidu: { template: 'template-aksk', akLabel: '百度云 AccessKey (AK)', skLabel: 'SecretKey (SK)', help: '百度智能云 BCM 资源权限凭据。' },
  aws_route53: { template: 'template-aksk', akLabel: 'AWS Access Key ID', skLabel: 'AWS Secret Access Key', help: '具备 AmazonRoute53FullAccess 的 IAM 凭据。' },
  godaddy: { template: 'template-aksk', akLabel: 'GoDaddy API Key', skLabel: 'GoDaddy API Secret', help: 'developer.godaddy.com 生成的 Production 密钥。' },
  namecheap: { template: 'template-aksk', akLabel: 'Namecheap API User', skLabel: 'API Key', help: 'Namecheap 需开启白名单 IP API 访问。' },
  gcp_dns: { template: 'template-aksk', akLabel: 'GCP Client Email / ID', skLabel: 'Private Key', help: 'Google Cloud DNS 服务账号凭据。' },
  azure_dns: { template: 'template-aksk', akLabel: 'Azure Client ID', skLabel: 'Client Secret', help: 'Azure Active Directory 服务主体凭据。' },
  oci_dns: { template: 'template-aksk', akLabel: 'Oracle OCI User OCID', skLabel: 'Private Key', help: 'Oracle Cloud 基础设施 API 密钥。' },

  // 专属独立模式
  dnspod: { template: 'template-dnspod' },
  porkbun: { template: 'template-porkbun' },
  powerdns: { template: 'template-server' },
  bind9_rfc2136: { template: 'template-server' },
  adguard_home: { template: 'template-server' },
};

// Wails 接口适配器
const backend = {
  isPinConfigured: async () => window.go?.main?.App?.IsPinConfigured() ?? false,
  setupPin: async (pin) => window.go?.main?.App?.SetupPin(pin),
  verifyPin: async (pin) => window.go?.main?.App?.VerifyPin(pin),
  changePin: async (oldP, newP) => window.go?.main?.App?.ChangePin(oldP, newP),
  lock: async () => window.go?.main?.App?.Lock(),
  getSettings: async () => window.go?.main?.App?.GetSettings() ?? {},
  saveSettings: async (s) => window.go?.main?.App?.SaveSettings(s),
  listAccounts: async () => window.go?.main?.App?.ListAccounts() ?? [],
  getAccountDetail: async (accId) => window.go?.main?.App?.GetAccountDetail(accId),
  addAccount: async (name, prov, creds, proxy) => window.go?.main?.App?.AddAccount(name, prov, creds, proxy),
  updateAccount: async (id, name, prov, creds, proxy) => window.go?.main?.App?.UpdateAccount(id, name, prov, creds, proxy),
  deleteAccount: async (id) => window.go?.main?.App?.DeleteAccount(id),
  testAccount: async (prov, creds) => window.go?.main?.App?.TestAccountConnection(prov, creds),
  listZones: async (accId) => window.go?.main?.App?.ListZones(accId) ?? [],
  listAllZones: async () => window.go?.main?.App?.ListAllZones() ?? [],
  checkZoneNS: async (domain, prov) => window.go?.main?.App?.CheckZoneNS(domain, prov),
  checkZonesNSBatch: async (items) => window.go?.main?.App?.CheckZonesNSBatch(items) ?? [],
  toggleHideZone: async (accId, zName, hide) => window.go?.main?.App?.ToggleHideZone(accId, zName, hide),
  listRecords: async (accId, zId, zName) => window.go?.main?.App?.ListRecords(accId, zId, zName) ?? [],
  createRecord: async (accId, zId, zName, rec) => window.go?.main?.App?.CreateRecord(accId, zId, zName, rec),
  updateRecord: async (accId, zId, zName, rec) => window.go?.main?.App?.UpdateRecord(accId, zId, zName, rec),
  deleteRecord: async (accId, zId, zName, rId) => window.go?.main?.App?.DeleteRecord(accId, zId, zName, rId),
};

// UI 元素索引
const el = {
  pinOverlay: document.getElementById('pin-overlay'),
  pinCard: document.getElementById('pin-card'),
  pinTitle: document.getElementById('pin-title'),
  pinSubtitle: document.getElementById('pin-subtitle'),
  pinInput: document.getElementById('pin-input'),
  pinDots: document.getElementById('pin-dots'),
  pinError: document.getElementById('pin-error'),
  btnSubmitPin: document.getElementById('btn-submit-pin'),

  globalSearch: document.getElementById('global-search'),
  proxyIndicator: document.getElementById('proxy-indicator'),
  proxyIndicatorText: document.getElementById('proxy-indicator-text'),
  btnToggleTheme: document.getElementById('btn-toggle-theme'),
  iconThemeSun: document.getElementById('icon-theme-sun'),
  iconThemeMoon: document.getElementById('icon-theme-moon'),
  btnOpenSettings: document.getElementById('btn-open-settings'),
  btnLockApp: document.getElementById('btn-lock-app'),

  accountsList: document.getElementById('accounts-list'),
  zonesList: document.getElementById('zones-list'),
  btnAddAccountModal: document.getElementById('btn-add-account-modal'),
  btnRefreshZones: document.getElementById('btn-refresh-zones'),
  btnCheckNS: document.getElementById('btn-check-ns'),
  hiddenCountBadge: document.getElementById('hidden-count-badge'),

  currentZoneTitle: document.getElementById('current-zone-title'),
  currentZoneProvider: document.getElementById('current-zone-provider'),
  recordsCountBadge: document.getElementById('records-count-badge'),
  btnRefreshRecords: document.getElementById('btn-refresh-records'),
  btnAddRecordModal: document.getElementById('btn-add-record-modal'),
  recordsTbody: document.getElementById('records-tbody'),

  // 模态框
  modalAccount: document.getElementById('modal-account'),
  modalRecord: document.getElementById('modal-record'),
  modalSettings: document.getElementById('modal-settings'),
  toastContainer: document.getElementById('toast-container'),

  // 下拉选择与设置组件
  accProviderSelect: document.getElementById('acc-provider-select'),
  settingThemeSelect: document.getElementById('setting-theme'),
  settingAggregateAll: document.getElementById('setting-aggregate-all'),
  settingPageSize: document.getElementById('setting-page-size'),
  settingAutoHideNS: document.getElementById('setting-auto-hide-ns'),
  btnRestoreAllHidden: document.getElementById('btn-restore-all-hidden'),

  // 右键菜单
  accountContextMenu: document.getElementById('account-context-menu'),
};

// ==========================================================================
// 主题控制 (Auto, Light, Dark mode)
// ==========================================================================

function applyTheme(theme) {
  const root = document.documentElement;
  let isDark = true;
  if (!theme || theme === 'auto') {
    isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    root.setAttribute('data-theme', isDark ? 'dark' : 'light');
  } else {
    isDark = (theme === 'dark');
    root.setAttribute('data-theme', theme);
  }

  // 同步原生 color-scheme
  root.style.colorScheme = isDark ? 'dark' : 'light';

  // 切换右上角图标：深色模式显示太阳（点击切浅色），浅色模式显示月亮（点击切深色）
  if (el.iconThemeSun && el.iconThemeMoon) {
    if (isDark) {
      el.iconThemeSun.style.display = 'block';
      el.iconThemeMoon.style.display = 'none';
      el.btnToggleTheme.title = '切换为浅色模式';
    } else {
      el.iconThemeSun.style.display = 'none';
      el.iconThemeMoon.style.display = 'block';
      el.btnToggleTheme.title = '切换为暗黑模式 (Dark mode)';
    }
  }
}

async function handleToggleThemeClick() {
  const currentTheme = document.documentElement.getAttribute('data-theme');
  const targetTheme = currentTheme === 'dark' ? 'light' : 'dark';
  applyTheme(targetTheme);
  state.settings.theme = targetTheme;
  if (el.settingThemeSelect) {
    el.settingThemeSelect.value = targetTheme;
    syncCustomSelect(el.settingThemeSelect);
  }
  await backend.saveSettings(state.settings);
  showToast(targetTheme === 'dark' ? '已切换为暗黑模式 (Dark mode)' : '已切换为浅色模式 (Light mode)', 'info');
}

// 监听系统主题变化
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
  if (state.settings.theme === 'auto' || !state.settings.theme) {
    applyTheme('auto');
  }
});

// Toast 提示
function showToast(message, type = 'info') {
  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.textContent = message;
  el.toastContainer.appendChild(toast);
  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transition = 'opacity 0.3s ease';
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}

// ==========================================================================
// Custom Select 下拉组件 (完全由 Webview 渲染，彻底根治 Linux WebKitGTK 白底菜单)
// ==========================================================================

function enhanceSelect(selectEl) {
  if (!selectEl || selectEl.dataset.customEnhanced) return;
  selectEl.dataset.customEnhanced = 'true';
  selectEl.style.display = 'none';

  const wrapper = document.createElement('div');
  wrapper.className = 'custom-select-wrapper';
  wrapper.dataset.targetId = selectEl.id;

  const trigger = document.createElement('div');
  trigger.className = 'custom-select-trigger';

  const labelSpan = document.createElement('span');
  const selectedOpt = selectEl.options[selectEl.selectedIndex];
  labelSpan.textContent = selectedOpt ? selectedOpt.text : '请选择';

  const arrowSvg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  arrowSvg.setAttribute('class', 'arrow');
  arrowSvg.setAttribute('viewBox', '0 0 24 24');
  arrowSvg.innerHTML = '<path d="M6 9l6 6 6-6" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round"/>';

  trigger.appendChild(labelSpan);
  trigger.appendChild(arrowSvg);

  const dropdown = document.createElement('div');
  dropdown.className = 'custom-select-dropdown';

  function buildOptions() {
    dropdown.innerHTML = '';
    Array.from(selectEl.children).forEach(child => {
      if (child.tagName === 'OPTGROUP') {
        const groupHeader = document.createElement('div');
        groupHeader.className = 'custom-select-optgroup';
        groupHeader.textContent = child.label;
        dropdown.appendChild(groupHeader);

        Array.from(child.children).forEach(opt => {
          const item = document.createElement('div');
          item.className = `custom-select-option ${opt.selected ? 'selected' : ''}`;
          item.dataset.value = opt.value;
          item.textContent = opt.text;

          item.addEventListener('click', (e) => {
            e.stopPropagation();
            selectEl.value = opt.value;
            selectEl.dispatchEvent(new Event('change', { bubbles: true }));
            labelSpan.textContent = opt.text;
            dropdown.querySelectorAll('.custom-select-option').forEach(o => o.classList.remove('selected'));
            item.classList.add('selected');
            closeAllDropdowns();
          });

          dropdown.appendChild(item);
        });
      } else if (child.tagName === 'OPTION') {
        const item = document.createElement('div');
        item.className = `custom-select-option ${child.selected ? 'selected' : ''}`;
        item.dataset.value = child.value;
        item.textContent = child.text;

        item.addEventListener('click', (e) => {
          e.stopPropagation();
          selectEl.value = child.value;
          selectEl.dispatchEvent(new Event('change', { bubbles: true }));
          labelSpan.textContent = child.text;
          dropdown.querySelectorAll('.custom-select-option').forEach(o => o.classList.remove('selected'));
          item.classList.add('selected');
          closeAllDropdowns();
        });

        dropdown.appendChild(item);
      }
    });
  }

  buildOptions();

  trigger.addEventListener('click', (e) => {
    e.stopPropagation();
    const isOpen = dropdown.classList.contains('open');
    closeAllDropdowns();
    if (!isOpen) {
      trigger.classList.add('open');
      dropdown.classList.add('open');
    }
  });

  wrapper.appendChild(trigger);
  wrapper.appendChild(dropdown);
  selectEl.parentNode.insertBefore(wrapper, selectEl.nextSibling);
}

function closeAllDropdowns() {
  document.querySelectorAll('.custom-select-trigger').forEach(t => t.classList.remove('open'));
  document.querySelectorAll('.custom-select-dropdown').forEach(d => d.classList.remove('open'));
}

function syncCustomSelect(selectEl) {
  if (!selectEl) return;
  const wrapper = selectEl.nextElementSibling;
  if (!wrapper || !wrapper.classList.contains('custom-select-wrapper')) return;

  const labelSpan = wrapper.querySelector('.custom-select-trigger span');
  const selectedOpt = selectEl.options[selectEl.selectedIndex];
  if (labelSpan && selectedOpt) {
    labelSpan.textContent = selectedOpt.text;
  }

  wrapper.querySelectorAll('.custom-select-option').forEach(optEl => {
    if (optEl.dataset.value === selectEl.value) {
      optEl.classList.add('selected');
    } else {
      optEl.classList.remove('selected');
    }
  });
}

function initCustomSelects() {
  document.querySelectorAll('select:not(.select-page-size)').forEach(enhanceSelect);
  document.addEventListener('click', closeAllDropdowns);
  window.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeAllDropdowns();
  });
}

// ==========================================================================
// PIN 验证与动态指示器
// ==========================================================================

async function initSecurity() {
  try {
    state.isPinConfigured = await backend.isPinConfigured();
    state.settings = await backend.getSettings();
    applyTheme(state.settings.theme || 'dark');

    if (!state.isPinConfigured) {
      state.pinMode = 'setup';
      el.pinTitle.textContent = '初始化安全 PIN 码';
      el.pinSubtitle.textContent = '请设置安全 PIN 码 (4~16位)';
      el.btnSubmitPin.textContent = '确认并初始化';
    } else {
      state.pinMode = 'unlock';
      el.pinTitle.textContent = '安全验证';
      el.pinSubtitle.textContent = '请输入安全 PIN 码解锁';
      el.btnSubmitPin.textContent = '解锁';
    }
    showPinOverlay();
  } catch (err) {
    showToast(`安全模块初始化异常: ${err}`, 'error');
  }
}

function showPinOverlay() {
  el.pinOverlay.classList.add('active');
  el.pinInput.value = '';
  updatePinDots();
  el.pinError.textContent = '';
  setTimeout(() => el.pinInput.focus(), 100);
}

function hidePinOverlay() {
  el.pinOverlay.classList.remove('active');
  el.pinInput.value = '';
  updatePinDots();
}

function updatePinDots() {
  const len = el.pinInput.value.length;
  if (len === 0) {
    el.pinDots.innerHTML = '<div class="placeholder-bar"></div>';
    return;
  }

  let html = '';
  for (let i = 0; i < len; i++) {
    html += '<span class="dot"></span>';
  }
  el.pinDots.innerHTML = html;
}

async function handlePinSubmit() {
  const pin = el.pinInput.value.trim();
  if (pin.length < 4) {
    showPinError('PIN 码长度至少 4 位');
    return;
  }

  el.pinError.textContent = '验证解密中...';

  try {
    if (state.pinMode === 'setup') {
      await backend.setupPin(pin);
      state.isPinConfigured = true;
      state.isUnlocked = true;
      hidePinOverlay();
      showToast('PIN 码设置成功，AnyZone 工作台已就绪', 'success');
      loadApp();
    } else {
      await backend.verifyPin(pin);
      state.isUnlocked = true;
      hidePinOverlay();
      showToast('解锁成功', 'success');
      loadApp();
    }
    resetIdleTimer();
  } catch (err) {
    showPinError(err.message || String(err));
  }
}

function showPinError(msg) {
  el.pinError.textContent = msg;
  el.pinCard.classList.add('shake');
  el.pinInput.value = '';
  updatePinDots();
  setTimeout(() => el.pinCard.classList.remove('shake'), 400);
  el.pinInput.focus();
}

function resetIdleTimer() {
  if (state.idleTimer) clearTimeout(state.idleTimer);
  const minutes = state.settings.auto_lock_minutes ?? 15;
  if (minutes > 0 && state.isUnlocked) {
    state.idleTimer = setTimeout(() => {
      lockApp();
      showToast('由于长时间无操作，AnyZone 已自动锁定', 'info');
    }, minutes * 60 * 1000);
  }
}

async function lockApp() {
  await backend.lock();
  state.isUnlocked = false;
  state.pinMode = 'unlock';
  el.pinTitle.textContent = '安全验证';
  el.pinSubtitle.textContent = '请输入安全 PIN 码解锁';
  el.btnSubmitPin.textContent = '解锁';
  showPinOverlay();
}

// ==========================================================================
// 业务数据加载与渲染
// ==========================================================================

async function loadApp() {
  await loadSettings();
  await loadAccounts();
}

async function loadSettings() {
  try {
    state.settings = await backend.getSettings();
    applyTheme(state.settings.theme || 'dark');
    updateProxyIndicatorUI();
  } catch (err) {
    console.error('加载设置失败', err);
  }
}

function updateProxyIndicatorUI() {
  if (state.settings.proxy_enabled && state.settings.proxy_url) {
    el.proxyIndicator.classList.add('active');
    el.proxyIndicator.classList.remove('disabled');
    const mode = state.settings.proxy_routing === 'smart' ? '智能分流' : '全局代理';
    el.proxyIndicatorText.textContent = `加速中 (${mode})`;
  } else {
    el.proxyIndicator.classList.remove('active');
    el.proxyIndicator.classList.add('disabled');
    el.proxyIndicatorText.textContent = '直连模式';
  }
}

async function loadAccounts() {
  try {
    state.accounts = await backend.listAccounts();
    renderAccounts();

    // 默认行为：全选所有厂商聚合模式
    const defaultAll = state.settings.aggregate_all_default !== false;
    if (defaultAll && state.accounts.length > 0) {
      await selectAllAccounts();
    } else if (state.accounts.length > 0) {
      await selectAccount(state.accounts[0]);
    }
  } catch (err) {
    showToast(`获取账号列表失败: ${err}`, 'error');
  }
}

let currentContextAccount = null;

function showAccountContextMenu(e, acc) {
  e.preventDefault();
  e.stopPropagation();
  currentContextAccount = acc;
  const menu = el.accountContextMenu;
  if (!menu) return;
  menu.style.display = 'flex';
  const menuW = 180;
  const menuH = 150;
  const x = Math.min(window.innerWidth - menuW - 10, Math.max(10, e.clientX));
  const y = Math.min(window.innerHeight - menuH - 10, Math.max(10, e.clientY));
  menu.style.left = `${x}px`;
  menu.style.top = `${y}px`;
}

function hideAccountContextMenu() {
  if (el.accountContextMenu) {
    el.accountContextMenu.style.display = 'none';
  }
}

function renderAccounts() {
  el.accountsList.innerHTML = '';
  if (state.accounts.length === 0) {
    el.accountsList.innerHTML = '<div class="empty-hint">暂无账号，点击上方新增</div>';
    el.zonesList.innerHTML = '<div class="empty-hint">请先添加账号</div>';
    return;
  }

  // 1. 顶部渲染“全部厂商”聚合入口
  const allItem = document.createElement('div');
  allItem.className = `nav-item all-accounts ${state.isAllAccountsMode ? 'active' : ''}`;
  allItem.innerHTML = `
    <div class="nav-item-left">
      <span class="provider-tag" style="background: rgba(99,102,241,0.25); color: #818cf8;">ALL</span>
      <span class="nav-item-name" style="font-weight: 600;">全部云厂商</span>
    </div>
    <span class="badge-counter">${state.accounts.length}</span>
  `;
  allItem.addEventListener('click', () => selectAllAccounts());
  el.accountsList.appendChild(allItem);

  // 2. 依次渲染各个账号
  state.accounts.forEach(acc => {
    const item = document.createElement('div');
    const isSingleActive = !state.isAllAccountsMode && state.selectedAccount?.id === acc.id;
    item.className = `nav-item ${isSingleActive ? 'active' : ''}`;

    const tagText = {
      cloudflare: 'CF',
      aliyun: '阿里',
      aliyun_esa: 'ESA',
      tencent_cloud: '腾讯',
      tencent_edgeone: 'EO',
      dnspod: 'DP',
      huawei: '华为',
      volcengine: '火山',
      baidu: '百度',
      aws_route53: 'AWS',
      gcp_dns: 'GCP',
      azure_dns: 'Azure',
      porkbun: 'Porkbun',
    }[acc.provider] || (acc.provider || '').slice(0, 4).toUpperCase();

    item.innerHTML = `
      <div class="nav-item-left">
        <span class="provider-tag ${acc.provider}">${tagText}</span>
        <span class="nav-item-name" title="${escapeHtml(acc.name)}">${escapeHtml(acc.name)}</span>
      </div>
      <div class="nav-item-actions">
        <button class="btn-item-more" title="修改配置与更多操作 (可直接右键)">···</button>
        <button class="btn-item-del" title="删除账号">&times;</button>
      </div>
    `;

    // 绑定右键菜单与操作按钮
    item.addEventListener('contextmenu', (e) => showAccountContextMenu(e, acc));

    const moreBtn = item.querySelector('.btn-item-more');
    if (moreBtn) {
      moreBtn.addEventListener('click', (e) => showAccountContextMenu(e, acc));
    }

    const delBtn = item.querySelector('.btn-item-del');
    if (delBtn) {
      delBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        handleDeleteAccount(acc);
      });
    }

    item.addEventListener('click', (e) => {
      if (e.target.closest('.nav-item-actions')) return;
      selectAccount(acc);
    });

    el.accountsList.appendChild(item);
  });
}

async function selectAllAccounts() {
  state.isAllAccountsMode = true;
  state.selectedAccount = null;
  renderAccounts();
  await loadZones(null);
}

async function selectAccount(acc) {
  state.isAllAccountsMode = false;
  state.selectedAccount = acc;
  renderAccounts();
  await loadZones(acc.id);
}

async function handleDeleteAccount(acc) {
  if (!confirm(`确定要移除账号 "${acc.name}" 吗？该操作不会影响云端数据。`)) return;
  try {
    await backend.deleteAccount(acc.id);
    showToast('账号已移除', 'success');
    if (state.selectedAccount?.id === acc.id) {
      state.selectedAccount = null;
    }
    await loadAccounts();
  } catch (err) {
    showToast(`移除失败: ${err}`, 'error');
  }
}

async function loadZones(accountId) {
  state.expandAllZones = false;
  const loadingText = accountId === null
    ? '正在聚合所有云厂商域名...'
    : '正在同步域名列表中...';
  el.zonesList.innerHTML = `<div class="empty-hint">${loadingText}</div>`;

  try {
    if (accountId === null) {
      state.zones = await backend.listAllZones();
    } else {
      state.zones = await backend.listZones(accountId);
    }

    // 预填后端已确认的权威 NS 状态（如 EdgeOne / ESA 官方 active 站点），避免闪烁
    state.zones.forEach(z => {
      const key = getZoneKey(z);
      if (z.ns_status && !state.nsCache[key]) {
        state.nsCache[key] = {
          account_id: z.account_id,
          domain: z.name,
          status: z.ns_status,
          actual_ns: z.actual_ns || [],
          detected_provider: z.detected_provider || z.provider,
          is_matched: z.ns_status === 'matched',
        };
      }
    });

    renderZones();

    // 后台静默并发探测权威 NS
    triggerBatchNSCheck(state.zones);
  } catch (err) {
    el.zonesList.innerHTML = `<div class="empty-hint" style="color: var(--color-danger)">同步失败: ${err.message || err}</div>`;
    showToast(`获取域名列表失败: ${err}`, 'error');
  }
}

function getZoneKey(z) {
  return (z.account_id || '') + ':' + z.name;
}

// 提供全局切换标签的方法，供空状态下的引导按钮调用
window.switchZoneFilter = function(filterName) {
  const targetTab = document.querySelector(`.filter-tab[data-filter="${filterName}"]`);
  if (targetTab) {
    targetTab.click();
  }
};

async function triggerBatchNSCheck(zones) {
  if (!zones || zones.length === 0) return;

  const toCheck = [];
  zones.forEach(z => {
    const key = getZoneKey(z);
    if (!state.nsCache[key]) {
      toCheck.push({ account_id: z.account_id, domain: z.name, provider: z.provider });
    }
  });

  if (toCheck.length === 0) return;

  try {
    const results = await backend.checkZonesNSBatch(toCheck);
    results.forEach(res => {
      if (res && res.domain) {
        const key = (res.account_id || '') + ':' + res.domain;
        state.nsCache[key] = res;
      }
    });

    // 重新渲染以更新 NS 徽章与过滤展示
    renderZones(false);
  } catch (err) {
    console.warn('NS 批量探测异常:', err);
  }
}

function renderZones(resetSelection = true) {
  el.zonesList.innerHTML = '';

  const filter = state.zoneFilter || 'all';

  // 统计隐藏与未生效数量（隐藏严格以用户手动操作为准）
  let hiddenCount = 0;
  let mismatchCount = 0;
  state.zones.forEach(z => {
    const key = getZoneKey(z);
    const nsInfo = state.nsCache[key];
    const isMismatched = nsInfo && nsInfo.status === 'mismatched';
    if (isMismatched) {
      mismatchCount++;
    }
    if (z.is_hidden) {
      hiddenCount++;
    }
  });
  if (el.hiddenCountBadge) {
    el.hiddenCountBadge.textContent = hiddenCount;
  }

  // 过滤展示列表
  const filtered = state.zones.filter(z => {
    const key = getZoneKey(z);
    const nsInfo = state.nsCache[key];
    const isMismatched = nsInfo && nsInfo.status === 'mismatched';

    if (filter === 'hidden') {
      // “已隐藏”标签仅展示用户主动隐藏的域名
      return z.is_hidden === true;
    }
    if (filter === 'mismatch') {
      return isMismatched; // 切换到“未生效”标签时展示权威 NS 未匹配的域名
    }
    // 用户手动隐藏的域名，在全部、正常、未生效下均不展示
    if (z.is_hidden) {
      return false;
    }
    if (filter === 'active') {
      return nsInfo ? nsInfo.status === 'matched' : true;
    }
    if (filter === 'all') {
      // “全部”标签展示该账号下所有未被手动隐藏的域名（包括 NS 正常的和未生效的）
      return true;
    }
    return true;
  });

  // 分页展示控制
  const limitSetting = parseInt(state.settings.zone_page_size, 10);
  const hasLimit = limitSetting > 0 && !state.expandAllZones;
  const renderLimit = hasLimit ? Math.min(limitSetting, filtered.length) : filtered.length;
  const displayZones = filtered.slice(0, renderLimit);

  if (filtered.length === 0) {
    let emptyMsg = '该账号下暂无域名';
    let actionBtnHtml = '';
    if (filter === 'hidden') {
      emptyMsg = '暂无已隐藏的域名';
    } else if (filter === 'mismatch') {
      emptyMsg = '暂无权威 NS 未生效的域名';
    } else if (filter === 'active') {
      emptyMsg = '暂无权威 NS 正常生效的域名';
      if (mismatchCount > 0) {
        actionBtnHtml = `<div style="margin-top: 10px;"><button class="btn btn-secondary btn-sm" onclick="switchZoneFilter('all')">查看全部域名 (含 ${mismatchCount} 个未生效)</button></div>`;
      }
    } else if (hiddenCount > 0) {
      emptyMsg = `该账号下有 ${hiddenCount} 个域名已被手动隐藏`;
      actionBtnHtml = `<div style="margin-top: 10px;"><button class="btn btn-secondary btn-sm" onclick="switchZoneFilter('hidden')">前往「已隐藏」查看与恢复</button></div>`;
    }
    el.zonesList.innerHTML += `<div class="empty-hint">${emptyMsg}${actionBtnHtml}</div>`;
    return;
  }

  displayZones.forEach(zone => {
    const item = document.createElement('div');
    const isSelected = state.selectedZone?.id === zone.id && state.selectedZone?.account_id === zone.account_id;
    item.className = `zone-item ${isSelected ? 'active' : ''}`;

    // 1. 接入类型徽章：仅 ESA 和 EdgeOne 在是 NS 时显示 NS，是 CNAME 时显示 CNAME；普通厂商一律不显示
    const isEdgeService = zone.provider === 'aliyun_esa' || zone.provider === 'tencent_edgeone';
    const accessType = (zone.access_type || '').toUpperCase();
    let accessBadgeHtml = '';
    if (isEdgeService) {
      if (accessType === 'CNAME') {
        accessBadgeHtml = '<span class="badge-access cname" title="CNAME 接入加速模式">CNAME</span>';
      } else {
        accessBadgeHtml = '<span class="badge-access ns" title="权威 NS 托管接入模式">NS</span>';
      }
    }

    // 2. 权威 NS 状态徽章计算
    const key = getZoneKey(zone);
    const nsInfo = state.nsCache[key];
    let nsBadgeHtml = '';
    if (isEdgeService && accessType === 'CNAME') {
      nsBadgeHtml = '<span class="badge-ns ok" title="CNAME 接入加速就绪">✓ 正常</span>';
    } else if (nsInfo) {
      if (nsInfo.status === 'matched') {
        nsBadgeHtml = '<span class="badge-ns ok" title="权威 NS 与当前服务商匹配正常">✓ 正常</span>';
      } else if (nsInfo.status === 'mismatched') {
        const actualTip = (nsInfo.actual_ns || []).slice(0, 2).join(', ');
        let targetName = '';
        if (nsInfo.detected_provider && nsInfo.detected_provider !== 'other') {
          targetName = PROVIDER_NAMES[nsInfo.detected_provider] || nsInfo.detected_provider;
        } else if (nsInfo.actual_ns && nsInfo.actual_ns.length > 0) {
          const firstHost = (nsInfo.actual_ns[0] || '').toLowerCase().replace(/\.$/, '');
          const parts = firstHost.split('.');
          targetName = parts.length >= 2 ? parts.slice(-2).join('.') : firstHost;
        }
        const provTip = targetName ? `已指向 ${targetName}` : '非本厂商';
        nsBadgeHtml = `<span class="badge-ns mismatch" title="实际权威 NS: ${actualTip} (${provTip})">! 未生效 (${provTip})</span>`;
      } else {
        nsBadgeHtml = '<span class="badge-ns" title="未检测到公网 NS">? 未生效</span>';
      }
    } else {
      nsBadgeHtml = '<span class="badge-ns querying">· 检测中</span>';
    }

    // 账号信息标签（全选模式下显示）
    const accTagHtml = state.isAllAccountsMode && zone.account_name
      ? `<span class="zone-acc-tag" title="${escapeHtml(zone.account_name)}">${escapeHtml(zone.account_name)}</span>`
      : '';

    // 操作按钮：隐藏 / 恢复（严格以用户手动操作为准）
    const isHidden = zone.is_hidden === true;
    const actionBtnHtml = isHidden
      ? '<button class="btn-zone-action btn-zone-restore" title="恢复在当前账号下显示此域名">↺ 恢复</button>'
      : '<button class="btn-zone-action btn-zone-hide" title="仅在当前账号下隐藏此域名">👁 隐藏</button>';

    item.innerHTML = `
      <div class="zone-item-header">
        <span class="zone-item-name" title="${escapeHtml(zone.name)}">${escapeHtml(zone.name)}</span>
        <div class="zone-item-actions">
          ${actionBtnHtml}
        </div>
      </div>
      <div class="zone-meta-line">
        ${accessBadgeHtml}
        ${accTagHtml}
        ${nsBadgeHtml}
      </div>
    `;

    // 绑定隐藏/恢复事件
    const hideBtn = item.querySelector('.btn-zone-hide');
    if (hideBtn) {
      hideBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        handleToggleHide(zone, true);
      });
    }

    const restoreBtn = item.querySelector('.btn-zone-restore');
    if (restoreBtn) {
      restoreBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        handleToggleHide(zone, false);
      });
    }

    item.addEventListener('click', () => selectZone(zone));
    el.zonesList.appendChild(item);
  });

  // 如果因设置数量限制未完全展示，提供一键展开全部按钮
  if (hasLimit && filtered.length > renderLimit) {
    const moreBtn = document.createElement('div');
    moreBtn.className = 'zones-load-more';
    moreBtn.textContent = `展开更多域名 (已显示 ${renderLimit} / 共 ${filtered.length} 个)`;
    moreBtn.addEventListener('click', () => {
      state.expandAllZones = true;
      renderZones(false);
    });
    el.zonesList.appendChild(moreBtn);
  }

  // 默认选中第一个
  if (resetSelection && (!state.selectedZone || !displayZones.some(z => z.id === state.selectedZone.id))) {
    selectZone(displayZones[0]);
  }
}

async function handleToggleHide(zone, hide) {
  try {
    await backend.toggleHideZone(zone.account_id, zone.name, hide);
    zone.is_hidden = hide;
    showToast(hide ? `已隐藏域名 ${zone.name}` : `已恢复显示域名 ${zone.name}`, 'info');
    renderZones(false);
  } catch (err) {
    showToast(`操作失败: ${err.message || err}`, 'error');
  }
}

async function selectZone(zone) {
  if (!zone) return;
  state.selectedZone = zone;

  // 若处于全选模式，根据 zone.account_id 自动定位到实际所属账号
  if (state.isAllAccountsMode) {
    state.selectedAccount = state.accounts.find(a => a.id === zone.account_id) || {
      id: zone.account_id,
      name: zone.account_name,
      provider: zone.provider,
    };
  }

  renderZones(false);

  el.currentZoneTitle.textContent = zone.name;
  el.currentZoneProvider.className = `provider-pill show ${zone.provider}`;
  const provText = state.isAllAccountsMode && zone.account_name
    ? `${zone.provider.toUpperCase()} · ${zone.account_name}`
    : zone.provider.toUpperCase();
  el.currentZoneProvider.textContent = provText;
  el.btnAddRecordModal.disabled = false;

  await loadRecords();
}

async function loadRecords() {
  if (!state.selectedAccount || !state.selectedZone) return;

  el.recordsTbody.innerHTML = '<tr><td colspan="7" class="empty-row">正在加载解析记录...</td></tr>';
  try {
    const res = await backend.listRecords(
      state.selectedAccount.id,
      state.selectedZone.id,
      state.selectedZone.name
    );
    state.records = Array.isArray(res) ? res : [];
    renderRecords();
  } catch (err) {
    state.records = [];
    el.recordsTbody.innerHTML = `<tr><td colspan="7" class="empty-row" style="color: var(--color-danger)">加载失败: ${err.message || err}</td></tr>`;
    showToast(`获取解析记录失败: ${err}`, 'error');
  }
}

function renderRecords() {
  const records = Array.isArray(state.records) ? state.records : [];
  const query = el.globalSearch.value.trim().toLowerCase();
  const filtered = records.filter(r => {
    if (!query) return true;
    return r.name.toLowerCase().includes(query) ||
           r.content.toLowerCase().includes(query) ||
           r.type.toLowerCase().includes(query) ||
           (r.comment && r.comment.toLowerCase().includes(query));
  });

  el.recordsCountBadge.textContent = `${filtered.length} 条记录`;
  el.recordsTbody.innerHTML = '';

  if (filtered.length === 0) {
    el.recordsTbody.innerHTML = '<tr><td colspan="7" class="empty-row">暂无符合条件的解析记录</td></tr>';
    return;
  }

  filtered.forEach(rec => {
    const tr = document.createElement('tr');

    let proxiedHtml = '-';
    if (rec.proxied !== undefined && rec.proxied !== null) {
      proxiedHtml = rec.proxied
        ? '<span class="proxy-toggle proxied">已开启</span>'
        : '<span class="proxy-toggle dns-only">仅 DNS</span>';
    }

    const ttlDisplay = rec.ttl === 1 ? '自动' : `${rec.ttl}s`;

    tr.innerHTML = `
      <td><span class="type-badge type-${rec.type}">${rec.type}</span></td>
      <td><span class="record-name">${escapeHtml(rec.name)}</span></td>
      <td><span class="record-content">${escapeHtml(rec.content)}</span></td>
      <td>${ttlDisplay}</td>
      <td>${proxiedHtml}</td>
      <td><span style="color: var(--text-muted); font-size: 12px;">${escapeHtml(rec.comment || '-')}</span></td>
      <td>
        <div class="table-actions">
          <button class="btn-action-sm btn-edit-record">编辑</button>
          <button class="btn-action-sm danger btn-del-record">删除</button>
        </div>
      </td>
    `;

    tr.querySelector('.btn-edit-record').addEventListener('click', () => openEditRecordModal(rec));
    tr.querySelector('.btn-del-record').addEventListener('click', () => handleDeleteRecord(rec));

    el.recordsTbody.appendChild(tr);
  });
}

function escapeHtml(text) {
  if (text === null || text === undefined) return '';
  return String(text)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

async function handleDeleteRecord(rec) {
  if (!confirm(`确定要删除记录 "${rec.name} -> ${rec.content}" 吗？`)) return;
  try {
    await backend.deleteRecord(
      state.selectedAccount.id,
      state.selectedZone.id,
      state.selectedZone.name,
      rec.id
    );
    showToast('解析记录已删除', 'success');
    await loadRecords();
  } catch (err) {
    showToast(`删除失败: ${err.message || err}`, 'error');
  }
}

// ==========================================================================
// 模态框交互：新增/编辑解析记录
// ==========================================================================

function openAddRecordModal() {
  document.getElementById('modal-record-title').textContent = '新增解析记录';
  document.getElementById('rec-id').value = '';
  document.getElementById('rec-name').value = '';
  document.getElementById('rec-content').value = '';
  document.getElementById('rec-type').value = 'A';
  document.getElementById('rec-ttl').value = '600';
  document.getElementById('rec-priority').value = '10';
  document.getElementById('rec-proxied').checked = false;
  document.getElementById('rec-comment').value = '';

  syncCustomSelect(document.getElementById('rec-type'));
  syncCustomSelect(document.getElementById('rec-ttl'));

  updateRecordModalFields();
  el.modalRecord.classList.add('active');
}

function openEditRecordModal(rec) {
  document.getElementById('modal-record-title').textContent = '编辑解析记录';
  document.getElementById('rec-id').value = rec.id;
  document.getElementById('rec-name').value = rec.name;
  document.getElementById('rec-content').value = rec.content;
  document.getElementById('rec-type').value = rec.type;
  document.getElementById('rec-ttl').value = String(rec.ttl);
  document.getElementById('rec-priority').value = rec.priority || '10';
  document.getElementById('rec-proxied').checked = !!rec.proxied;
  document.getElementById('rec-comment').value = rec.comment || '';

  syncCustomSelect(document.getElementById('rec-type'));
  syncCustomSelect(document.getElementById('rec-ttl'));

  updateRecordModalFields();
  el.modalRecord.classList.add('active');
}

function updateRecordModalFields() {
  const type = document.getElementById('rec-type').value;
  const isCloudflare = state.selectedZone?.provider === 'cloudflare';

  document.getElementById('group-rec-priority').style.display = (type === 'MX' || type === 'SRV') ? 'block' : 'none';
  document.getElementById('group-rec-proxied').style.display = (isCloudflare && (type === 'A' || type === 'AAAA' || type === 'CNAME')) ? 'block' : 'none';
}

document.getElementById('rec-type').addEventListener('change', updateRecordModalFields);

async function handleSaveRecord() {
  const id = document.getElementById('rec-id').value.trim();
  const type = document.getElementById('rec-type').value;
  const name = document.getElementById('rec-name').value.trim();
  const content = document.getElementById('rec-content').value.trim();
  const ttl = parseInt(document.getElementById('rec-ttl').value, 10);
  const priority = parseInt(document.getElementById('rec-priority').value, 10);
  const proxied = document.getElementById('rec-proxied').checked;
  const comment = document.getElementById('rec-comment').value.trim();

  if (!name || !content) {
    showToast('主机记录和记录值不能为空', 'error');
    return;
  }

  const recPayload = {
    id,
    type,
    name,
    content,
    ttl,
    priority: (type === 'MX' || type === 'SRV') ? priority : 0,
    comment,
  };

  if (state.selectedZone?.provider === 'cloudflare') {
    recPayload.proxied = proxied;
  }

  try {
    if (id) {
      await backend.updateRecord(
        state.selectedAccount.id,
        state.selectedZone.id,
        state.selectedZone.name,
        recPayload
      );
      showToast('记录修改成功', 'success');
    } else {
      await backend.createRecord(
        state.selectedAccount.id,
        state.selectedZone.id,
        state.selectedZone.name,
        recPayload
      );
      showToast('记录新增成功', 'success');
    }
    el.modalRecord.classList.remove('active');
    await loadRecords();
  } catch (err) {
    showToast(`保存记录失败: ${err.message || err}`, 'error');
  }
}

// ==========================================================================
// 模态框交互：添加云厂商账号 (下拉选择 20+ 平台动态驱动)
// ==========================================================================

function updateProviderFormDisplay() {
  const selectedProvider = el.accProviderSelect.value;
  const conf = providerTemplates[selectedProvider] || { template: 'template-token' };

  // 隐藏所有模版
  document.querySelectorAll('.provider-form').forEach(f => f.classList.remove('active'));

  // 激活对应模版
  const targetForm = document.getElementById(conf.template);
  if (targetForm) targetForm.classList.add('active');

  // 动态更新表单提示文案
  if (conf.template === 'template-token') {
    if (conf.label) document.getElementById('label-token-input').textContent = conf.label;
    if (conf.help) document.getElementById('help-token-input').textContent = conf.help;
  } else if (conf.template === 'template-aksk') {
    if (conf.akLabel) document.getElementById('label-ak-input').textContent = conf.akLabel;
    if (conf.skLabel) document.getElementById('label-sk-input').textContent = conf.skLabel;
    if (conf.help) document.getElementById('help-aksk-input').textContent = conf.help;
  }

  // 扩展字段切换 (Zone ID 针对 Cloudflare；Region 针对 AWS/华为云)
  const tokenZoneGroup = document.getElementById('group-token-zone-id');
  const akskRegionGroup = document.getElementById('group-aksk-region');

  if (tokenZoneGroup) {
    tokenZoneGroup.style.display = (selectedProvider === 'cloudflare') ? 'block' : 'none';
  }
  if (akskRegionGroup) {
    akskRegionGroup.style.display = (selectedProvider === 'aws_route53' || selectedProvider === 'huawei' || selectedProvider === 'aliyun_esa') ? 'block' : 'none';
  }
}

// DNSPod 二级认证方式切换 (Token 模式 vs 腾讯云 API 密钥模式)
function switchDNSPodAuthMode(mode) {
  const tokenSubform = document.getElementById('dnspod-subform-token');
  const secretSubform = document.getElementById('dnspod-subform-secret');
  const pillToken = document.getElementById('pill-dp-auth-token');
  const pillSecret = document.getElementById('pill-dp-auth-secret');

  if (mode === 'secret') {
    if (tokenSubform) tokenSubform.style.display = 'none';
    if (secretSubform) secretSubform.style.display = 'block';
    if (pillToken) pillToken.classList.remove('active');
    if (pillSecret) pillSecret.classList.add('active');
    const rSecret = document.querySelector('input[name="dnspod-auth-mode"][value="secret"]');
    if (rSecret) rSecret.checked = true;
  } else {
    if (tokenSubform) tokenSubform.style.display = 'block';
    if (secretSubform) secretSubform.style.display = 'none';
    if (pillToken) pillToken.classList.add('active');
    if (pillSecret) pillSecret.classList.remove('active');
    const rToken = document.querySelector('input[name="dnspod-auth-mode"][value="token"]');
    if (rToken) rToken.checked = true;
  }
}

function resetPasswordVisibility() {
  document.querySelectorAll('.input-eye-group').forEach((group) => {
    const input = group.querySelector('input');
    const btn = group.querySelector('.btn-toggle-eye');
    if (input) input.type = 'password';
    if (btn) btn.classList.remove('is-visible');
  });
}

function openAddAccountModal() {
  resetPasswordVisibility();
  document.getElementById('acc-edit-id').value = '';
  document.getElementById('modal-account-title').textContent = '添加云厂商账号';
  document.getElementById('btn-save-account').textContent = '保存账号';

  document.getElementById('acc-name').value = '';
  document.getElementById('acc-proxy').value = '';

  document.getElementById('form-token-input').value = '';
  const tokenZoneInput = document.getElementById('form-token-zone-id');
  if (tokenZoneInput) tokenZoneInput.value = '';

  document.getElementById('form-ak-input').value = '';
  document.getElementById('form-sk-input').value = '';
  const akskRegionInput = document.getElementById('form-aksk-region');
  if (akskRegionInput) akskRegionInput.value = '';

  document.getElementById('form-dp-id').value = '';
  document.getElementById('form-dp-token').value = '';
  const dpSecretId = document.getElementById('form-dp-secret-id');
  if (dpSecretId) dpSecretId.value = '';
  const dpSecretKey = document.getElementById('form-dp-secret-key');
  if (dpSecretKey) dpSecretKey.value = '';
  switchDNSPodAuthMode('token');

  document.getElementById('form-pb-key').value = '';
  document.getElementById('form-pb-secret').value = '';
  document.getElementById('form-srv-url').value = '';
  document.getElementById('form-srv-key').value = '';

  syncCustomSelect(el.accProviderSelect);
  updateProviderFormDisplay();
  el.modalAccount.classList.add('active');
}

async function openEditAccountModal(acc) {
  hideAccountContextMenu();
  if (!acc) return;
  resetPasswordVisibility();

  try {
    showToast(`正在读取账号 "${acc.name}" 的凭据与配置...`, 'info');
    const detail = await backend.getAccountDetail(acc.id);
    if (!detail) throw new Error('未获取到该账号的详细信息');

    const creds = detail.credentials || {};

    // 模态框设为编辑态
    document.getElementById('acc-edit-id').value = detail.id;
    document.getElementById('modal-account-title').textContent = `修改账号配置 - ${detail.name}`;
    document.getElementById('btn-save-account').textContent = '保存修改';

    // 基础信息
    document.getElementById('acc-name').value = detail.name || '';
    document.getElementById('acc-proxy').value = detail.custom_proxy || '';

    // 切换厂商并触发对应模板
    el.accProviderSelect.value = detail.provider;
    syncCustomSelect(el.accProviderSelect);
    updateProviderFormDisplay();

    // 回显各平台凭据
    // 1. Token 模板
    const tokenInput = document.getElementById('form-token-input');
    if (tokenInput) tokenInput.value = creds.api_token || '';
    const zoneIdInput = document.getElementById('form-token-zone-id');
    if (zoneIdInput) zoneIdInput.value = creds.zone_id || '';

    // 2. Key + Secret 模板
    const akInput = document.getElementById('form-ak-input');
    if (akInput) akInput.value = creds.access_key_id || creds.api_key || creds.token_id || '';
    const skInput = document.getElementById('form-sk-input');
    if (skInput) skInput.value = creds.access_key_secret || creds.api_secret || creds.token || '';
    const regionInput = document.getElementById('form-aksk-region');
    if (regionInput) regionInput.value = creds.region || '';

    // 3. DNSPod 模板 (支持 Token 与 腾讯云 API 密钥双模)
    const dpId = document.getElementById('form-dp-id');
    if (dpId) dpId.value = creds.token_id || '';
    const dpToken = document.getElementById('form-dp-token');
    if (dpToken) dpToken.value = creds.token || '';

    const dpSecretId = document.getElementById('form-dp-secret-id');
    if (dpSecretId) dpSecretId.value = creds.access_key_id || creds.api_key || '';
    const dpSecretKey = document.getElementById('form-dp-secret-key');
    if (dpSecretKey) dpSecretKey.value = creds.access_key_secret || creds.api_secret || '';

    if (creds.access_key_id || creds.api_key) {
      switchDNSPodAuthMode('secret');
    } else {
      switchDNSPodAuthMode('token');
    }

    // 4. Porkbun 模板
    const pbKey = document.getElementById('form-pb-key');
    if (pbKey) pbKey.value = creds.api_key || '';
    const pbSecret = document.getElementById('form-pb-secret');
    if (pbSecret) pbSecret.value = creds.api_secret || '';

    // 5. Server 模板
    const srvUrl = document.getElementById('form-srv-url');
    if (srvUrl) srvUrl.value = creds.server_url || '';
    const srvKey = document.getElementById('form-srv-key');
    if (srvKey) srvKey.value = creds.api_key || '';

    el.modalAccount.classList.add('active');
  } catch (err) {
    showToast(`读取账号详情失败: ${err.message || err}`, 'error');
  }
}

function getCredentialsPayload() {
  const selectedProvider = el.accProviderSelect.value;
  const conf = providerTemplates[selectedProvider] || { template: 'template-token' };

  if (conf.template === 'template-token') {
    const token = document.getElementById('form-token-input').value.trim();
    if (!token) throw new Error('请输入 API Token / 访问令牌');
    const tokenZoneId = document.getElementById('form-token-zone-id') ? document.getElementById('form-token-zone-id').value.trim() : '';
    return JSON.stringify({ api_token: token, zone_id: tokenZoneId });
  } else if (conf.template === 'template-aksk') {
    const ak = document.getElementById('form-ak-input').value.trim();
    const sk = document.getElementById('form-sk-input').value.trim();
    if (!ak || !sk) throw new Error('请输入 Key ID 和 Secret 密钥');
    const chkESA = document.getElementById('form-enable-esa');
    const chkEO = document.getElementById('form-enable-edgeone');
    const region = document.getElementById('form-aksk-region') ? document.getElementById('form-aksk-region').value.trim() : '';
    return JSON.stringify({
      access_key_id: ak,
      access_key_secret: sk,
      api_key: ak,
      api_secret: sk,
      region: region,
      enable_esa: selectedProvider === 'aliyun_esa',
      enable_edgeone: selectedProvider === 'tencent_edgeone',
    });
  } else if (conf.template === 'template-dnspod') {
    const selectedMode = document.querySelector('input[name="dnspod-auth-mode"]:checked')?.value || 'token';
    if (selectedMode === 'secret') {
      const secretId = document.getElementById('form-dp-secret-id').value.trim();
      const secretKey = document.getElementById('form-dp-secret-key').value.trim();
      if (!secretId || !secretKey) throw new Error('请输入 SecretId 和 SecretKey (腾讯云 API 密钥)');
      return JSON.stringify({
        access_key_id: secretId,
        access_key_secret: secretKey,
        api_key: secretId,
        api_secret: secretKey,
        enable_edgeone: false,
      });
    } else {
      const id = document.getElementById('form-dp-id').value.trim();
      const token = document.getElementById('form-dp-token').value.trim();
      if (!id || !token) throw new Error('请输入 DNSPod Token ID 和 Token 密钥');
      return JSON.stringify({ token_id: id, token: token });
    }
  } else if (conf.template === 'template-porkbun') {
    const key = document.getElementById('form-pb-key').value.trim();
    const secret = document.getElementById('form-pb-secret').value.trim();
    if (!key || !secret) throw new Error('请输入 Porkbun API Key 和 Secret API Key');
    return JSON.stringify({ api_key: key, secret_api_key: secret });
  } else if (conf.template === 'template-server') {
    const url = document.getElementById('form-srv-url').value.trim();
    const key = document.getElementById('form-srv-key').value.trim();
    if (!url || !key) throw new Error('请输入服务器 API 地址和访问 Key');
    return JSON.stringify({ server_url: url, api_token: key });
  }

  throw new Error('未知凭据模版类型');
}

async function handleTestAccount() {
  try {
    const providerType = el.accProviderSelect.value;
    const creds = getCredentialsPayload();
    showToast('正在发起连通性探测...', 'info');
    await backend.testAccount(providerType, creds);
    showToast('连接测试成功，凭据有效！', 'success');
  } catch (err) {
    showToast(`连通性测试失败: ${err.message || err}`, 'error');
  }
}

async function handleSaveAccount() {
  const editId = document.getElementById('acc-edit-id').value.trim();
  const name = document.getElementById('acc-name').value.trim();
  const providerType = el.accProviderSelect.value;
  const customProxy = document.getElementById('acc-proxy').value.trim();

  if (!name) {
    showToast('请输入账号备注名称', 'error');
    return;
  }

  try {
    const creds = getCredentialsPayload();
    if (editId) {
      await backend.updateAccount(editId, name, providerType, creds, customProxy);
      showToast(`账号 "${name}" 配置已成功更新`, 'success');
      el.modalAccount.classList.remove('active');
      await loadAccounts();
      if (state.selectedAccount?.id === editId) {
        state.selectedAccount.name = name;
        state.selectedAccount.provider = providerType;
        renderAccounts();
        await loadZones(editId);
      }
    } else {
      await backend.addAccount(name, providerType, creds, customProxy);
      showToast('云账号已成功添加并加密保存', 'success');
      el.modalAccount.classList.remove('active');
      await loadAccounts();
    }
  } catch (err) {
    showToast(`保存账号失败: ${err.message || err}`, 'error');
  }
}

// ==========================================================================
// 模态框交互：系统设置
// ==========================================================================

function openSettingsModal() {
  el.settingThemeSelect.value = state.settings.theme || 'auto';
  syncCustomSelect(el.settingThemeSelect);

  // 聚合与域名设置
  if (el.settingAggregateAll) {
    el.settingAggregateAll.checked = state.settings.aggregate_all_default !== false;
  }
  if (el.settingPageSize) {
    el.settingPageSize.value = String(state.settings.zone_page_size ?? 0);
    syncCustomSelect(el.settingPageSize);
  }
  if (el.settingAutoHideNS) {
    el.settingAutoHideNS.checked = !!state.settings.auto_hide_mismatched_ns;
  }

  document.getElementById('setting-proxy-enabled').checked = !!state.settings.proxy_enabled;
  document.getElementById('setting-proxy-url').value = state.settings.proxy_url || '';
  document.getElementById('setting-proxy-routing').value = state.settings.proxy_routing || 'smart';
  syncCustomSelect(document.getElementById('setting-proxy-routing'));

  document.getElementById('setting-autolock').value = String(state.settings.auto_lock_minutes ?? 15);
  syncCustomSelect(document.getElementById('setting-autolock'));

  document.getElementById('change-pin-section').style.display = 'none';
  el.modalSettings.classList.add('active');
}

async function handleSaveSettings() {
  const selectedTheme = el.settingThemeSelect.value;
  const newPageSize = el.settingPageSize ? parseInt(el.settingPageSize.value, 10) : 0;
  state.expandAllZones = false;

  const updated = {
    ...state.settings,
    theme: selectedTheme,
    aggregate_all_default: el.settingAggregateAll ? el.settingAggregateAll.checked : true,
    zone_page_size: newPageSize,
    auto_hide_mismatched_ns: el.settingAutoHideNS ? el.settingAutoHideNS.checked : false,
    proxy_enabled: document.getElementById('setting-proxy-enabled').checked,
    proxy_url: document.getElementById('setting-proxy-url').value.trim(),
    proxy_routing: document.getElementById('setting-proxy-routing').value,
    auto_lock_minutes: parseInt(document.getElementById('setting-autolock').value, 10),
  };

  try {
    await backend.saveSettings(updated);
    state.settings = updated;
    applyTheme(selectedTheme);
    updateProxyIndicatorUI();
    resetIdleTimer();
    renderZones(false);
    showToast('系统设置已更新', 'success');
    el.modalSettings.classList.remove('active');
  } catch (err) {
    showToast(`保存设置失败: ${err.message || err}`, 'error');
  }
}

async function handleRestoreAllHidden() {
  if (!confirm('确定要恢复所有已手动隐藏的域名吗？')) return;
  try {
    state.settings.hidden_zones = [];
    await backend.saveSettings(state.settings);
    state.zones.forEach(z => { z.is_hidden = false; });
    renderZones(false);
    showToast('已恢复所有隐藏的域名', 'success');
  } catch (err) {
    showToast(`恢复失败: ${err.message || err}`, 'error');
  }
}

document.getElementById('btn-change-pin-toggle').addEventListener('click', () => {
  const sec = document.getElementById('change-pin-section');
  sec.style.display = sec.style.display === 'none' ? 'block' : 'none';
});

document.getElementById('btn-execute-change-pin').addEventListener('click', async () => {
  const oldPin = document.getElementById('old-pin').value.trim();
  const newPin = document.getElementById('new-pin').value.trim();
  if (!oldPin || !newPin) {
    showToast('原 PIN 码和新 PIN 码均不能为空', 'error');
    return;
  }
  try {
    await backend.changePin(oldPin, newPin);
    showToast('安全 PIN 码修改成功', 'success');
    document.getElementById('old-pin').value = '';
    document.getElementById('new-pin').value = '';
    document.getElementById('change-pin-section').style.display = 'none';
  } catch (err) {
    showToast(`修改失败: ${err.message || err}`, 'error');
  }
});

// ==========================================================================
// 全局事件监听与初始化绑定
// ==========================================================================

function bindEvents() {
  el.pinInput.addEventListener('input', () => {
    updatePinDots();
    const len = el.pinInput.value.length;
    if (state.pinMode === 'unlock' && state.settings.pin_length && len === state.settings.pin_length) {
      handlePinSubmit();
    }
  });

  el.pinInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') handlePinSubmit();
  });

  el.btnSubmitPin.addEventListener('click', handlePinSubmit);

  window.addEventListener('keydown', (e) => {
    resetIdleTimer();
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'l') {
      e.preventDefault();
      lockApp();
    }
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
      e.preventDefault();
      el.globalSearch.focus();
    }
  });

  ['mousemove', 'mousedown', 'keydown', 'touchstart'].forEach(evt => {
    window.addEventListener(evt, resetIdleTimer, { passive: true });
  });

  // 顶部操作
  const versionLink = document.getElementById('link-app-version');
  if (versionLink) {
    versionLink.addEventListener('click', (e) => {
      e.preventDefault();
      const url = versionLink.getAttribute('href') || 'https://github.com/eallion/anyzone';
      if (window.runtime && window.runtime.BrowserOpenURL) {
        window.runtime.BrowserOpenURL(url);
      } else {
        window.open(url, '_blank');
      }
    });
  }

  el.btnToggleTheme.addEventListener('click', handleToggleThemeClick);
  el.btnLockApp.addEventListener('click', lockApp);
  el.btnOpenSettings.addEventListener('click', openSettingsModal);
  el.proxyIndicator.addEventListener('click', openSettingsModal);
  el.globalSearch.addEventListener('input', renderRecords);

  // 账号操作与凭据明密文切换
  document.querySelectorAll('.btn-toggle-eye').forEach((btn) => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      const group = btn.closest('.input-eye-group');
      if (!group) return;
      const input = group.querySelector('input');
      if (!input) return;
      const isCurrentlyPassword = input.type === 'password';
      input.type = isCurrentlyPassword ? 'text' : 'password';
      btn.classList.toggle('is-visible', isCurrentlyPassword);
    });
  });

  el.btnAddAccountModal.addEventListener('click', openAddAccountModal);
  document.getElementById('btn-close-account-modal').addEventListener('click', () => el.modalAccount.classList.remove('active'));
  document.getElementById('btn-test-account').addEventListener('click', handleTestAccount);
  document.getElementById('btn-save-account').addEventListener('click', handleSaveAccount);

  // DNSPod 二级认证方式切换单选监听
  document.querySelectorAll('input[name="dnspod-auth-mode"]').forEach(r => {
    r.addEventListener('change', (e) => {
      switchDNSPodAuthMode(e.target.value);
    });
  });

  // 域名与 NS 操作
  el.btnRefreshZones.addEventListener('click', () => {
    if (state.isAllAccountsMode) {
      loadZones(null);
    } else if (state.selectedAccount) {
      loadZones(state.selectedAccount.id);
    }
  });

  if (el.btnCheckNS) {
    el.btnCheckNS.addEventListener('click', async () => {
      state.nsCache = {};
      showToast('正在并发检测所有域名的公网权威 NS...', 'info');
      renderZones(false);
      await triggerBatchNSCheck(state.zones);
      showToast('权威 NS 检测已完成', 'success');
    });
  }

  // 域名筛选选项卡
  document.querySelectorAll('.filter-tab').forEach(tab => {
    tab.addEventListener('click', () => {
      document.querySelectorAll('.filter-tab').forEach(t => t.classList.remove('active'));
      tab.classList.add('active');
      state.zoneFilter = tab.dataset.filter;
      state.expandAllZones = false;
      renderZones(true);
    });
  });

  // 右键快捷菜单全局关闭与项点击
  document.addEventListener('click', hideAccountContextMenu);
  document.addEventListener('contextmenu', (e) => {
    if (!e.target.closest('.nav-item')) hideAccountContextMenu();
  });

  const ctxEdit = document.getElementById('ctx-edit-account');
  if (ctxEdit) {
    ctxEdit.addEventListener('click', () => {
      if (currentContextAccount) openEditAccountModal(currentContextAccount);
    });
  }

  const ctxRefresh = document.getElementById('ctx-refresh-account');
  if (ctxRefresh) {
    ctxRefresh.addEventListener('click', () => {
      if (currentContextAccount) {
        selectAccount(currentContextAccount);
        hideAccountContextMenu();
      }
    });
  }

  const ctxTest = document.getElementById('ctx-test-account');
  if (ctxTest) {
    ctxTest.addEventListener('click', async () => {
      if (!currentContextAccount) return;
      hideAccountContextMenu();
      try {
        showToast(`正在测试账号 "${currentContextAccount.name}" 的凭据连通性...`, 'info');
        const detail = await backend.getAccountDetail(currentContextAccount.id);
        const credsJSON = JSON.stringify(detail.credentials || {});
        await backend.testAccount(currentContextAccount.provider, credsJSON);
        showToast(`账号 "${currentContextAccount.name}" 连通性测试通过！`, 'success');
      } catch (err) {
        showToast(`测试失败: ${err.message || err}`, 'error');
      }
    });
  }

  const ctxDel = document.getElementById('ctx-delete-account');
  if (ctxDel) {
    ctxDel.addEventListener('click', () => {
      if (currentContextAccount) {
        const target = currentContextAccount;
        hideAccountContextMenu();
        handleDeleteAccount(target);
      }
    });
  }

  // 解析记录操作
  el.btnRefreshRecords.addEventListener('click', loadRecords);
  el.btnAddRecordModal.addEventListener('click', openAddRecordModal);
  document.getElementById('btn-close-record-modal').addEventListener('click', () => el.modalRecord.classList.remove('active'));
  document.getElementById('btn-cancel-record').addEventListener('click', () => el.modalRecord.classList.remove('active'));
  document.getElementById('btn-save-record').addEventListener('click', handleSaveRecord);

  // 设置模态框关闭与批量恢复
  document.getElementById('btn-close-settings-modal').addEventListener('click', () => el.modalSettings.classList.remove('active'));
  document.getElementById('btn-save-settings').addEventListener('click', handleSaveSettings);
  if (el.btnRestoreAllHidden) {
    el.btnRestoreAllHidden.addEventListener('click', handleRestoreAllHidden);
  }
}

// 启动入口
window.addEventListener('DOMContentLoaded', () => {
  bindEvents();
  initCustomSelects();
  initSecurity();
});
