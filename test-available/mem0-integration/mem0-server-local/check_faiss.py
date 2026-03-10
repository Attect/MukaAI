"""
检查 FAISS 索引维度
"""
import faiss
import os

index_path = "./data/faiss_memories/mem0_memories.faiss"

if os.path.exists(index_path):
    index = faiss.read_index(index_path)
    print(f"✓ FAISS 索引已加载")
    print(f"索引维度：{index.d}")
    print(f"向量数量：{index.ntotal}")
else:
    print(f"✗ FAISS 索引不存在：{index_path}")
