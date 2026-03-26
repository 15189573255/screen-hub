# Screen Hub

跨平台远程屏幕监控与控制系统。通过浏览器实时查看和操控远程桌面。

## 架构

```
┌─────────┐    WebSocket     ┌─────────┐    WebSocket     ┌─────────┐
│ Browser │ ◄──────────────► │ Server  │ ◄──────────────► │  Agent  │
└────┬────┘                  └─────────┘                  └────┬────┘
     │                                                         │
     └──────────── WebRTC P2P (video + control) ───────────────┘
```

- **Server** — 中心信令服务器 + Web UI 托管
- **Agent** — 被控端，安装在每台电脑上，负责截屏推流和输入模拟
- **Browser** — 通过 Web UI 查看在线 Agent 列表，点击即可远程控制

### 工作流程

1. Agent 通过 WebSocket 连接 Server 并注册，定期发送缩略图
2. 浏览器打开 Web UI，通过 WebSocket 获取在线 Agent 列表和实时缩略图
3. 点击某个 Agent 后，通过 Server 中继 WebRTC 信令，建立浏览器与 Agent 的 P2P 连接
4. `video` 数据通道：Agent 以 15 FPS 发送 JPEG 帧
5. `control` 数据通道：浏览器发送鼠标/键盘事件（JSON）

## 环境要求

- Go 1.24+
- **Linux**: 需要安装 `xdotool`（用于输入模拟）
- **Windows**: 无额外依赖
- **macOS**: 需要 CGO 支持（使用 CoreGraphics）

## 编译

### 从源码编译

```bash
git clone https://github.com/15189573255/screen-hub.git
cd screen-hub
```

**编译 Server：**

```bash
go build -o server ./cmd/server
```

**编译 Agent：**

```bash
go build -o agent ./cmd/agent
```

### 交叉编译示例

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o server-linux ./cmd/server
GOOS=linux GOARCH=amd64 go build -o agent-linux ./cmd/agent

# Windows
GOOS=windows GOARCH=amd64 go build -o server.exe ./cmd/server
GOOS=windows GOARCH=amd64 go build -o agent.exe ./cmd/agent

# macOS (需要 CGO 交叉编译工具链)
GOOS=darwin GOARCH=amd64 go build -o server-darwin ./cmd/server
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o agent-darwin ./cmd/agent
```

> **注意**: macOS 上的 Agent 依赖 CoreGraphics (CGO)，交叉编译需要对应的工具链。Server 端无此限制。

## 安装与配置

### Linux

```bash
# 安装输入模拟依赖 (Agent 端需要)
# Debian/Ubuntu
sudo apt install xdotool

# RHEL/CentOS/Fedora
sudo dnf install xdotool

# Arch Linux
sudo pacman -S xdotool

# 下载或编译二进制文件，放置到 /usr/local/bin
sudo cp server /usr/local/bin/screen-hub-server
sudo cp agent /usr/local/bin/screen-hub-agent
sudo chmod +x /usr/local/bin/screen-hub-server /usr/local/bin/screen-hub-agent

