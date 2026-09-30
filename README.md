# AnyZone

AnyZone 是一个专为解决多云平台、多账号频繁切换痛点而设计的跨平台本地 DNS 域名解析管理客户端。

基于 Go (Wails v2) 与现代原生前端技术栈构建，支持在单一工作台内统一管理全球主流公有云、边缘安全加速平台（ESA / EdgeOne）以及企业私有自建 DNS 解析记录。

---

## 核心功能与特性

### 1. 全球主流公有云与边缘加速平台深度聚合
AnyZone 内置了针对各平台 API 特性的专属驱动，实现了 25+ 主流平台与协议的统一接入：
- **国内公有云与边缘安全加速平台**：
  - **阿里云 DNS** (Alidns 云解析)
  - **阿里云 ESA** (边缘安全加速，支持华东 1 杭州及海外各地域节点，支持 NS 与 CNAME 接入)
  - **腾讯云 DNS** (腾讯云 CAM 密钥云解析)
  - **腾讯云 EdgeOne** (边缘安全加速，支持权威 NS 与 CNAME 加速域名自动同步)
  - **DNSPod** (独立平台，支持经典 Token 与腾讯云 API 密钥双模认证)
  - **华为云 DNS** (Huawei Cloud DNS)
  - **火山引擎** (TrafficRoute 字节跳动云解析)
  - **百度智能云** (BCM DNS)
- **国际主流云巨头**：
  - **AWS Route 53**
  - **Google Cloud DNS**
  - **Microsoft Azure DNS**
  - **Oracle Cloud (OCI DNS)**
- **海外主流域名注册商与专业解析商**：
  - **Cloudflare** (支持全球 Anycast 与 CDN Proxy 代理加速开关)
  - **Porkbun**、**Hetzner DNS**、**DigitalOcean**、**Vultr DNS**、**Linode (Akamai)**
  - **GoDaddy**、**Namecheap**、**DNSimple**、**Bunny.net DNS**、**deSEC** (开源安全 DNS)
- **企业私有与内网自建 DNS**：
  - **PowerDNS** (基于 REST HTTP API)
  - **BIND 9** (基于 RFC 2136 动态更新协议)
  - **AdGuard Home / CoreDNS**

### 2. 权威 NS 探测与边缘接入模式智能识别
- **实时公网权威 NS 探测**：后台并发对域名进行真实权威 DNS 检测，精准比对当前服务商。
- **目标归属精准诊断**：对于未生效域名，自动解析并清晰标注实际指向的目标服务商（例如：`! 未生效 (已指向 Cloudflare)`），彻底消除解析盲区。
- **边缘接入精细化区分**：对于阿里云 ESA 与腾讯云 EdgeOne 边缘安全加速产品，自动识别并呈现 `[NS]` 权威接入与 `[CNAME]` 加速接入模式；对于普通 DNS 服务商自动隐藏接入标志，界面纯净规范。
- **EdgeOne 加速域名自动提取**：对于 CNAME 接入的 EdgeOne 站点，自动拉取分配的专属 CNAME 地址与部署状态（已生效、部署中、已停用），直接呈现在解析列表中。

### 3. 本地金融级安全与凭据保护
- **全流程本地加密存储**：
  - 核心凭据采用 **Argon2id** (RFC 9106) 高内存抵御算法从主 PIN 码派生 256 位加密密钥。
  - 所有 API Token、AccessKey Secret、Private Key 等敏感字段均通过 **AES-256-GCM** (AEAD 认证加密) 密文落盘存储。
  - 数据文件与目录在 Linux / macOS 下严格受限为 `0600` / `0700` 权限，防止跨用户窃取。
- **防暴力破解与闲置保护**：
  - 连续输入错误触发安全冷却与延时锁定机制（3 次错误锁定 30 秒，5 次错误锁定 5 分钟）。
  - 支持配置闲置超时自动上锁，锁定后内存密钥立即清空。
- **内嵌明密文安全查看**：
  - 所有凭据配置输入框内部右侧内嵌眼睛查看按钮，默认密码掩码防护，点击可在明文与掩码间无缝切换，兼顾安全性与输入核对体验。
