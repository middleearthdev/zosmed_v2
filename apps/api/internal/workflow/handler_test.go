package workflow_test

// handler_test.go exercises the HTTP layer with an in-memory dbgen.DBTX
// stub, following the same pattern as apps/api/internal/auth/fakedb_test.go:
// a hand-rolled fake that dispatches on the (stable, sqlc-generated) SQL
// text, so Handler can be tested without a real Postgres connection.
//
// Store.Save (the transactional replace, ADR-004 R4) needs a real
// *pgxpool.Pool and is NOT exercised here — it was verified end-to-end
// against a real dev database via `go run ./apps/api/cmd/seed` (which
// creates + activates a workflow through this exact Store), and is a
// reasonable candidate for a future TxRunner-style seam (mirroring
// libs/kits/seller.TxRunner) if a hermetic test becomes necessary.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zosmed/zosmed/apps/api/internal/auth"
	"github.com/zosmed/zosmed/apps/api/internal/httpx"
	apiworkflow "github.com/zosmed/zosmed/apps/api/internal/workflow"
	"github.com/zosmed/zosmed/libs/platform/dbgen"
	"github.com/zosmed/zosmed/libs/platform/uuidx"
)

// ── in-memory dbgen.DBTX stub ─────────────────────────────────────────────────

type fakeWorkflowDBTX struct {
	mu            sync.Mutex
	accountByUser map[pgtype.UUID]dbgen.Account
	workflows     map[pgtype.UUID]dbgen.Workflow
	nodes         map[pgtype.UUID][]dbgen.WorkflowNode
	edges         map[pgtype.UUID][]dbgen.WorkflowEdge
}

func newFakeWorkflowDBTX() *fakeWorkflowDBTX {
	return &fakeWorkflowDBTX{
		accountByUser: map[pgtype.UUID]dbgen.Account{},
		workflows:     map[pgtype.UUID]dbgen.Workflow{},
		nodes:         map[pgtype.UUID][]dbgen.WorkflowNode{},
		edges:         map[pgtype.UUID][]dbgen.WorkflowEdge{},
	}
}

func (f *fakeWorkflowDBTX) Exec(_ context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if strings.Contains(sql, "DELETE FROM workflow WHERE id = $1 AND account_id = $2") {
		id := args[0].(pgtype.UUID)
		accountID := args[1].(pgtype.UUID)
		wf, ok := f.workflows[id]
		if !ok || wf.AccountID != accountID {
			return pgconn.NewCommandTag("DELETE 0"), nil
		}
		delete(f.workflows, id)
		return pgconn.NewCommandTag("DELETE 1"), nil
	}
	panic("fakeWorkflowDBTX.Exec: unhandled query: " + sql)
}

func (f *fakeWorkflowDBTX) Query(_ context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case strings.Contains(sql, "FROM workflow_node"):
		workflowID := args[0].(pgtype.UUID)
		rows := f.nodes[workflowID]
		fns := make([]func(...any) error, len(rows))
		for i, n := range rows {
			fns[i] = nodeScanFn(n)
		}
		return &fakeRows{scanFns: fns}, nil

	case strings.Contains(sql, "FROM workflow_edge WHERE workflow_id"):
		workflowID := args[0].(pgtype.UUID)
		rows := f.edges[workflowID]
		fns := make([]func(...any) error, len(rows))
		for i, e := range rows {
			fns[i] = edgeScanFn(e)
		}
		return &fakeRows{scanFns: fns}, nil

	case strings.Contains(sql, "LEFT JOIN workflow_node n"):
		accountID := args[0].(pgtype.UUID)
		var fns []func(...any) error
		for _, wf := range f.workflows {
			if wf.AccountID != accountID {
				continue
			}
			count := int32(len(f.nodes[wf.ID]))
			fns = append(fns, summaryScanFn(wf, count))
		}
		return &fakeRows{scanFns: fns}, nil
	}
	panic("fakeWorkflowDBTX.Query: unhandled query: " + sql)
}

