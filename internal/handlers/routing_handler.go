package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/2bleere/voice-ferry/pkg/config"
	"github.com/2bleere/voice-ferry/pkg/etcd"
	"github.com/2bleere/voice-ferry/pkg/routing"
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
)

// RoutingHandler implements the RoutingRuleService
type RoutingHandler struct {
	v1.UnimplementedRoutingRuleServiceServer
	cfg           *config.Config
	routingEngine *routing.Engine
	etcdClient    *etcd.Client
	rules         map[string]*v1.RoutingRule // in-memory cache/fallback
}

// NewRoutingHandler creates a new routing handler
func NewRoutingHandler(cfg *config.Config, routingEngine *routing.Engine) *RoutingHandler {
	handler := &RoutingHandler{
		cfg:           cfg,
		routingEngine: routingEngine,
		rules:         make(map[string]*v1.RoutingRule),
	}

	// Initialize etcd client if enabled
	if cfg.Etcd.Enabled {
		etcdClient, err := etcd.NewClient(&cfg.Etcd)
		if err != nil {
			log.Printf("Failed to initialize etcd client: %v", err)
		} else {
			handler.etcdClient = etcdClient
			// Load existing rules from etcd
			handler.loadRulesFromEtcd()
			// Start watching for changes
			go handler.watchRulesChanges()
		}
	}

	return handler
}

// AddRoutingRule adds a new routing rule
func (h *RoutingHandler) AddRoutingRule(ctx context.Context, req *v1.AddRoutingRuleRequest) (*v1.RoutingRuleResponse, error) {
	log.Printf("Adding routing rule: %s", req.Rule.Name)

	rule := req.Rule
	if rule.RuleId == "" {
		rule.RuleId = generateRuleID()
	}
	rule.CreatedAt = timestamppb.Now()
	rule.UpdatedAt = timestamppb.Now()

	h.rules[rule.RuleId] = rule

	// Add to routing engine
	h.routingEngine.AddRule(rule)

	// Persist to etcd
	if h.etcdClient != nil {
		if err := h.etcdClient.StoreRoutingRule(ctx, rule); err != nil {
			log.Printf("Failed to save routing rule to etcd: %v", err)
		}
	}

	return &v1.RoutingRuleResponse{
		Rule: rule,
	}, nil
}

// GetRoutingRule retrieves a routing rule by ID
func (h *RoutingHandler) GetRoutingRule(ctx context.Context, req *v1.GetRoutingRuleRequest) (*v1.RoutingRuleResponse, error) {
	log.Printf("Getting routing rule: %s", req.RuleId)

	rule, exists := h.rules[req.RuleId]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "routing rule not found")
	}

	return &v1.RoutingRuleResponse{
		Rule: rule,
	}, nil
}

// UpdateRoutingRule updates an existing routing rule
func (h *RoutingHandler) UpdateRoutingRule(ctx context.Context, req *v1.UpdateRoutingRuleRequest) (*v1.RoutingRuleResponse, error) {
	log.Printf("Updating routing rule: %s", req.Rule.RuleId)

	if _, exists := h.rules[req.Rule.RuleId]; !exists {
		return nil, status.Errorf(codes.NotFound, "routing rule not found")
	}

	rule := req.Rule
	rule.UpdatedAt = timestamppb.Now()
	h.rules[rule.RuleId] = rule

	// Update in routing engine
	err := h.routingEngine.UpdateRule(rule)
	if err != nil {
		log.Printf("Failed to update rule in routing engine: %v", err)
	}

	// Update in etcd
	if h.etcdClient != nil {
		if err := h.etcdClient.StoreRoutingRule(ctx, rule); err != nil {
			log.Printf("Failed to update routing rule in etcd: %v", err)
		}
	}

	return &v1.RoutingRuleResponse{
		Rule: rule,
	}, nil
}

// DeleteRoutingRule deletes a routing rule
func (h *RoutingHandler) DeleteRoutingRule(ctx context.Context, req *v1.DeleteRoutingRuleRequest) (*emptypb.Empty, error) {
	log.Printf("Deleting routing rule: %s", req.RuleId)

	if _, exists := h.rules[req.RuleId]; !exists {
		return nil, status.Errorf(codes.NotFound, "routing rule not found")
	}

	delete(h.rules, req.RuleId)

	// Remove from routing engine
	err := h.routingEngine.RemoveRule(req.RuleId)
	if err != nil {
		log.Printf("Failed to remove rule from routing engine: %v", err)
	}

	// Remove from etcd
	if h.etcdClient != nil {
		if err := h.etcdClient.DeleteRoutingRule(ctx, req.RuleId); err != nil {
			log.Printf("Failed to delete routing rule from etcd: %v", err)
		}
	}

	return &emptypb.Empty{}, nil
}

// ListRoutingRules lists all routing rules
func (h *RoutingHandler) ListRoutingRules(ctx context.Context, req *v1.ListRoutingRulesRequest) (*v1.ListRoutingRulesResponse, error) {
	log.Printf("Listing routing rules")

	var rules []*v1.RoutingRule
	for _, rule := range h.rules {
		rules = append(rules, rule)
	}

	return &v1.ListRoutingRulesResponse{
		Rules: rules,
		// TODO: Implement pagination
	}, nil
}

// loadRulesFromEtcd loads routing rules from etcd on startup
func (h *RoutingHandler) loadRulesFromEtcd() {
	ctx := context.Background()
	rules, err := h.etcdClient.ListRoutingRules(ctx)
	if err != nil {
		log.Printf("Failed to load routing rules from etcd: %v", err)
		return
	}

	log.Printf("Loaded %d routing rules from etcd", len(rules))
	for _, rule := range rules {
		h.rules[rule.RuleId] = rule
		h.routingEngine.AddRule(rule)
	}
}

// watchRulesChanges watches for routing rule changes in etcd
func (h *RoutingHandler) watchRulesChanges() {
	ctx := context.Background()
	watchChan := h.etcdClient.WatchRoutingRules(ctx)

	for resp := range watchChan {
		for _, event := range resp.Events {
			ruleID := etcd.ExtractRuleIDFromKey(string(event.Kv.Key))

			switch event.Type {
			case clientv3.EventTypePut:
				// Rule added or updated
				var rule v1.RoutingRule
				if err := json.Unmarshal(event.Kv.Value, &rule); err != nil {
					log.Printf("Failed to unmarshal routing rule from etcd: %v", err)
					continue
				}

				h.rules[ruleID] = &rule
				if err := h.routingEngine.UpdateRule(&rule); err != nil {
					// If update fails, it means rule doesn't exist, so add it
					h.routingEngine.AddRule(&rule)
				}
				log.Printf("Updated routing rule %s from etcd watch", ruleID)

			case clientv3.EventTypeDelete:
				// Rule deleted
				delete(h.rules, ruleID)
				if err := h.routingEngine.RemoveRule(ruleID); err != nil {
					log.Printf("Failed to remove routing rule %s from engine: %v", ruleID, err)
				}
				log.Printf("Deleted routing rule %s from etcd watch", ruleID)
			}
		}
	}
}

// generateRuleID generates a unique rule ID
func generateRuleID() string {
	return fmt.Sprintf("rule-%d", timestamppb.Now().GetSeconds())
}
