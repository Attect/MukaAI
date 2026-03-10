@echo off
REM Mem0 Server 构建脚本 (Windows)
REM 此脚本将 mem0 server 打包为独立的可执行文件

echo ============================================================
echo Mem0 Server - 构建脚本
echo ============================================================
echo.

REM 检查 Python 是否安装
python --version >nul 2>&1
if errorlevel 1 (
    echo [错误] 未找到 Python，请先安装 Python 3.9+
    pause
    exit /b 1
)

echo [1/5] 检查 Python 环境...
python --version

echo.
echo [2/5] 创建虚拟环境...
if exist venv (
    echo [提示] 虚拟环境已存在，将重新创建
    rmdir /s /q venv
)
python -m venv venv
if errorlevel 1 (
    echo [错误] 创建虚拟环境失败
    pause
    exit /b 1
)

echo.
echo [3/5] 激活虚拟环境并安装依赖...
call venv\Scripts\activate.bat
pip install --upgrade pip >nul 2>&1
pip install -r requirements.txt
if errorlevel 1 (
    echo [错误] 安装依赖失败
    pause
    exit /b 1
)

echo.
echo [4/5] 清理旧的构建文件...
if exist build rmdir /s /q build
if exist dist rmdir /s /q dist
if exist *.log del *.log

echo.
echo [5/5] 开始构建可执行文件...
echo [提示] 这可能需要几分钟...
pyinstaller --clean --log-level INFO mem0_server.spec
if errorlevel 1 (
    echo [错误] 构建失败
    pause
    exit /b 1
)

echo.
echo ============================================================
echo 构建完成!
echo ============================================================
echo.
echo 可执行文件位置：
echo   - 目录模式：dist\mem0-server\mem0-server.exe
echo   - 单文件模式：dist\mem0-server.exe
echo.
echo 使用方法:
echo   1. 复制 dist\mem0-server 目录到目标位置
echo   2. 创建 .env 文件配置环境变量
echo   3. 运行 mem0-server.exe 启动服务器
echo.
echo ============================================================

pause
