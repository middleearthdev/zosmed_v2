package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/apps/api/internal/auth"
	"github.com/zosmed/zosmed/apps/api/internal/httpx"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

// defaultRunLimit is the default page size for the Runs endpoints when the
// caller omits ?limit=.
const defaultRunLimit = int32(50)

// Handler handles the workflow builder REST endpoints (ADR-004 §3). It reads
// via dbgen directly for simple lookups and delegates the transactional save
// + activate validation to Store (SoC §12a-3: handler.go is transport-only).
type Handler struct {
	queries *dbgen.Queries
	store   *Store
}

// NewHandler returns a Handler backed by queries (reads) and store (writes).
func NewHandler(queries *dbgen.Queries, store *Store) *Handler {
	return &Handler{queries: queries, store: store}
}

// ── GET /api/v1/workflows ─────────────────────────────────────────────────────

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.resolveAccountID(w, r)
	if !ok {
		return
	}

	rows, err := h.queries.ListWorkflowsByAccount(ctx, accountID)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal memuat daftar workflow")
		return
	}

	out := make([]WorkflowSummaryDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapSummaryDTO(row))
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ── POST /api/v1/workflows ────────────────────────────────────────────────────

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.resolveAccountID(w, r)
	if !ok {
		return
	}

	var body CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Err(w, http.StatusBadRequest, "invalid_body", "Body permintaan tidak valid")
		return
	}
	if !auth.ValidSegments[body.Segment] {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Segmen tidak dikenal (gunakan seller/creator/booking)")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "Workflow baru"
	}

	wf, err := h.queries.CreateWorkflow(ctx, dbgen.CreateWorkflowParams{
		AccountID: accountID,
		Name:      name,
		Segment:   body.Segment,
	})
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal membuat workflow")
		return
	}

	httpx.JSON(w, http.StatusCreated, mapWorkflowDTO(wf, nil, nil))
}

// ── GET /api/v1/workflows/{id} ────────────────────────────────────────────────

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.resolveAccountID(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	wf, err := h.queries.GetWorkflowByID(ctx, dbgen.GetWorkflowByIDParams{ID: id, AccountID: accountID})
	if err != nil {
		if isNoRows(err) {
			httpx.Err(w, http.StatusNotFound, "not_found", "Workflow tidak ditemukan")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal memuat workflow")
		return
	}

	nodeRows, edgeRows, ok := h.loadGraph(w, r, id)
	if !ok {
		return
	}
	httpx.JSON(w, http.StatusOK, mapWorkflowDTO(wf, nodeRows, edgeRows))
}

// ── PUT /api/v1/workflows/{id} ────────────────────────────────────────────────

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.resolveAccountID(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	var body SaveWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Err(w, http.StatusBadRequest, "invalid_body", "Body permintaan tidak valid")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		httpx.Err(w, http.StatusBadRequest, "invalid_request", "Nama workflow tidak boleh kosong")
		return
	}

	nodeInputs := make([]NodeInput, 0, len(body.Nodes))
	for i, n := range body.Nodes {
		category := n.Node.Category
		if category != "trigger" && category != "filter" && category != "action" {
			httpx.Err(w, http.StatusBadRequest, "invalid_request",
				fmt.Sprintf("kategori node tidak dikenal: %q", category))
			return
		}
		clientID := n.ID
		if clientID == "" {
			// Local-only key for correlating edges within THIS request; never
			// persisted (ADR-004 R4 — the server always assigns a fresh UUID).
			clientID = fmt.Sprintf("__local_%d", i)
		}
		cfg := n.Config
		if len(cfg) == 0 {
			cfg = json.RawMessage(`{}`)
		}
		nodeInputs = append(nodeInputs, NodeInput{
			ClientID:  clientID,
			Category:  category,
			NodeType:  n.Node.Kind,
			Config:    cfg,
			PositionX: n.Position.X,
			PositionY: n.Position.Y,
		})
	}

	edgeInputs := make([]EdgeInput, 0, len(body.Edges))
	for _, e := range body.Edges {
		edgeInputs = append(edgeInputs, EdgeInput{From: e.From, To: e.To})
	}

	wf, nodeRows, edgeRows, err := h.store.Save(ctx, id, accountID, name, nodeInputs, edgeInputs)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.Err(w, http.StatusNotFound, "not_found", "Workflow tidak ditemukan")
		case errors.Is(err, ErrUnknownEdgeNode):
			httpx.Err(w, http.StatusBadRequest, "invalid_request",
				"Ada edge yang mereferensikan node id di luar daftar node pada permintaan ini")
		default:
			httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal menyimpan workflow")
		}
		return
	}

	httpx.JSON(w, http.StatusOK, mapWorkflowDTO(wf, nodeRows, edgeRows))
}