func (f *fakeWorkflowDBTX) QueryRow(_ context.Context, sql string, args ...interface{}) pgx.Row {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case strings.Contains(sql, "FROM account WHERE user_id"):
		userID := args[0].(pgtype.UUID)
		acc, ok := f.accountByUser[userID]
		if !ok {
			return &fakeRow{err: pgx.ErrNoRows}
		}
		return &fakeRow{scan: accountScanFn(acc)}

	case strings.Contains(sql, "SET status = 'live'"): // ActivateWorkflow
		id := args[0].(pgtype.UUID)
		accountID := args[1].(pgtype.UUID)
		wf, ok := f.workflows[id]
		if !ok || wf.AccountID != accountID {
			return &fakeRow{err: pgx.ErrNoRows}
		}
		wf.Status = dbgen.WorkflowStatusLive
		wf.Version++
		f.workflows[id] = wf
		return &fakeRow{scan: workflowScanFn(wf)}

	case strings.Contains(sql, "SET status = $1::workflow_status"): // SetWorkflowStatus (pause)
		status := args[0].(dbgen.WorkflowStatus)
		id := args[1].(pgtype.UUID)
		accountID := args[2].(pgtype.UUID)
		wf, ok := f.workflows[id]
		if !ok || wf.AccountID != accountID {
			return &fakeRow{err: pgx.ErrNoRows}
		}
		wf.Status = status
		f.workflows[id] = wf
		return &fakeRow{scan: workflowScanFn(wf)}

	case strings.Contains(sql, "FROM workflow WHERE id = $1 AND account_id = $2"): // GetWorkflowByID
		id := args[0].(pgtype.UUID)
		accountID := args[1].(pgtype.UUID)
		wf, ok := f.workflows[id]
		if !ok || wf.AccountID != accountID {
			return &fakeRow{err: pgx.ErrNoRows}
		}
		return &fakeRow{scan: workflowScanFn(wf)}
	}
	panic("fakeWorkflowDBTX.QueryRow: unhandled query: " + sql)
}

// ── row scanners ──────────────────────────────────────────────────────────────

type fakeRow struct {
	err  error
	scan func(dest ...any) error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return r.scan(dest...)
}

// fakeRows implements pgx.Rows over a slice of pre-bound scan functions.
type fakeRows struct {
	idx     int
	scanFns []func(dest ...any) error
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool                                   { return r.idx < len(r.scanFns) }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }
func (r *fakeRows) Scan(dest ...any) error {
	fn := r.scanFns[r.idx]
	r.idx++
	return fn(dest...)
}

func accountScanFn(a dbgen.Account) func(dest ...any) error {
	return func(dest ...any) error {
		*(dest[0].(*pgtype.UUID)) = a.ID
		*(dest[1].(*string)) = a.IgUserID
		*(dest[2].(*string)) = a.Handle
		*(dest[3].(*string)) = a.DisplayName
		*(dest[4].(*string)) = a.Status
		*(dest[5].(*pgtype.Timestamptz)) = a.CreatedAt
		*(dest[6].(*string)) = a.AccessToken
		*(dest[7].(*string)) = a.TokenType
		*(dest[8].(*[]string)) = a.Scopes
		*(dest[9].(*pgtype.Timestamptz)) = a.TokenExpiresAt
		*(dest[10].(*pgtype.Timestamptz)) = a.TokenRefreshedAt
		*(dest[11].(*pgtype.UUID)) = a.UserID
		return nil
	}
}

func workflowScanFn(wf dbgen.Workflow) func(dest ...any) error {
	return func(dest ...any) error {
		*(dest[0].(*pgtype.UUID)) = wf.ID
		*(dest[1].(*pgtype.UUID)) = wf.AccountID
		*(dest[2].(*string)) = wf.Name
		*(dest[3].(*dbgen.WorkflowStatus)) = wf.Status
		*(dest[4].(*string)) = wf.Segment
		*(dest[5].(*int32)) = wf.Version
		*(dest[6].(*pgtype.Timestamptz)) = wf.CreatedAt
		*(dest[7].(*pgtype.Timestamptz)) = wf.UpdatedAt
		return nil
	}
}

func summaryScanFn(wf dbgen.Workflow, nodeCount int32) func(dest ...any) error {
	return func(dest ...any) error {
		*(dest[0].(*pgtype.UUID)) = wf.ID
		*(dest[1].(*pgtype.UUID)) = wf.AccountID
		*(dest[2].(*string)) = wf.Name
		*(dest[3].(*dbgen.WorkflowStatus)) = wf.Status
		*(dest[4].(*string)) = wf.Segment
		*(dest[5].(*int32)) = wf.Version
		*(dest[6].(*pgtype.Timestamptz)) = wf.CreatedAt
		*(dest[7].(*pgtype.Timestamptz)) = wf.UpdatedAt
		*(dest[8].(*int32)) = nodeCount
		return nil
	}
}

func nodeScanFn(n dbgen.WorkflowNode) func(dest ...any) error {
	return func(dest ...any) error {
		*(dest[0].(*pgtype.UUID)) = n.ID
		*(dest[1].(*pgtype.UUID)) = n.WorkflowID
		*(dest[2].(*string)) = n.Category
		*(dest[3].(*string)) = n.NodeType
		*(dest[4].(*[]byte)) = n.Config
		*(dest[5].(*int32)) = n.PositionX
		*(dest[6].(*int32)) = n.PositionY
		*(dest[7].(*pgtype.Timestamptz)) = n.CreatedAt
		return nil
	}
}

