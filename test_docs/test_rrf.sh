#!/bin/bash

# RRF优化测试脚本
# 测试不同k值对分数的影响

echo "=== RRF分数优化测试 ==="
echo "测试查询: 'What are the workspace options?'"
echo ""

# 测试原始k=60的情况（通过环境变量覆盖）
echo "1. 测试 k=60 (原始值):"
RAGO_RRF_K=60 RAGO_RRF_RELEVANCE_THRESHOLD=0.016 ./rago query "What are the workspace options?" --show-sources | grep -E "(Score|相关性|找到|chunks)"

echo ""

# 测试新的k=10
echo "2. 测试 k=10 (优化值):"
RAGO_RRF_K=10 RAGO_RRF_RELEVANCE_THRESHOLD=0.05 ./rago query "What are the workspace options?" --show-sources | grep -E "(Score|相关性|找到|chunks)"

echo ""

# 测试k=5（更激进的优化）
echo "3. 测试 k=5 (更激进优化):"
RAGO_RRF_K=5 RAGO_RRF_RELEVANCE_THRESHOLD=0.1 ./rago query "What are the workspace options?" --show-sources | grep -E "(Score|相关性|找到|chunks)"

echo ""
echo "=== 分析 ==="
echo "k值越小，RRF分数越高："
echo "- k=60: 最高分约 1/(60+1) = 0.016 (1.6%)"
echo "- k=10: 最高分约 1/(10+1) = 0.091 (9.1%)"
echo "- k=5:  最高分约 1/(5+1)  = 0.167 (16.7%)"
