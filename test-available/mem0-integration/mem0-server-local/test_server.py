#!/usr/bin/env python
"""
Mem0 Server 测试脚本
用于验证 mem0 server 的 API 功能
"""
import requests
import json
import time

BASE_URL = "http://localhost:8000"

def print_response(title, response):
    """打印响应信息"""
    print(f"\n{'='*60}")
    print(f"{title}")
    print(f"{'='*60}")
    print(f"状态码：{response.status_code}")
    print(f"响应内容：{json.dumps(response.json(), indent=2, ensure_ascii=False)}")
    print()

def test_health():
    """测试健康检查"""
    print(">>> 测试健康检查...")
    response = requests.get(f"{BASE_URL}/")
    if response.status_code == 200:
        print("[✓] 健康检查通过")
        return True
    else:
        print(f"[✗] 健康检查失败：{response.status_code}")
        return False

def test_create_memory():
    """测试创建记忆"""
    print(">>> 测试创建记忆...")
    data = {
        "messages": [
            {"role": "user", "content": "我喜欢看科幻电影，特别是星际穿越和银翼杀手"},
            {"role": "assistant", "content": "好的，我记住了您喜欢科幻电影，尤其是星际穿越和银翼杀手"}
        ],
        "user_id": "test_user_001"
    }
    response = requests.post(f"{BASE_URL}/memories", json=data)
    print_response("创建记忆", response)
    return response.status_code == 200

def test_get_memories():
    """测试获取记忆"""
    print(">>> 测试获取记忆...")
    params = {"user_id": "test_user_001"}
    response = requests.get(f"{BASE_URL}/memories", params=params)
    print_response("获取记忆", response)
    return response.status_code == 200

def test_search_memory():
    """测试搜索记忆"""
    print(">>> 测试搜索记忆...")
    data = {
        "query": "电影偏好",
        "user_id": "test_user_001"
    }
    response = requests.post(f"{BASE_URL}/search", json=data)
    print_response("搜索记忆", response)
    return response.status_code == 200

def test_delete_memory(memory_id=None):
    """测试删除记忆"""
    if not memory_id:
        print(">>> 跳过删除测试（无 memory_id）")
        return True
    
    print(f">>> 测试删除记忆 {memory_id}...")
    response = requests.delete(f"{BASE_URL}/memories/{memory_id}")
    print_response("删除记忆", response)
    return response.status_code == 200

def test_delete_all_memories():
    """测试删除所有记忆"""
    print(">>> 测试删除所有记忆...")
    params = {"user_id": "test_user_001"}
    response = requests.delete(f"{BASE_URL}/memories", params=params)
    print_response("删除所有记忆", response)
    return response.status_code == 200

def run_all_tests():
    """运行所有测试"""
    print("="*60)
    print("Mem0 Server 测试套件")
    print("="*60)
    print()
    
    # 等待服务器启动
    print("等待服务器启动...")
    for i in range(10):
        try:
            if test_health():
                break
        except:
            pass
        time.sleep(1)
    else:
        print("[✗] 服务器未响应，请检查服务器是否已启动")
        return
    
    # 运行测试
    results = []
    
    results.append(("创建记忆", test_create_memory()))
    results.append(("获取记忆", test_get_memories()))
    results.append(("搜索记忆", test_search_memory()))
    # results.append(("删除记忆", test_delete_memory()))
    results.append(("删除所有记忆", test_delete_all_memories()))
    
    # 输出测试报告
    print("\n" + "="*60)
    print("测试报告")
    print("="*60)
    passed = sum(1 for _, result in results if result)
    total = len(results)
    
    for name, result in results:
        status = "✓" if result else "✗"
        print(f"{status} {name}")
    
    print(f"\n总计：{passed}/{total} 通过")
    print("="*60)

if __name__ == "__main__":
    run_all_tests()
