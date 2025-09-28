package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	pb "github.com/liliang-cn/rago/v2/proto/rago"
	"github.com/liliang-cn/rago/v2/pkg/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// ConversationService implements the gRPC ConversationService
type ConversationService struct {
	pb.UnimplementedConversationServiceServer
	store *store.ConversationStore
}

// NewConversationService creates a new conversation service
func NewConversationService(convStore *store.ConversationStore) *ConversationService {
	return &ConversationService{
		store: convStore,
	}
}

// SaveConversation saves or updates a conversation
func (s *ConversationService) SaveConversation(ctx context.Context, req *pb.SaveConversationRequest) (*pb.SaveConversationResponse, error) {
	// Generate UUID if not provided
	conversationID := req.Id
	if conversationID == "" {
		conversationID = uuid.New().String()
	} else {
		// Validate UUID format
		if _, err := uuid.Parse(conversationID); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid conversation ID format: %v", err)
		}
	}

	// Convert protobuf messages to store format
	messages := make([]store.ConversationMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, store.ConversationMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			Metadata:  msg.Metadata,
		})
	}

	// Convert metadata
	metadata := make(map[string]interface{})
	for k, v := range req.Metadata {
		metadata[k] = v.AsInterface()
	}

	// Create conversation object
	conversation := &store.Conversation{
		ID:        conversationID,
		Title:     req.Title,
		Messages:  messages,
		Metadata:  metadata,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	// Check if it's an update
	if req.Id != "" {
		if existing, err := s.store.GetConversation(conversationID); err == nil && existing != nil {
			conversation.CreatedAt = existing.CreatedAt
		}
	}

	// Save conversation
	if err := s.store.SaveConversation(conversation); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save conversation: %v", err)
	}

	return &pb.SaveConversationResponse{
		Id:      conversationID,
		Success: true,
	}, nil
}

// GetConversation retrieves a conversation by ID
func (s *ConversationService) GetConversation(ctx context.Context, req *pb.GetConversationRequest) (*pb.GetConversationResponse, error) {
	// Validate UUID format
	if _, err := uuid.Parse(req.Id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conversation ID format: %v", err)
	}

	conversation, err := s.store.GetConversation(req.Id)
	if err != nil {
		if err == store.ErrConversationNotFound {
			return nil, status.Errorf(codes.NotFound, "conversation not found: %s", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "failed to get conversation: %v", err)
	}

	// Convert messages to protobuf format
	messages := make([]*pb.ConversationMessage, 0, len(conversation.Messages))
	for _, msg := range conversation.Messages {
		messages = append(messages, &pb.ConversationMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			Metadata:  msg.Metadata,
		})
	}

	// Convert metadata to protobuf format
	metadata := make(map[string]*structpb.Value)
	for k, v := range conversation.Metadata {
		val, err := structpb.NewValue(v)
		if err != nil {
			continue // Skip invalid values
		}
		metadata[k] = val
	}

	return &pb.GetConversationResponse{
		Id:        conversation.ID,
		Title:     conversation.Title,
		Messages:  messages,
		Metadata:  metadata,
		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
	}, nil
}

// ListConversations lists conversations with pagination
func (s *ConversationService) ListConversations(ctx context.Context, req *pb.ListConversationsRequest) (*pb.ListConversationsResponse, error) {
	// Set default values
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	// Get conversations
	conversations, total, err := s.store.ListConversations(int(page), int(pageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list conversations: %v", err)
	}

	// Convert to protobuf format
	summaries := make([]*pb.ConversationSummary, 0, len(conversations))
	for _, conv := range conversations {
		summaries = append(summaries, &pb.ConversationSummary{
			Id:           conv.ID,
			Title:        conv.Title,
			MessageCount: int32(len(conv.Messages)),
			CreatedAt:    conv.CreatedAt,
			UpdatedAt:    conv.UpdatedAt,
		})
	}

	return &pb.ListConversationsResponse{
		Conversations: summaries,
		Total:         int32(total),
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

// DeleteConversation deletes a conversation by ID
func (s *ConversationService) DeleteConversation(ctx context.Context, req *pb.DeleteConversationRequest) (*pb.DeleteConversationResponse, error) {
	// Validate UUID format
	if _, err := uuid.Parse(req.Id); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conversation ID format: %v", err)
	}

	if err := s.store.DeleteConversation(req.Id); err != nil {
		if err == store.ErrConversationNotFound {
			return nil, status.Errorf(codes.NotFound, "conversation not found: %s", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "failed to delete conversation: %v", err)
	}

	return &pb.DeleteConversationResponse{
		Success: true,
		Message: "Conversation deleted successfully",
	}, nil
}