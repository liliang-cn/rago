# Fake Data Example for RAGO

This directory contains completely fictional data for testing the RAGO system. The documents in `fake_documents.txt` contain made-up information about imaginary technologies, alien species, and fictional concepts.

## Contents

- `fake_documents.txt`: 5 fictional documents about made-up topics

## Usage

You can use these documents to test RAGO's ingestion and querying capabilities:

```bash
# Initialize RAGO (if not already done)
rago init

# Ingest the fake documents
rago ingest --file examples/fake_data/fake_documents.txt

# Query the documents
rago query "What are Zorblaxian crystal mining techniques?"
rago query "How do temporal bakers create yesterday's bread?"
rago query "Explain the quantum linguistics of the Snurflax"
```

These queries will test RAGO's ability to understand and retrieve information from completely fictional content.