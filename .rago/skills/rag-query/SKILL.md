---
name: rag-query
description: Perform semantic search queries on your ingested documents
version: 1.0.0
category: rag
tags: [search, query, rag]
variables:
  - name: query
    type: string
    required: true
    description: The question or topic to search for
  - name: top_k
    type: number
    default: 5
    description: Number of results to return
  - name: temperature
    type: number
    default: 0.7
    description: LLM temperature for response generation
---

# RAG Query Skill

## Step 1: Define Your Query

Please enter the question or topic you want to search for in your documents.

**Example**: "What are the key findings from the Q3 report?"

```input:query
```

## Step 2: Configure Search Options (Optional)

You can refine your search by adjusting parameters:

- **Top K Results**: `{{top_k}}` (number of documents to retrieve)
- **Temperature**: `{{temperature}}` (creativity level for AI response)

## Step 3: Review Results

The skill will execute the query and display:
1. The AI-generated answer based on retrieved context
2. Source documents with relevance scores
3. Thinking process (if enabled)

---

**Confirmation Required**: true
**Interactive**: true
