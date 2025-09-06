package router

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// ProviderRouter intelligently routes requests to the best available provider
type ProviderRouter struct {
	mu          sync.RWMutex
	providers   map[string]ProviderInfo
	policies    []RoutingPolicy
	costTracker *CostTracker
	metrics     *PerformanceMetrics
	config      *RouterConfig
	
	// Circuit breaker for failed providers
	breakers    map[string]*CircuitBreaker
	
	// Provider health monitoring
	healthCheck *HealthChecker
}

// RouterConfig holds router configuration
type RouterConfig struct {
	EnableCostOptimization bool
	EnableLoadBalancing    bool
	EnableFailover        bool
	MaxRetries            int
	RetryDelay            time.Duration
	HealthCheckInterval   time.Duration
	CostUpdateInterval    time.Duration
}

// DefaultRouterConfig returns default configuration
func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		EnableCostOptimization: true,
		EnableLoadBalancing:    true,
		EnableFailover:        true,
		MaxRetries:            3,
		RetryDelay:            time.Second,
		HealthCheckInterval:   30 * time.Second,
		CostUpdateInterval:    5 * time.Minute,
	}
}

// ProviderInfo holds information about a provider
type ProviderInfo struct {
	Name         string                 `json:"name"`
	Type         domain.ProviderType    `json:"type"`
	Models       []ModelInfo            `json:"models"`
	Capabilities []string               `json:"capabilities"`
	Status       ProviderStatus         `json:"status"`
	Priority     int                    `json:"priority"`
	Metadata     map[string]interface{} `json:"metadata"`
	
	// Performance metrics
	Latency      time.Duration         `json:"latency"`
	SuccessRate  float64               `json:"success_rate"`
	LastUsed     time.Time             `json:"last_used"`
	
	// Cost information
	CostPerToken float64               `json:"cost_per_token"`
	CostPerCall  float64               `json:"cost_per_call"`
	
	// Provider instance
	Provider     interface{}           // Actual provider implementation
}

// ModelInfo holds information about a specific model
type ModelInfo struct {
	Name           string    `json:"name"`
	Type           ModelType `json:"type"`
	ContextLength  int       `json:"context_length"`
	CostPerToken   float64   `json:"cost_per_token"`
	Speed          float64   `json:"speed"` // Tokens per second
	Quality        float64   `json:"quality"` // 0-1 quality score
	Capabilities   []string  `json:"capabilities"`
}

// ModelType defines the type of model
type ModelType string

const (
	ModelTypeLLM       ModelType = "llm"
	ModelTypeEmbedding ModelType = "embedding"
	ModelTypeVision    ModelType = "vision"
	ModelTypeAudio     ModelType = "audio"
)

// ProviderStatus represents provider availability
type ProviderStatus string

const (
	ProviderStatusHealthy   ProviderStatus = "healthy"
	ProviderStatusDegraded  ProviderStatus = "degraded"
	ProviderStatusUnhealthy ProviderStatus = "unhealthy"
	ProviderStatusOffline   ProviderStatus = "offline"
)

// RoutingPolicy defines how requests are routed
type RoutingPolicy interface {
	SelectProvider(ctx context.Context, request *RoutingRequest, providers []ProviderInfo) (*ProviderInfo, error)
	Name() string
}

