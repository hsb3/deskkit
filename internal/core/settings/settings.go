// Package settings reads and guards the desk's store-backed LLM settings — the one row a
// browser can edit to point a desk at a provider/model and hand it an API key.
//
// It lives in the store rather than the machine-wide config file because a desk's store is the
// only state that is guaranteed to survive a container redeploy: the central YAML resolves under
// the operator's XDG config home, which on a hosted deployment is outside the mounted data
// volume and is wiped with the container, while the store sits ON that volume.
//
// This package deliberately imports NOTHING from internal/core/config: config consumes these
// settings as a resolution leg, so the dependency runs one way only.
package settings

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The store shape. These names are the contract the migration creates, the hooks guard, and
// the browser PATCHes, so they are declared once here and referenced everywhere else.
const (
	// Collection is the settings collection name. All of its API rules are nil, which in
	// PocketBase means superuser-only — that IS the auth posture, enforced by the framework
	// rather than by any middleware of ours.
	Collection = "settings"

	// RecordID is the fixed id of the ONE settings row. The collection is a singleton: the
	// migration seeds this row, every reader reads this id and no other, and the create hook
	// rejects any record carrying a different id.
	RecordID = "settings0000000"

	FieldProvider   = "llm_provider"
	FieldModel      = "llm_model"
	FieldAPIKey     = "llm_api_key"
	FieldAPIKeyHint = "llm_api_key_hint"
)

// hintLen is how much of the key the visible hint may expose. Four characters is enough for an
// operator to recognize which key is installed and far too little to reconstruct one.
const hintLen = 4

// Settings is the resolved content of the singleton row. A zero value means "nothing stored",
// which every consumer must treat as "fall through to the next resolution leg".
type Settings struct {
	LLMProvider string
	LLMModel    string
	LLMAPIKey   string
}

// KeyHint derives the display hint for a key: its last four characters, empty for an unset key.
// The hook recomputes the stored hint with this on every write, so the hint always describes the
// key actually stored — a browser-supplied hint is never trusted, because a client can send any
// string it likes while storing a different key.
func KeyHint(key string) string {
	if len(key) <= hintLen {
		return key
	}
	return key[len(key)-hintLen:]
}

// Load reads the singleton row through the app. It is deliberately TOLERANT: a store whose
// migrations predate this collection, or one whose row is missing, yields a zero value and no
// error, because config resolution runs on every command and must not break on an older store.
// A genuine database failure still surfaces.
func Load(app core.App) (*Settings, error) {
	if app == nil {
		return &Settings{}, nil
	}
	if _, err := app.FindCollectionByNameOrId(Collection); err != nil {
		return &Settings{}, nil // store predates the settings migration
	}
	rec, err := app.FindRecordById(Collection, RecordID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &Settings{}, nil
		}
		return nil, fmt.Errorf("settings: read %s/%s: %w", Collection, RecordID, err)
	}
	return &Settings{
		LLMProvider: rec.GetString(FieldProvider),
		LLMModel:    rec.GetString(FieldModel),
		LLMAPIKey:   rec.GetString(FieldAPIKey),
	}, nil
}

// LoadFromDir reads the singleton row straight out of a store directory's data.db, for the
// surfaces that must NOT open a full app — `config show` opens no store and creates nothing, and
// a PocketBase bootstrap would MkdirAll the data dir and run system migrations against it.
//
// An absent store is not an error: it yields a zero value, so a display surface can consult the
// leg honestly instead of guessing at it. Reads are plain SELECTs; nothing here writes.
func LoadFromDir(dir string) (*Settings, error) {
	if dir == "" {
		return &Settings{}, nil
	}
	dbPath := filepath.Join(dir, "data.db")
	if _, err := os.Stat(dbPath); err != nil {
		return &Settings{}, nil
	}
	db, err := core.DefaultDBConnect(dbPath)
	if err != nil {
		return nil, fmt.Errorf("settings: open %s: %w", dbPath, err)
	}
	defer db.Close()

	var row struct {
		Provider string `db:"llm_provider"`
		Model    string `db:"llm_model"`
		APIKey   string `db:"llm_api_key"`
	}
	err = db.Select(FieldProvider, FieldModel, FieldAPIKey).
		From(Collection).
		Where(dbx.HashExp{"id": RecordID}).
		One(&row)
	if err != nil {
		// A missing table (older store) and a missing row are the same answer: nothing stored.
		return &Settings{}, nil
	}
	return &Settings{LLMProvider: row.Provider, LLMModel: row.Model, LLMAPIKey: row.APIKey}, nil
}

// BindHooks installs the guards on the settings collection. Bound under serve, BEFORE the router
// starts serving, because one of them is load-bearing for the key's confidentiality.
//
//   - the key is re-hidden from every enriched API response. `Hidden: true` on the field is not
//     sufficient on its own: PocketBase deliberately UNHIDES every hidden field for a superuser
//     caller (apis.autoResolveRecordsFlags), and a superuser is the ONLY caller this collection
//     admits — so without this hook the key would come back in full on every read. Re-verify this
//     hook against apis/record_helpers.go on a PocketBase bump.
//   - the hint is recomputed SERVER-SIDE from the submitted key on both create and update, so a
//     client cannot make the displayed hint disagree with the stored key.
//   - a create carrying any id but the canonical one is refused, which is what keeps the
//     collection a singleton (the migration seeds the one row; nothing else may add another).
func BindHooks(app core.App) {
	app.OnRecordEnrich(Collection).BindFunc(func(e *core.RecordEnrichEvent) error {
		// Captured before e.Next(): the event struct is REUSED across the records of a list
		// response, so e.Record no longer refers to this record once the chain returns.
		rec := e.Record
		if err := e.Next(); err != nil {
			return err
		}
		rec.Hide(FieldAPIKey)
		return nil
	})
	app.OnRecordCreate(Collection).BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id != RecordID {
			return fmt.Errorf("%s is a singleton: the only permitted record id is %q", Collection, RecordID)
		}
		e.Record.Set(FieldAPIKeyHint, KeyHint(e.Record.GetString(FieldAPIKey)))
		return e.Next()
	})
	app.OnRecordUpdate(Collection).BindFunc(func(e *core.RecordEvent) error {
		e.Record.Set(FieldAPIKeyHint, KeyHint(e.Record.GetString(FieldAPIKey)))
		return e.Next()
	})
}
