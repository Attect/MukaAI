"""
Mem0 Server 启动脚本
此脚本用于启动 mem0 REST API 服务器
"""
import uvicorn

if __name__ == "__main__":
    print("=" * 60)
    print("Mem0 REST API Server (Local)")
    print("=" * 60)
    print("启动服务器...")
    print("API 文档：http://localhost:8000/docs")
    print("健康检查：http://localhost:8000")
    print("=" * 60)
    
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8000,
        reload=False,
        log_level="info"
    )