// RoutingRequest contains request information for routing
type RoutingRequest struct {
	Type          RequestType            `json:"type"`
	Model         string                 `json:"model,omitempty"`
	Requirements  []string               `json:"requirements,omitempty"`
	MaxCost       float64                `json:"max_cost,omitempty"`
	MaxLatency    time.Duration          `json:"max_latency,omitempty"`
	TokenEstimate int                    `json:"token_estimate,omitempty"`
	Priority      int                    `json:"priority"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// RequestType defines the type of request
type RequestType string

const (
	RequestTypeGeneration RequestType = "generation"
	RequestTypeEmbedding  RequestType = "embedding"
	RequestTypeChat       RequestType = "chat"
	RequestTypeCompletion RequestType = "completion"
)

// CostTracker tracks and optimizes costs
type CostTracker struct {
	mu          sync.RWMutex
	usage       map[string]*UsageStats
	budgets     map[string]*Budget
	alerts      []CostAlert
	storage     CostStorage
}

// UsageStats tracks usage statistics
type UsageStats struct {
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	TotalTokens  int64         `json:"total_tokens"`
	TotalCalls   int64         `json:"total_calls"`
	TotalCost    float64       `json:"total_cost"`
	Period       time.Duration `json:"period"`
	LastReset    time.Time     `json:"last_reset"`
}

// Budget defines spending limits
type Budget struct {
	Name         string        `json:"name"`
	Provider     string        `json:"provider,omitempty"`
	Model        string        `json:"model,omitempty"`
	Limit        float64       `json:"limit"`
	Period       BudgetPeriod  `json:"period"`
	Spent        float64       `json:"spent"`
	LastReset    time.Time     `json:"last_reset"`
	AlertPercent float64       `json:"alert_percent"`
}

// BudgetPeriod defines budget period
type BudgetPeriod string

const (
	BudgetPeriodHourly  BudgetPeriod = "hourly"
	BudgetPeriodDaily   BudgetPeriod = "daily"
	BudgetPeriodWeekly  BudgetPeriod = "weekly"
	BudgetPeriodMonthly BudgetPeriod = "monthly"
)

// CostAlert represents a cost alert
type CostAlert struct {
	ID        string    `json:"id"`
	Type      AlertType `json:"type"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
	Budget    *Budget   `json:"budget,omitempty"`
}

// AlertType defines alert types
type AlertType string

const (
	AlertTypeBudgetWarning  AlertType = "budget_warning"
	AlertTypeBudgetExceeded AlertType = "budget_exceeded"
	AlertTypeCostSpike      AlertType = "cost_spike"
)

// PerformanceMetrics tracks provider performance
type PerformanceMetrics struct {
	mu       sync.RWMutex
	metrics  map[string]*ProviderMetrics
	window   time.Duration
}

// ProviderMetrics holds metrics for a provider
type ProviderMetrics struct {
	Provider         string        `json:"provider"`
	RequestCount     int64         `json:"request_count"`
	SuccessCount     int64         `json:"success_count"`
	FailureCount     int64         `json:"failure_count"`
	TotalLatency     time.Duration `json:"total_latency"`
	AverageLatency   time.Duration `json:"average_latency"`
	P95Latency       time.Duration `json:"p95_latency"`
	P99Latency       time.Duration `json:"p99_latency"`
	LastUpdate       time.Time     `json:"last_update"`
	LatencySamples   []time.Duration
}

// CircuitBreaker prevents cascading failures
type CircuitBreaker struct {
	provider      string
	failureCount  int
	lastFailure   time.Time
	state         BreakerState
	timeout       time.Duration
	threshold     int
	mu            sync.RWMutex
}

// BreakerState represents circuit breaker state
type BreakerState string

const (
	BreakerStateClosed   BreakerState = "closed"
	BreakerStateOpen     BreakerState = "open"
	BreakerStateHalfOpen BreakerState = "half_open"
)

// HealthChecker monitors provider health
type HealthChecker struct {
	providers map[string]ProviderInfo
	interval  time.Duration
	checker   func(provider ProviderInfo) error
	mu        sync.RWMutex
}

// CostStorage interface for persisting cost data
type CostStorage interface {
	SaveUsageStats(stats *UsageStats) error
	LoadUsageStats(provider string) (*UsageStats, error)
	SaveBudget(budget *Budget) error
	LoadBudgets() ([]*Budget, error)
	SaveAlert(alert *CostAlert) error
}

// NewProviderRouter creates a new provider router
func NewProviderRouter(config *RouterConfig, storage CostStorage) *ProviderRouter {
	if config == nil {
		config = DefaultRouterConfig()
	}

	router := &ProviderRouter{
		providers: make(map[string]ProviderInfo),
		breakers:  make(map[string]*CircuitBreaker),
		config:    config,
		costTracker: &CostTracker{
			usage:   make(map[string]*UsageStats),
			budgets: make(map[string]*Budget),
			storage: storage,
		},
		metrics: &PerformanceMetrics{
			metrics: make(map[string]*ProviderMetrics),
			window:  time.Hour,
		},
	}

	// Initialize routing policies
	router.initializePolicies()

	// Start background tasks
	if config.HealthCheckInterval > 0 {
		go router.startHealthChecker()
	}
	if config.CostUpdateInterval > 0 {
		go router.startCostUpdater()
	}

	return router
}

