# Free News Sources for RAGO (No API Key Required)

## RSS Feeds (XML Format)
These can be fetched directly with the `fetch` tool:

### Technology News
- **BBC Technology**: https://feeds.bbci.co.uk/news/technology/rss.xml
- **Reuters Technology**: https://www.reutersagency.com/feed/?taxonomy=best-sectors&post_type=best
- **The Verge**: https://www.theverge.com/rss/index.xml
- **Ars Technica**: https://feeds.arstechnica.com/arstechnica/index
- **TechCrunch**: https://techcrunch.com/feed/

### General News
- **BBC World**: https://feeds.bbci.co.uk/news/world/rss.xml
- **CNN Top Stories**: http://rss.cnn.com/rss/cnn_topstories.rss
- **The Guardian**: https://www.theguardian.com/world/rss
- **NPR News**: https://feeds.npr.org/1001/rss.xml

### Business News
- **Yahoo Finance**: https://finance.yahoo.com/news/rssindex
- **MarketWatch**: http://feeds.marketwatch.com/marketwatch/topstories/

## JSON APIs (No Key Required)

### Hacker News
- **Top Stories IDs**: https://hacker-news.firebaseio.com/v0/topstories.json
- **Story Details**: https://hacker-news.firebaseio.com/v0/item/{id}.json
- **Best Stories**: https://hacker-news.firebaseio.com/v0/beststories.json

### Reddit (Public JSON)
- **r/news**: https://www.reddit.com/r/news/.json
- **r/worldnews**: https://www.reddit.com/r/worldnews/.json
- **r/technology**: https://www.reddit.com/r/technology/.json
- **r/programming**: https://www.reddit.com/r/programming/.json

## Example RAGO Commands

### Fetch and Summarize Tech News
```bash
rago agent run "fetch the latest technology news from BBC RSS feed and create a summary"
```

### Monitor Hacker News
```bash
rago agent run "get the top 5 stories from Hacker News and summarize them"
```

### Reddit Analysis
```bash
rago agent run "fetch recent posts from r/technology subreddit and identify trending topics"
```

## Workflow Example

Here's what RAGO generates when you ask for news:

```json
{
  "steps": [
    {
      "id": "fetch_news",
      "name": "Fetch RSS Feed",
      "tool": "fetch",
      "inputs": {
        "url": "https://feeds.bbci.co.uk/news/technology/rss.xml"
      },
      "outputs": {
        "data": "rss_content"
      }
    },
    {
      "id": "analyze",
      "name": "Analyze and Summarize",
      "tool": "sequential-thinking",
      "inputs": {
        "prompt": "Extract the top 5 stories from this RSS feed and create a brief summary",
        "data": "{{rss_content}}"
      }
    }
  ]
}
```

## Limitations

1. **RSS Feeds**: Provide titles and descriptions but not full article content
2. **Reddit JSON**: Rate limited if accessed too frequently
3. **HTML Scraping**: Many news sites block automated access or require JavaScript
4. **Content Quality**: RSS typically has summaries, not full articles

## Best Practices

1. Use RSS feeds for reliable, structured data
2. Combine multiple sources for comprehensive coverage
3. Use sequential-thinking to extract and summarize key points
4. Cache results to avoid hitting rate limits
5. Save summaries locally for later reference

## Alternative: Local News Aggregation

Instead of fetching directly, you could:
1. Use a local RSS reader that exports OPML/JSON
2. Save articles as markdown files locally
3. Use RAGO to process local files: `rago ingest *.md`
4. Query your local knowledge base: `rago query "latest tech news"`

This approach gives you full control over your news consumption without depending on external APIs.