@echo off
chcp 65001 >nul 2>&1
setlocal EnableDelayedExpansion

echo ========================================
echo   MC Launcher WebUI - Windows 安装程序
echo ========================================
echo.

REM 检查 Git
where git >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ 错误: 未找到 git
    echo    下载地址: https://git-scm.com/download/win
    pause
    exit /b 1
)
echo ✅ Git 已安装

REM 检查 Go
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ 错误: 未找到 Go
    echo    下载地址: https://go.dev/dl/
    pause
    exit /b 1
)
for /f "tokens=*" %%i in ('go version') do set "GO_VERSION=%%i"
echo ✅ Go 已安装: !GO_VERSION!

REM 设置安装目录
set "REPO_URL=https://github.com/NekoMymaomao/mc-launcher-webui.git"
set "TARGET_DIR=%USERPROFILE%\mc-launcher"
set "BACKEND_DIR=%TARGET_DIR%\backend"

REM 检查是否已安装
if exist "%TARGET_DIR%" (
    echo 📁 检测到已有安装目录: %TARGET_DIR%
    set /p OVERWRITE="是否覆盖现有安装? (Y/N): "
    if /i not "!OVERWRITE!"=="Y" (
        echo 取消安装
        pause
        exit /b 0
    )
    rmdir /s /q "%TARGET_DIR%"
)

echo.
echo 📥 正在克隆仓库...
git clone --depth 1 --recursive "%REPO_URL%" "%TARGET_DIR%"

if not exist "%TARGET_DIR%" (
    echo ❌ 克隆失败
    pause
    exit /b 1
)

echo ✅ 仓库已克隆到: %TARGET_DIR%

REM 编译后端
echo.
echo 🔧 正在编译后端服务...
cd /d "%BACKEND_DIR%"
call go mod tidy
call go build -ldflags="-s -w" -o "%TARGET_DIR%\mc-launcher-server.exe" .

if not exist "%TARGET_DIR%\mc-launcher-server.exe" (
    echo ❌ 编译失败
    pause
    exit /b 1
)

echo ✅ 后端编译成功

REM 复制前端文件
echo.
echo 📁 准备前端文件...
mkdir "%TARGET_DIR%\static" 2>nul
xcopy "%TARGET_DIR%\frontend\*" "%TARGET_DIR%\static\" /E /I /Y >nul

REM 创建启动脚本
(
echo @echo off
echo chcp 65001 >nul 2>&1
echo title MC Launcher WebUI
echo echo ========================================
echo echo   MC Launcher WebUI
echo echo ========================================
echo echo.
echo echo 🚀 启动 MC Launcher 服务器...
echo echo    地址: http://localhost:9876
echo echo    按 Ctrl+C 停止
echo echo.
echo cd /d "%%~dp0"
echo start "" "mc-launcher-server.exe"
echo timeout /t 2 >nul
echo start "" "http://localhost:9876"
echo echo.
echo echo ✅ 服务器已启动，浏览器应该已打开
echo echo    如果浏览器未打开，请手动访问: http://localhost:9876
echo echo.
echo pause
) > "%TARGET_DIR%\launcher.bat"

echo ✅ 启动脚本已创建: launcher.bat

REM 创建停止脚本
(
echo @echo off
echo for /f "tokens=*" %%i in ('tasklist /fo csv ^| findstr /i "mc-launcher-server.exe"') do (
echo     for /f "tokens=2 delims=," %%j in ("%%i") do taskkill /pid %%~nj /f
echo )
echo echo ✅ 已停止 MC Launcher 服务器
echo pause
) > "%TARGET_DIR%\stop.bat"

echo ✅ 停止脚本已创建: stop.bat

REM 创建 README
(
echo # MC Launcher WebUI - Windows 安装说明
echo.
echo ## 安装完成
echo.
echo ✅ 已成功安装到: %TARGET_DIR%
echo.
echo ## 使用方法
echo.
echo ### 启动服务器
echo ```
echo cd %TARGET_DIR%
echo .\launcher.bat
echo ```
echo.
echo ### 停止服务器
echo ```
echo cd %TARGET_DIR%
echo .\stop.bat
echo ```
echo.
echo ### 访问 Web UI
echo 启动服务器后，在浏览器打开:
echo - http://localhost:9876
echo - http://127.0.0.1:9876
echo.
echo ### 更新版本
echo ```
echo cd %TARGET_DIR%\backend
echo go mod tidy
echo go build -ldflags^="-s -w" -o ..\mc-launcher-server.exe .
echo ```
echo.
echo ## 文件说明
echo - launcher.bat - 一键启动脚本
echo - stop.bat - 一键停止脚本
echo - mc-launcher-server.exe - 后端服务
echo - static/ - Web UI 静态文件
echo - Core/mc-launcher/ - 核心源码
echo - .github/workflows/ - CI/CD 配置
echo.
echo ## 支持的系统
echo - ✅ Windows x64
echo.
echo ## 故障排查
echo 如果启动失败:
echo ```
echo cd %TARGET_DIR%\backend
echo go mod tidy
echo go build -ldflags^="-s -w" -o ..\mc-launcher-server.exe .
echo ```
echo.
echo ## 卸载
echo 删除目录: %TARGET_DIR%
) > "%TARGET_DIR%\README-INSTALL.md"

echo.
echo ========================================
echo   ✅ 安装完成!
echo ========================================
echo.
echo 📁 安装位置: %TARGET_DIR%
echo 🚀 启动命令: cd %TARGET_DIR% ^&^& .\launcher.bat
echo 🛑 停止命令: cd %TARGET_DIR% ^&^& .\stop.bat
echo 🌐 访问地址: http://localhost:9876
echo.
echo ========================================
pause