// initializePolicies sets up routing policies
func (r *ProviderRouter) initializePolicies() {
	r.policies = []RoutingPolicy{
		&CostOptimizedPolicy{},
		&LatencyOptimizedPolicy{},
		&QualityOptimizedPolicy{},
		&LoadBalancedPolicy{},
		&FallbackPolicy{},
	}
}

// RegisterProvider registers a new provider
func (r *ProviderRouter) RegisterProvider(info ProviderInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if info.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	r.providers[info.Name] = info

	// Initialize circuit breaker
	r.breakers[info.Name] = &CircuitBreaker{
		provider:  info.Name,
		state:     BreakerStateClosed,
		timeout:   30 * time.Second,
		threshold: 5,
	}

	// Initialize metrics
	r.metrics.initProvider(info.Name)

	return nil
}

// Route selects the best provider for a request
func (r *ProviderRouter) Route(ctx context.Context, request *RoutingRequest) (*ProviderInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get available providers
	available := r.getAvailableProviders(request)
	if len(available) == 0 {
		return nil, fmt.Errorf("no available providers for request")
	}

	// Check budget constraints
	if r.config.EnableCostOptimization {
		available = r.filterByBudget(available, request)
	}

	// Select policy based on request
	policy := r.selectPolicy(request)

	// Select provider using policy
	provider, err := policy.SelectProvider(ctx, request, available)
	if err != nil {
		return nil, fmt.Errorf("failed to select provider: %w", err)
	}

	// Record selection
	r.recordProviderSelection(provider, request)

	return provider, nil
}

// getAvailableProviders returns providers that can handle the request
func (r *ProviderRouter) getAvailableProviders(request *RoutingRequest) []ProviderInfo {
	var available []ProviderInfo

	for _, provider := range r.providers {
		// Check if provider is healthy
		if provider.Status == ProviderStatusOffline || provider.Status == ProviderStatusUnhealthy {
			continue
		}

		// Check circuit breaker
		if breaker := r.breakers[provider.Name]; breaker != nil {
			if !breaker.CanRequest() {
				continue
			}
		}

		// Check capabilities
		if !r.providerMeetsRequirements(provider, request) {
			continue
		}

		available = append(available, provider)
	}

	return available
}

// providerMeetsRequirements checks if provider meets request requirements
func (r *ProviderRouter) providerMeetsRequirements(provider ProviderInfo, request *RoutingRequest) bool {
	// Check model availability
	if request.Model != "" {
		hasModel := false
		for _, model := range provider.Models {
			if model.Name == request.Model {
				hasModel = true
				break
			}
		}
		if !hasModel {
			return false
		}
	}

	// Check required capabilities
	for _, req := range request.Requirements {
		hasCapability := false
		for _, cap := range provider.Capabilities {
			if cap == req {
				hasCapability = true
				break
			}
		}
		if !hasCapability {
			return false
		}
	}

	return true
}

// filterByBudget filters providers based on budget constraints
func (r *ProviderRouter) filterByBudget(providers []ProviderInfo, request *RoutingRequest) []ProviderInfo {
	var filtered []ProviderInfo

	for _, provider := range providers {
		// Estimate cost
		estimatedCost := r.estimateCost(provider, request)

		// Check request budget
		if request.MaxCost > 0 && estimatedCost > request.MaxCost {
			continue
		}

		// Check global budget
		if !r.costTracker.CheckBudget(provider.Name, estimatedCost) {
			continue
		}

		filtered = append(filtered, provider)
	}

	return filtered
}

// estimateCost estimates the cost for a request
func (r *ProviderRouter) estimateCost(provider ProviderInfo, request *RoutingRequest) float64 {
	if request.TokenEstimate > 0 {
		return float64(request.TokenEstimate) * provider.CostPerToken
	}
	return provider.CostPerCall
}

