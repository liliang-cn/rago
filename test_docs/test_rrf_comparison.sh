#!/bin/bash

# RRF优化前后对比测试
echo "=== RRF优化前后对比测试 ==="
echo ""

# 测试同一个中文查询在不同k值下的表现
echo "测试查询: '音书酒吧有什么招牌饮品？'"
echo ""

echo "📉 【优化前】k=60, threshold=0.3:"
echo "预期结果: 分数过低，无相关内容返回"
RAGO_RRF_K=60 RAGO_RRF_RELEVANCE_THRESHOLD=0.3 ./rago query "音书酒吧有什么招牌饮品？" --show-sources --top-k 3 2>/dev/null | grep -E "(Score|无|找不到|没有相关)"
echo ""

echo "📈 【优化后】k=10, threshold=0.05:"
echo "预期结果: 分数合理，正确返回饮品信息"
RAGO_RRF_K=10 RAGO_RRF_RELEVANCE_THRESHOLD=0.05 ./rago query "音书酒吧有什么招牌饮品？" --show-sources --top-k 3 2>/dev/null | grep -E "(Score|书香马天尼|音符威士忌)"
echo ""

echo "🚀 【激进优化】k=5, threshold=0.1:"
echo "预期结果: 分数更高，更强的相关性"
RAGO_RRF_K=5 RAGO_RRF_RELEVANCE_THRESHOLD=0.1 ./rago query "音书酒吧有什么招牌饮品？" --show-sources --top-k 3 2>/dev/null | grep -E "(Score|书香马天尼|音符威士忌)"
echo ""

echo "📊 分数理论值对比:"
echo "k=60: 最高分 1/(60+1) = 0.0164 (1.64%)"
echo "k=10: 最高分 1/(10+1) = 0.0909 (9.09%)"
echo "k=5:  最高分 1/(5+1)  = 0.1667 (16.67%)"
