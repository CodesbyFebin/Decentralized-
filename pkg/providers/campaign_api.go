package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"decentralized.host/pkg/audit"
)

// CampaignAPIHandler provides HTTP endpoints for campaign management.
type CampaignAPIHandler struct {
	store  *CampaignStore
	ledger *audit.Ledger
}

// NewCampaignAPIHandler creates a new campaign API handler.
func NewCampaignAPIHandler(db *sql.DB, ledger *audit.Ledger) (*CampaignAPIHandler, error) {
	store, err := NewCampaignStore(db, ledger)
	if err != nil {
		return nil, err
	}

	return &CampaignAPIHandler{
		store: store,
		ledger: ledger,
	}, nil
}

// StartCampaignRequest is the request body for starting a qualification campaign.
type StartCampaignRequest struct {
	ResourceID string `json:"resource_id"`
	SourceSHA  string `json:"source_sha"`
}

// HandleStartCampaign starts a new qualification campaign.
// POST /api/v1/qualification/campaign
func (h *CampaignAPIHandler) HandleStartCampaign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StartCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ResourceID == "" || req.SourceSHA == "" {
		http.Error(w, "resource_id and source_sha required", http.StatusBadRequest)
		return
	}

	// Create campaign ID
	campaignID := "qual-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	response := map[string]interface{}{
		"campaign_id": campaignID,
		"resource_id": req.ResourceID,
		"status": "RUNNING",
		"start_time": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetCampaign retrieves a campaign by ID.
// GET /api/v1/qualification/campaign/{id}
func (h *CampaignAPIHandler) HandleGetCampaign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	campaignID := r.PathValue("id")
	if campaignID == "" {
		http.Error(w, "campaign id required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	campaign, err := h.store.GetCampaign(ctx, campaignID)
	if err != nil {
		http.Error(w, "campaign not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(campaign)
}

// HandleListCampaigns lists campaigns with optional filtering.
// GET /api/v1/qualification/campaigns?resource_id=X&status=Y&level=Z&limit=10&offset=0
func (h *CampaignAPIHandler) HandleListCampaigns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resourceID := r.URL.Query().Get("resource_id")
	status := r.URL.Query().Get("status")
	level := r.URL.Query().Get("level")

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	campaigns, err := h.store.QueryCampaigns(ctx, resourceID, status, level, limit, offset)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"campaigns": campaigns,
		"count": len(campaigns),
		"limit": limit,
		"offset": offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleVerifyCampaign verifies a campaign's cryptographic evidence.
// POST /api/v1/qualification/verify
func (h *CampaignAPIHandler) HandleVerifyCampaign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var evidence QualifiedEvidence
	if err := json.NewDecoder(r.Body).Decode(&evidence); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Verify cryptographic signature
	err := VerifyEvidence(&evidence)
	verified := err == nil

	response := map[string]interface{}{
		"verified": verified,
		"signer_id": evidence.SignerID,
		"campaign_id": evidence.CampaignID,
		"p1_core_passed": evidence.P1_CORE_Passed,
		"p1_qemu_passed": evidence.P1_QEMU_Passed,
		"p1_k8s_passed": evidence.P1_K8S_Passed,
		"p2_multi_passed": evidence.P2_Multi_Passed,
	}

	if !verified {
		response["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleDeleteCampaign deletes a campaign.
// DELETE /api/v1/qualification/campaign/{id}
func (h *CampaignAPIHandler) HandleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	campaignID := r.PathValue("id")
	if campaignID == "" {
		http.Error(w, "campaign id required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.store.DeleteCampaign(ctx, campaignID); err != nil {
		http.Error(w, "failed to delete campaign", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
		"campaign_id": campaignID,
	})
}

// CampaignStatsResponse represents campaign statistics.
type CampaignStatsResponse struct {
	TotalCampaigns   int64                      `json:"total_campaigns"`
	PassedCampaigns  int64                      `json:"passed_campaigns"`
	FailedCampaigns  int64                      `json:"failed_campaigns"`
	ByLevel          map[string]int64           `json:"by_qualification_level"`
	ByResource       map[string]int64           `json:"by_resource_id"`
}

// HandleCampaignStats returns statistics about campaigns.
// GET /api/v1/qualification/stats
func (h *CampaignAPIHandler) HandleCampaignStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	stats := CampaignStatsResponse{
		ByLevel: make(map[string]int64),
		ByResource: make(map[string]int64),
	}

	// Count total campaigns
	row := h.store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM qualification_campaigns")
	row.Scan(&stats.TotalCampaigns)

	// Count passed/failed
	row = h.store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM qualification_campaigns WHERE status = 'PASSED'")
	row.Scan(&stats.PassedCampaigns)

	row = h.store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM qualification_campaigns WHERE status = 'FAILED'")
	row.Scan(&stats.FailedCampaigns)

	// Count by qualification level
	rows, _ := h.store.db.QueryContext(ctx, "SELECT qualification_level, COUNT(*) FROM qualification_campaigns WHERE qualification_level IS NOT NULL GROUP BY qualification_level")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var level string
			var count int64
			if rows.Scan(&level, &count) == nil {
				stats.ByLevel[level] = count
			}
		}
	}

	// Count by resource ID
	rows, _ = h.store.db.QueryContext(ctx, "SELECT resource_id, COUNT(*) FROM qualification_campaigns GROUP BY resource_id LIMIT 20")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var resID string
			var count int64
			if rows.Scan(&resID, &count) == nil {
				stats.ByResource[resID] = count
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
