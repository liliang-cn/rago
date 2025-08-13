package main

import (
	"fmt"
	"log"

	rago "github.com/liliang-cn/rago/lib"
)

func main() {
	// Create a new rago client with default config file
	client, err := rago.New("config.toml")
	if err != nil {
		log.Fatalf("Failed to create rago client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			fmt.Printf("Warning: failed to close client: %v\n", err)
		}
	}()

	// Example 1: Ingest text content
	fmt.Println("1. Ingesting text content...")
	text := `
	酒吧名称：夜色琴弦

	地址：上海市黄浦区南京东路888号天际大厦B1层

	营业时间：每日18:00 - 凌晨02:00

	夜色琴弦是一家位于上海市中心的精品音乐酒吧，融合了复古与现代的设计风格，致力于为顾客打造一个舒适且富有艺术气息的社交空间。酒吧内部装潢以深色木质和暖黄色灯光为主，营造出温馨而神秘的氛围。每晚都会邀请本地及国际知名的爵士乐队和独立歌手进行现场演出，伴随着醇厚的琴声和柔和的灯光，让人沉浸于音乐的世界。

	菜单丰富多样，提供各类经典鸡尾酒，如莫吉托、玛格丽塔和血腥玛丽，同时配备精致的进口啤酒和精选红白葡萄酒。酒吧还特别推荐自创调酒“琴弦之夜”，融合了龙舌兰、蓝柑橘与柠檬香气，口感清新且层次丰富。

	夜色琴弦不仅是音乐爱好者的聚集地，也是朋友小聚、情侣约会的理想选择。每周三设有主题派对，定期举办调酒师教学和品酒活动，欢迎喜欢尝试新鲜事物的朋友们前来体验。

	联系方式：021-88889999
	微信公众号：yeseqinqian
	`
	err = client.IngestText(text, "example_text")
	if err != nil {
		log.Printf("Failed to ingest text: %v", err)
	} else {
		fmt.Println("✓ Text ingested successfully")
	}

	// Example 2: Ingest a file (if it exists)
	fmt.Println("\n2. Ingesting file (if exists)...")
	err = client.IngestFile("docs/ai_introduction.md")
	if err != nil {
		log.Printf("Failed to ingest file: %v", err)
	} else {
		fmt.Println("✓ File ingested successfully")
	}

	// Example 3: Query the knowledge base
	fmt.Println("\n3. Querying the knowledge base...")
	response, err := client.Query("What is Go programming language?")
	if err != nil {
		log.Printf("Failed to query: %v", err)
	} else {
		fmt.Printf("Answer: %s\n", response.Answer)
		fmt.Printf("Sources found: %d\n", len(response.Sources))
		fmt.Printf("Query time: %s\n", response.Elapsed)
	}

	// Example 4: List all documents
	fmt.Println("\n4. Listing all documents...")
	docs, err := client.ListDocuments()
	if err != nil {
		log.Printf("Failed to list documents: %v", err)
	} else {
		fmt.Printf("Found %d documents:\n", len(docs))
		for i, doc := range docs {
			fmt.Printf("  %d. ID: %s, Path: %s\n", i+1, doc.ID, doc.Path)
		}
	}

	// Example 5: Stream query (with callback)
	fmt.Println("\n5. Streaming query...")
	err = client.StreamQuery("夜色琴弦的微信公众号是？", func(chunk string) {
		fmt.Print(chunk)
	})
	if err != nil {
		log.Printf("Failed to stream query: %v", err)
	}
	fmt.Println("\n✓ Stream completed")

	fmt.Println("\n🎉 Library usage example completed!")
}