// selectPolicy selects the appropriate routing policy
func (r *ProviderRouter) selectPolicy(request *RoutingRequest) RoutingPolicy {
	// Simple policy selection - can be enhanced
	if request.MaxCost > 0 && r.config.EnableCostOptimization {
		return &CostOptimizedPolicy{}
	}
	if request.MaxLatency > 0 {
		return &LatencyOptimizedPolicy{}
	}
	if r.config.EnableLoadBalancing {
		return &LoadBalancedPolicy{}
	}
	return &FallbackPolicy{}
}

// recordProviderSelection records metrics for provider selection
func (r *ProviderRouter) recordProviderSelection(provider *ProviderInfo, request *RoutingRequest) {
	provider.LastUsed = time.Now()
	r.metrics.RecordRequest(provider.Name)
}

// ExecuteWithFallback executes a request with automatic fallback
func (r *ProviderRouter) ExecuteWithFallback(ctx context.Context, request *RoutingRequest, 
	executor func(provider *ProviderInfo) error) error {
	
	maxRetries := r.config.MaxRetries
	if !r.config.EnableFailover {
		maxRetries = 1
	}

	var lastErr error
	triedProviders := make(map[string]bool)

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Select provider
		provider, err := r.Route(ctx, request)
		if err != nil {
			return fmt.Errorf("routing failed: %w", err)
		}

		// Mark provider as tried (but don't skip if it's the only one)
		triedProviders[provider.Name] = true

		// Execute request
		startTime := time.Now()
		err = executor(provider)
		duration := time.Since(startTime)

		if err == nil {
			// Success - record metrics
			r.recordSuccess(provider.Name, duration)
			return nil
		}

		// Failure - record and handle
		lastErr = err
		r.recordFailure(provider.Name, err)

		// Wait before retry
		if attempt < maxRetries-1 {
			time.Sleep(r.config.RetryDelay)
		}
	}

	return fmt.Errorf("all providers failed: %w", lastErr)
}

// recordSuccess records successful request
func (r *ProviderRouter) recordSuccess(provider string, duration time.Duration) {
	r.metrics.RecordSuccess(provider, duration)
	if breaker := r.breakers[provider]; breaker != nil {
		breaker.RecordSuccess()
	}
}

// recordFailure records failed request
func (r *ProviderRouter) recordFailure(provider string, err error) {
	r.metrics.RecordFailure(provider)
	if breaker := r.breakers[provider]; breaker != nil {
		breaker.RecordFailure()
	}
}

// startHealthChecker starts background health checking
func (r *ProviderRouter) startHealthChecker() {
	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.checkProviderHealth()
	}
}

// checkProviderHealth checks health of all providers
func (r *ProviderRouter) checkProviderHealth() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, provider := range r.providers {
		// Simple health check - can be enhanced
		// This would actually ping the provider
		provider.Status = ProviderStatusHealthy
		r.providers[name] = provider
	}
}

// startCostUpdater starts background cost tracking
func (r *ProviderRouter) startCostUpdater() {
	ticker := time.NewTicker(r.config.CostUpdateInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.costTracker.UpdateCosts()
	}
}

// CircuitBreaker methods

// CanRequest checks if requests can be made
func (cb *CircuitBreaker) CanRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case BreakerStateClosed:
		return true
	case BreakerStateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = BreakerStateHalfOpen
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case BreakerStateHalfOpen:
		return true
	}
	return false
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	if cb.state == BreakerStateHalfOpen {
		cb.state = BreakerStateClosed
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailure = time.Now()

	if cb.threshold > 0 && cb.failureCount >= cb.threshold {
		cb.state = BreakerStateOpen
	}
}

// CostTracker methods

// CheckBudget checks if a cost is within budget
func (ct *CostTracker) CheckBudget(provider string, cost float64) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	for _, budget := range ct.budgets {
		if budget.Provider == "" || budget.Provider == provider {
			if budget.Spent+cost > budget.Limit {
				return false
			}
		}
	}
	return true
}

