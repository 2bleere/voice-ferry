package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/2bleere/voice-ferry/pkg/config"
	v1 "github.com/2bleere/voice-ferry/proto/gen/b2bua/v1"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	// Key prefixes for different data types
	RoutingRulesPrefix = "/voice-ferry-c4/routing-rules/"
	ConfigPrefix       = "/voice-ferry-c4/config/"
	SessionPrefix      = "/voice-ferry-c4/sessions/"
)

// Client wraps etcd client with domain-specific methods
type Client struct {
	client *clientv3.Client
	cfg    *config.EtcdConfig
}

// NewClient creates a new etcd client
func NewClient(cfg *config.EtcdConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("etcd is disabled")
	}

	etcdConfig := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout,
	}

	if cfg.Username != "" {
		etcdConfig.Username = cfg.Username
		etcdConfig.Password = cfg.Password
	}

	if cfg.TLS.Enabled {
		// TODO: Add TLS configuration
		log.Printf("Warning: TLS configuration for etcd not implemented yet")
	}

	client, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &Client{
		client: client,
		cfg:    cfg,
	}, nil
}

// Close closes the etcd client connection
func (c *Client) Close() error {
	return c.client.Close()
}

// HealthCheck performs a health check on the etcd cluster
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.DialTimeout)
	defer cancel()

	_, err := c.client.Status(ctx, c.cfg.Endpoints[0])
	return err
}

// StoreRoutingRule stores a routing rule in etcd
func (c *Client) StoreRoutingRule(ctx context.Context, rule *v1.RoutingRule) error {
	key := RoutingRulesPrefix + rule.RuleId

	data, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal routing rule: %w", err)
	}

	_, err = c.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to store routing rule in etcd: %w", err)
	}

	log.Printf("Stored routing rule %s in etcd", rule.RuleId)
	return nil
}

// GetRoutingRule retrieves a routing rule from etcd
func (c *Client) GetRoutingRule(ctx context.Context, ruleID string) (*v1.RoutingRule, error) {
	key := RoutingRulesPrefix + ruleID

	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get routing rule from etcd: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("routing rule %s not found", ruleID)
	}

	var rule v1.RoutingRule
	err = json.Unmarshal(resp.Kvs[0].Value, &rule)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal routing rule: %w", err)
	}

	return &rule, nil
}

// ListRoutingRules retrieves all routing rules from etcd
func (c *Client) ListRoutingRules(ctx context.Context) ([]*v1.RoutingRule, error) {
	resp, err := c.client.Get(ctx, RoutingRulesPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list routing rules from etcd: %w", err)
	}

	var rules []*v1.RoutingRule
	for _, kv := range resp.Kvs {
		var rule v1.RoutingRule
		if err := json.Unmarshal(kv.Value, &rule); err != nil {
			log.Printf("Failed to unmarshal routing rule from key %s: %v", string(kv.Key), err)
			continue
		}
		rules = append(rules, &rule)
	}

	return rules, nil
}

// DeleteRoutingRule deletes a routing rule from etcd
func (c *Client) DeleteRoutingRule(ctx context.Context, ruleID string) error {
	key := RoutingRulesPrefix + ruleID

	_, err := c.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete routing rule from etcd: %w", err)
	}

	log.Printf("Deleted routing rule %s from etcd", ruleID)
	return nil
}

// WatchRoutingRules watches for changes to routing rules
func (c *Client) WatchRoutingRules(ctx context.Context) clientv3.WatchChan {
	return c.client.Watch(ctx, RoutingRulesPrefix, clientv3.WithPrefix())
}

// StoreConfig stores configuration in etcd
func (c *Client) StoreConfig(ctx context.Context, key string, value interface{}) error {
	fullKey := ConfigPrefix + key

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal config value: %w", err)
	}

	_, err = c.client.Put(ctx, fullKey, string(data))
	if err != nil {
		return fmt.Errorf("failed to store config in etcd: %w", err)
	}

	return nil
}

// GetConfig retrieves configuration from etcd
func (c *Client) GetConfig(ctx context.Context, key string, target interface{}) error {
	fullKey := ConfigPrefix + key

	resp, err := c.client.Get(ctx, fullKey)
	if err != nil {
		return fmt.Errorf("failed to get config from etcd: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("config key %s not found", key)
	}

	err = json.Unmarshal(resp.Kvs[0].Value, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config value: %w", err)
	}

	return nil
}

// StoreSession stores session data in etcd
func (c *Client) StoreSession(ctx context.Context, sessionID string, data interface{}) error {
	key := SessionPrefix + sessionID

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Set session with TTL (e.g., 1 hour)
	lease, err := c.client.Grant(ctx, 3600)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	_, err = c.client.Put(ctx, key, string(jsonData), clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("failed to store session in etcd: %w", err)
	}

	return nil
}

// GetSession retrieves session data from etcd
func (c *Client) GetSession(ctx context.Context, sessionID string, target interface{}) error {
	key := SessionPrefix + sessionID

	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get session from etcd: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("session %s not found", sessionID)
	}

	err = json.Unmarshal(resp.Kvs[0].Value, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return nil
}

// DeleteSession deletes session data from etcd
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	key := SessionPrefix + sessionID

	_, err := c.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete session from etcd: %w", err)
	}

	return nil
}

// GetMemberID returns the member ID of this etcd client
func (c *Client) GetMemberID(ctx context.Context) (uint64, error) {
	resp, err := c.client.MemberList(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get member list: %w", err)
	}

	// Return the first member ID for simplicity
	if len(resp.Members) > 0 {
		return resp.Members[0].ID, nil
	}

	return 0, fmt.Errorf("no members found")
}

// ExtractRuleIDFromKey extracts rule ID from an etcd key
func ExtractRuleIDFromKey(key string) string {
	return strings.TrimPrefix(key, RoutingRulesPrefix)
}
