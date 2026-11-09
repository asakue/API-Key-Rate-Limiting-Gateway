package telemetry

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricEvent struct {
	Timestamp  time.Time
	KeyID      string
	RouteID    string
	StatusCode int
	LatencyMs  int64
}

type AggregatedKey struct {
	MinuteBucket time.Time
	KeyID        string
	RouteID      string
	StatusBucket string
	StatusCode   int
}

type AggregatedValue struct {
	Count        int64
	TotalLatency int64
}

type Collector struct {
	dbPool        *pgxpool.Pool
	eventChan     chan MetricEvent
	flushInterval time.Duration
	maxBatchSize  int
	droppedEvents atomic.Int64
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

func NewCollector(dbPool *pgxpool.Pool, bufferSize int, flushInterval time.Duration, maxBatchSize int) *Collector {
	c := &Collector{
		dbPool:        dbPool,
		eventChan:     make(chan MetricEvent, bufferSize),
		flushInterval: flushInterval,
		maxBatchSize:  maxBatchSize,
		stopChan:      make(chan struct{}),
	}

	c.wg.Add(1)
	go c.workerLoop()

	return c
}

// Record queues an event non-blockingly. If buffer is full, drops with atomic counter.
func (c *Collector) Record(event MetricEvent) {
	select {
	case c.eventChan <- event:
	default:
		c.droppedEvents.Add(1)
	}
}

func (c *Collector) workerLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	batch := make(map[AggregatedKey]*AggregatedValue)
	batchCount := 0

	flush := func() {
		if len(batch) == 0 {
			return
		}
		c.flushToDB(batch)
		batch = make(map[AggregatedKey]*AggregatedValue)
		batchCount = 0
	}

	for {
		select {
		case <-c.stopChan:
			// Drain channel
			for {
				select {
				case ev := <-c.eventChan:
					c.aggregate(batch, ev)
				default:
					flush()
					return
				}
			}

		case ev := <-c.eventChan:
			c.aggregate(batch, ev)
			batchCount++
			if batchCount >= c.maxBatchSize {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

func (c *Collector) aggregate(batch map[AggregatedKey]*AggregatedValue, ev MetricEvent) {
	bucketTime := ev.Timestamp.Truncate(time.Minute)
	statusBucket := "2xx"
	if ev.StatusCode >= 500 {
		statusBucket = "5xx"
	} else if ev.StatusCode == 429 {
		statusBucket = "429"
	} else if ev.StatusCode >= 400 {
		statusBucket = "4xx"
	}

	k := AggregatedKey{
		MinuteBucket: bucketTime,
		KeyID:        ev.KeyID,
		RouteID:      ev.RouteID,
		StatusBucket: statusBucket,
		StatusCode:   ev.StatusCode,
	}

	val, exists := batch[k]
	if !exists {
		val = &AggregatedValue{}
		batch[k] = val
	}
	val.Count++
	val.TotalLatency += ev.LatencyMs
}

func (c *Collector) flushToDB(batch map[AggregatedKey]*AggregatedValue) {
	if c.dbPool == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	batchRows := make([][]interface{}, 0, len(batch))
	for k, v := range batch {
		batchRows = append(batchRows, []interface{}{
			k.MinuteBucket, k.KeyID, k.RouteID, k.StatusBucket, k.StatusCode, v.Count, v.TotalLatency,
		})
	}

	// Use pgx CopyFrom for highest bulk write speed
	_, err := c.dbPool.CopyFrom(
		ctx,
		pgx.Identifier{"request_aggregates"},
		[]string{"bucket_time", "key_id", "route_id", "status_bucket", "status_code", "request_count", "total_latency_ms"},
		pgx.CopyFromRows(batchRows),
	)
	if err != nil {
		log.Printf("[WARN] Failed to flush telemetry batch: %v", err)
	}
}

func (c *Collector) Close() {
	close(c.stopChan)
	c.wg.Wait()
	if dropped := c.droppedEvents.Load(); dropped > 0 {
		log.Printf("[INFO] Collector stopped. Total dropped telemetry events: %d", dropped)
	}
}
