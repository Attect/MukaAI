@echo off
REM LM Studio 测试运行脚本

echo 正在启动 LM Studio 测试...
echo.

REM 设置类路径
set CLASSPATH=build\classes\kotlin\jvm\main

REM 添加所有依赖到类路径
for /f "delims=" %%i in ('gradlew.bat -q dependencies --configuration jvmRuntimeClasspath 2^>nul ^| findstr /r "\\.[jJ][aA][rR]"') do (
    set CLASSPATH=!CLASSPATH!;%%i
)

REM 运行测试
java -cp "%CLASSPATH%" com.assistant.test.lmstudio.LMStudioTestKt

echo.
pause