- **网络错误脱敏机制**：
  - 严格拦截网络请求错误中的敏感 Query 参数，避免 AccessKey 与签名在错误日志或界面回显中外泄。

### 4. 聚合视图与高效解析管理
- **全局聚合检索**：默认启动即可聚合所有已配置云厂商账号下的全部域名，支持通过快捷键 `Ctrl+F` 实时跨厂商全文搜索主机头、记录类型、解析值与备注。
- **域名多态筛选与隐藏保护**：
  - 支持“全部 / 正常 / 未生效 / 已隐藏”四态一键切换。
  - 支持单个域名独立隐藏与恢复，各云平台显示状态互不干扰。
  - 提供已隐藏域名的批量一键恢复功能与列表分页滚动控制。
- **完整 CRUD 记录操作**：
  - 支持 A、AAAA、CNAME、TXT、MX、NS、SRV、CAA 等全类型解析记录的新增、修改与删除。
  - 完整保留各平台专属特性（如 Cloudflare CDN 代理橙色云朵开关、阿里云与 DNSPod 解析备注、优先级参数等）。

### 5. API 代理加速与智能分流
- 支持配置全局 HTTP、HTTPS 与 SOCKS5 代理。
- 内置**智能分流引擎**：海外服务商（Cloudflare、AWS、Hetzner、DigitalOcean、Porkbun 等）自动经由代理通道加速；国内服务商（阿里云、腾讯云、DNSPod、华为云、火山引擎、百度云）与私有自建服务（PowerDNS）默认直连，严防触发国内公有云风控拦截。
- 支持针对单一账号单独配置专属代理覆盖全局设置。

### 6. 便携模式 (Portable Mode)
- 启动时自动优先检测程序运行目录下的 `./anyzone_data/` 目录；
- 若存在该目录，所有加密数据与用户配置将完全保存在该目录内，可直接放入加密 U 盘或便携介质中实现绿色随身携带。

---

## 项目架构与目录结构

```
anyzone/
├── .github/workflows/
│   └── build-and-release.yml   # GitHub Actions 跨平台矩阵编译与 Release 自动发布流水线
├── frontend/                   # 原生轻量化前端工作台 (HTML5 + Vanilla CSS + ES6)
│   ├── index.html              # 主布局结构 (PIN 验证层、主导航、域名列表、记录面板、配置模态框)
│   ├── src/
│   │   ├── style.css           # 纯原生 CSS，支持暗黑/浅色模式、高对比度下拉框与响应式流体布局
│   │   ├── main.js             # 前端控制中枢、事件委托、实时搜索过滤与 Wails 状态绑定
│   │   └── assets/             # 矢量图标与 Logo 资源
│   └── wailsjs/                # Wails 自动生成的 Go 后端运行时桥接绑定
├── internal/
│   ├── core/
│   │   ├── config/             # 账号实体、凭据模型与应用全局配置规范
│   │   ├── dnsutil/            # 公网权威 NS 递归探测与厂商特征智能分析引擎
│   │   ├── network/            # HTTP / SOCKS5 代理智能分流网络传输层
│   │   ├── security/           # Argon2id 密钥派生、AES-256-GCM 加解密与防爆破状态机
│   │   └── store/              # 便携与系统本地持久化仓储（文件权限保护）
│   └── provider/               # 云厂商驱动平铺包 (实现 DNSProvider 统一接口)
│       ├── interface.go        # 标准 DNSProvider 抽象接口与 Record 数据定义
│       ├── factory.go          # 云厂商驱动工厂与多态分发
│       ├── aliyun.go           # 阿里云 Alidns 与 ESA 边缘安全加速 RPC 驱动
│       ├── tencent.go          # 腾讯云 CAM、EdgeOne 边缘安全加速与 DNSPod 驱动
│       ├── dnspod.go           # DNSPod 经典 API 驱动
│       ├── cloudflare.go       # Cloudflare REST API v4 驱动
│       ├── huawei.go           # 华为云 DNS 驱动
│       ├── porkbun.go          # Porkbun API v3 驱动
│       ├── hetzner.go          # Hetzner DNS REST 驱动
│       ├── digitalocean.go     # DigitalOcean Networking DNS 驱动
│       ├── godaddy.go          # GoDaddy Domains API 驱动
│       └── powerdns.go         # PowerDNS HTTP API 驱动
├── app.go                      # Wails 应用后端生命周期与前端暴露 API 控制器
├── main.go                     # 桌面原生窗口启动入口与配置项
├── wails.json                  # Wails 桌面打包与项目规范定义
├── nfpm.yaml                   # Linux (.deb / .rpm) 打包规范文件
├── go.mod                      # Go 依赖版本管理
└── LICENSE                     # 开源授权协议
```

