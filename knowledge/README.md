# Knowledge Test Data

This folder contains sample documents for testing RAGO's RAG (Retrieval-Augmented Generation) functionality.

## Contents

- `coffee_shop.md` - Coffee shop description (English)
- `tech_hub.md` - Technology hub information
- `sound_and_book_bar.md` - Sound and book bar description
- `音书酒吧.md` - Sound and book bar (Chinese)
- `chuan_yue_lou_en.pdf` - Restaurant information (PDF format)

## Purpose

These files are used for:
1. Testing document ingestion in different formats (Markdown, PDF)
2. Testing multilingual support (English and Chinese)
3. Demonstrating RAG capabilities with real-world content
4. Vector search and semantic retrieval testing

## Usage

```bash
# Ingest documents using RAGO CLI
rago ingest knowledge/coffee_shop.md

# Or programmatically
ragoClient.RAG().IngestDocument(ctx, core.IngestRequest{
    DocumentID: "coffee-shop",
    Content:    content,
    Metadata:   metadata,
})
```

## Note

This is test data only. Replace with your own documents for production use.