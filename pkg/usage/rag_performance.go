package usage

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// RAGPerformanceAnalyzer provides comprehensive performance analysis for RAG operations
type RAGPerformanceAnalyzer struct {
	repository Repository
}

// NewRAGPerformanceAnalyzer creates a new RAG performance analyzer
func NewRAGPerformanceAnalyzer(repo Repository) *RAGPerformanceAnalyzer {
	return &RAGPerformanceAnalyzer{
		repository: repo,
	}
}

// RAGPerformanceReport represents a comprehensive performance analysis report
type RAGPerformanceReport struct {
	// Time range
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	
	// Overall metrics
	TotalQueries      int     `json:"total_queries"`
	SuccessRate       float64 `json:"success_rate"`
	AvgLatency        float64 `json:"avg_latency"`
	MedianLatency     float64 `json:"median_latency"`
	P95Latency        float64 `json:"p95_latency"`
	P99Latency        float64 `json:"p99_latency"`
	
	// Retrieval performance
	AvgRetrievalTime  float64 `json:"avg_retrieval_time"`
	AvgGenerationTime float64 `json:"avg_generation_time"`
	RetrievalRatio    float64 `json:"retrieval_ratio"` // Retrieval time / Total time
	
	// Quality metrics
	AvgChunksFound     float64 `json:"avg_chunks_found"`
	AvgTopScore        float64 `json:"avg_top_score"`
	AvgSourceUtil      float64 `json:"avg_source_utilization"`
	AvgConfidence      float64 `json:"avg_confidence"`
	AvgFactuality      float64 `json:"avg_factuality"`
	
	// Performance trends
	LatencyTrend       Trend `json:"latency_trend"`
	QualityTrend       Trend `json:"quality_trend"`
	
	// Performance bottlenecks
	SlowQueries        []QueryPerformanceIssue `json:"slow_queries"`
	LowQualityQueries  []QueryPerformanceIssue `json:"low_quality_queries"`
	
	// Recommendations
	Recommendations    []PerformanceRecommendation `json:"recommendations"`
}

// Trend represents a performance trend (improving, declining, stable)
type Trend string

const (
	TrendImproving Trend = "improving"
	TrendDeclining Trend = "declining"
	TrendStable    Trend = "stable"
)

// QueryPerformanceIssue represents a performance issue with a specific query
type QueryPerformanceIssue struct {
	QueryID     string  `json:"query_id"`
	Query       string  `json:"query"`
	Latency     int64   `json:"latency"`
	Score       float64 `json:"score"`
	Issue       string  `json:"issue"`
	Severity    string  `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
}

// PerformanceRecommendation represents an optimization recommendation
type PerformanceRecommendation struct {
	Type        string  `json:"type"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Priority    string  `json:"priority"`
	Metrics     map[string]interface{} `json:"metrics"`
}

// GeneratePerformanceReport generates a comprehensive performance analysis report
func (a *RAGPerformanceAnalyzer) GeneratePerformanceReport(ctx context.Context, filter *RAGSearchFilter) (*RAGPerformanceReport, error) {
	// Set default time range if not specified
	if filter.StartTime.IsZero() && filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
		filter.StartTime = filter.EndTime.AddDate(0, 0, -7) // Last 7 days
	}
	
	// Get all queries in the time range
	queries, err := a.repository.ListRAGQueries(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RAG queries: %w", err)
	}
	
	if len(queries) == 0 {
		return &RAGPerformanceReport{
			StartTime: filter.StartTime,
			EndTime:   filter.EndTime,
		}, nil
	}
	
	report := &RAGPerformanceReport{
		StartTime:    filter.StartTime,
		EndTime:      filter.EndTime,
		TotalQueries: len(queries),
	}
	
	// Calculate overall metrics
	a.calculateOverallMetrics(report, queries)
	
	// Calculate quality metrics
	err = a.calculateQualityMetrics(ctx, report, queries)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate quality metrics: %w", err)
	}
	
	// Calculate trends
	a.calculateTrends(report, queries)
	
	// Identify performance issues
	a.identifyPerformanceIssues(report, queries)
	
	// Generate recommendations
	a.generateRecommendations(report, queries)
	
	return report, nil
}

// calculateOverallMetrics calculates basic performance metrics
func (a *RAGPerformanceAnalyzer) calculateOverallMetrics(report *RAGPerformanceReport, queries []*RAGQueryRecord) {
	var totalLatency, totalRetrievalTime, totalGenerationTime int64
	var successCount int
	var latencies []int64
	
	for _, query := range queries {
		totalLatency += query.TotalLatency
		totalRetrievalTime += query.RetrievalTime
		totalGenerationTime += query.GenerationTime
		latencies = append(latencies, query.TotalLatency)
		
		if query.Success {
			successCount++
		}
	}
	
	report.SuccessRate = float64(successCount) / float64(len(queries))
	report.AvgLatency = float64(totalLatency) / float64(len(queries))
	report.AvgRetrievalTime = float64(totalRetrievalTime) / float64(len(queries))
	report.AvgGenerationTime = float64(totalGenerationTime) / float64(len(queries))
	
	if totalLatency > 0 {
		report.RetrievalRatio = float64(totalRetrievalTime) / float64(totalLatency)
	}
	
	// Calculate percentiles
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	
	if len(latencies) > 0 {
		report.MedianLatency = float64(latencies[len(latencies)/2])
		p95Index := int(float64(len(latencies)) * 0.95)
		if p95Index >= len(latencies) {
			p95Index = len(latencies) - 1
		}
		report.P95Latency = float64(latencies[p95Index])
		
		p99Index := int(float64(len(latencies)) * 0.99)
		if p99Index >= len(latencies) {
			p99Index = len(latencies) - 1
		}
		report.P99Latency = float64(latencies[p99Index])
	}
}