func edgeScanFn(e dbgen.WorkflowEdge) func(dest ...any) error {
	return func(dest ...any) error {
		*(dest[0].(*pgtype.UUID)) = e.ID
		*(dest[1].(*pgtype.UUID)) = e.WorkflowID
		*(dest[2].(*pgtype.UUID)) = e.FromNodeID
		*(dest[3].(*pgtype.UUID)) = e.ToNodeID
		return nil
	}
}

// ── test fixtures / helpers ───────────────────────────────────────────────────

func testUUID(n int) pgtype.UUID {
	var b [16]byte
	b[14] = byte(n >> 8)
	b[15] = byte(n)
	return pgtype.UUID{Bytes: b, Valid: true}
}

func newTestRouter(h *apiworkflow.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/workflows", h.List)
	r.Get("/workflows/{id}", h.Get)
	r.Post("/workflows/{id}/activate", h.Activate)
	r.Post("/workflows/{id}/pause", h.Pause)
	r.Delete("/workflows/{id}", h.Delete)
	return r
}

func authedRequest(method, path string, userID pgtype.UUID) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := auth.WithUser(req.Context(), auth.UserDTO{ID: uuidx.Format(userID)})
	return req.WithContext(ctx)
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) httpx.Envelope {
	t.Helper()
	var env httpx.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v (body=%s)", err, rec.Body.String())
	}
	return env
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestActivate_ValidGraphGoesLive(t *testing.T) {
	userID := testUUID(1)
	accountID := testUUID(2)
	workflowID := testUUID(3)
	triggerNodeID := testUUID(4)
	actionNodeID := testUUID(5)

	db := newFakeWorkflowDBTX()
	db.accountByUser[userID] = dbgen.Account{ID: accountID, UserID: userID, Status: "connected"}
	db.workflows[workflowID] = dbgen.Workflow{
		ID: workflowID, AccountID: accountID, Name: "Demo", Status: dbgen.WorkflowStatusDraft,
		Segment: "seller", Version: 1,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	db.nodes[workflowID] = []dbgen.WorkflowNode{
		{ID: triggerNodeID, WorkflowID: workflowID, Category: "trigger", NodeType: "comment-received", Config: []byte(`{}`)},
		{ID: actionNodeID, WorkflowID: workflowID, Category: "action", NodeType: "send-whatsapp-link", Config: []byte(`{}`)},
	}

	queries := dbgen.New(db)
	h := apiworkflow.NewHandler(queries, nil)
	router := newTestRouter(h)

	req := authedRequest(http.MethodPost, fmt.Sprintf("/workflows/%s/activate", uuidx.Format(workflowID)), userID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Error != nil {
		t.Fatalf("unexpected error envelope: %+v", env.Error)
	}
}

func TestActivate_NoTriggerReturns422(t *testing.T) {
	userID := testUUID(1)
	accountID := testUUID(2)
	workflowID := testUUID(3)
	actionNodeID := testUUID(4)

	db := newFakeWorkflowDBTX()
	db.accountByUser[userID] = dbgen.Account{ID: accountID, UserID: userID, Status: "connected"}
	db.workflows[workflowID] = dbgen.Workflow{
		ID: workflowID, AccountID: accountID, Name: "Demo", Status: dbgen.WorkflowStatusDraft,
		Segment: "seller", Version: 1,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	db.nodes[workflowID] = []dbgen.WorkflowNode{
		{ID: actionNodeID, WorkflowID: workflowID, Category: "action", NodeType: "send-whatsapp-link", Config: []byte(`{}`)},
	}

	queries := dbgen.New(db)
	h := apiworkflow.NewHandler(queries, nil)
	router := newTestRouter(h)

	req := authedRequest(http.MethodPost, fmt.Sprintf("/workflows/%s/activate", uuidx.Format(workflowID)), userID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Error == nil || env.Error.Reason != apiworkflow.ReasonNoTrigger {
		t.Fatalf("error = %+v, want reason %q", env.Error, apiworkflow.ReasonNoTrigger)
	}
}

func TestActivate_UnknownNodeTypeReturns422(t *testing.T) {
	userID := testUUID(1)
	accountID := testUUID(2)
	workflowID := testUUID(3)
	triggerNodeID := testUUID(4)
	actionNodeID := testUUID(5)

	db := newFakeWorkflowDBTX()
	db.accountByUser[userID] = dbgen.Account{ID: accountID, UserID: userID, Status: "connected"}
	db.workflows[workflowID] = dbgen.Workflow{
		ID: workflowID, AccountID: accountID, Name: "Demo", Status: dbgen.WorkflowStatusDraft,
		Segment: "seller", Version: 1,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	db.nodes[workflowID] = []dbgen.WorkflowNode{
		// §4b DO-NOT-list capability — must never be in the feasible catalog.
		{ID: triggerNodeID, WorkflowID: workflowID, Category: "trigger", NodeType: "new-follower", Config: []byte(`{}`)},
		{ID: actionNodeID, WorkflowID: workflowID, Category: "action", NodeType: "send-whatsapp-link", Config: []byte(`{}`)},
	}

	queries := dbgen.New(db)
	h := apiworkflow.NewHandler(queries, nil)
	router := newTestRouter(h)

	req := authedRequest(http.MethodPost, fmt.Sprintf("/workflows/%s/activate", uuidx.Format(workflowID)), userID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Error == nil || env.Error.Reason != apiworkflow.ReasonUnknownNodeType {
		t.Fatalf("error = %+v, want reason %q", env.Error, apiworkflow.ReasonUnknownNodeType)
	}
}

func TestActivate_WrongAccountIsNotFound(t *testing.T) {
	userID := testUUID(1)
	accountID := testUUID(2)
	otherAccountID := testUUID(99)
	workflowID := testUUID(3)

	db := newFakeWorkflowDBTX()
	db.accountByUser[userID] = dbgen.Account{ID: accountID, UserID: userID, Status: "connected"}
	db.workflows[workflowID] = dbgen.Workflow{
		ID: workflowID, AccountID: otherAccountID, Name: "Not yours", Status: dbgen.WorkflowStatusDraft,
		Segment: "seller", Version: 1,
	}

	queries := dbgen.New(db)
	h := apiworkflow.NewHandler(queries, nil)
	router := newTestRouter(h)

	req := authedRequest(http.MethodPost, fmt.Sprintf("/workflows/%s/activate", uuidx.Format(workflowID)), userID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestList_ReturnsSummariesWithNodeCount(t *testing.T) {
	userID := testUUID(1)
	accountID := testUUID(2)
	workflowID := testUUID(3)

	db := newFakeWorkflowDBTX()
	db.accountByUser[userID] = dbgen.Account{ID: accountID, UserID: userID, Status: "connected"}
	db.workflows[workflowID] = dbgen.Workflow{
		ID: workflowID, AccountID: accountID, Name: "Demo", Status: dbgen.WorkflowStatusLive,
		Segment: "seller", Version: 2,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	db.nodes[workflowID] = []dbgen.WorkflowNode{
		{ID: testUUID(10), WorkflowID: workflowID, Category: "trigger", NodeType: "comment-received"},
		{ID: testUUID(11), WorkflowID: workflowID, Category: "action", NodeType: "send-whatsapp-link"},
	}

	queries := dbgen.New(db)
	h := apiworkflow.NewHandler(queries, nil)
	router := newTestRouter(h)

	req := authedRequest(http.MethodGet, "/workflows", userID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	list, ok := env.Data.([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("data = %#v, want a single-element list", env.Data)
	}
	summary := list[0].(map[string]any)
	if summary["nodeCount"].(float64) != 2 {
		t.Errorf("nodeCount = %v, want 2", summary["nodeCount"])
	}
	if summary["status"].(string) != "live" {
		t.Errorf("status = %v, want live", summary["status"])
	}
}

func TestDelete_RemovesWorkflow(t *testing.T) {
	userID := testUUID(1)
	accountID := testUUID(2)
	workflowID := testUUID(3)

	db := newFakeWorkflowDBTX()
	db.accountByUser[userID] = dbgen.Account{ID: accountID, UserID: userID, Status: "connected"}
	db.workflows[workflowID] = dbgen.Workflow{ID: workflowID, AccountID: accountID, Name: "Demo"}

	queries := dbgen.New(db)
	h := apiworkflow.NewHandler(queries, nil)
	router := newTestRouter(h)

	req := authedRequest(http.MethodDelete, fmt.Sprintf("/workflows/%s", uuidx.Format(workflowID)), userID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if _, exists := db.workflows[workflowID]; exists {
		t.Error("expected workflow to be removed from the store")
	}
}

func TestActivate_UnauthenticatedReturns401(t *testing.T) {
	db := newFakeWorkflowDBTX()
	queries := dbgen.New(db)
	h := apiworkflow.NewHandler(queries, nil)
	router := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/workflows/%s/activate", uuidx.Format(testUUID(1))), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body=%s)", rec.Code, rec.Body.String())
	}
}
