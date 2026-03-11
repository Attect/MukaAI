# Compose Material 3 主题切换 + multiplatform-settings 可行性测试

## 测试目标

验证以下技术在 Kotlin Multiplatform 项目中的可行性：

1. **Compose Material 3** 在 Desktop 和 Web (Wasm) 平台的主题切换功能
2. **multiplatform-settings** 在各平台的兼容性

## 测试内容

### 1. 主题切换功能
- [ ] 亮色/暗色主题切换
- [ ] 跟随系统主题自动切换
- [ ] 主题状态持久化
- [ ] 主题切换动画效果

### 2. multiplatform-settings
- [ ] Desktop (JVM) 平台设置读写
- [ ] Web (Wasm) 平台设置读写
- [ ] 跨平台设置同步
- [ ] 设置变更监听

## 技术栈

- Kotlin Multiplatform
- Compose Multiplatform (Desktop + Web)
- Compose Material 3
- multiplatform-settings

## 构建和运行

### Desktop
```bash
./gradlew :composeApp:run
```

### Web
```bash
./gradlew :composeApp:wasmJsBrowserRun
```

## 测试结果

详见 [verification-report.md](verification-report.md)
