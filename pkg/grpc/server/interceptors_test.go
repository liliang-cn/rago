package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Mock handler for testing interceptors
type MockUnaryHandler struct {
	mock.Mock
}

func (m *MockUnaryHandler) Handle(ctx context.Context, req interface{}) (interface{}, error) {
	args := m.Called(ctx, req)
	return args.Get(0), args.Error(1)
}

// Mock server info
type MockServerInfo struct {
	FullMethod string
}

func TestLoggingInterceptor(t *testing.T) {
	handler := &MockUnaryHandler{}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/rago.RAGService/Query",
	}
	ctx := context.Background()
	req := "test request"
	
	t.Run("successful request", func(t *testing.T) {
		expectedResp := "test response"
		handler.On("Handle", ctx, req).Return(expectedResp, nil)
		
		resp, err := loggingInterceptor(ctx, req, info, handler.Handle)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResp, resp)
		handler.AssertExpectations(t)
	})
	
	t.Run("request with error", func(t *testing.T) {
		expectedErr := status.Error(codes.Internal, "test error")
		handler.On("Handle", ctx, req).Return(nil, expectedErr)
		
		resp, err := loggingInterceptor(ctx, req, info, handler.Handle)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, expectedErr, err)
		handler.AssertExpectations(t)
	})
	
	t.Run("request with client IP", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"x-forwarded-for": "192.168.1.1",
		})
		ctxWithMD := metadata.NewIncomingContext(ctx, md)
		
		expectedResp := "test response"
		handler.On("Handle", ctxWithMD, req).Return(expectedResp, nil)
		
		resp, err := loggingInterceptor(ctxWithMD, req, info, handler.Handle)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResp, resp)
		handler.AssertExpectations(t)
	})
}

func TestRecoveryInterceptor(t *testing.T) {
	info := &grpc.UnaryServerInfo{
		FullMethod: "/rago.RAGService/Query",
	}
	ctx := context.Background()
	req := "test request"
	
	t.Run("normal execution", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "success", nil
		}
		
		resp, err := recoveryInterceptor(ctx, req, info, handler)
		
		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})
	
	t.Run("panic recovery", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			panic("test panic")
		}
		
		resp, err := recoveryInterceptor(ctx, req, info, handler)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "internal server error")
	})
}

func TestAuthInterceptor(t *testing.T) {
	token := "test-token"
	interceptor := authInterceptor(token)
	handler := &MockUnaryHandler{}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/rago.RAGService/Query",
	}
	ctx := context.Background()
	req := "test request"
	
	t.Run("valid token", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"authorization": "Bearer test-token",
		})
		ctxWithMD := metadata.NewIncomingContext(ctx, md)
		
		expectedResp := "success"
		handler.On("Handle", ctxWithMD, req).Return(expectedResp, nil)
		
		resp, err := interceptor(ctxWithMD, req, info, handler.Handle)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResp, resp)
		handler.AssertExpectations(t)
	})
	
	t.Run("invalid token", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"authorization": "Bearer wrong-token",
		})
		ctxWithMD := metadata.NewIncomingContext(ctx, md)
		
		resp, err := interceptor(ctxWithMD, req, info, handler.Handle)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Contains(t, st.Message(), "invalid token")
	})
	
	t.Run("missing authorization header", func(t *testing.T) {
		md := metadata.New(map[string]string{})
		ctxWithMD := metadata.NewIncomingContext(ctx, md)
		
		resp, err := interceptor(ctxWithMD, req, info, handler.Handle)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Contains(t, st.Message(), "missing authorization token")
	})
	
	t.Run("missing metadata", func(t *testing.T) {
		resp, err := interceptor(ctx, req, info, handler.Handle)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Contains(t, st.Message(), "missing metadata")
	})
	
	t.Run("health check bypass", func(t *testing.T) {
		healthInfo := &grpc.UnaryServerInfo{
			FullMethod: "/rago.RAGService/HealthCheck",
		}
		
		expectedResp := "health ok"
		handler.On("Handle", ctx, req).Return(expectedResp, nil)
		
		resp, err := interceptor(ctx, req, healthInfo, handler.Handle)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResp, resp)
		handler.AssertExpectations(t)
	})
}

func TestValidationInterceptor(t *testing.T) {
	handler := &MockUnaryHandler{}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/rago.RAGService/Query",
	}
	ctx := context.Background()
	
	t.Run("valid request", func(t *testing.T) {
		req := "test request"
		expectedResp := "success"
		handler.On("Handle", ctx, req).Return(expectedResp, nil)
		
		resp, err := validationInterceptor(ctx, req, info, handler.Handle)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResp, resp)
		handler.AssertExpectations(t)
	})
	
	t.Run("request with validator", func(t *testing.T) {
		// Mock request that implements Validate method
		req := &MockValidatableRequest{valid: true}
		expectedResp := "success"
		handler.On("Handle", ctx, req).Return(expectedResp, nil)
		
		resp, err := validationInterceptor(ctx, req, info, handler.Handle)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResp, resp)
		handler.AssertExpectations(t)
	})
	
	t.Run("invalid request", func(t *testing.T) {
		req := &MockValidatableRequest{valid: false}
		
		resp, err := validationInterceptor(ctx, req, info, handler.Handle)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "validation failed")
	})
}

// Mock request that implements validation
type MockValidatableRequest struct {
	valid bool
}

func (m *MockValidatableRequest) Validate() error {
	if !m.valid {
		return assert.AnError
	}
	return nil
}

