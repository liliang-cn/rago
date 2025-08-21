package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/liliang-cn/rago/internal/config"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/embedder"
	"github.com/liliang-cn/rago/internal/llm"
	"github.com/liliang-cn/rago/internal/store"
	"github.com/liliang-cn/rago/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMIntegration(t *testing.T) {
	// Skip if running in CI or no Ollama available
	if testing.Short() {
		t.Skip("Skipping LLM integration test in short mode")
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "rago-llm-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test configuration
	cfg := &config.Config{
		Ollama: config.OllamaConfig{
			BaseURL:        "http://localhost:11434",
			EmbeddingModel: "nomic-embed-text",
			LLMModel:       "qwen3",
		},
		Sqvect: config.SqvectConfig{
			DBPath: filepath.Join(tempDir, "llm_test.db"),
		},
	}

	// Initialize components
	embedderClient, err := embedder.NewOllamaService(cfg.Ollama.BaseURL, cfg.Ollama.EmbeddingModel)
	require.NoError(t, err)

	llmClient, err := llm.NewOllamaService(cfg.Ollama.BaseURL, cfg.Ollama.LLMModel)
	require.NoError(t, err)

	vectorStore, err := store.NewSQLiteStore(cfg.Sqvect.DBPath, 768, 10, 100)
	require.NoError(t, err)
	defer vectorStore.Close()

	docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

	ctx := context.Background()

	// Fictional but specific knowledge base documents - like 音书酒吧
	knowledgeDocs := []domain.Document{
		{
			ID:      "starlight-tech-company",
			Path:    "company/starlight-tech.md",
			Content: "StarLight Tech 是一家成立于2019年的虚拟现实技术公司，总部位于深圳南山区科技园北区A4栋15楼。公司CEO是张明轩，CTO是李雨婷。主要产品包括VR教育平台'学海无涯'和企业培训系统'职场先锋'。公司员工编号以SLT开头，如SLT-2024-001。客服热线：0755-8899-7766，企业邮箱：contact@starlight-tech.cn。公司吉祥物是一只蓝色的星光小熊，名叫'启明'。",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"type":     "company_info",
				"industry": "technology",
				"location": "shenzhen",
				"founded":  "2019",
			},
		},
		{
			ID:      "moonbeam-restaurant",
			Path:    "business/moonbeam-restaurant.md",
			Content: "月光餐厅是位于北京朝阳区工体北路328号的高端法式料理餐厅。主厨马克·杜邦（Marc Dupont）来自法国里昂，招牌菜是'月光松露牛排'，售价688元。餐厅电话：010-6588-9900，营业时间：11:30-14:00, 17:30-22:00。包厢最低消费：2888元。会员卡分为银月卡（充值1万）、金月卡（充值3万）和钻月卡（充值10万）。餐厅养了一只法国斗牛犬叫'奶油'，是餐厅吉祥物。",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"type":     "restaurant",
				"cuisine":  "french",
				"location": "beijing",
				"price":    "high_end",
			},
		},
		{
			ID:      "dr-chen-profile",
			Path:    "staff/dr-chen-weihua.md",
			Content: "陈维华博士是虚构的人工智能研究专家，工号AI-RESEARCHER-0089，在北京理工大学计算机学院任教授。他发明了'智慧算法优化器'理论，发表论文156篇。手机号：138-0108-6789，办公室：信息楼A508，每周三下午2-5点为学生答疑时间。他的宠物金毛犬叫'算法'，经常带到实验室。陈教授最喜欢的咖啡是蓝山咖啡，办公桌上总放着一个红色的'福'字马克杯。",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"type":           "person",
				"profession":     "professor",
				"department":     "computer_science",
				"specialization": "ai_research",
			},
		},
		{
			ID:      "rainbow-pharmacy",
			Path:    "pharmacy/rainbow-health.md",
			Content: "彩虹健康药房是虚构的连锁药房，总店位于上海徐汇区淮海中路1567号。店长王小华，员工工号以RH开头。药房电话：021-5432-1098，24小时服务热线：400-888-1234。特色服务包括免费血压测量、慢病管理咨询。会员积分兑换：100积分=10元现金。药房吉祥物是彩虹小象'健健'。常备药品包括：板蓝根颗粒（RH-BLG-001）、阿莫西林胶囊（RH-AMX-002）。药房与上海第一人民医院有合作关系。",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"type":     "pharmacy",
				"location": "shanghai",
				"service":  "24hour",
				"chain":    "true",
			},
		},
		{
			ID:      "cloudsky-hotel",
			Path:    "hotel/cloudsky-resort.md",
			Content: "云天度假酒店位于青岛崂山区仙霞路888号，是五星级海景酒店。总经理刘海燕，前台经理张小雨。酒店电话：0532-8877-6655，客房数量：288间。标准海景房价格：1288元/晚，豪华套房：3888元/晚。酒店设施包括无边际泳池、SPA中心'云端轻语'、中餐厅'海韵轩'。酒店吉祥物是海豚'云朵'。会员等级：铜云卡、银云卡、金云卡、钻云卡。酒店提供免费接机服务（提前24小时预约）。",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"type":     "hotel",
				"location": "qingdao",
				"rating":   "5star",
				"view":     "ocean",
			},
		},
	}

	t.Run("Knowledge Base Setup", func(t *testing.T) {
		// Store all knowledge documents
		for _, doc := range knowledgeDocs {
			// Store document
			err := docStore.Store(ctx, doc)
			require.NoError(t, err, "Failed to store document %s", doc.ID)

			// Create embedding and store chunk
			vector, err := embedderClient.Embed(ctx, doc.Content)
			require.NoError(t, err, "Failed to embed content for %s", doc.ID)

			chunk := domain.Chunk{
				ID:         doc.ID + "_main",
				DocumentID: doc.ID,
				Content:    doc.Content,
				Vector:     vector,
				Metadata:   doc.Metadata,
			}

			err = vectorStore.Store(ctx, []domain.Chunk{chunk})
			require.NoError(t, err, "Failed to store chunk for %s", doc.ID)
		}

		// Verify documents are stored
		docs, err := docStore.List(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(docs), 4, "Should have at least 4 documents")
	})

	t.Run("RAG Question Answering", func(t *testing.T) {
		// Test questions with our fictional knowledge base
		testCases := []struct {
			question       string
			expectedFilter map[string]interface{}
			description    string
		}{
			{
				question:       "StarLight Tech公司的CEO是谁？客服电话是多少？",
				expectedFilter: map[string]interface{}{"type": "company_info"},
				description:    "StarLight Tech company info",
			},
			{
				question:       "月光餐厅的招牌菜是什么？主厨叫什么名字？",
				expectedFilter: map[string]interface{}{"type": "restaurant"},
				description:    "Moonbeam restaurant specialties",
			},
			{
				question:       "陈维华博士的办公室在哪里？他的宠物叫什么？",
				expectedFilter: map[string]interface{}{"type": "person"},
				description:    "Dr. Chen personal details",
			},
			{
				question:       "彩虹健康药房的24小时服务热线是多少？吉祥物叫什么？",
				expectedFilter: map[string]interface{}{"type": "pharmacy"},
				description:    "Rainbow pharmacy contact info",
			},
			{
				question:       "云天度假酒店标准海景房多少钱一晚？SPA中心叫什么名字？",
				expectedFilter: map[string]interface{}{"type": "hotel"},
				description:    "CloudSky hotel pricing and facilities",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				// Step 1: Create query embedding
				queryVector, err := embedderClient.Embed(ctx, tc.question)
				require.NoError(t, err)

				// Step 2: Retrieve relevant chunks with filter
				chunks, err := vectorStore.SearchWithFilters(ctx, queryVector, 3, tc.expectedFilter)
				require.NoError(t, err)
				assert.Greater(t, len(chunks), 0, "Should find relevant chunks")

				// Step 3: Build context from chunks
				var contexts []string
				for _, chunk := range chunks {
					contexts = append(contexts, chunk.Content)
				}
				context := strings.Join(contexts, "\n\n")

				// Step 4: Generate answer using LLM
				prompt := fmt.Sprintf(`基于以下上下文信息回答问题，请准确引用具体的信息。直接回答，不要包含思考过程。

上下文：
%s

问题：%s

请直接回答：`, context, tc.question)

				opts := &domain.GenerationOptions{
					Temperature: 0.1, // Low temperature for precise answers
					MaxTokens:   150,
					Think:       func() *bool { b := false; return &b }(),
				}

				answer, err := llmClient.Generate(ctx, prompt, opts)
				require.NoError(t, err)
				assert.NotEmpty(t, answer, "Should generate an answer")

				fmt.Printf("Q: %s\n", tc.question)
				fmt.Printf("A: %s\n", strings.TrimSpace(answer))
				fmt.Printf("---\n")

				// Basic verification that we got a non-empty answer
				assert.NotEmpty(t, answer, "Should generate a non-empty answer")
			})
		}
	})

	t.Run("Metadata Extraction with LLM", func(t *testing.T) {
		// Test document for metadata extraction - fictional business info
		testDoc := `
		星辰咖啡馆是一家新开业的精品咖啡店，位于成都锦江区春熙路步行街125号。
		店主李晓雯拥有10年咖啡烘焙经验，主推手冲咖啡和意式浓缩。店内有20个座位，
		提供免费WiFi，营业时间8:00-22:00。招牌饮品包括'晨光拿铁'和'星夜摩卡'。
		目标客群是年轻白领和咖啡爱好者。联系电话：028-8765-4321。
		`

		// Use LLM to extract metadata from fictional content
		extractionPrompt := fmt.Sprintf(`分析以下文档并提取元数据，用JSON格式回答。请直接返回JSON，不要包含思考过程。

文档内容：
%s

请提取以下元数据：
- business_name (商家名称)
- location (具体地址) 
- owner (店主姓名)
- phone (联系电话)
- business_hours (营业时间)
- signature_drinks (招牌饮品，数组格式)

直接返回有效的JSON格式：`, testDoc)

		opts := &domain.GenerationOptions{
			Temperature: 0.1,
			MaxTokens:   300,
			Think:       func() *bool { b := false; return &b }(),
		}

		result, err := llmClient.Generate(ctx, extractionPrompt, opts)
		require.NoError(t, err)
		assert.NotEmpty(t, result, "Should extract metadata")

		fmt.Printf("Document: %s\n", strings.TrimSpace(testDoc))
		fmt.Printf("Extracted Metadata: %s\n", strings.TrimSpace(result))

		// Basic validation that we got JSON-like output
		assert.True(t, strings.Contains(result, "{"), "Should contain JSON structure")
		assert.True(t, strings.Contains(result, "}"), "Should contain JSON structure")
	})

	t.Run("Semantic Similarity with LLM", func(t *testing.T) {
		// Test similar concepts using our fictional knowledge base
		similarPairs := []struct {
			text1           string
			text2           string
			shouldBeSimilar bool
			description     string
		}{
			{
				text1:           "StarLight Tech的CEO张明轩",
				text2:           "StarLight Tech公司的首席执行官张明轩",
				shouldBeSimilar: true,
				description:     "CEO synonyms in Chinese",
			},
			{
				text1:           "月光餐厅的松露牛排",
				text2:           "月光餐厅招牌菜月光松露牛排",
				shouldBeSimilar: true,
				description:     "Restaurant signature dish",
			},
			{
				text1:           "陈维华博士的宠物狗算法",
				text2:           "陈教授的宠物狗算法",
				shouldBeSimilar: true,
				description:     "Pet description variations",
			},
			{
				text1:           "彩虹健康药房的24小时热线",
				text2:           "StarLight Tech的客服电话",
				shouldBeSimilar: false,
				description:     "Different company phone lines",
			},
			{
				text1:           "云天度假酒店的海景房",
				text2:           "月光餐厅的法式料理",
				shouldBeSimilar: false,
				description:     "Hotel vs restaurant services",
			},
		}

		for _, pair := range similarPairs {
			t.Run(pair.description, func(t *testing.T) {
				similar, err := utils.IsAlmostSameWithModel(ctx, pair.text1, pair.text2, cfg.Ollama.LLMModel)
				require.NoError(t, err)

				fmt.Printf("Text1: %s\n", pair.text1)
				fmt.Printf("Text2: %s\n", pair.text2)
				fmt.Printf("Similar: %v (expected: %v)\n", similar, pair.shouldBeSimilar)
				fmt.Printf("---\n")

				assert.Equal(t, pair.shouldBeSimilar, similar,
					"Similarity result should match expectation for %s", pair.description)
			})
		}
	})

	t.Run("Multi-turn Conversation", func(t *testing.T) {
		// Test fictional knowledge conversations
		conversation := []struct {
			question string
			filter   map[string]interface{}
		}{
			{
				question: "告诉我StarLight Tech公司的基本信息",
				filter:   map[string]interface{}{"type": "company_info"},
			},
			{
				question: "这家公司的产品有哪些？",
				filter:   map[string]interface{}{"type": "company_info"},
			},
			{
				question: "如果我想联系这家公司，应该拨打什么电话？",
				filter:   map[string]interface{}{"type": "company_info"},
			},
		}

		var conversationHistory []string

		for i, turn := range conversation {
			t.Run(fmt.Sprintf("Turn_%d", i+1), func(t *testing.T) {
				// Get context from knowledge base
				queryVector, err := embedderClient.Embed(ctx, turn.question)
				require.NoError(t, err)

				chunks, err := vectorStore.SearchWithFilters(ctx, queryVector, 2, turn.filter)
				require.NoError(t, err)

				var contexts []string
				for _, chunk := range chunks {
					contexts = append(contexts, chunk.Content)
				}
				context := strings.Join(contexts, "\n\n")

				// Build prompt with conversation history
				prompt := fmt.Sprintf(`你是一个智能助手，根据提供的上下文信息回答问题。请直接回答，不要包含思考过程。

上下文信息：
%s

对话历史：
%s

当前问题：%s

请直接回答：`, context, strings.Join(conversationHistory, "\n"), turn.question)

				opts := &domain.GenerationOptions{
					Temperature: 0.4,
					MaxTokens:   150,
					Think:       func() *bool { b := false; return &b }(),
				}

				answer, err := llmClient.Generate(ctx, prompt, opts)
				require.NoError(t, err)
				assert.NotEmpty(t, answer, "Should generate answer")

				// Add to conversation history
				conversationHistory = append(conversationHistory,
					fmt.Sprintf("Q: %s\nA: %s", turn.question, strings.TrimSpace(answer)))

				fmt.Printf("Turn %d - Q: %s\n", i+1, turn.question)
				fmt.Printf("Turn %d - A: %s\n", i+1, strings.TrimSpace(answer))
				fmt.Printf("---\n")

				// Basic verification that conversation continues
				assert.NotEmpty(t, answer, "Should generate conversation answer")
			})
		}
	})

	t.Run("LLM Performance and Health", func(t *testing.T) {
		// Test LLM response time with fictional knowledge
		start := time.Now()

		_, err := llmClient.Generate(ctx, "StarLight Tech公司在哪里？", &domain.GenerationOptions{
			Temperature: 0.1,
			MaxTokens:   50,
			Think:       func() *bool { b := false; return &b }(),
		})

		duration := time.Since(start)
		require.NoError(t, err)

		fmt.Printf("LLM Response Time: %v\n", duration)
		assert.Less(t, duration, 30*time.Second, "Response should be reasonably fast")

		// Test embedding performance
		start = time.Now()
		_, err = embedderClient.Embed(ctx, "测试虚构知识库的嵌入性能")
		embeddingDuration := time.Since(start)

		require.NoError(t, err)
		fmt.Printf("Embedding Time: %v\n", embeddingDuration)
		assert.Less(t, embeddingDuration, 10*time.Second, "Embedding should be fast")
	})
}
