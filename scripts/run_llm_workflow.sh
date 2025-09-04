#!/bin/bash

echo "🤖 Running LLM-Powered Workflow Example"
echo "========================================"
echo ""
echo "This workflow will use the LLM to analyze data!"
echo ""

# First, let's create a simple workflow that uses LLM
cat > testdata/llm_test_workflow.json << 'EOF'
{
  "steps": [
    {
      "id": "create_sample_data",
      "name": "Create Sample Data",
      "type": "tool",
      "tool": "filesystem",
      "inputs": {
        "action": "write",
        "path": "./sample_data.txt",
        "content": "Product: iPhone 15 Pro\nOriginal Price: $999\nCurrent Price: $899\nDiscount: 10%\nStock: Limited\nReviews: 4.5/5 stars (2,341 reviews)"
      }
    },
    {
      "id": "read_data",
      "name": "Read Product Data",
      "type": "tool",
      "tool": "filesystem",
      "inputs": {
        "action": "read",
        "path": "./sample_data.txt"
      },
      "outputs": {
        "content": "product_data"
      }
    },
    {
      "id": "analyze_with_llm",
      "name": "LLM Analysis of Product",
      "type": "tool",
      "tool": "sequential-thinking",
      "inputs": {
        "task": "Analyze this product data and provide: 1) Whether this is a good deal, 2) Key selling points, 3) Potential concerns, 4) Buy recommendation (YES/NO/WAIT)",
        "data": "{{product_data}}"
      },
      "outputs": {
        "result": "llm_analysis"
      }
    },
    {
      "id": "generate_summary",
      "name": "Generate Purchase Summary",
      "type": "tool",
      "tool": "sequential-thinking",
      "inputs": {
        "task": "Create a concise 2-sentence summary for a buyer based on this analysis",
        "analysis": "{{llm_analysis}}"
      },
      "outputs": {
        "result": "buyer_summary"
      }
    },
    {
      "id": "save_analysis",
      "name": "Save LLM Analysis",
      "type": "tool",
      "tool": "filesystem",
      "inputs": {
        "action": "write",
        "path": "./product_analysis.md",
        "content": "# Product Analysis (LLM Generated)\n\n## Product Data\n{{product_data}}\n\n## AI Analysis\n{{llm_analysis}}\n\n## Quick Summary\n{{buyer_summary}}"
      }
    }
  ]
}
EOF

echo "✅ Created workflow: testdata/llm_test_workflow.json"
echo ""
echo "This workflow will:"
echo "  1. Create sample product data"
echo "  2. Read the data"
echo "  3. 🧠 Call LLM to analyze if it's a good deal"
echo "  4. 🧠 Call LLM to generate a buyer summary"
echo "  5. Save the analysis"
echo ""
echo "To run this workflow:"
echo ""
echo "1. Create an agent with this workflow:"
echo "   go run ./cmd/rago agent create --name \"Product Analyzer\" --type workflow --workflow-file testdata/llm_test_workflow.json"
echo ""
echo "2. Execute the agent (this will call the LLM):"
echo "   go run ./cmd/rago agent execute [agent-id]"
echo ""
echo "The 'sequential-thinking' tool steps will make actual LLM API calls!"
echo "You'll see the LLM analyzing the product data and generating insights."