// RecordCost records a cost
func (ct *CostTracker) RecordCost(provider string, cost float64, tokens int64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	stats, exists := ct.usage[provider]
	if !exists {
		stats = &UsageStats{
			Provider:  provider,
			LastReset: time.Now(),
		}
		ct.usage[provider] = stats
	}

	stats.TotalCost += cost
	stats.TotalTokens += tokens
	stats.TotalCalls++

	// Update budgets
	for _, budget := range ct.budgets {
		if budget.Provider == "" || budget.Provider == provider {
			budget.Spent += cost
			ct.checkBudgetAlert(budget)
		}
	}

	// Save to storage
	if ct.storage != nil {
		ct.storage.SaveUsageStats(stats)
	}
}

// checkBudgetAlert checks if budget alert should be triggered
func (ct *CostTracker) checkBudgetAlert(budget *Budget) {
	percentUsed := (budget.Spent / budget.Limit) * 100

	if percentUsed >= 100 {
		ct.createAlert(AlertTypeBudgetExceeded, fmt.Sprintf("Budget %s exceeded: $%.2f of $%.2f", 
			budget.Name, budget.Spent, budget.Limit), budget)
	} else if percentUsed >= budget.AlertPercent {
		ct.createAlert(AlertTypeBudgetWarning, fmt.Sprintf("Budget %s at %.0f%%: $%.2f of $%.2f", 
			budget.Name, percentUsed, budget.Spent, budget.Limit), budget)
	}
}

// createAlert creates a cost alert
func (ct *CostTracker) createAlert(alertType AlertType, message string, budget *Budget) {
	alert := CostAlert{
		Type:      alertType,
		Message:   message,
		Severity:  "warning",
		Timestamp: time.Now(),
		Budget:    budget,
	}

	ct.alerts = append(ct.alerts, alert)

	if ct.storage != nil {
		ct.storage.SaveAlert(&alert)
	}
}

// UpdateCosts updates cost tracking
func (ct *CostTracker) UpdateCosts() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Reset budgets based on period
	for _, budget := range ct.budgets {
		if ct.shouldResetBudget(budget) {
			budget.Spent = 0
			budget.LastReset = time.Now()
		}
	}
}

// shouldResetBudget checks if budget should be reset
func (ct *CostTracker) shouldResetBudget(budget *Budget) bool {
	elapsed := time.Since(budget.LastReset)

	switch budget.Period {
	case BudgetPeriodHourly:
		return elapsed >= time.Hour
	case BudgetPeriodDaily:
		return elapsed >= 24*time.Hour
	case BudgetPeriodWeekly:
		return elapsed >= 7*24*time.Hour
	case BudgetPeriodMonthly:
		return elapsed >= 30*24*time.Hour
	}

	return false
}

// PerformanceMetrics methods

// initProvider initializes metrics for a provider
func (pm *PerformanceMetrics) initProvider(provider string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.metrics[provider] = &ProviderMetrics{
		Provider:       provider,
		LastUpdate:     time.Now(),
		LatencySamples: make([]time.Duration, 0, 1000),
	}
}

// RecordRequest records a request
func (pm *PerformanceMetrics) RecordRequest(provider string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if metrics, exists := pm.metrics[provider]; exists {
		metrics.RequestCount++
	}
}

// RecordSuccess records a successful request
func (pm *PerformanceMetrics) RecordSuccess(provider string, latency time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if metrics, exists := pm.metrics[provider]; exists {
		metrics.SuccessCount++
		metrics.TotalLatency += latency
		metrics.LatencySamples = append(metrics.LatencySamples, latency)
		
		// Keep only recent samples
		if len(metrics.LatencySamples) > 1000 {
			metrics.LatencySamples = metrics.LatencySamples[len(metrics.LatencySamples)-1000:]
		}

		// Update statistics
		pm.updateLatencyStats(metrics)
	}
}

// RecordFailure records a failed request
func (pm *PerformanceMetrics) RecordFailure(provider string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if metrics, exists := pm.metrics[provider]; exists {
		metrics.FailureCount++
	}
}

