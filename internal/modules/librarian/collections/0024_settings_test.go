package collections

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	pbcore "github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/hsb3/deskkit/internal/core/settings"
)

const (
	testSuperEmail = "operator@desk.test"
	testSuperPass  = "a-sufficiently-long-password"
	testAPIKey     = "sk-test-0123456789-ABCDWXYZ"
)

// settingsAPI stands up the real record API over a migrated test store and returns a superuser
// auth token. The HTTP path is the point: the auth posture and the hidden key are properties of
// what the SHIPPED collection actually serves, not of a struct field a unit test could read.
func settingsAPI(t *testing.T) (*tests.TestApp, *httptest.Server, string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	settings.BindHooks(app)

	col, err := app.FindCollectionByNameOrId(pbcore.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("find superusers: %v", err)
	}
	su := pbcore.NewRecord(col)
	su.SetEmail(testSuperEmail)
	su.SetPassword(testSuperPass)
	if err := app.Save(su); err != nil {
		t.Fatalf("save superuser: %v", err)
	}

	r, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("apis.NewRouter: %v", err)
	}
	mux, err := r.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	code, body := settingsPost(t, srv.URL+"/api/collections/"+pbcore.CollectionNameSuperusers+"/auth-with-password",
		`{"identity":"`+testSuperEmail+`","password":"`+testSuperPass+`"}`, "")
	if code != http.StatusOK {
		t.Fatalf("superuser login status = %d; body: %s", code, body)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil || out.Token == "" {
		t.Fatalf("no superuser token: err=%v body=%s", err, body)
	}
	return app, srv, out.Token
}

func settingsRequest(t *testing.T, method, url, body, token string) (int, string) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, string(b)
}

func settingsPost(t *testing.T, url, body, token string) (int, string) {
	t.Helper()
	return settingsRequest(t, http.MethodPost, url, body, token)
}

func settingsRecordURL(srv *httptest.Server) string {
	return srv.URL + "/api/collections/" + settings.Collection + "/records/" + settings.RecordID
}

// TestMigration0024_SeedsSingletonRow: the migration itself creates the ONE row at the fixed id,
// so no surface ever has to decide whether to create it.
func TestMigration0024_SeedsSingletonRow(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	rec, err := app.FindRecordById(settings.Collection, settings.RecordID)
	if err != nil {
		t.Fatalf("the migration must seed %s/%s: %v", settings.Collection, settings.RecordID, err)
	}
	if rec.GetString(settings.FieldAPIKey) != "" {
		t.Fatal("the seeded row must carry no key")
	}
	n, err := app.CountRecords(settings.Collection)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("settings holds %d rows, want exactly 1", n)
	}
}

// TestMigration0024_RulesAreSuperuserOnly: every API rule nil is PocketBase's superuser-only
// posture. This is the whole access-control story for the collection — there is no middleware.
func TestMigration0024_RulesAreSuperuserOnly(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	col, err := app.FindCollectionByNameOrId(settings.Collection)
	if err != nil {
		t.Fatalf("find settings: %v", err)
	}
	rules := map[string]*string{
		"list": col.ListRule, "view": col.ViewRule, "create": col.CreateRule,
		"update": col.UpdateRule, "delete": col.DeleteRule,
	}
	for name, rule := range rules {
		if rule != nil {
			t.Errorf("settings.%sRule = %q, want nil (superuser only)", name, *rule)
		}
	}
}

// TestMigration0024_APIKeyNeverLeavesOverHTTP is the DoD proof: a SUPERUSER — the strongest
// caller there is — reads the settings record over the real record API after a key is stored, and
// the response carries no key field at all. `Hidden: true` is what enforces it; the visible hint
// is what the browser renders instead.
func TestMigration0024_APIKeyNeverLeavesOverHTTP(t *testing.T) {
	_, srv, token := settingsAPI(t)

	code, body := settingsRequest(t, http.MethodPatch, settingsRecordURL(srv),
		`{"`+settings.FieldProvider+`":"anthropic","`+settings.FieldAPIKey+`":"`+testAPIKey+`"}`, token)
	if code != http.StatusOK {
		t.Fatalf("PATCH settings status = %d, want 200; body: %s", code, body)
	}
	assertNoKeyInBody(t, "PATCH response", body)

	code, body = settingsRequest(t, http.MethodGet, settingsRecordURL(srv), "", token)
	if code != http.StatusOK {
		t.Fatalf("GET settings status = %d, want 200; body: %s", code, body)
	}
	assertNoKeyInBody(t, "GET response", body)

	code, body = settingsRequest(t, http.MethodGet,
		srv.URL+"/api/collections/"+settings.Collection+"/records", "", token)
	if code != http.StatusOK {
		t.Fatalf("LIST settings status = %d, want 200; body: %s", code, body)
	}
	assertNoKeyInBody(t, "LIST response", body)
}

