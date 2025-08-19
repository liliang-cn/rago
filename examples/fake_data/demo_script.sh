#!/bin/bash

# RAGO Demo Script
# This script demonstrates the capabilities of RAGO with fake data

echo "=== RAGO Demo with Fake Data ==="
echo ""

# Check if rago is installed
if ! command -v rago &> /dev/null
then
    echo "rago could not be found. Please install it first."
    exit 1
fi

# Create a temporary directory for this demo
DEMO_DIR="./rago_demo"
mkdir -p $DEMO_DIR
cd $DEMO_DIR

echo "1. Initializing RAGO..."
echo ""

# Initialize RAGO (if needed)
# rago init

echo "2. Creating sample documents..."
echo ""

# Copy the documents file to the demo directory
cp ../documents.txt ./sample_documents.txt

echo "3. Ingesting documents into RAGO..."
echo ""
rago ingest sample_documents.txt --chunk-size 200 --overlap 30

echo ""
echo "4. Listing ingested documents..."
echo ""
rago list

echo ""
echo "5. Performing queries..."
echo ""

echo "Query 1: What is quantum computing?"
echo ""
rago query "What is quantum computing?"

echo ""
echo "Query 2: History of artificial intelligence"
echo ""
rago query "History of artificial intelligence"

echo ""
echo "Query 3: How does blockchain work?"
echo ""
rago query "How does blockchain work?"

echo ""
echo "Query 4: Types of renewable energy"
echo ""
rago query "Types of renewable energy"

echo ""
echo "Query 5: Machine learning basics"
echo ""
rago query "Machine learning basics"

echo ""
echo "6. Exporting the knowledge base..."
echo ""
rago export knowledge_base.json

echo ""
echo "Demo completed! Check the knowledge_base.json file for exported data."
echo ""

# Clean up
cd ..
# rm -rf $DEMO_DIR

echo "=== End of Demo ==="