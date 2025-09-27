package server

import (
	"context"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// loggingInterceptor logs all gRPC requests
func loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// Get request metadata
	md, _ := metadata.FromIncomingContext(ctx)
	clientIP := getClientIP(md)

	// Call the handler
	resp, err := handler(ctx, req)

	// Log the request
	duration := time.Since(start)
	statusCode := codes.OK
	if err != nil {
		statusCode = status.Code(err)
	}

	log.Printf("[gRPC] %s | %s | %s | %v | %v",
		clientIP,
		info.FullMethod,
		statusCode,
		duration,
		err,
	)

	return resp, err
}

// recoveryInterceptor recovers from panics
func recoveryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[gRPC Panic] %s: %v\n%s", info.FullMethod, r, debug.Stack())
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()

	return handler(ctx, req)
}

// authInterceptor validates authentication tokens
func authInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for health check
		if strings.HasSuffix(info.FullMethod, "/HealthCheck") {
			return handler(ctx, req)
		}

		// Get token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authorization := md.Get("authorization")
		if len(authorization) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		// Validate token
		authToken := strings.TrimPrefix(authorization[0], "Bearer ")
		if authToken != token {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(ctx, req)
	}
}

// streamAuthInterceptor validates authentication for streaming RPCs
func streamAuthInterceptor(token string) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Skip auth for health check
		if strings.HasSuffix(info.FullMethod, "/HealthCheck") {
			return handler(srv, stream)
		}

		// Get token from metadata
		md, ok := metadata.FromIncomingContext(stream.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		authorization := md.Get("authorization")
		if len(authorization) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization token")
		}

		// Validate token
		authToken := strings.TrimPrefix(authorization[0], "Bearer ")
		if authToken != token {
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(srv, stream)
	}
}

// rateLimitInterceptor implements rate limiting
func rateLimitInterceptor(limiter RateLimiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Get client identifier
		md, _ := metadata.FromIncomingContext(ctx)
		clientID := getClientID(md)

		// Check rate limit
		if !limiter.Allow(clientID) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// metricsInterceptor collects metrics
func metricsInterceptor(metrics MetricsCollector) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		metrics.RecordRequest(info.FullMethod, statusCode, duration)

		return resp, err
	}
}

// validationInterceptor validates requests
func validationInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// Validate request if it implements Validator interface
	if validator, ok := req.(interface{ Validate() error }); ok {
		if err := validator.Validate(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
		}
	}

	return handler(ctx, req)
}

// getClientIP extracts client IP from metadata
func getClientIP(md metadata.MD) string {
	if xForwardedFor := md.Get("x-forwarded-for"); len(xForwardedFor) > 0 {
		return xForwardedFor[0]
	}
	if xRealIP := md.Get("x-real-ip"); len(xRealIP) > 0 {
		return xRealIP[0]
	}
	return "unknown"
}

// getClientID extracts client identifier from metadata
func getClientID(md metadata.MD) string {
	if clientID := md.Get("client-id"); len(clientID) > 0 {
		return clientID[0]
	}
	return getClientIP(md)
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(clientID string) bool
}

// TokenBucketLimiter implements token bucket rate limiting
type TokenBucketLimiter struct {
	buckets map[string]*TokenBucket
	rate    int
	burst   int
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens    int
	lastCheck time.Time
}

// NewTokenBucketLimiter creates a new token bucket limiter
func NewTokenBucketLimiter(rate, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		buckets: make(map[string]*TokenBucket),
		rate:    rate,
		burst:   burst,
	}
}

// Allow checks if a request is allowed
func (l *TokenBucketLimiter) Allow(clientID string) bool {
	now := time.Now()

	bucket, exists := l.buckets[clientID]
	if !exists {
		bucket = &TokenBucket{
			tokens:    l.burst,
			lastCheck: now,
		}
		l.buckets[clientID] = bucket
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(bucket.lastCheck)
	tokensToAdd := int(elapsed.Seconds() * float64(l.rate))
	bucket.tokens = min(bucket.tokens+tokensToAdd, l.burst)
	bucket.lastCheck = now

	// Check if we have tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MetricsCollector interface for collecting metrics
type MetricsCollector interface {
	RecordRequest(method string, statusCode codes.Code, duration time.Duration)
}

// SimpleMetricsCollector implements basic metrics collection
type SimpleMetricsCollector struct {
	requests map[string]int64
	errors   map[string]int64
	duration map[string]time.Duration
}

// NewSimpleMetricsCollector creates a new simple metrics collector
func NewSimpleMetricsCollector() *SimpleMetricsCollector {
	return &SimpleMetricsCollector{
		requests: make(map[string]int64),
		errors:   make(map[string]int64),
		duration: make(map[string]time.Duration),
	}
}

// RecordRequest records a request metric
func (c *SimpleMetricsCollector) RecordRequest(method string, statusCode codes.Code, duration time.Duration) {
	c.requests[method]++
	if statusCode != codes.OK {
		c.errors[method]++
	}
	c.duration[method] += duration
}

// GetMetrics returns collected metrics
func (c *SimpleMetricsCollector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"requests": c.requests,
		"errors":   c.errors,
		"duration": c.duration,
	}
}