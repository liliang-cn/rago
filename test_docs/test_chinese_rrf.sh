#!/bin/bash

# 中文查询RRF优化测试脚本
echo "=== 中文查询RRF分数优化测试 ==="
echo "测试知识库：音书酒吧.md"
echo ""

# 确保编译最新版本
echo "编译最新版本..."
go build -o rago main.go
echo ""

# 测试1：询问酒吧地址
echo "🏠 测试1: 酒吧地址查询"
echo "查询: '音书酒吧在哪里？地址是什么？'"
echo "---"
./rago query "音书酒吧在哪里？地址是什么？" --show-sources --top-k 3
echo ""
echo "=================="

# 测试2：询问营业时间
echo "🕐 测试2: 营业时间查询"
echo "查询: '音书酒吧什么时候营业？营业时间是？'"
echo "---"
./rago query "音书酒吧什么时候营业？营业时间是？" --show-sources --top-k 3
echo ""
echo "=================="

# 测试3：询问特色饮品
echo "🍸 测试3: 特色饮品查询"
echo "查询: '音书酒吧有什么招牌饮品？'"
echo "---"
./rago query "音书酒吧有什么招牌饮品？" --show-sources --top-k 3
echo ""
echo "=================="

# 测试4：询问特色活动
echo "📚 测试4: 特色活动查询"
echo "查询: '音书酒吧有什么特别活动吗？'"
echo "---"
./rago query "音书酒吧有什么特别活动吗？" --show-sources --top-k 3
echo ""
echo "=================="

# 测试5：模糊查询
echo "🎵 测试5: 模糊特征查询"
echo "查询: '有音乐和书的地方'"
echo "---"
./rago query "有音乐和书的地方" --show-sources --top-k 3
echo ""
echo "=================="

echo "📊 测试总结："
echo "- 观察RRF分数是否在合理范围（0.05-0.18）"
echo "- 检查中文查询的相关性匹配"
echo "- 验证高质量匹配是否能正确通过阈值"
