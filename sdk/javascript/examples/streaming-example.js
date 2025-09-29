/**
 * RAGO SDK Streaming Example
 * 
 * This example demonstrates streaming functionality:
 * - Real-time RAG query streaming
 * - LLM generation streaming  
 * - Chat streaming
 */

const { createRAGO } = require('@rago/javascript-sdk');

async function streamingExample() {
  console.log('🌊 RAGO SDK Streaming Example\n');

  const rago = createRAGO({
    baseURL: 'http://localhost:7127',
    timeout: 60000, // Longer timeout for streaming
  });

  try {
    // Health check
    const isHealthy = await rago.healthCheck();
    if (!isHealthy) {
      console.log('❌ Server is not available. Please start the RAGO server first.');
      return;
    }

    console.log('✅ Server is healthy\n');

    // 1. RAG Query Streaming
    console.log('🔍 RAG Query Streaming Example');
    console.log('─'.repeat(50));
    console.log('Query: "Explain machine learning in detail"\n');

    let ragStreamContent = '';
    
    await rago.rag.queryStream({
      query: 'Explain machine learning in detail with examples',
      top_k: 3,
      temperature: 0.7,
      show_sources: true
    }, {
      onData: (chunk) => {
        process.stdout.write(chunk);
        ragStreamContent += chunk;
      },
      onComplete: () => {
        console.log('\n\n✅ RAG streaming completed\n');
      },
      onError: (error) => {
        console.error('\n❌ RAG streaming error:', error.message);
      }
    });

    // 2. LLM Generation Streaming
    console.log('🤖 LLM Generation Streaming Example');
    console.log('─'.repeat(50));
    console.log('Prompt: "Write a creative story about AI"\n');

    let llmStreamContent = '';

    await rago.llm.generateStream({
      prompt: 'Write a creative short story about an AI assistant that learns to understand human emotions. Make it engaging and thoughtful.',
      temperature: 0.8,
      max_tokens: 500
    }, {
      onData: (chunk) => {
        process.stdout.write(chunk);
        llmStreamContent += chunk;
      },
      onComplete: () => {
        console.log('\n\n✅ LLM generation streaming completed\n');
      },
      onError: (error) => {
        console.error('\n❌ LLM streaming error:', error.message);
      }
    });

    // 3. Chat Streaming
    console.log('💬 Chat Streaming Example');
    console.log('─'.repeat(50));

    const chatMessages = [
      { role: 'user', content: 'Hello! Can you help me understand the difference between AI and machine learning?' },
      { role: 'assistant', content: 'Hello! I\'d be happy to help explain the difference between AI and machine learning.' },
      { role: 'user', content: 'Please provide a detailed explanation with practical examples' }
    ];

    console.log('Chat History:');
    chatMessages.forEach((msg, i) => {
      console.log(`${msg.role}: ${msg.content}`);
    });
    console.log('\nAssistant response (streaming):\n');

    let chatStreamContent = '';

    await rago.llm.chatStream({
      messages: chatMessages,
      temperature: 0.7,
      max_tokens: 400
    }, {
      onData: (chunk) => {
        process.stdout.write(chunk);
        chatStreamContent += chunk;
      },
      onComplete: () => {
        console.log('\n\n✅ Chat streaming completed\n');
      },
      onError: (error) => {
        console.error('\n❌ Chat streaming error:', error.message);
      }
    });

    // 4. Performance comparison
    console.log('📊 Streaming vs Non-streaming Performance Comparison');
    console.log('─'.repeat(50));

    // Non-streaming query
    console.log('Testing non-streaming query...');
    const startTime = Date.now();
    const nonStreamResult = await rago.rag.query({
      query: 'What are the benefits of machine learning?',
      top_k: 3
    });
    const nonStreamTime = Date.now() - startTime;
    console.log(`Non-streaming: ${nonStreamTime}ms (${nonStreamResult.answer.length} chars)`);

    // Streaming query (simulated timing)
    console.log('Testing streaming query...');
    const streamStartTime = Date.now();
    let streamCharCount = 0;
    let firstChunkTime = null;

    await rago.rag.queryStream({
      query: 'What are the benefits of machine learning?',
      top_k: 3
    }, {
      onData: (chunk) => {
        if (firstChunkTime === null) {
          firstChunkTime = Date.now() - streamStartTime;
        }
        streamCharCount += chunk.length;
      },
      onComplete: () => {
        const totalStreamTime = Date.now() - streamStartTime;
        console.log(`Streaming: ${totalStreamTime}ms total (${streamCharCount} chars)`);
        console.log(`Time to first chunk: ${firstChunkTime}ms`);
        console.log(`Streaming advantage: ${((nonStreamTime - firstChunkTime) / nonStreamTime * 100).toFixed(1)}% faster perceived response`);
      },
      onError: (error) => {
        console.error('Streaming test error:', error.message);
      }
    });

    console.log('\n✅ Streaming examples completed successfully!');

  } catch (error) {
    console.error('❌ Error during streaming example:', error.message);
    if (error.response) {
      console.error('Response data:', error.response);
    }
  }
}

// Helper function to simulate typing effect
function typeText(text, delay = 50) {
  return new Promise((resolve) => {
    let i = 0;
    const interval = setInterval(() => {
      process.stdout.write(text[i]);
      i++;
      if (i >= text.length) {
        clearInterval(interval);
        resolve();
      }
    }, delay);
  });
}

// Run the example
if (require.main === module) {
  streamingExample().catch(console.error);
}

module.exports = { streamingExample };