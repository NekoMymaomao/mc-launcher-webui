#!/bin/bash
# MC Launcher WebUI - 安装脚本 (仅支持 Linux 和 Windows)

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case "$OS" in
    linux*)
        echo "检测到 Linux 系统"
        exec bash "$(dirname "$0")/install-linux.sh"
        ;;
    msys*|cygwin*|mingw*)
        echo "检测到 Windows 系统 (Git Bash)"
        echo "💡 建议使用原生 Windows 安装脚本:"
        echo "   下载 install-windows.bat 并在 cmd 中运行"
        echo ""
        exec bash "$(dirname "$0")/install-linux.sh"
        ;;
    *)
        echo "❌ 不支持的操作系统: $OS"
        echo ""
        echo "支持的系统:"
        echo "  ✅ Linux (x86_64, amd64, aarch64, arm64)"
        echo "  ✅ Windows (x64)"
        echo ""
        echo "其他系统 (macOS, macOS, iOS, Android) 暂不支持"
        exit 1
        ;;
esac
