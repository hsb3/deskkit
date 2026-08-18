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
)

// usersAPI stands up the real record API over a migrated test store, so these tests exercise the
// SHIPPED rule strings through the same endpoints a browser or a curl transcript would hit —
// not a hand-rolled re-check of the rule semantics.
func usersAPI(t *testing.T) (*tests.TestApp, *httptest.Server) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// The test-app fixture ships `users` with MFA on, which turns every successful first factor
	// into a 401 + mfaId challenge and would mask the approval gate's own verdict. MFA is
	// orthogonal to this migration (which touches only the rules and the `approved` field), so
	// switch it off here to isolate what is under test.
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users: %v", err)
	}
	if usersCol.MFA.Enabled {
		usersCol.MFA.Enabled = false
		if err := app.Save(usersCol); err != nil {
			t.Fatalf("disable MFA on the users fixture: %v", err)
		}
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
	return app, srv
}

func postAPI(t *testing.T, url, body string) (int, string) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, string(b)
}

const (
	testUserEmail = "member@desk.test"
	testUserPass  = "a-sufficiently-long-password"
)

// TestMigration0023_ApprovalGate is the DoD's approval-gate proof: a self-created user cannot
// authenticate; after an operator (superuser-side, i.e. server-side) marks it verified AND
// approved, the very same credentials authenticate. Both halves run through
// /api/collections/users/auth-with-password.
func TestMigration0023_ApprovalGate(t *testing.T) {
	app, srv := usersAPI(t)
	signup := srv.URL + "/api/collections/users/records"
	login := srv.URL + "/api/collections/users/auth-with-password"

	// Public signup is still allowed (a hosted desk needs a way in).
	code, body := postAPI(t, signup, `{"email":"`+testUserEmail+`","password":"`+testUserPass+
		`","passwordConfirm":"`+testUserPass+`"}`)
	if code != http.StatusOK {
		t.Fatalf("signup status = %d, want 200; body: %s", code, body)
	}

	// ...but the fresh account cannot authenticate: it is neither verified nor approved.
	code, body = postAPI(t, login, `{"identity":"`+testUserEmail+`","password":"`+testUserPass+`"}`)
	if code == http.StatusOK {
		t.Fatalf("an unapproved user authenticated (status 200) — the approval gate is not enforced; body: %s", body)
	}
	if code != http.StatusForbidden && code != http.StatusUnauthorized {
		t.Fatalf("unapproved login status = %d, want 401 or 403; body: %s", code, body)
	}

	// The operator approves it server-side (the only place `approved` can be set — see the
	// self-approval tests below).
	rec, err := app.FindAuthRecordByEmail("users", testUserEmail)
	if err != nil {
		t.Fatalf("find the signed-up record: %v", err)
	}
	if rec.GetBool("approved") {
		t.Fatal("a freshly signed-up record must NOT be approved")
	}
	rec.SetVerified(true)
	rec.Set("approved", true)
	if err := app.Save(rec); err != nil {
		t.Fatalf("approve: %v", err)
	}

	// Same credentials, now accepted, and the response really carries a token.
	code, body = postAPI(t, login, `{"identity":"`+testUserEmail+`","password":"`+testUserPass+`"}`)
	if code != http.StatusOK {
		t.Fatalf("approved login status = %d, want 200; body: %s", code, body)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil || out.Token == "" {
		t.Fatalf("approved login must return a token; err=%v body=%s", err, body)
	}
}

// TestMigration0023_VerifiedAloneIsNotEnough: `approved` is a real second factor, not decoration.
// A verified-but-unapproved account — the state PocketBase's own email-verification flow produces
// unaided — must still be refused.
func TestMigration0023_VerifiedAloneIsNotEnough(t *testing.T) {
	app, srv := usersAPI(t)
	login := srv.URL + "/api/collections/users/auth-with-password"

	code, body := postAPI(t, srv.URL+"/api/collections/users/records",
		`{"email":"`+testUserEmail+`","password":"`+testUserPass+`","passwordConfirm":"`+testUserPass+`"}`)
	if code != http.StatusOK {
		t.Fatalf("signup status = %d, want 200; body: %s", code, body)
	}

	rec, err := app.FindAuthRecordByEmail("users", testUserEmail)
	if err != nil {
		t.Fatalf("find record: %v", err)
	}
	rec.SetVerified(true) // verified but NOT approved
	if err := app.Save(rec); err != nil {
		t.Fatalf("verify: %v", err)
	}

	code, body = postAPI(t, login, `{"identity":"`+testUserEmail+`","password":"`+testUserPass+`"}`)
	if code == http.StatusOK {
		t.Fatalf("a verified-but-unapproved user authenticated — `approved` is not part of the gate; body: %s", body)
	}
}

// TestMigration0023_CannotSelfApproveAtSignup: the create rule rejects the PRESENCE of `approved`
// in the request body, so a client cannot bootstrap itself past the gate. Both true and false are
// rejected: allowing `false` would let a client establish the field and then patch it.
func TestMigration0023_CannotSelfApproveAtSignup(t *testing.T) {
	_, srv := usersAPI(t)
	signup := srv.URL + "/api/collections/users/records"

	for _, val := range []string{"true", "false"} {
		code, body := postAPI(t, signup, `{"email":"self`+val+`@desk.test","password":"`+testUserPass+
			`","passwordConfirm":"`+testUserPass+`","approved":`+val+`}`)
		if code == http.StatusOK {
			t.Fatalf("signup carrying approved:%s was accepted — a client must never set its own approval; body: %s", val, body)
		}
	}
}

// TestMigration0023_CannotSelfApproveOnUpdate: an APPROVED, authenticated user (the strongest
// non-superuser position available) still cannot flip `approved` on its own record. The update
// rule's no-self-approval clause is what keeps approval an operator-only fact.
func TestMigration0023_CannotSelfApproveOnUpdate(t *testing.T) {
	app, srv := usersAPI(t)

	code, body := postAPI(t, srv.URL+"/api/collections/users/records",
		`{"email":"`+testUserEmail+`","password":"`+testUserPass+`","passwordConfirm":"`+testUserPass+`"}`)
	if code != http.StatusOK {
		t.Fatalf("signup status = %d; body: %s", code, body)
	}
	rec, err := app.FindAuthRecordByEmail("users", testUserEmail)
	if err != nil {
		t.Fatalf("find record: %v", err)
	}
	rec.SetVerified(true)
	rec.Set("approved", true)
	if err := app.Save(rec); err != nil {
		t.Fatalf("approve: %v", err)
	}

	code, body = postAPI(t, srv.URL+"/api/collections/users/auth-with-password",
		`{"identity":"`+testUserEmail+`","password":"`+testUserPass+`"}`)
	if code != http.StatusOK {
		t.Fatalf("login status = %d; body: %s", code, body)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(body), &out); err != nil || out.Token == "" {
		t.Fatalf("no token in login response: %s", body)
	}

	req, err := http.NewRequest(http.MethodPatch,
		srv.URL+"/api/collections/users/records/"+rec.Id, strings.NewReader(`{"approved":false}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", out.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("an authenticated user changed its own `approved` flag — approval must be operator-only")
	}
}

// TestMigration0023_DeleteIsSuperuserOnly: the stock collection lets a user delete its own record.
// Under an approval gate that is a bypass — a rejected account could churn itself back to a fresh
// signup — so DeleteRule must be nil (superuser only).
func TestMigration0023_DeleteIsSuperuserOnly(t *testing.T) {
	app, _ := usersAPI(t)
	c, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users: %v", err)
	}
	if c.DeleteRule != nil {
		t.Fatalf("users.DeleteRule = %q, want nil (superuser only)", *c.DeleteRule)
	}
}

// TestMigration0023_UpDownUp proves the migration is reversible and re-appliable: down restores
// the dependency's stock posture (no `approved` field, open auth rule) and up re-hardens it.
func TestMigration0023_UpDownUp(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	approvedField := func() pbcore.Field {
		c, ferr := app.FindCollectionByNameOrId("users")
		if ferr != nil {
			t.Fatalf("find users: %v", ferr)
		}
		return c.Fields.GetByName("approved")
	}
	authRule := func() string {
		c, ferr := app.FindCollectionByNameOrId("users")
		if ferr != nil {
			t.Fatalf("find users: %v", ferr)
		}
		if c.AuthRule == nil {
			return "<nil>"
		}
		return *c.AuthRule
	}

	if approvedField() == nil {
		t.Fatal("fresh store: users.approved missing")
	}
	if got := authRule(); got != usersAuthRule {
		t.Fatalf("fresh store: users.AuthRule = %q, want %q", got, usersAuthRule)
	}

	mig := findMigration(t, "0023_users_approval")

	if err := mig.Down(app); err != nil {
		t.Fatalf("down: %v", err)
	}
	if approvedField() != nil {
		t.Fatal("after down, users.approved must be gone")
	}
	if got := authRule(); got != usersStockAuthRule {
		t.Fatalf("after down, users.AuthRule = %q, want the stock %q", got, usersStockAuthRule)
	}

	if err := mig.Up(app); err != nil {
		t.Fatalf("up: %v", err)
	}
	if approvedField() == nil {
		t.Fatal("after up->down->up, users.approved must be present again")
	}
	if got := authRule(); got != usersAuthRule {
		t.Fatalf("after up->down->up, users.AuthRule = %q, want %q", got, usersAuthRule)
	}
}
