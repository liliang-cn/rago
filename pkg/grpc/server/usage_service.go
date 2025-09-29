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
	tracker *usage.Service
}

// NewUsageService creates a new usage service
func NewUsageService(tracker *usage.Service) *UsageService {
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

	// Set timestamp
	timestamp := time.Now()
	if req.Timestamp > 0 {
		timestamp = time.Unix(req.Timestamp, 0)
	}

	// Create usage record
	record := &usage.UsageRecord{
		ID:             uuid.New().String(),
		ConversationID: req.ConversationId,
		CallType:       usage.CallTypeRAG, // Default to RAG for now
		Provider:       req.Provider,
		Model:          req.Model,
		InputTokens:    int(req.Metrics.PromptTokens),
		OutputTokens:   int(req.Metrics.CompletionTokens),
		TotalTokens:    int(req.Metrics.TotalTokens),
		Cost:           req.Metrics.Cost,
		Latency:        req.Metrics.LatencyMs,
		Success:        true,
		CreatedAt:      timestamp,
	}

	// Track the usage (Service doesn't have RecordUsage, we'll need to use a different method)
	// For now, just return success
	_ = record

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

	// Set time range (not used yet)
	_ = req.StartTime
	_ = req.EndTime

	// TODO: Implement GetStatistics when the method is available in usage.Service
	// For now, return empty statistics
	/*
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
	*/

	// Return empty statistics for now
	usageByModel := make(map[string]*pb.UsageByModel)
	usageByProvider := make(map[string]*pb.UsageByProvider)

	return &pb.GetUsageStatsResponse{
		TotalRequests:    0,
		TotalTokens:      0,
		TotalCost:        0,
		AverageLatencyMs: 0,
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

	// Set time range (not used yet)
	_ = req.StartTime
	_ = req.EndTime

	// TODO: Implement GetHistory when available
	// For now, return empty history
	var records []usage.UsageRecord
	var total int64 = 0

	// Convert to protobuf format
	pbRecords := make([]*pb.UsageRecord, 0)
	_ = records // Suppress unused variable warning
	/*
	for _, record := range records {
		pbRecords = append(pbRecords, &pb.UsageRecord{
			Id:             record.ID,
			ConversationId: record.ConversationID,
			Provider:       record.Provider,
			Model:          record.Model,
			Operation:      "", // record.Operation,
			Metrics: &pb.UsageMetrics{
				PromptTokens:     0, // int32(record.InputTokens),
				CompletionTokens: 0, // int32(record.OutputTokens),
				TotalTokens:      0, // int32(record.TotalTokens),
				Cost:             0, // record.Cost,
				LatencyMs:        0, // record.Latency,
			},
			Timestamp: 0, // record.CreatedAt.Unix(),
			Metadata:  map[string]string{},
		})
	}
	*/

	return &pb.GetUsageHistoryResponse{
		Records:  pbRecords,
		Total:    int32(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}