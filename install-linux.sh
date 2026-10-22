#!/bin/bash
# MC Launcher WebUI - Linux 一键安装脚本
# 作者: NekoMymaomao
# 版本: 2.0.0
# 支持系统: Linux (x86_64, arm64)

set -e

REPO_URL="https://github.com/NekoMymaomao/mc-launcher-webui.git"
TARGET_DIR="${HOME}/mc-launcher"
BACKEND_DIR="$TARGET_DIR/backend"

echo "========================================"
echo "  MC Launcher WebUI - Linux 安装程序"
echo "========================================"
echo ""

# 检查依赖
if ! command -v git &> /dev/null; then
    echo "❌ 未找到 git，请先安装 Git"
    echo "   Ubuntu/Debian: sudo apt install git"
    exit 1
fi
echo "✅ Git 已安装"

if ! command -v go &> /dev/null; then
    echo "❌ 未找到 Go"
    echo "   请安装 Go 1.22+: https://go.dev/dl/"
    exit 1
fi
echo "✅ Go 已安装: $(go version)"

# 克隆仓库
if [ -d "$TARGET_DIR" ]; then
    echo "📁 检测到已有安装目录: $TARGET_DIR"
    read -p "是否覆盖现有安装? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$TARGET_DIR"
    else
        echo "取消安装"
        exit 0
    fi
fi

echo ""
echo "📥 正在克隆仓库..."
git clone --depth 1 --recursive "$REPO_URL" "$TARGET_DIR"

if [ ! -d "$TARGET_DIR" ]; then
    echo "❌ 克隆失败"
    exit 1
fi

echo "✅ 仓库已克隆到: $TARGET_DIR"

# 编译后端
echo ""
echo "🔧 正在编译后端服务..."
cd "$BACKEND_DIR"

go mod tidy
go build -ldflags="-s -w" -o "$TARGET_DIR/mc-launcher-server" .

if [ ! -f "$TARGET_DIR/mc-launcher-server" ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 后端编译成功"

# 复制前端文件
echo ""
echo "📁 准备前端文件..."
mkdir -p "$TARGET_DIR/static"
cp -r "$TARGET_DIR/frontend/"* "$TARGET_DIR/static/"

# 创建启动脚本
cat > "$TARGET_DIR/launcher.sh" << 'LAUNCHER'
#!/bin/bash
# MC Launcher WebUI - 启动脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "========================================"
echo "  MC Launcher WebUI"
echo "========================================"
echo ""
echo "🚀 启动 MC Launcher 服务器..."
echo "   地址: http://localhost:9876"
echo "   按 Ctrl+C 停止"
echo ""

exec ./mc-launcher-server
LAUNCHER
chmod +x "$TARGET_DIR/launcher.sh"

# 创建停止脚本
cat > "$TARGET_DIR/stop.sh" << 'STOP'
#!/bin/bash
PID=$(pgrep -f "mc-launcher-server")
if [ -n "$PID" ]; then
    kill "$PID"
    echo "✅ 已停止 MC Launcher 服务器 (PID: $PID)"
else
    echo "ℹ️  MC Launcher 服务器未运行"
fi
STOP
chmod +x "$TARGET_DIR/stop.sh"

# 创建安装说明
cat > "$TARGET_DIR/README-INSTALL.md" << README
# MC Launcher WebUI - Linux 安装说明

## 安装完成

✅ 已成功安装到: $TARGET_DIR

## 使用方法

### 启动服务器
\`\`\`bash
cd $TARGET_DIR
./launcher.sh
\`\`\`

### 停止服务器
\`\`\`bash
cd $TARGET_DIR
./stop.sh
\`\`\`

### 访问 Web UI
启动服务器后，在浏览器打开:
- http://localhost:9876
- http://127.0.0.1:9876

### 更新版本
\`\`\`bash
cd $TARGET_DIR
git pull && cd backend && go mod tidy && go build -ldflags="-s -w" -o ../mc-launcher-server .
\`\`\`

## 文件说明
- \`launcher.sh\` - 一键启动脚本
- \`stop.sh\` - 一键停止脚本
- \`mc-launcher-server\` - 后端服务
- \`static/\` - Web UI 静态文件
- \`Core/mc-launcher/\` - 核心源码
- \`.github/workflows/\` - CI/CD 配置

## 支持的系统
- ✅ Linux x86_64 (amd64)
- ✅ Linux ARM64 (aarch64)

## 故障排查
如果启动失败:
\`\`\`bash
cd $TARGET_DIR/backend
go mod tidy
go build -ldflags="-s -w" -o ../mc-launcher-server .
\`\`\`

## 卸载
\`\`\`bash
rm -rf $TARGET_DIR
\`\`\`
README

echo ""
echo "========================================"
echo "  ✅ 安装完成!"
echo "========================================"
echo ""
echo "📁 安装位置: $TARGET_DIR"
echo "🚀 启动命令: cd $TARGET_DIR && ./launcher.sh"
echo "🌐 访问地址: http://localhost:9876"
echo ""
echo "========================================"
