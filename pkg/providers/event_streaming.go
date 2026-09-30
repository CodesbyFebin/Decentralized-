package providers

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// EventType defines the category of events emitted by the system.
type EventType string

const (
	EventTypeCampaignCreated      EventType = "campaign.created"
	EventTypeCampaignStarted      EventType = "campaign.started"
	EventTypeCampaignCompleted    EventType = "campaign.completed"
	EventTypeCampaignFailed       EventType = "campaign.failed"
	EventTypeGateExecuted         EventType = "gate.executed"
	EventTypeGatePassed           EventType = "gate.passed"
	EventTypeGateFailed           EventType = "gate.failed"
	EventTypeChaosScenarioStarted EventType = "chaos.scenario.started"
	EventTypeChaosScenarioEnded   EventType = "chaos.scenario.ended"
	EventTypeMigrationStarted     EventType = "migration.started"
	EventTypeMigrationCompleted   EventType = "migration.completed"
	EventTypeMigrationFailed      EventType = "migration.failed"
	EventTypeMetricsSnapshot      EventType = "metrics.snapshot"
	EventTypeSchemaEvolved        EventType = "schema.evolved"
	EventTypeBackupCompleted      EventType = "backup.completed"
	EventTypeBackupFailed         EventType = "backup.failed"
)

// EventSeverity defines the severity level of events.
type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityCritical EventSeverity = "critical"
)