func TestGetClientIP(t *testing.T) {
	t.Run("x-forwarded-for header", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"x-forwarded-for": "192.168.1.1",
		})
		
		ip := getClientIP(md)
		assert.Equal(t, "192.168.1.1", ip)
	})
	
	t.Run("x-real-ip header", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"x-real-ip": "10.0.0.1",
		})
		
		ip := getClientIP(md)
		assert.Equal(t, "10.0.0.1", ip)
	})
	
	t.Run("no headers", func(t *testing.T) {
		md := metadata.New(map[string]string{})
		
		ip := getClientIP(md)
		assert.Equal(t, "unknown", ip)
	})
	
	t.Run("prefer x-forwarded-for", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"x-forwarded-for": "192.168.1.1",
			"x-real-ip":       "10.0.0.1",
		})
		
		ip := getClientIP(md)
		assert.Equal(t, "192.168.1.1", ip)
	})
}

func TestGetClientID(t *testing.T) {
	t.Run("client-id header", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"client-id": "client-123",
		})
		
		id := getClientID(md)
		assert.Equal(t, "client-123", id)
	})
	
	t.Run("fallback to IP", func(t *testing.T) {
		md := metadata.New(map[string]string{
			"x-forwarded-for": "192.168.1.1",
		})
		
		id := getClientID(md)
		assert.Equal(t, "192.168.1.1", id)
	})
}

func TestTokenBucketLimiter(t *testing.T) {
	limiter := NewTokenBucketLimiter(5, 10) // 5 requests per second, burst of 10
	
	t.Run("initial burst", func(t *testing.T) {
		// Should allow initial burst
		for i := 0; i < 10; i++ {
			assert.True(t, limiter.Allow("client1"))
		}
		
		// Next request should be denied
		assert.False(t, limiter.Allow("client1"))
	})
	
	t.Run("token refill", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(5, 1) // 5 per second, burst of 1
		
		// Use up the token
		assert.True(t, limiter.Allow("client2"))
		assert.False(t, limiter.Allow("client2"))
		
		// Wait for token refill (simulate time passing)
		bucket := limiter.buckets["client2"]
		bucket.lastCheck = bucket.lastCheck.Add(-time.Second)
		
		// Should allow request after refill
		assert.True(t, limiter.Allow("client2"))
	})
	
	t.Run("different clients", func(t *testing.T) {
		limiter := NewTokenBucketLimiter(1, 1)
		
		assert.True(t, limiter.Allow("client1"))
		assert.True(t, limiter.Allow("client2"))
		assert.False(t, limiter.Allow("client1"))
		assert.False(t, limiter.Allow("client2"))
	})
}

func TestRateLimitInterceptor(t *testing.T) {
	limiter := NewTokenBucketLimiter(1, 1) // Very restrictive for testing
	interceptor := rateLimitInterceptor(limiter)
	handler := &MockUnaryHandler{}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/rago.RAGService/Query",
	}
	ctx := context.Background()
	req := "test request"
	
	t.Run("allowed request", func(t *testing.T) {
		expectedResp := "success"
		handler.On("Handle", ctx, req).Return(expectedResp, nil)
		
		resp, err := interceptor(ctx, req, info, handler.Handle)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResp, resp)
		handler.AssertExpectations(t)
	})
	
	t.Run("rate limited request", func(t *testing.T) {
		// Second request should be rate limited
		resp, err := interceptor(ctx, req, info, handler.Handle)
		
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.ResourceExhausted, st.Code())
		assert.Contains(t, st.Message(), "rate limit exceeded")
	})
}

func TestSimpleMetricsCollector(t *testing.T) {
	collector := NewSimpleMetricsCollector()
	
	t.Run("record successful request", func(t *testing.T) {
		collector.RecordRequest("/test/method", codes.OK, 100*time.Millisecond)
		
		metrics := collector.GetMetrics()
		requests := metrics["requests"].(map[string]int64)
		errors := metrics["errors"].(map[string]int64)
		duration := metrics["duration"].(map[string]time.Duration)
		
		assert.Equal(t, int64(1), requests["/test/method"])
		assert.Equal(t, int64(0), errors["/test/method"])
		assert.Equal(t, 100*time.Millisecond, duration["/test/method"])
	})
	
	t.Run("record error request", func(t *testing.T) {
		collector.RecordRequest("/test/method", codes.Internal, 50*time.Millisecond)
		
		metrics := collector.GetMetrics()
		requests := metrics["requests"].(map[string]int64)
		errors := metrics["errors"].(map[string]int64)
		duration := metrics["duration"].(map[string]time.Duration)
		
		assert.Equal(t, int64(2), requests["/test/method"])
		assert.Equal(t, int64(1), errors["/test/method"])
		assert.Equal(t, 150*time.Millisecond, duration["/test/method"])
	})
}

func TestMetricsInterceptor(t *testing.T) {
	collector := NewSimpleMetricsCollector()
	interceptor := metricsInterceptor(collector)
	handler := &MockUnaryHandler{}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/rago.RAGService/Query",
	}
	ctx := context.Background()
	req := "test request"
	
	t.Run("successful request metrics", func(t *testing.T) {
		expectedResp := "success"
		handler.On("Handle", ctx, req).Return(expectedResp, nil)
		
		resp, err := interceptor(ctx, req, info, handler.Handle)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedResp, resp)
		
		metrics := collector.GetMetrics()
		requests := metrics["requests"].(map[string]int64)
		assert.Equal(t, int64(1), requests["/rago.RAGService/Query"])
		handler.AssertExpectations(t)
	})
}