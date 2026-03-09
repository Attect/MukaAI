@echo off
REM LM Studio 测试运行脚本
echo 正在启动 LM Studio 测试...
echo.

REM 收集依赖
call gradlew.bat copyDependencies --no-daemon 2>nul

REM 运行测试
java -cp "build/classes/kotlin/jvm/main;libs/*" com.assistant.test.lmstudio.LMStudioTestKt

echo.
pause