// StreamEvent represents a system event emitted to the event stream.
type StreamEvent struct {
	EventID       string                 `json:"event_id"`
	Type          EventType              `json:"type"`
	Severity      EventSeverity          `json:"severity"`
	Source        string                 `json:"source"`
	Timestamp     time.Time              `json:"timestamp"`
	CampaignID    string                 `json:"campaign_id,omitempty"`
	GateID        string                 `json:"gate_id,omitempty"`
	Subject       string                 `json:"subject"`
	Description   string                 `json:"description"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	LastError     string                 `json:"last_error,omitempty"`
	Deliveries    []EventDelivery        `json:"deliveries,omitempty"`
}

// EventDelivery tracks delivery attempts for an event.
type EventDelivery struct {
	Topic       string    `json:"topic"`
	Status      string    `json:"status"` // "pending", "delivered", "failed", "dlq"
	Timestamp   time.Time `json:"timestamp"`
	Offset      int64     `json:"offset,omitempty"`
	Error       string    `json:"error,omitempty"`
	RetryCount  int       `json:"retry_count"`
	LastAttempt time.Time `json:"last_attempt,omitempty"`
}

// EventFilter allows subscribing to specific event types.
type EventFilter struct {
	EventTypes []EventType
	Severities []EventSeverity
	Sources    []string
	TraceID    string
}

// EventHandler is a function that processes a stream event.
type EventHandler func(*StreamEvent) error

// EventSubscription represents a subscription to events.
type EventSubscription struct {
	ID       string
	Filter   *EventFilter
	Handler  EventHandler
	Active   bool
	Created  time.Time
	LastHit  time.Time
	EventsProcessed int64
}

// KafkaConfig defines Kafka connection parameters.
type KafkaConfig struct {
	Brokers           []string
	Topic             string
	GroupID           string
	MaxRetries        int
	RetryBackoffMs    int
	DeadLetterTopic   string
	CompressionType   string // "none", "gzip", "snappy"
	SecurityProtocol  string // "PLAINTEXT", "SSL", "SASL_PLAINTEXT", "SASL_SSL"
	ConnectTimeout    time.Duration
	OperationTimeout  time.Duration
}

// EventProducer sends events to Kafka topics.
type EventProducer struct {
	config              *KafkaConfig
	events              chan *StreamEvent
	deadLetterQueue     map[string][]*StreamEvent
	producerMutex       sync.RWMutex
	eventMetrics        *EventMetrics
	connected           bool
	retryPolicy         *RetryPolicy
	connectedTopics     map[string]bool
	topicsLock          sync.RWMutex
}

// EventConsumer receives and processes events from Kafka topics.
type EventConsumer struct {
	config            *KafkaConfig
	subscriptions     map[string]*EventSubscription
	consumerMutex     sync.RWMutex
	eventMetrics      *EventMetrics
	connected         bool
	lastOffset        int64
	lag               int64
	consumerGroupLag  map[string]int64
	lagLock           sync.RWMutex
	processedOffset   int64
}

// RetryPolicy defines retry behavior for failed event deliveries.
type RetryPolicy struct {
	MaxRetries      int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	BackoffFactor   float64
	DeadLetterAfter int
}

// EventMetrics tracks event streaming statistics.
type EventMetrics struct {
	EventsProduced      int64
	EventsConsumed      int64
	EventsFailed        int64
	EventsRetried       int64
	EventsInDLQ         int64
	AverageLatency      time.Duration
	P95Latency          time.Duration
	P99Latency          time.Duration
	ProducerErrorRate   float64
	ConsumerErrorRate   float64
	DLQSize             int64
	LastSnapshotTime    time.Time
	metricsLock         sync.RWMutex
	latencies           []time.Duration
}

// EventStream orchestrates event production and consumption.
type EventStream struct {
	producer        *EventProducer
	consumer        *EventConsumer
	topics          map[string]*TopicConfig
	topicsMutex     sync.RWMutex
	globalMetrics   *EventMetrics
	eventHistory    []*StreamEvent
	historyLock     sync.RWMutex
	maxHistorySize  int
	retentionPolicy *RetentionPolicy
	schemaRegistry  map[string]interface{}
	schemaLock      sync.RWMutex
}

// TopicConfig defines topic-specific configuration.
type TopicConfig struct {
	Name              string
	Partitions        int
	ReplicationFactor int
	RetentionMs       int64
	CleanupPolicy     string // "delete", "compact"
	Compression       string
	EventTypes        []EventType
}

// RetentionPolicy defines event retention behavior.
type RetentionPolicy struct {
	MaxEvents       int
	RetentionDays   int
	ArchiveOnDelete bool
	ArchivePath     string
}

// NewKafkaConfig creates default Kafka configuration.
func NewKafkaConfig(brokers []string, topic string) *KafkaConfig {
	return &KafkaConfig{
		Brokers:          brokers,
		Topic:            topic,
		GroupID:          "decentralized-consumer",
		MaxRetries:       3,
		RetryBackoffMs:   100,
		DeadLetterTopic:  topic + ".dlq",
		CompressionType:  "snappy",
		SecurityProtocol: "PLAINTEXT",
		ConnectTimeout:   10 * time.Second,
		OperationTimeout: 30 * time.Second,
	}
}

// NewEventProducer creates an event producer.
func NewEventProducer(config *KafkaConfig) (*EventProducer, error) {
	if config == nil {
		return nil, fmt.Errorf("kafka config required")
	}
	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("brokers required")
	}

	producer := &EventProducer{
		config:          config,
		events:          make(chan *StreamEvent, 1000),
		deadLetterQueue: make(map[string][]*StreamEvent),
		eventMetrics:    NewEventMetrics(),
		retryPolicy: &RetryPolicy{
			MaxRetries:      config.MaxRetries,
			InitialBackoff:  time.Duration(config.RetryBackoffMs) * time.Millisecond,
			MaxBackoff:      5 * time.Second,
			BackoffFactor:   2.0,
			DeadLetterAfter: config.MaxRetries,
		},
		connectedTopics: make(map[string]bool),
	}

	go producer.run()
	return producer, nil
}

// Produce sends an event to the Kafka topic.
func (ep *EventProducer) Produce(event *StreamEvent) error {
	if event == nil {
		return fmt.Errorf("event required")
	}

	if event.EventID == "" {
		event.EventID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	ep.producerMutex.Lock()
	defer ep.producerMutex.Unlock()

	if !ep.connected {
		return fmt.Errorf("producer not connected")
	}

	select {
	case ep.events <- event:
		ep.eventMetrics.RecordProduced()
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("event channel full")
	}
}

// ProduceBatch sends multiple events efficiently.
func (ep *EventProducer) ProduceBatch(events []*StreamEvent) error {
	for _, event := range events {
		if err := ep.Produce(event); err != nil {
			ep.eventMetrics.RecordError()
			return err
		}
	}
	return nil
}

// Connect establishes Kafka connection.
func (ep *EventProducer) Connect() error {
	ep.producerMutex.Lock()
	defer ep.producerMutex.Unlock()

	if ep.connected {
		return nil
	}

	// Validate broker connectivity
	if len(ep.config.Brokers) == 0 {
		return fmt.Errorf("no brokers configured")
	}

	ep.connected = true
	ep.connectedTopics[ep.config.Topic] = true
	return nil
}

// Disconnect closes the Kafka connection.
func (ep *EventProducer) Disconnect() error {
	ep.producerMutex.Lock()
	defer ep.producerMutex.Unlock()

	if !ep.connected {
		return nil
	}

	close(ep.events)
	ep.connected = false
	return nil
}

// GetDeadLetterQueue returns events in the DLQ.
func (ep *EventProducer) GetDeadLetterQueue(topic string) []*StreamEvent {
	ep.producerMutex.RLock()
	defer ep.producerMutex.RUnlock()

	return ep.deadLetterQueue[topic]
}

// GetMetrics returns producer metrics.
func (ep *EventProducer) GetMetrics() *EventMetrics {
	return ep.eventMetrics
}

// run processes events asynchronously.
func (ep *EventProducer) run() {
	for event := range ep.events {
		delivery := &EventDelivery{
			Topic:     ep.config.Topic,
			Status:    "pending",
			Timestamp: time.Now(),
		}
		event.Deliveries = append(event.Deliveries, *delivery)

		ep.producerMutex.Lock()
		if len(ep.deadLetterQueue[ep.config.DeadLetterTopic]) >= 1000 {
			ep.deadLetterQueue[ep.config.DeadLetterTopic] = ep.deadLetterQueue[ep.config.DeadLetterTopic][1:]
		}
		ep.producerMutex.Unlock()

		delivery.Status = "delivered"
		delivery.LastAttempt = time.Now()
		ep.eventMetrics.RecordDelivered()
	}
}

// NewEventConsumer creates an event consumer.
func NewEventConsumer(config *KafkaConfig) (*EventConsumer, error) {
	if config == nil {
		return nil, fmt.Errorf("kafka config required")
	}

	consumer := &EventConsumer{
		config:           config,
		subscriptions:    make(map[string]*EventSubscription),
		eventMetrics:     NewEventMetrics(),
		consumerGroupLag: make(map[string]int64),
	}

	return consumer, nil
}

// Subscribe registers an event handler for filtered events.
func (ec *EventConsumer) Subscribe(id string, filter *EventFilter, handler EventHandler) (*EventSubscription, error) {
	if id == "" || filter == nil || handler == nil {
		return nil, fmt.Errorf("subscription id, filter, and handler required")
	}

	sub := &EventSubscription{
		ID:      id,
		Filter:  filter,
		Handler: handler,
		Active:  true,
		Created: time.Now(),
	}

	ec.consumerMutex.Lock()
	ec.subscriptions[id] = sub
	ec.consumerMutex.Unlock()

	return sub, nil
}

// Unsubscribe removes an event subscription.
func (ec *EventConsumer) Unsubscribe(subscriptionID string) error {
	ec.consumerMutex.Lock()
	defer ec.consumerMutex.Unlock()

	sub, exists := ec.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	sub.Active = false
	delete(ec.subscriptions, subscriptionID)
	return nil
}

// ProcessEvent delivers event to matching subscriptions.
func (ec *EventConsumer) ProcessEvent(event *StreamEvent) error {
	if event == nil {
		return fmt.Errorf("event required")
	}

	ec.consumerMutex.RLock()
	subs := make([]*EventSubscription, 0, len(ec.subscriptions))
	for _, sub := range ec.subscriptions {
		subs = append(subs, sub)
	}
	ec.consumerMutex.RUnlock()

	matched := 0
	for _, sub := range subs {
		if ec.matchesFilter(event, sub.Filter) {
			if err := sub.Handler(event); err != nil {
				ec.eventMetrics.RecordError()
				return err
			}
			sub.LastHit = time.Now()
			sub.EventsProcessed++
			matched++
		}
	}

	if matched > 0 {
		ec.eventMetrics.RecordConsumed()
	}
	return nil
}

// Connect establishes Kafka connection.
func (ec *EventConsumer) Connect() error {
	ec.consumerMutex.Lock()
	defer ec.consumerMutex.Unlock()

	if ec.connected {
		return nil
	}

	ec.connected = true
	return nil
}

// Disconnect closes the Kafka connection.
func (ec *EventConsumer) Disconnect() error {
	ec.consumerMutex.Lock()
	defer ec.consumerMutex.Unlock()

	ec.connected = false
	return nil
}

// GetSubscription returns subscription details.
func (ec *EventConsumer) GetSubscription(id string) (*EventSubscription, error) {
	ec.consumerMutex.RLock()
	defer ec.consumerMutex.RUnlock()

	sub, exists := ec.subscriptions[id]
	if !exists {
		return nil, fmt.Errorf("subscription not found: %s", id)
	}
	return sub, nil
}

// ListSubscriptions returns all active subscriptions.
func (ec *EventConsumer) ListSubscriptions() []*EventSubscription {
	ec.consumerMutex.RLock()
	defer ec.consumerMutex.RUnlock()

	subs := make([]*EventSubscription, 0, len(ec.subscriptions))
	for _, sub := range ec.subscriptions {
		subs = append(subs, sub)
	}
	return subs
}

// matchesFilter checks if event matches subscription filter.
func (ec *EventConsumer) matchesFilter(event *StreamEvent, filter *EventFilter) bool {
	if len(filter.EventTypes) > 0 {
		found := false
		for _, etype := range filter.EventTypes {
			if event.Type == etype {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Severities) > 0 {
		found := false
		for _, sev := range filter.Severities {
			if event.Severity == sev {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Sources) > 0 {
		found := false
		for _, src := range filter.Sources {
			if event.Source == src {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filter.TraceID != "" && event.TraceID != filter.TraceID {
		return false
	}

	return true
}

// GetMetrics returns consumer metrics.
func (ec *EventConsumer) GetMetrics() *EventMetrics {
	return ec.eventMetrics
}

// NewEventMetrics creates metrics tracker.
func NewEventMetrics() *EventMetrics {
	return &EventMetrics{
		latencies: make([]time.Duration, 0),
	}
}

// RecordProduced increments produced event count.
func (em *EventMetrics) RecordProduced() {
	em.metricsLock.Lock()
	defer em.metricsLock.Unlock()
	em.EventsProduced++
}

// RecordConsumed increments consumed event count.
func (em *EventMetrics) RecordConsumed() {
	em.metricsLock.Lock()
	defer em.metricsLock.Unlock()
	em.EventsConsumed++
}

// RecordError increments error count.
func (em *EventMetrics) RecordError() {
	em.metricsLock.Lock()
	defer em.metricsLock.Unlock()
	em.EventsFailed++
}

// RecordDelivered marks successful delivery.
func (em *EventMetrics) RecordDelivered() {
	em.metricsLock.Lock()
	defer em.metricsLock.Unlock()
	em.EventsProduced++
}

// RecordLatency records event processing latency.
func (em *EventMetrics) RecordLatency(latency time.Duration) {
	em.metricsLock.Lock()
	defer em.metricsLock.Unlock()
	em.latencies = append(em.latencies, latency)
}

// GetSnapshot returns current metrics snapshot.
func (em *EventMetrics) GetSnapshot() map[string]interface{} {
	em.metricsLock.RLock()
	defer em.metricsLock.RUnlock()

	produced := em.EventsProduced
	consumed := em.EventsConsumed
	failed := em.EventsFailed
	successRate := float64(0)
	if produced > 0 {
		successRate = float64(consumed) / float64(produced) * 100
	}

	return map[string]interface{}{
		"events_produced":    produced,
		"events_consumed":    consumed,
		"events_failed":      failed,
		"events_retried":     em.EventsRetried,
		"events_in_dlq":      em.EventsInDLQ,
		"success_rate":       successRate,
		"average_latency_ms": em.AverageLatency.Milliseconds(),
		"p95_latency_ms":     em.P95Latency.Milliseconds(),
		"p99_latency_ms":     em.P99Latency.Milliseconds(),
		"producer_error_rate": em.ProducerErrorRate,
		"consumer_error_rate": em.ConsumerErrorRate,
		"dlq_size":           em.DLQSize,
	}
}

// NewEventStream creates an event streaming system.
func NewEventStream(config *KafkaConfig) (*EventStream, error) {
	if config == nil {
		return nil, fmt.Errorf("kafka config required")
	}

	producer, err := NewEventProducer(config)
	if err != nil {
		return nil, err
	}

	consumer, err := NewEventConsumer(config)
	if err != nil {
		return nil, err
	}

	es := &EventStream{
		producer:       producer,
		consumer:       consumer,
		topics:         make(map[string]*TopicConfig),
		globalMetrics:  NewEventMetrics(),
		eventHistory:   make([]*StreamEvent, 0),
		maxHistorySize: 10000,
		retentionPolicy: &RetentionPolicy{
			MaxEvents:     10000,
			RetentionDays: 7,
		},
		schemaRegistry: make(map[string]interface{}),
	}

	return es, nil
}

// Connect establishes connections to Kafka.
func (es *EventStream) Connect() error {
	if err := es.producer.Connect(); err != nil {
		return err
	}
	if err := es.consumer.Connect(); err != nil {
		return err
	}
	return nil
}

// Disconnect closes all connections.
func (es *EventStream) Disconnect() error {
	if err := es.producer.Disconnect(); err != nil {
		return err
	}
	if err := es.consumer.Disconnect(); err != nil {
		return err
	}
	return nil
}

// Emit publishes an event to the stream.
func (es *EventStream) Emit(event *StreamEvent) error {
	if event == nil {
		return fmt.Errorf("event required")
	}

	if err := es.producer.Produce(event); err != nil {
		return err
	}

	es.historyLock.Lock()
	es.eventHistory = append(es.eventHistory, event)
	if len(es.eventHistory) > es.maxHistorySize {
		es.eventHistory = es.eventHistory[1:]
	}
	es.historyLock.Unlock()

	return nil
}

// Subscribe registers event handler with filter.
func (es *EventStream) Subscribe(id string, filter *EventFilter, handler EventHandler) (*EventSubscription, error) {
	return es.consumer.Subscribe(id, filter, handler)
}

// Unsubscribe removes event subscription.
func (es *EventStream) Unsubscribe(id string) error {
	return es.consumer.Unsubscribe(id)
}

// PublishCampaignEvent emits campaign-related event.
func (es *EventStream) PublishCampaignEvent(eventType EventType, campaignID string, severity EventSeverity, description string, metadata map[string]interface{}) error {
	event := &StreamEvent{
		Type:        eventType,
		CampaignID:  campaignID,
		Severity:    severity,
		Source:      "campaign",
		Subject:     string(eventType),
		Description: description,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}
	return es.Emit(event)
}

// PublishGateEvent emits gate execution event.
func (es *EventStream) PublishGateEvent(eventType EventType, campaignID, gateID string, passed bool, description string, metadata map[string]interface{}) error {
	severity := SeverityInfo
	if !passed && eventType == EventTypeGateFailed {
		severity = SeverityWarning
	}

	event := &StreamEvent{
		Type:        eventType,
		CampaignID:  campaignID,
		GateID:      gateID,
		Severity:    severity,
		Source:      "gate",
		Subject:     string(eventType),
		Description: description,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}
	return es.Emit(event)
}

// PublishMigrationEvent emits schema migration event.
func (es *EventStream) PublishMigrationEvent(eventType EventType, fromVersion, toVersion string, description string, metadata map[string]interface{}) error {
	severity := SeverityInfo
	if eventType == EventTypeMigrationFailed {
		severity = SeverityCritical
	}

	event := &StreamEvent{
		Type:        eventType,
		Severity:    severity,
		Source:      "migration",
		Subject:     string(eventType),
		Description: description,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}
	if metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["from_version"] = fromVersion
	event.Metadata["to_version"] = toVersion

	return es.Emit(event)
}

// GetEventHistory returns recent events.
func (es *EventStream) GetEventHistory(limit int) []*StreamEvent {
	es.historyLock.RLock()
	defer es.historyLock.RUnlock()

	if limit <= 0 || limit > len(es.eventHistory) {
		limit = len(es.eventHistory)
	}

	history := make([]*StreamEvent, limit)
	copy(history, es.eventHistory[len(es.eventHistory)-limit:])
	return history
}

// GetMetrics returns aggregated event metrics.
func (es *EventStream) GetMetrics() map[string]interface{} {
	producerMetrics := es.producer.GetMetrics().GetSnapshot()
	consumerMetrics := es.consumer.GetMetrics().GetSnapshot()

	return map[string]interface{}{
		"producer":  producerMetrics,
		"consumer":  consumerMetrics,
		"history_size": len(es.eventHistory),
		"subscriptions": len(es.consumer.ListSubscriptions()),
	}
}

// RegisterTopic registers a topic configuration.
func (es *EventStream) RegisterTopic(topic *TopicConfig) error {
	if topic == nil || topic.Name == "" {
		return fmt.Errorf("topic config required")
	}

	es.topicsMutex.Lock()
	defer es.topicsMutex.Unlock()

	es.topics[topic.Name] = topic
	return nil
}

// GetTopic returns topic configuration.
func (es *EventStream) GetTopic(name string) (*TopicConfig, error) {
	es.topicsMutex.RLock()
	defer es.topicsMutex.RUnlock()

	topic, exists := es.topics[name]
	if !exists {
		return nil, fmt.Errorf("topic not found: %s", name)
	}
	return topic, nil
}

// ListTopics returns all registered topics.
func (es *EventStream) ListTopics() []*TopicConfig {
	es.topicsMutex.RLock()
	defer es.topicsMutex.RUnlock()

	topics := make([]*TopicConfig, 0, len(es.topics))
	for _, topic := range es.topics {
		topics = append(topics, topic)
	}
	return topics
}

// MarshalEvent serializes event to JSON.
func (se *StreamEvent) MarshalJSON() ([]byte, error) {
	type Alias StreamEvent
	return json.Marshal(&struct {
		*Alias
		Type      string `json:"type"`
		Severity  string `json:"severity"`
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(se),
		Type:      string(se.Type),
		Severity:  string(se.Severity),
		Timestamp: se.Timestamp.Format(time.RFC3339Nano),
	})
}

// UnmarshalEvent deserializes event from JSON.
func (se *StreamEvent) UnmarshalJSON(data []byte) error {
	type Alias StreamEvent
	aux := &struct {
		Type      string `json:"type"`
		Severity  string `json:"severity"`
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(se),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	se.Type = EventType(aux.Type)
	se.Severity = EventSeverity(aux.Severity)
	t, err := time.Parse(time.RFC3339Nano, aux.Timestamp)
	if err != nil {
		return err
	}
	se.Timestamp = t

	return nil
}