---

## 开发与构建

### 运行环境要求
- **Go**: 1.22 或更高版本
- **Wails CLI**: v2.9+ (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- **Node.js**: 20+ (用于辅助前端资产检查)

### Linux 系统依赖安装 (以 Ubuntu 24.04 / 26.04 为准)
```bash
sudo apt update
sudo apt install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```

### 启动本地实时开发调试
```bash
wails dev -tags webkit2_41
```

### 本地打包编译生成二进制
```bash
# Linux 本地构建
wails build -platform linux/amd64 -tags webkit2_41 -o anyzone

# Windows 跨平台或本地构建 (支持 NSIS 安装包)
wails build -platform windows/amd64 -nsis -o anyzone.exe

# macOS 本地构建 (Universal 架构)
wails build -platform darwin/universal -o anyzone
```

---

## 跨平台打包与自动化发布

本项目通过 GitHub Actions 实现了全自动多平台持续集成与发布流水线（配置文件位于 `.github/workflows/build-and-release.yml`）。

### 自动构建产物清单
每次发布 Release 均会自动生成经过 SHA256 校验的跨平台分发包：
- **Linux (x86_64 / arm64)**：
  - `anyzone_<version>_linux_amd64.AppImage` / `anyzone_<version>_linux_arm64.AppImage` (通用独立便携运行包)
  - `anyzone_<version>_linux_amd64.deb` / `anyzone_<version>_linux_arm64.deb` (适用 Ubuntu / Debian，自动注册桌面与图标)
  - `anyzone_<version>_linux_amd64.rpm` / `anyzone_<version>_linux_arm64.rpm` (适用 Fedora / RHEL / openSUSE)
  - `anyzone_<version>_linux_amd64.tar.gz` / `anyzone_<version>_linux_arm64.tar.gz` (通用免安装归档)
- **Windows (x86_64 / arm64)**：
  - `anyzone_<version>_windows_amd64.exe` / `anyzone_<version>_windows_arm64.exe` (独立单文件绿色免安装程序，双击直接运行)
  - `anyzone_<version>_windows_amd64_installer.exe` (NSIS 安装向导安装包)
  - `anyzone_<version>_windows_amd64_portable.zip` / `anyzone_<version>_windows_arm64_portable.zip` (绿色便携压缩包)
- **macOS (Apple Silicon & Intel)**：
  - `anyzone_<version>_darwin_arm64.dmg` (Apple Silicon M 系列专用镜像)
  - `anyzone_<version>_darwin_universal.dmg` (通用二进制镜像，兼容 Apple Silicon 与 Intel 芯片)
- **安全校验**：
  - `checksums.txt` (包含所有二进制分发包的 SHA256 完整性哈希清单)

### 发布新版本流程
支持以下两种发布方式：
1. **推 Tag 自动发布**：在本地 Git 仓库创建符合语义化版本格式的 Tag 并推送至远程即可：
   ```bash
   git tag -a v1.0.0 -m "Release version v1.0.0"
   git push origin v1.0.0
   ```
2. **网页端手动触发发布**：进入 GitHub 仓库的 **Actions** 标签页，选择 **Build and Release AnyZone** 工作流，点击 **Run workflow**（可自定义 Tag 版本号，默认为 `v1.0.0`）即可自动打包并同步发布到 Releases 页面。

---

## 开源协议

本项目基于 [MIT License](LICENSE) 开源。