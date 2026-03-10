# Mem0 集成可行性测试

## 测试概述

本测试验证将 mem0 AI 记忆系统集成到项目中的可行性，使用独立可执行文件的方式部署。

## 测试目标

1. ✅ 验证 mem0 可以本地部署
2. ✅ 验证 mem0 支持 FAISS 作为本地向量存储
3. ✅ 验证 mem0 支持通过 Ollama 协议连接本地 LLM（如 LM Studio）
4. ✅ 验证 mem0 可以打包为独立可执行文件
5. ✅ 验证 mem0 REST API 的可用性

## 测试目录结构

```
test-available/mem0-integration/
├── mem0/                          # 克隆的 mem0 官方仓库
└── mem0-server-local/             # 本地化部署版本
    ├── main.py                    # 服务器主程序
    ├── run_server.py              # 启动脚本
    ├── requirements.txt           # Python 依赖
    ├── mem0_server.spec           # PyInstaller 配置
    ├── build.bat                  # Windows 构建脚本
    ├── test_server.py             # API 测试脚本
    ├── .env.example               # 环境变量示例
    └── README.md                  # 使用说明
```

## 技术栈

- **Mem0**: AI 记忆管理框架
- **FastAPI**: REST API 框架
- **FAISS**: Facebook AI 相似性搜索库（向量存储）
- **Ollama Protocol**: LLM 通信协议（兼容 LM Studio）
- **PyInstaller**: Python 应用打包工具
- **UVicorn**: ASGI 服务器

## 部署方案

### 方案 1：独立可执行文件（推荐）

**优点**:
- 无需安装 Python 环境
- 部署简单，直接运行
- 适合生产环境

**构建步骤**:
```bash
cd test-available/mem0-integration/mem0-server-local
build.bat
```

**运行**:
```bash
dist\mem0-server\mem0-server.exe
```

### 方案 2：Python 脚本运行

**优点**:
- 开发调试方便
- 文件体积小

**运行**:
```bash
pip install -r requirements.txt
python run_server.py
```

## API 接口

| 接口 | 方法 | 描述 |
|------|------|------|
| `/memories` | POST | 创建记忆 |
| `/memories` | GET | 获取所有记忆 |
| `/memories/{id}` | GET | 获取特定记忆 |
| `/memories/{id}` | PUT | 更新记忆 |
| `/memories/{id}/history` | GET | 获取记忆历史 |
| `/memories/{id}` | DELETE | 删除记忆 |
| `/memories` | DELETE | 删除所有记忆 |
| `/search` | POST | 搜索记忆 |
| `/reset` | POST | 重置所有记忆 |
| `/configure` | POST | 配置服务器 |

## 配置说明

### 环境变量

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `FAISS_PATH` | FAISS 向量存储路径 | `./data/faiss_memories` |
| `OLLAMA_BASE_URL` | LLM 服务地址 | `http://localhost:1234` |
| `OLLAMA_MODEL` | 模型名称 | `local-model` |
| `EMBEDDER_MODEL` | Embedder 模型 | `text-embedding-3-small` |
| `EMBEDDER_API_KEY` | Embedder API 密钥 | `not-needed` |

### LM Studio 配置

1. 启动 LM Studio
2. 加载模型（如 `qwen3.5-9b-uncensored-hauhaucs-aggressive`）
3. 启动本地服务器（端口 1234）
4. 确保开启 Function Calling 支持

## 使用示例

### 创建记忆

```bash
curl -X POST "http://localhost:8000/memories" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "我喜欢看科幻电影"},
      {"role": "assistant", "content": "好的，我记住了"}
    ],
    "user_id": "user123"
  }'
```

### 搜索记忆

```bash
curl -X POST "http://localhost:8000/search" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "电影偏好",
    "user_id": "user123"
  }'
```

### 获取记忆

```bash
curl "http://localhost:8000/memories?user_id=user123"
```

## 测试结果

### ✅ 可行性验证

1. **Mem0 本地部署**: 成功
   - 使用 FAISS 作为本地向量存储
   - 使用 LM Studio 作为 LLM 和 Embedder
   - 无需依赖云服务

2. **REST API**: 成功
   - 提供完整的 CRUD 接口
   - 支持搜索和过滤
   - 成功测试创建、获取、搜索记忆

3. **独立打包**: 成功
   - 使用 PyInstaller 打包为 exe
   - 文件大小：约 50 MB
   - 无需 Python 环境即可运行

4. **LM Studio 集成**: 成功
   - LLM: qwen3.5-9b-uncensored-hauhaucs-aggressive
   - Embedding: nomic-embed-text-v1.5 (768 维)
   - 支持 Function Calling

## 构建结果

### 可执行文件
- **位置**: `dist/mem0-server.exe`
- **大小**: 约 50 MB
- **类型**: 独立 Windows 可执行文件
- **依赖**: 无需 Python 环境

### 使用方法

1. **复制文件**:
   ```bash
   # 将 dist/mem0-server.exe 复制到目标位置
   ```

2. **创建配置文件** (.env):
   ```bash
   LM_STUDIO_BASE_URL=http://192.168.8.100:11452
   LM_STUDIO_MODEL=qwen3.5-9b-uncensored-hauhaucs-aggressive
   EMBEDDER_MODEL=nomic-embed-text-v1.5
   FAISS_PATH=./data/faiss_memories
   ```

3. **启动服务器**:
   ```bash
   mem0-server.exe
   ```

4. **访问 API**:
   - API 文档：http://localhost:8000/docs
   - 健康检查：http://localhost:8000

## 与 Kotlin 项目集成

### 集成方案

1. **启动 Mem0 Server**:
   - 随项目启动时自动运行
   - 作为独立进程管理

2. **Kotlin 客户端**:
   - 使用 Ktor Client 调用 REST API
   - 封装为 MemoryService 模块

3. **数据流**:
   ```
   Kotlin App <-> Ktor Client <-> Mem0 Server (REST API) <-> LM Studio
   ```

### 下一步

1. ✅ 完成 Mem0 Server 的构建和测试
2. ⏳ 在 Kotlin 项目中实现 REST API 客户端
3. ⏳ 集成到项目启动流程
4. ⏳ 编写集成测试

## 参考文档

- [Mem0 官方文档](https://docs.mem0.ai)
- [Mem0 GitHub](https://github.com/mem0ai/mem0)
- [FAISS 文档](https://github.com/facebookresearch/faiss)
- [LM Studio 文档](https://lmstudio.ai)
- [PyInstaller 文档](https://pyinstaller.org)

## 注意事项

1. **首次运行**: 会创建 FAISS 索引，可能需要几秒钟
2. **模型要求**: LLM 需要支持 Function Calling
3. **端口占用**: 默认使用 8000 端口
4. **数据持久化**: FAISS 数据存储在配置的路径

## 问题与解决方案

### 问题 1: 打包后文件过大

**解决**: 使用 UPX 压缩，或优化 hiddenimports

### 问题 2: 无法连接 LLM

**解决**: 检查 LM Studio 是否启动，端口是否正确

### 问题 3: FAISS 初始化失败

**解决**: 检查 FAISS_PATH 路径是否有写权限

## 结论

✅ **Mem0 可以成功集成为独立可执行文件**

通过本可行性测试，验证了以下关键点：

1. Mem0 可以本地部署，无需依赖云服务
2. 使用 FAISS 作为向量存储，完全本地化
3. 通过 Ollama 协议兼容 LM Studio
4. 可以打包为独立 exe，部署简单
5. REST API 接口完整，易于集成

**推荐方案**: 使用 PyInstaller 打包为独立可执行文件，随项目启动时运行。
