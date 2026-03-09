# 可行性测试 01: Ktor 服务端 + 多平台客户端架构验证

## 测试目的
验证使用 Ktor 构建原生服务端，并通过 KMP 实现 desktop/android/ios/web(wasm) 多平台客户端的技术可行性。

## 测试内容
1. 创建 Ktor 服务端基础结构
2. 实现简单的 RPC 接口
3. 创建 KMP 共享模块
4. 验证多平台客户端调用服务端

## 技术栈
- Kotlin 2.3.10+
- Ktor 3.4.1+
- Kotlinx Serialization
- Kotlin Coroutines

## 构建和运行
```bash
# 构建服务端
./gradlew :test-available:01-ktor-multiplatform:server:run

# 构建客户端 (根据平台选择)
./gradlew :test-available:01-ktor-multiplatform:client-desktop:run
```

## 预期结果
- 服务端成功启动并监听端口
- 多平台客户端能够成功连接并调用 RPC 接口
- 数据模型在服务端和客户端之间正确序列化/反序列化

## 参考
- [Ktor Server Documentation](../../references/ktor/ktor-server.md)
- [Ktor Client Documentation](../../references/ktor/ktor-client.md)
- [Ktor Integration Guide](../../references/ktor/ktor-integration.md)
