"""
测试 LM Studio Embedding
"""
import requests
import json

url = "http://192.168.8.100:11452/v1/embeddings"
headers = {"Content-Type": "application/json"}
data = {
    "model": "nomic-embed-text-v1.5",
    "input": "Hello, world!"
}

response = requests.post(url, json=data, headers=headers)

if response.status_code == 200:
    result = response.json()
    embedding = result["data"][0]["embedding"]
    print(f"✓ Embedding 成功!")
    print(f"向量维度：{len(embedding)}")
    print(f"前 10 个值：{embedding[:10]}")
else:
    print(f"✗ 失败：{response.status_code}")
    print(response.text)