// calculateQualityMetrics calculates quality-related metrics
func (a *RAGPerformanceAnalyzer) calculateQualityMetrics(ctx context.Context, report *RAGPerformanceReport, queries []*RAGQueryRecord) error {
	var totalChunks int
	var totalTopScore, totalSourceUtil, totalConfidence, totalFactuality float64
	var qualityCount int
	
	for _, query := range queries {
		totalChunks += query.ChunksFound
		
		// Get visualization data for quality metrics
		viz, err := a.repository.GetRAGVisualization(ctx, query.ID)
		if err != nil {
			continue // Skip if visualization data not available
		}
		
		// Get top score from chunk hits
		if len(viz.ChunkHits) > 0 {
			topScore := viz.ChunkHits[0].Score
			for _, hit := range viz.ChunkHits {
				if hit.Score > topScore {
					topScore = hit.Score
				}
			}
			totalTopScore += topScore
		}
		
		// Add quality metrics
		totalSourceUtil += viz.QualityMetrics.SourceUtilization
		totalConfidence += viz.QualityMetrics.ConfidenceScore
		totalFactuality += viz.QualityMetrics.FactualityScore
		qualityCount++
	}
	
	if len(queries) > 0 {
		report.AvgChunksFound = float64(totalChunks) / float64(len(queries))
	}
	
	if qualityCount > 0 {
		report.AvgTopScore = totalTopScore / float64(qualityCount)
		report.AvgSourceUtil = totalSourceUtil / float64(qualityCount)
		report.AvgConfidence = totalConfidence / float64(qualityCount)
		report.AvgFactuality = totalFactuality / float64(qualityCount)
	}
	
	return nil
}

// calculateTrends analyzes performance trends over time
func (a *RAGPerformanceAnalyzer) calculateTrends(report *RAGPerformanceReport, queries []*RAGQueryRecord) {
	if len(queries) < 10 {
		report.LatencyTrend = TrendStable
		report.QualityTrend = TrendStable
		return
	}
	
	// Sort queries by time
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].CreatedAt.Before(queries[j].CreatedAt)
	})
	
	// Split into first and second half
	midpoint := len(queries) / 2
	firstHalf := queries[:midpoint]
	secondHalf := queries[midpoint:]
	
	// Calculate average latency for each half
	var firstAvgLatency, secondAvgLatency float64
	for _, q := range firstHalf {
		firstAvgLatency += float64(q.TotalLatency)
	}
	firstAvgLatency /= float64(len(firstHalf))
	
	for _, q := range secondHalf {
		secondAvgLatency += float64(q.TotalLatency)
	}
	secondAvgLatency /= float64(len(secondHalf))
	
	// Determine latency trend
	latencyChange := (secondAvgLatency - firstAvgLatency) / firstAvgLatency
	if math.Abs(latencyChange) < 0.1 { // Less than 10% change
		report.LatencyTrend = TrendStable
	} else if latencyChange < 0 {
		report.LatencyTrend = TrendImproving // Lower latency is better
	} else {
		report.LatencyTrend = TrendDeclining
	}
	
	// Calculate success rate for each half for quality trend
	firstSuccess := 0
	for _, q := range firstHalf {
		if q.Success {
			firstSuccess++
		}
	}
	firstSuccessRate := float64(firstSuccess) / float64(len(firstHalf))
	
	secondSuccess := 0
	for _, q := range secondHalf {
		if q.Success {
			secondSuccess++
		}
	}
	secondSuccessRate := float64(secondSuccess) / float64(len(secondHalf))
	
	// Determine quality trend
	qualityChange := secondSuccessRate - firstSuccessRate
	if math.Abs(qualityChange) < 0.05 { // Less than 5% change
		report.QualityTrend = TrendStable
	} else if qualityChange > 0 {
		report.QualityTrend = TrendImproving
	} else {
		report.QualityTrend = TrendDeclining
	}
}

