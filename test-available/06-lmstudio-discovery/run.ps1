# LM Studio 测试运行脚本
Write-Host "正在启动 LM Studio 测试..." -ForegroundColor Green
Write-Host ""

# 获取依赖类路径
$dependencies = (.\gradlew.bat -q dependencies --configuration jvmRuntimeClasspath 2>$null | Select-String -Pattern "\.jar$" | ForEach-Object { $_.Line.Trim() })

# 构建类路径
$classpath = "build/classes/kotlin/jvm/main"
foreach ($dep in $dependencies) {
    if ($dep -match "\\") {
        $classpath += ";$dep"
    }
}

# 运行测试
Write-Host "运行测试..." -ForegroundColor Yellow
java -cp "$classpath" com.assistant.test.lmstudio.LMStudioTestKt

Write-Host ""
Write-Host "测试完成!" -ForegroundColor Green