// ── DELETE /api/v1/workflows/{id} ─────────────────────────────────────────────

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.resolveAccountID(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	n, err := h.queries.DeleteWorkflow(ctx, dbgen.DeleteWorkflowParams{ID: id, AccountID: accountID})
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal menghapus workflow")
		return
	}
	if n == 0 {
		httpx.Err(w, http.StatusNotFound, "not_found", "Workflow tidak ditemukan")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ── POST /api/v1/workflows/{id}/activate ──────────────────────────────────────

// Activate validates the graph against ADR-004 §3's five rules and, only if
// valid, flips status -> 'live' (bumping version, ADR-004 §2.1). Invalid
// graphs return 422 validation_failed with a machine-readable reason.
func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.resolveAccountID(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	if _, err := h.queries.GetWorkflowByID(ctx, dbgen.GetWorkflowByIDParams{ID: id, AccountID: accountID}); err != nil {
		if isNoRows(err) {
			httpx.Err(w, http.StatusNotFound, "not_found", "Workflow tidak ditemukan")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal memuat workflow")
		return
	}

	nodeRows, edgeRows, ok := h.loadGraph(w, r, id)
	if !ok {
		return
	}

	if reason, valid := validateForActivate(nodeRows, edgeRows); !valid {
		httpx.ErrWithReason(w, http.StatusUnprocessableEntity, "validation_failed", validationMessage(reason), reason)
		return
	}

	wf, err := h.queries.ActivateWorkflow(ctx, dbgen.ActivateWorkflowParams{ID: id, AccountID: accountID})
	if err != nil {
		if isNoRows(err) {
			httpx.Err(w, http.StatusNotFound, "not_found", "Workflow tidak ditemukan")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal mengaktifkan workflow")
		return
	}
	httpx.JSON(w, http.StatusOK, mapWorkflowDTO(wf, nodeRows, edgeRows))
}

// ── POST /api/v1/workflows/{id}/pause ─────────────────────────────────────────

func (h *Handler) Pause(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.resolveAccountID(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	wf, err := h.queries.SetWorkflowStatus(ctx, dbgen.SetWorkflowStatusParams{
		Status:    dbgen.WorkflowStatusPaused,
		ID:        id,
		AccountID: accountID,
	})
	if err != nil {
		if isNoRows(err) {
			httpx.Err(w, http.StatusNotFound, "not_found", "Workflow tidak ditemukan")
			return
		}
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal menjeda workflow")
		return
	}

	nodeRows, edgeRows, ok := h.loadGraph(w, r, id)
	if !ok {
		return
	}
	httpx.JSON(w, http.StatusOK, mapWorkflowDTO(wf, nodeRows, edgeRows))
}

// ── GET /api/v1/workflows/{id}/runs ───────────────────────────────────────────

func (h *Handler) ListRunsForWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.resolveAccountID(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}

	rows, err := h.queries.ListRunsByWorkflow(ctx, dbgen.ListRunsByWorkflowParams{
		WorkflowID: id,
		AccountID:  accountID,
		Lim:        parseLimit(r),
	})
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal memuat riwayat run")
		return
	}

	out := make([]RunSummaryDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapRunSummaryDTO(row))
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ── GET /api/v1/runs ──────────────────────────────────────────────────────────

func (h *Handler) ListRunsForAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID, ok := h.resolveAccountID(w, r)
	if !ok {
		return
	}

	rows, err := h.queries.ListRunsByAccount(ctx, dbgen.ListRunsByAccountParams{
		AccountID: accountID,
		Lim:       parseLimit(r),
	})
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal memuat riwayat run")
		return
	}

	out := make([]RunSummaryDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapRunSummaryDTO(row))
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ── shared helpers ────────────────────────────────────────────────────────────

// resolveAccountID resolves the caller's own IG account (ADR-004 R5: MVP is
// 1 user <-> 1 account, scoped via GetAccountByUserID — no accountId query
// param). On failure it writes the HTTP error and returns ok=false.
func (h *Handler) resolveAccountID(w http.ResponseWriter, r *http.Request) (pgtype.UUID, bool) {
	userDTO, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.Err(w, http.StatusUnauthorized, "unauthorized", "Silakan login terlebih dahulu")
		return pgtype.UUID{}, false
	}
	userID, err := uuidx.Parse(userDTO.ID)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "internal_error", "ID pengguna tidak valid")
		return pgtype.UUID{}, false
	}
	account, err := h.queries.GetAccountByUserID(r.Context(), userID)
	if err != nil {
		if isNoRows(err) {
			httpx.Err(w, http.StatusConflict, "account_not_connected", "Hubungkan akun Instagram kamu terlebih dahulu")
			return pgtype.UUID{}, false
		}
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal memuat akun")
		return pgtype.UUID{}, false
	}
	return account.ID, true
}

// loadGraph loads the node/edge rows for a workflow already confirmed to
// belong to the caller's account (used by Get/Activate/Pause to avoid
// re-deriving the same two queries three times, §12a-1).
func (h *Handler) loadGraph(w http.ResponseWriter, r *http.Request, workflowID pgtype.UUID) ([]dbgen.WorkflowNode, []dbgen.WorkflowEdge, bool) {
	ctx := r.Context()
	nodeRows, err := h.queries.ListNodesByWorkflow(ctx, workflowID)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal memuat node workflow")
		return nil, nil, false
	}
	edgeRows, err := h.queries.ListEdgesByWorkflow(ctx, workflowID)
	if err != nil {
		httpx.Err(w, http.StatusInternalServerError, "db_error", "Gagal memuat edge workflow")
		return nil, nil, false
	}
	return nodeRows, edgeRows, true
}

// parseLimit reads ?limit= from the request, falling back to defaultRunLimit
// for missing/invalid/non-positive values.
func parseLimit(r *http.Request) int32 {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultRunLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultRunLimit
	}
	return int32(n)
}

// parseUUIDParam parses a UUID string from a query/path param. On parse
// failure it writes a 400 response and returns ok=false so callers can do an
// early return (mirrors apps/api/internal/commentorder's helper of the same
// name — small enough duplication to not warrant a shared package, §12a-4).
func parseUUIDParam(w http.ResponseWriter, raw, paramName string) (pgtype.UUID, bool) {
	id, err := uuidx.Parse(raw)
	if err != nil {
		httpx.Err(w, http.StatusBadRequest, "invalid_param",
			fmt.Sprintf("invalid UUID for param %q: %s", paramName, raw))
		return pgtype.UUID{}, false
	}
	return id, true
}