// updateLatencyStats updates latency statistics
func (pm *PerformanceMetrics) updateLatencyStats(metrics *ProviderMetrics) {
	if len(metrics.LatencySamples) == 0 {
		return
	}

	// Calculate average
	if metrics.SuccessCount > 0 {
		metrics.AverageLatency = metrics.TotalLatency / time.Duration(metrics.SuccessCount)
	}

	// Calculate percentiles
	sorted := make([]time.Duration, len(metrics.LatencySamples))
	copy(sorted, metrics.LatencySamples)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	p95Index := int(math.Ceil(float64(len(sorted)) * 0.95)) - 1
	p99Index := int(math.Ceil(float64(len(sorted)) * 0.99)) - 1

	if p95Index >= 0 && p95Index < len(sorted) {
		metrics.P95Latency = sorted[p95Index]
	}
	if p99Index >= 0 && p99Index < len(sorted) {
		metrics.P99Latency = sorted[p99Index]
	}

	metrics.LastUpdate = time.Now()
}

// Routing Policy Implementations

// CostOptimizedPolicy selects the cheapest provider
type CostOptimizedPolicy struct{}

func (p *CostOptimizedPolicy) Name() string { return "cost_optimized" }

func (p *CostOptimizedPolicy) SelectProvider(ctx context.Context, request *RoutingRequest, 
	providers []ProviderInfo) (*ProviderInfo, error) {
	
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Make a copy to avoid modifying the original slice
	providersCopy := make([]ProviderInfo, len(providers))
	copy(providersCopy, providers)

	// Sort by cost
	sort.Slice(providersCopy, func(i, j int) bool {
		return providersCopy[i].CostPerToken < providersCopy[j].CostPerToken
	})

	return &providersCopy[0], nil
}

// LatencyOptimizedPolicy selects the fastest provider
type LatencyOptimizedPolicy struct{}

func (p *LatencyOptimizedPolicy) Name() string { return "latency_optimized" }

func (p *LatencyOptimizedPolicy) SelectProvider(ctx context.Context, request *RoutingRequest,
	providers []ProviderInfo) (*ProviderInfo, error) {
	
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Make a copy to avoid modifying the original slice
	providersCopy := make([]ProviderInfo, len(providers))
	copy(providersCopy, providers)

	// Sort by latency
	sort.Slice(providersCopy, func(i, j int) bool {
		return providersCopy[i].Latency < providersCopy[j].Latency
	})

	return &providersCopy[0], nil
}

// QualityOptimizedPolicy selects the highest quality provider
type QualityOptimizedPolicy struct{}

func (p *QualityOptimizedPolicy) Name() string { return "quality_optimized" }

func (p *QualityOptimizedPolicy) SelectProvider(ctx context.Context, request *RoutingRequest,
	providers []ProviderInfo) (*ProviderInfo, error) {
	
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Make a copy to avoid modifying the original slice
	providersCopy := make([]ProviderInfo, len(providers))
	copy(providersCopy, providers)

	// Sort by success rate and priority
	sort.Slice(providersCopy, func(i, j int) bool {
		if providersCopy[i].SuccessRate == providersCopy[j].SuccessRate {
			return providersCopy[i].Priority > providersCopy[j].Priority
		}
		return providersCopy[i].SuccessRate > providersCopy[j].SuccessRate
	})

	return &providersCopy[0], nil
}

// LoadBalancedPolicy distributes load across providers
type LoadBalancedPolicy struct {
	lastIndex int
	mu        sync.Mutex
}

func (p *LoadBalancedPolicy) Name() string { return "load_balanced" }

func (p *LoadBalancedPolicy) SelectProvider(ctx context.Context, request *RoutingRequest,
	providers []ProviderInfo) (*ProviderInfo, error) {
	
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Round-robin selection
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastIndex = (p.lastIndex + 1) % len(providers)
	return &providers[p.lastIndex], nil
}

// FallbackPolicy provides simple fallback selection
type FallbackPolicy struct{}

func (p *FallbackPolicy) Name() string { return "fallback" }

func (p *FallbackPolicy) SelectProvider(ctx context.Context, request *RoutingRequest,
	providers []ProviderInfo) (*ProviderInfo, error) {
	
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Return first available
	return &providers[0], nil
}