# Web 静态文件
sudo mkdir -p /opt/screen-hub/web
sudo cp -r web/* /opt/screen-hub/web/
```

### Windows

```powershell
# 无额外依赖，编译后直接运行
# 建议放置到固定目录，例如：
mkdir C:\screen-hub
copy server.exe C:\screen-hub\
copy agent.exe C:\screen-hub\
xcopy /E web C:\screen-hub\web\
```

### macOS

```bash
# macOS Agent 使用 CoreGraphics，无需额外安装
# 需要在「系统设置 → 隐私与安全性 → 屏幕录制」中授权 Agent
# 需要在「系统设置 → 隐私与安全性 → 辅助功能」中授权 Agent（用于输入控制）

sudo cp server /usr/local/bin/screen-hub-server
sudo cp agent /usr/local/bin/screen-hub-agent
sudo chmod +x /usr/local/bin/screen-hub-server /usr/local/bin/screen-hub-agent

sudo mkdir -p /opt/screen-hub/web
sudo cp -r web/* /opt/screen-hub/web/
```

## 使用

### 启动 Server

```bash
# 默认监听 :8080，Web 静态文件从 ./web 目录读取
./server

# 自定义端口和 Web 目录
./server -addr :9090 -web /opt/screen-hub/web
```

启动后浏览器访问 `http://<server-ip>:8080` 即可打开管理界面。

### 启动 Agent

```bash
# 连接到 Server
./agent -server ws://<server-ip>:8080

# 自定义显示名称和显示器索引
./agent -server ws://<server-ip>:8080 -name "办公室电脑" -display 0
```

**参数说明：**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-server` | `ws://localhost:8080` | Server 的 WebSocket 地址 |
| `-name` | 主机名 | Agent 在管理界面中的显示名称 |
| `-display` | `0` | 要捕获的显示器索引（多显示器时使用） |

## 部署为系统服务

### Linux (systemd)

**Server 服务：**

```bash
sudo tee /etc/systemd/system/screen-hub-server.service > /dev/null <<'EOF'
[Unit]
Description=Screen Hub Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/screen-hub-server -addr :8080 -web /opt/screen-hub/web
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now screen-hub-server
```

**Agent 服务：**

```bash
sudo tee /etc/systemd/system/screen-hub-agent.service > /dev/null <<'EOF'
[Unit]
Description=Screen Hub Agent
After=network.target

[Service]
Type=simple
Environment=DISPLAY=:0
ExecStart=/usr/local/bin/screen-hub-agent -server ws://<server-ip>:8080 -name "my-linux-pc"
Restart=always
RestartSec=5

[Install]
WantedBy=graphical.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now screen-hub-agent
```

> **注意**: Agent 需要访问图形显示，`Environment=DISPLAY=:0` 确保能连接到 X11 显示服务器。如果使用 Wayland，可能需要额外配置。

**常用管理命令：**

```bash
# 查看状态
sudo systemctl status screen-hub-server
sudo systemctl status screen-hub-agent

# 查看日志
sudo journalctl -u screen-hub-server -f
sudo journalctl -u screen-hub-agent -f

# 停止 / 重启
sudo systemctl stop screen-hub-agent
sudo systemctl restart screen-hub-server
```

### Windows 服务

Windows 没有原生方式将普通 exe 注册为服务，推荐使用 [NSSM](https://nssm.cc/) 或 [WinSW](https://github.com/winsw/winsw)。

#### 方式一：使用 NSSM

```powershell
# 下载 nssm: https://nssm.cc/download
# 解压后将 nssm.exe 放入 PATH 或与程序同目录

# 安装 Server 为服务
nssm install ScreenHubServer C:\screen-hub\server.exe -addr :8080 -web C:\screen-hub\web
nssm set ScreenHubServer DisplayName "Screen Hub Server"
nssm set ScreenHubServer Start SERVICE_AUTO_START
nssm start ScreenHubServer

# 安装 Agent 为服务
nssm install ScreenHubAgent C:\screen-hub\agent.exe -server ws://<server-ip>:8080 -name "my-windows-pc"
nssm set ScreenHubAgent DisplayName "Screen Hub Agent"
nssm set ScreenHubAgent Start SERVICE_AUTO_START
nssm start ScreenHubAgent
```

> **注意**: Windows 服务默认运行在 Session 0，无法访问用户桌面。Agent 需要截屏和模拟输入，必须配置为交互式服务或改用「计划任务」方式。

#### 方式二：使用计划任务（推荐用于 Agent）

由于 Agent 需要访问用户桌面，使用计划任务在用户登录时自动启动更可靠：

```powershell
# 创建 Agent 开机自启计划任务（当前用户登录时运行）
schtasks /create /tn "ScreenHubAgent" /tr "C:\screen-hub\agent.exe -server ws://<server-ip>:8080 -name my-pc" /sc onlogon /rl highest

# 立即运行
schtasks /run /tn "ScreenHubAgent"

# 删除任务
schtasks /delete /tn "ScreenHubAgent" /f
```

#### 方式三：使用 WinSW（适用于 Server）

下载 [WinSW](https://github.com/winsw/winsw/releases)，重命名为 `screen-hub-server-svc.exe`，放到 `C:\screen-hub\`，创建同名配置文件 `screen-hub-server-svc.xml`：

```xml
<service>
  <id>ScreenHubServer</id>
  <name>Screen Hub Server</name>
  <description>Screen Hub 信令服务器</description>
  <executable>C:\screen-hub\server.exe</executable>
  <arguments>-addr :8080 -web C:\screen-hub\web</arguments>
  <startmode>Automatic</startmode>
  <log mode="roll-by-size">
    <sizeThreshold>10240</sizeThreshold>
    <keepFiles>3</keepFiles>
  </log>
</service>
```

```powershell
# 安装并启动服务
C:\screen-hub\screen-hub-server-svc.exe install
C:\screen-hub\screen-hub-server-svc.exe start
```

### macOS (launchd)

**Server（系统级守护进程）：**

```bash
sudo tee /Library/LaunchDaemons/com.screen-hub.server.plist > /dev/null <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.screen-hub.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/screen-hub-server</string>
        <string>-addr</string>
        <string>:8080</string>
        <string>-web</string>
        <string>/opt/screen-hub/web</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/screen-hub-server.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/screen-hub-server.log</string>
</dict>
</plist>
EOF

sudo launchctl load /Library/LaunchDaemons/com.screen-hub.server.plist
```

**Agent（用户级，登录后自动运行）：**

```bash
tee ~/Library/LaunchAgents/com.screen-hub.agent.plist > /dev/null <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.screen-hub.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/screen-hub-agent</string>
        <string>-server</string>
        <string>ws://<server-ip>:8080</string>
        <string>-name</string>
        <string>my-mac</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/screen-hub-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/screen-hub-agent.log</string>
</dict>
</plist>
EOF

launchctl load ~/Library/LaunchAgents/com.screen-hub.agent.plist
```

> **注意**: macOS Agent 需要「屏幕录制」和「辅助功能」权限。首次运行时系统会弹窗提示授权，在「系统设置 → 隐私与安全性」中手动开启。

**macOS 常用管理命令：**

```bash
# 查看状态
launchctl list | grep screen-hub

# 停止
launchctl unload ~/Library/LaunchAgents/com.screen-hub.agent.plist
sudo launchctl unload /Library/LaunchDaemons/com.screen-hub.server.plist

# 重启（先 unload 再 load）
launchctl unload ~/Library/LaunchAgents/com.screen-hub.agent.plist
launchctl load ~/Library/LaunchAgents/com.screen-hub.agent.plist
```

## 技术栈

| 组件 | 技术 |
|------|------|
| WebRTC | [pion/webrtc](https://github.com/pion/webrtc) v4 |
| WebSocket | [gorilla/websocket](https://github.com/gorilla/websocket) |
| 截屏 | [kbinani/screenshot](https://github.com/kbinani/screenshot) |
| 图片缩放 | [nfnt/resize](https://github.com/nfnt/resize) |
| 前端 | 原生 HTML / CSS / JavaScript |

## 跨平台输入模拟

| 平台 | 实现方式 |
|------|----------|
| Windows | `user32.dll` SendInput（纯 Go syscall，无 CGO） |
| Linux | `xdotool` 命令行调用（需安装 xdotool） |
| macOS | CoreGraphics CGEvent（需要 CGO） |

## 目录结构

```
screen-hub/
├── cmd/
│   ├── server/main.go     # Server 入口
│   └── agent/main.go      # Agent 入口
├── internal/
│   ├── proto/             # 消息类型定义
│   ├── capture/           # 截屏封装
│   └── input/             # 跨平台输入模拟
├── web/                   # 前端静态文件
├── go.mod
└── go.sum
```

## License

[Apache-2.0](LICENSE)