// identifyPerformanceIssues identifies slow queries and quality issues
func (a *RAGPerformanceAnalyzer) identifyPerformanceIssues(report *RAGPerformanceReport, queries []*RAGQueryRecord) {
	// Find slow queries (P95 threshold)
	slowThreshold := report.P95Latency
	
	for _, query := range queries {
		if float64(query.TotalLatency) > slowThreshold {
			issue := QueryPerformanceIssue{
				QueryID:   query.ID,
				Query:     truncateString(query.Query, 100),
				Latency:   query.TotalLatency,
				Issue:     "High latency",
				Severity:  getSeverity(float64(query.TotalLatency), slowThreshold),
				Timestamp: query.CreatedAt,
			}
			report.SlowQueries = append(report.SlowQueries, issue)
		}
		
		// Check for low success rate (failed queries)
		if !query.Success {
			issue := QueryPerformanceIssue{
				QueryID:   query.ID,
				Query:     truncateString(query.Query, 100),
				Latency:   query.TotalLatency,
				Issue:     fmt.Sprintf("Query failed: %s", query.ErrorMessage),
				Severity:  "high",
				Timestamp: query.CreatedAt,
			}
			report.LowQualityQueries = append(report.LowQualityQueries, issue)
		}
	}
	
	// Sort issues by severity and timestamp
	sort.Slice(report.SlowQueries, func(i, j int) bool {
		if report.SlowQueries[i].Severity != report.SlowQueries[j].Severity {
			return getSeverityPriority(report.SlowQueries[i].Severity) > getSeverityPriority(report.SlowQueries[j].Severity)
		}
		return report.SlowQueries[i].Timestamp.After(report.SlowQueries[j].Timestamp)
	})
	
	sort.Slice(report.LowQualityQueries, func(i, j int) bool {
		return report.LowQualityQueries[i].Timestamp.After(report.LowQualityQueries[j].Timestamp)
	})
	
	// Limit to top 10 issues
	if len(report.SlowQueries) > 10 {
		report.SlowQueries = report.SlowQueries[:10]
	}
	if len(report.LowQualityQueries) > 10 {
		report.LowQualityQueries = report.LowQualityQueries[:10]
	}
}

// generateRecommendations generates performance optimization recommendations
func (a *RAGPerformanceAnalyzer) generateRecommendations(report *RAGPerformanceReport, queries []*RAGQueryRecord) {
	recommendations := []PerformanceRecommendation{}
	
	// High latency recommendation
	if report.AvgLatency > 5000 { // 5 seconds
		rec := PerformanceRecommendation{
			Type:        "latency",
			Title:       "Optimize Query Response Time",
			Description: "Average query latency is high. Consider optimizing embedding models, chunk size, or implementing caching.",
			Impact:      "Reduce average latency by 30-50%",
			Priority:    "high",
			Metrics: map[string]interface{}{
				"current_avg_latency": report.AvgLatency,
				"target_latency":      3000,
			},
		}
		recommendations = append(recommendations, rec)
	}
	
	// Low success rate recommendation
	if report.SuccessRate < 0.9 {
		rec := PerformanceRecommendation{
			Type:        "reliability",
			Title:       "Improve Query Success Rate",
			Description: "Query success rate is below 90%. Review error patterns and improve error handling.",
			Impact:      "Increase success rate to 95%+",
			Priority:    "high",
			Metrics: map[string]interface{}{
				"current_success_rate": report.SuccessRate,
				"target_success_rate":  0.95,
			},
		}
		recommendations = append(recommendations, rec)
	}
	
	// Retrieval optimization recommendation
	if report.RetrievalRatio > 0.7 {
		rec := PerformanceRecommendation{
			Type:        "retrieval",
			Title:       "Optimize Retrieval Performance",
			Description: "Retrieval takes up most of the query time. Consider optimizing vector search or reducing chunk count.",
			Impact:      "Reduce retrieval time by 20-40%",
			Priority:    "medium",
			Metrics: map[string]interface{}{
				"retrieval_ratio":     report.RetrievalRatio,
				"avg_retrieval_time":  report.AvgRetrievalTime,
			},
		}
		recommendations = append(recommendations, rec)
	}
	
	// Quality improvement recommendation
	if report.AvgConfidence < 0.7 {
		rec := PerformanceRecommendation{
			Type:        "quality",
			Title:       "Improve Answer Quality",
			Description: "Average confidence score is low. Consider improving chunking strategy or fine-tuning retrieval parameters.",
			Impact:      "Increase answer confidence and accuracy",
			Priority:    "medium",
			Metrics: map[string]interface{}{
				"avg_confidence": report.AvgConfidence,
				"target_confidence": 0.8,
			},
		}
		recommendations = append(recommendations, rec)
	}
	
	// Source utilization recommendation
	if report.AvgSourceUtil < 0.5 {
		rec := PerformanceRecommendation{
			Type:        "efficiency",
			Title:       "Improve Source Utilization",
			Description: "Low source utilization indicates retrieved chunks may not be relevant. Optimize retrieval parameters.",
			Impact:      "Better resource efficiency and answer quality",
			Priority:    "low",
			Metrics: map[string]interface{}{
				"avg_source_util": report.AvgSourceUtil,
				"target_source_util": 0.7,
			},
		}
		recommendations = append(recommendations, rec)
	}
	
	report.Recommendations = recommendations
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func getSeverity(value, threshold float64) string {
	ratio := value / threshold
	if ratio > 2.0 {
		return "critical"
	} else if ratio > 1.5 {
		return "high"
	} else if ratio > 1.2 {
		return "medium"
	}
	return "low"
}

func getSeverityPriority(severity string) int {
	switch severity {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}