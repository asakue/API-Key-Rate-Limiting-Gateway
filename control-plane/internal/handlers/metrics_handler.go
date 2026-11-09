package handlers

import (
	"net/http"
	"time"

	"api-gateway/control-plane/internal/database"
	"api-gateway/control-plane/internal/models"
)

type MetricsHandler struct {
	db *database.DB
}

func NewMetricsHandler(db *database.DB) *MetricsHandler {
	return &MetricsHandler{db: db}
}

func (h *MetricsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	summaryQuery := `
		SELECT 
			COALESCE(SUM(request_count), 0) AS total_requests,
			COALESCE(SUM(CASE WHEN status_bucket = '2xx' THEN request_count ELSE 0 END), 0) AS total_2xx,
			COALESCE(SUM(CASE WHEN status_bucket = '4xx' AND status_code != 429 THEN request_count ELSE 0 END), 0) AS total_4xx,
			COALESCE(SUM(CASE WHEN status_bucket = '5xx' THEN request_count ELSE 0 END), 0) AS total_5xx,
			COALESCE(SUM(CASE WHEN status_code = 429 THEN request_count ELSE 0 END), 0) AS total_429,
			COALESCE(SUM(total_latency_ms), 0) AS total_latency
		FROM request_aggregates
		WHERE bucket_time >= NOW() - INTERVAL '24 HOURS'
	`

	var summary models.MetricSummary
	var totalLatency int64
	err := h.db.Pool.QueryRow(ctx, summaryQuery).Scan(
		&summary.TotalRequests,
		&summary.Total2xx,
		&summary.Total4xx,
		&summary.Total5xx,
		&summary.Total429,
		&totalLatency,
	)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query summary: " + err.Error()})
		return
	}

	if summary.TotalRequests > 0 {
		summary.AverageLatencyMs = float64(totalLatency) / float64(summary.TotalRequests)
	}

	// Count active keys
	_ = h.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_keys WHERE status = 'active'`).Scan(&summary.ActiveKeysCount)

	respondJSON(w, http.StatusOK, summary)
}

func (h *MetricsHandler) TimeSeries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := `
		SELECT 
			date_trunc('minute', bucket_time) AS minute_bucket,
			COALESCE(SUM(request_count), 0) AS total,
			COALESCE(SUM(CASE WHEN status_bucket = '2xx' THEN request_count ELSE 0 END), 0) AS success,
			COALESCE(SUM(CASE WHEN status_bucket = '4xx' AND status_code != 429 THEN request_count ELSE 0 END), 0) AS client_err,
			COALESCE(SUM(CASE WHEN status_code = 429 THEN request_count ELSE 0 END), 0) AS throttled,
			COALESCE(SUM(CASE WHEN status_bucket = '5xx' THEN request_count ELSE 0 END), 0) AS server_err
		FROM request_aggregates
		WHERE bucket_time >= NOW() - INTERVAL '1 HOUR'
		GROUP BY minute_bucket
		ORDER BY minute_bucket ASC
	`

	rows, err := h.db.Pool.Query(ctx, query)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query timeseries"})
		return
	}
	defer rows.Close()

	points := make([]models.TimeSeriesPoint, 0)
	for rows.Next() {
		var p models.TimeSeriesPoint
		if err := rows.Scan(&p.Timestamp, &p.Total, &p.Success, &p.ClientErr, &p.Throttled, &p.ServerErr); err == nil {
			points = append(points, p)
		}
	}

	// If no data points in DB yet, return last 10 minutes with 0 for nice UI rendering
	if len(points) == 0 {
		now := time.Now().UTC().Truncate(time.Minute)
		for i := 9; i >= 0; i-- {
			points = append(points, models.TimeSeriesPoint{
				Timestamp: now.Add(-time.Duration(i) * time.Minute),
				Total:     0,
				Success:   0,
				ClientErr: 0,
				Throttled: 0,
				ServerErr: 0,
			})
		}
	}

	respondJSON(w, http.StatusOK, points)
}

