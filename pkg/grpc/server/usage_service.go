package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/liliang-cn/rago/v2/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UsageService implements the gRPC UsageService
type UsageService struct {
	pb.UnimplementedUsageServiceServer
	tracker *usage.Tracker
}

// NewUsageService creates a new usage service
func NewUsageService(tracker *usage.Tracker) *UsageService {
	return &UsageService{
		tracker: tracker,
	}
}

// RecordUsage records usage metrics
func (s *UsageService) RecordUsage(ctx context.Context, req *pb.RecordUsageRequest) (*pb.RecordUsageResponse, error) {
	// Validate conversation ID if provided
	if req.ConversationId != "" {
		if _, err := uuid.Parse(req.ConversationId); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid conversation ID format: %v", err)
		}
	}

	// Convert protobuf metrics to domain format
	metrics := usage.Metrics{
		PromptTokens:     int(req.Metrics.PromptTokens),
		CompletionTokens: int(req.Metrics.CompletionTokens),
		TotalTokens:      int(req.Metrics.TotalTokens),
		Cost:             req.Metrics.Cost,
		LatencyMS:        req.Metrics.LatencyMs,
	}

	// Set timestamp
	timestamp := time.Now()
	if req.Timestamp > 0 {
		timestamp = time.Unix(req.Timestamp, 0)
	}

	// Record usage
	err := s.tracker.RecordUsage(usage.UsageRecord{
		ConversationID: req.ConversationId,
		Provider:       req.Provider,
		Model:          req.Model,
		Operation:      req.Operation,
		Metrics:        metrics,
		Timestamp:      timestamp,
		Metadata:       req.Metadata,
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record usage: %v", err)
	}

	return &pb.RecordUsageResponse{
		Success: true,
	}, nil
}

// GetUsageStats returns usage statistics
func (s *UsageService) GetUsageStats(ctx context.Context, req *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error) {
	// Validate conversation ID if provided
	if req.ConversationId != "" {
		if _, err := uuid.Parse(req.ConversationId); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid conversation ID format: %v", err)
		}
	}

	// Set time range
	startTime := time.Unix(0, 0)
	if req.StartTime > 0 {
		startTime = time.Unix(req.StartTime, 0)
	}

	endTime := time.Now()
	if req.EndTime > 0 {
		endTime = time.Unix(req.EndTime, 0)
	}

	// Get statistics
	stats, err := s.tracker.GetStatistics(usage.StatsQuery{
		ConversationID: req.ConversationId,
		StartTime:      startTime,
		EndTime:        endTime,
		Provider:       req.Provider,
		Model:          req.Model,
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get usage stats: %v", err)
	}

	// Convert to protobuf format
	usageByModel := make(map[string]*pb.UsageByModel)
	for model, modelStats := range stats.ByModel {
		usageByModel[model] = &pb.UsageByModel{
			Model:        model,
			RequestCount: int64(modelStats.RequestCount),
			TotalTokens:  int64(modelStats.TotalTokens),
			TotalCost:    modelStats.TotalCost,
		}
	}

	usageByProvider := make(map[string]*pb.UsageByProvider)
	for provider, providerStats := range stats.ByProvider {
		usageByProvider[provider] = &pb.UsageByProvider{
			Provider:     provider,
			RequestCount: int64(providerStats.RequestCount),
			TotalTokens:  int64(providerStats.TotalTokens),
			TotalCost:    providerStats.TotalCost,
		}
	}

	return &pb.GetUsageStatsResponse{
		TotalRequests:    int64(stats.TotalRequests),
		TotalTokens:      int64(stats.TotalTokens),
		TotalCost:        stats.TotalCost,
		AverageLatencyMs: stats.AverageLatencyMS,
		UsageByModel:     usageByModel,
		UsageByProvider:  usageByProvider,
	}, nil
}

// GetUsageHistory returns usage history with pagination
func (s *UsageService) GetUsageHistory(ctx context.Context, req *pb.GetUsageHistoryRequest) (*pb.GetUsageHistoryResponse, error) {
	// Validate conversation ID if provided
	if req.ConversationId != "" {
		if _, err := uuid.Parse(req.ConversationId); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid conversation ID format: %v", err)
		}
	}

	// Set default values
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	// Set time range
	startTime := time.Unix(0, 0)
	if req.StartTime > 0 {
		startTime = time.Unix(req.StartTime, 0)
	}

	endTime := time.Now()
	if req.EndTime > 0 {
		endTime = time.Unix(req.EndTime, 0)
	}

	// Get usage history
	records, total, err := s.tracker.GetHistory(usage.HistoryQuery{
		ConversationID: req.ConversationId,
		StartTime:      startTime,
		EndTime:        endTime,
		Page:           int(page),
		PageSize:       int(pageSize),
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get usage history: %v", err)
	}

	// Convert to protobuf format
	pbRecords := make([]*pb.UsageRecord, 0, len(records))
	for _, record := range records {
		pbRecords = append(pbRecords, &pb.UsageRecord{
			Id:             record.ID,
			ConversationId: record.ConversationID,
			Provider:       record.Provider,
			Model:          record.Model,
			Operation:      record.Operation,
			Metrics: &pb.UsageMetrics{
				PromptTokens:     int32(record.Metrics.PromptTokens),
				CompletionTokens: int32(record.Metrics.CompletionTokens),
				TotalTokens:      int32(record.Metrics.TotalTokens),
				Cost:             record.Metrics.Cost,
				LatencyMs:        record.Metrics.LatencyMS,
			},
			Timestamp: record.Timestamp.Unix(),
			Metadata:  record.Metadata,
		})
	}

	return &pb.GetUsageHistoryResponse{
		Records:  pbRecords,
		Total:    int32(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}