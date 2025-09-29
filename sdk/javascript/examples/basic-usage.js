/**
 * Basic RAGO SDK Usage Example
 * 
 * This example demonstrates the core RAG functionality:
 * - Document ingestion
 * - Querying with context retrieval
 * - Analytics
 */

const { createRAGO } = require('@rago/javascript-sdk');

async function basicRAGExample() {
  console.log('🚀 RAGO SDK Basic Usage Example\n');

  // Initialize the SDK
  const rago = createRAGO({
    baseURL: 'http://localhost:7127',
    timeout: 30000,
  });

  try {
    // 1. Health check
    console.log('📡 Checking server health...');
    const isHealthy = await rago.healthCheck();
    console.log(`Server status: ${isHealthy ? '✅ Healthy' : '❌ Unavailable'}\n`);

    if (!isHealthy) {
      console.log('❌ Server is not available. Please start the RAGO server first.');
      return;
    }

    // 2. Get server info
    console.log('ℹ️  Getting server information...');
    const serverInfo = await rago.getServerInfo();
    console.log('Server info:', serverInfo);
    console.log();

    // 3. Ingest sample documents
    console.log('📄 Ingesting sample documents...');
    
    const documents = [
      {
        content: `
          Machine Learning is a subset of artificial intelligence that enables computers to learn and make decisions without being explicitly programmed. 
          It uses algorithms to analyze data, identify patterns, and make predictions or decisions based on the learned patterns.
          
          Key types of machine learning include:
          - Supervised learning: Learning from labeled data
          - Unsupervised learning: Finding patterns in unlabeled data  
          - Reinforcement learning: Learning through interaction and feedback
          
          Popular algorithms include linear regression, decision trees, neural networks, and support vector machines.
        `,
        filename: 'machine-learning-basics.txt',
        metadata: { category: 'AI', topic: 'Machine Learning' }
      },
      {
        content: `
          Natural Language Processing (NLP) is a branch of artificial intelligence that helps computers understand, interpret, and manipulate human language.
          
          NLP combines computational linguistics with statistical, machine learning, and deep learning models to enable computers to process human language in a valuable way.
          
          Common NLP tasks include:
          - Text classification
          - Named entity recognition
          - Sentiment analysis
          - Machine translation
          - Question answering
          - Text summarization
          
          Applications include chatbots, language translation services, sentiment analysis tools, and voice assistants.
        `,
        filename: 'nlp-overview.txt',
        metadata: { category: 'AI', topic: 'NLP' }
      }
    ];

    for (const doc of documents) {
      const result = await rago.rag.ingest(doc);
      console.log(`✅ Ingested "${doc.filename}" - ${result.chunks_created} chunks created`);
    }
    console.log();

    // 4. List documents
    console.log('📋 Listing ingested documents...');
    const docList = await rago.rag.listDocuments();
    console.log(`Found ${docList.length} documents:`);
    docList.forEach(doc => {
      console.log(`  - ${doc.filename} (${doc.chunk_count} chunks)`);
    });
    console.log();

    // 5. Query examples
    console.log('🔍 Running query examples...\n');
    
    const queries = [
      'What is machine learning?',
      'Explain the types of machine learning',
      'What are common NLP tasks?',
      'How does reinforcement learning work?'
    ];

    for (const query of queries) {
      console.log(`Query: "${query}"`);
      console.log('─'.repeat(50));
      
      const result = await rago.rag.query({
        query,
        top_k: 3,
        show_sources: true,
        temperature: 0.7
      });

      console.log(`Answer: ${result.answer}\n`);
      
      if (result.sources && result.sources.length > 0) {
        console.log('📚 Sources:');
        result.sources.forEach((source, i) => {
          console.log(`  ${i + 1}. Score: ${source.score.toFixed(3)} - ${source.content.substring(0, 100)}...`);
        });
      }
      console.log('\n' + '='.repeat(80) + '\n');
    }

    // 6. Search without generation
    console.log('🔎 Testing search-only functionality...');
    const searchResult = await rago.rag.search({
      query: 'neural networks algorithms',
      top_k: 5
    });
    
    console.log(`Found ${searchResult.results.length} relevant chunks:`);
    searchResult.results.forEach((chunk, i) => {
      console.log(`  ${i + 1}. Score: ${chunk.score.toFixed(3)} - ${chunk.content.substring(0, 100)}...`);
    });
    console.log();

    // 7. Analytics
    console.log('📊 Getting usage analytics...');
    const analytics = await rago.analytics.getRAGAnalytics();
    console.log('RAG Analytics:');
    console.log(`  - Total queries: ${analytics.total_queries}`);
    console.log(`  - Success rate: ${(analytics.success_rate * 100).toFixed(1)}%`);
    console.log(`  - Average latency: ${analytics.avg_latency.toFixed(0)}ms`);
    console.log(`  - Average chunks per query: ${analytics.avg_chunks.toFixed(1)}`);
    console.log();

    const usageStats = await rago.analytics.getUsageStats();
    console.log('Usage Statistics:');
    console.log(`  - Total tokens: ${usageStats.total_tokens.toLocaleString()}`);
    console.log(`  - Total cost: $${usageStats.total_cost.toFixed(6)}`);
    console.log(`  - Average latency: ${usageStats.average_latency.toFixed(0)}ms`);
    console.log();

    console.log('✅ Basic RAG example completed successfully!');

  } catch (error) {
    console.error('❌ Error during example execution:', error.message);
    if (error.response) {
      console.error('Response data:', error.response);
    }
  }
}

// Run the example
if (require.main === module) {
  basicRAGExample().catch(console.error);
}

module.exports = { basicRAGExample };