func assertNoKeyInBody(t *testing.T, what, body string) {
	t.Helper()
	if strings.Contains(body, testAPIKey) {
		t.Fatalf("%s leaked the stored API key: %s", what, body)
	}
	if strings.Contains(body, `"`+settings.FieldAPIKey+`"`) {
		t.Fatalf("%s carries the %s field at all — it must be hidden from the API: %s",
			what, settings.FieldAPIKey, body)
	}
	if !strings.Contains(body, settings.FieldAPIKeyHint) {
		t.Fatalf("%s must still carry %s so a browser can show which key is installed: %s",
			what, settings.FieldAPIKeyHint, body)
	}
}

// TestMigration0024_UnauthenticatedCannotRead: nil rules mean the record API refuses anyone who
// is not a superuser, so the key's collection is not merely hidden-field-protected.
func TestMigration0024_UnauthenticatedCannotRead(t *testing.T) {
	_, srv, _ := settingsAPI(t)
	code, body := settingsRequest(t, http.MethodGet, settingsRecordURL(srv), "", "")
	if code == http.StatusOK {
		t.Fatalf("an unauthenticated caller read the settings record: %s", body)
	}
}

// TestMigration0024_HintIsRecomputedServerSide: the hook derives the hint from the key that was
// actually stored, so a client that submits a flattering hint alongside a different key still
// gets the truth back.
func TestMigration0024_HintIsRecomputedServerSide(t *testing.T) {
	app, srv, token := settingsAPI(t)

	code, body := settingsRequest(t, http.MethodPatch, settingsRecordURL(srv),
		`{"`+settings.FieldAPIKey+`":"`+testAPIKey+`","`+settings.FieldAPIKeyHint+`":"LIES"}`, token)
	if code != http.StatusOK {
		t.Fatalf("PATCH status = %d; body: %s", code, body)
	}
	rec, err := app.FindRecordById(settings.Collection, settings.RecordID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got, want := rec.GetString(settings.FieldAPIKeyHint), settings.KeyHint(testAPIKey); got != want {
		t.Fatalf("stored hint = %q, want %q (recomputed from the stored key, never the client's)", got, want)
	}
	if rec.GetString(settings.FieldAPIKey) != testAPIKey {
		t.Fatal("the key itself must be stored verbatim")
	}

	// Clearing the key clears the hint: a hint left behind would claim a key that is gone.
	code, body = settingsRequest(t, http.MethodPatch, settingsRecordURL(srv),
		`{"`+settings.FieldAPIKey+`":""}`, token)
	if code != http.StatusOK {
		t.Fatalf("PATCH clear status = %d; body: %s", code, body)
	}
	rec, err = app.FindRecordById(settings.Collection, settings.RecordID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if got := rec.GetString(settings.FieldAPIKeyHint); got != "" {
		t.Fatalf("hint after clearing the key = %q, want empty", got)
	}
}

// TestMigration0024_SingletonGuardRejectsExtraRow: the fixed id is enforced, not merely
// conventional — a second row would give readers (who read the canonical id only) a silently
// ignored place to put settings.
func TestMigration0024_SingletonGuardRejectsExtraRow(t *testing.T) {
	_, srv, token := settingsAPI(t)
	code, body := settingsPost(t, srv.URL+"/api/collections/"+settings.Collection+"/records",
		`{"id":"settings0000001","`+settings.FieldProvider+`":"openai"}`, token)
	if code == http.StatusOK {
		t.Fatalf("a second settings row was created — the singleton guard is not enforced: %s", body)
	}
}

// TestMigration0024_UpDownUp proves the migration is reversible and re-appliable, and that the
// seeded singleton comes back with it.
func TestMigration0024_UpDownUp(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	mig := findMigration(t, "0024_settings")

	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, err := app.FindCollectionByNameOrId(settings.Collection); err == nil {
		t.Fatal("after down, the settings collection must be gone")
	}
	// Load must stay tolerant of exactly this state: a store whose migrations predate 0024.
	s, err := settings.Load(app)
	if err != nil || s.LLMProvider != "" {
		t.Fatalf("settings.Load on a pre-0024 store = (%+v, %v), want (zero, nil)", s, err)
	}

	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}
	if _, err := app.FindRecordById(settings.Collection, settings.RecordID); err != nil {
		t.Fatalf("after up->down->up, the seeded row must be back: %v", err)
	}
}
