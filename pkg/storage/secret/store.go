package secret

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/authlib/claims"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	secret "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/storage/unified/sql/sqltemplate"
	"github.com/grafana/grafana/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type SecureValueStore interface {
	Create(ctx context.Context, s *secret.SecureValue) (*secret.SecureValue, error)
	Read(ctx context.Context, ns string, name string) (*secret.SecureValue, error)
	Update(ctx context.Context, s *secret.SecureValue) (*secret.SecureValue, error)
	Delete(ctx context.Context, ns string, name string) (*secret.SecureValue, bool, error)
	List(ctx context.Context, ns string, options *internalversion.ListOptions) (*secret.SecureValueList, error)

	// Decrypt a single value -- the identity is in context
	Decrypt(ctx context.Context, ns string, name string) (string, error)

	// Show the history for a single value
	History(ctx context.Context, ns string, name string, continueToken string) (*secret.SecureValueActivity, error)
}

func ProvideSecureValueStore(db db.DB, keeper SecretKeeper, cfg *setting.Cfg) (SecureValueStore, error) {
	// Run SQL migrations
	err := MigrateSecretStore(context.Background(), db.GetEngine(), cfg)
	if err != nil {
		return nil, err
	}

	// One version of DB?
	return &secureStore{
		keeper:  keeper,
		db:      db,
		dialect: sqltemplate.DialectForDriver(string(db.GetDBType())),
	}, nil
}

var (
	_ SecureValueStore = (*secureStore)(nil)

	//go:embed *.sql
	sqlTemplatesFS embed.FS

	sqlTemplates = template.Must(template.New("sql").ParseFS(sqlTemplatesFS, `*.sql`))

	// The SQL Commands
	sqlSecureValueInsert = mustTemplate("secure_value_insert.sql")
	sqlSecureValueUpdate = mustTemplate("secure_value_update.sql")
)

func mustTemplate(filename string) *template.Template {
	if t := sqlTemplates.Lookup(filename); t != nil {
		return t
	}
	panic(fmt.Sprintf("template file not found: %s", filename))
}

type secureStore struct {
	keeper  SecretKeeper
	db      db.DB
	dialect sqltemplate.Dialect
}

type secretValueRow struct {
	UID         string
	Namespace   string
	Name        string
	Title       string
	Salt        string
	Value       string
	Keeper      string
	Addr        string
	Created     int64
	CreatedBy   string
	Updated     int64
	UpdatedBy   string
	Annotations string // map[string]string
	Labels      string // map[string]string
	APIs        string // []string
}

// Create implements SecureValueStore.
func (v *secretValueRow) toK8s() (*secret.SecureValue, error) {
	val := &secret.SecureValue{
		ObjectMeta: v1.ObjectMeta{
			Name:              v.Name,
			Namespace:         v.Namespace,
			UID:               types.UID(v.UID),
			CreationTimestamp: v1.NewTime(time.UnixMilli(v.Created)),
		},
		Spec: secret.SecureValueSpec{
			Title: v.Title,
		},
	}

	if v.APIs != "" {
		err := json.Unmarshal([]byte(v.APIs), &val.Spec.APIs)
		if err != nil {
			return nil, err
		}
	}
	if v.Annotations != "" {
		err := json.Unmarshal([]byte(v.Annotations), &val.Annotations)
		if err != nil {
			return nil, err
		}
	}
	if v.Labels != "" {
		err := json.Unmarshal([]byte(v.Labels), &val.Labels)
		if err != nil {
			return nil, err
		}
	}

	meta, err := utils.MetaAccessor(val)
	if err != nil {
		return nil, err
	}
	meta.SetCreatedBy(v.CreatedBy)
	if v.Updated != v.Created {
		meta.SetUpdatedBy(v.UpdatedBy)
		meta.SetUpdatedTimestampMillis(v.Updated)
	}
	meta.SetResourceVersionInt64(v.Updated) // yes millis RV
	return val, nil
}

type createSecureValue struct {
	sqltemplate.SQLTemplate
	Row *secretValueRow
}

func (r createSecureValue) Validate() error {
	return nil // TODO
}

type updateSecureValue struct {
	sqltemplate.SQLTemplate
	Row *secretValueRow
}

func (r updateSecureValue) Validate() error {
	return nil // TODO
}

// Create implements SecureValueStore.
func (s *secureStore) Create(ctx context.Context, v *secret.SecureValue) (*secret.SecureValue, error) {
	authInfo, ok := claims.From(ctx)
	if !ok {
		return nil, fmt.Errorf("missing auth info in context")
	}

	now := time.Now().UnixMilli()
	row := &secretValueRow{
		UID:       uuid.NewString(),
		Namespace: v.Namespace,
		Name:      v.Name,
		Title:     v.Spec.Title,
		Salt:      util.GenerateShortUID(), // not exposed in the UI
		Value:     v.Spec.Value,
		Created:   now,
		Updated:   now,
		CreatedBy: authInfo.GetUID(),
		UpdatedBy: authInfo.GetUID(),
	}
	if row.Name == "" {
		row.Name = util.GenerateShortUID()
	}
	if len(v.Labels) > 0 {
		v, err := json.Marshal(v.Labels)
		if err != nil {
			return nil, err
		}
		row.Labels = string(v)
	}
	if len(v.Spec.APIs) > 0 {
		v, err := json.Marshal(v.Spec.APIs)
		if err != nil {
			return nil, err
		}
		row.APIs = string(v)
	}
	if len(v.Annotations) > 0 {
		v, err := json.Marshal(v.Annotations)
		if err != nil {
			return nil, err
		}
		row.Annotations = string(v)
	}

	// insert
	req := &createSecureValue{
		SQLTemplate: sqltemplate.New(s.dialect),
		Row:         row,
	}
	q, err := sqltemplate.Execute(sqlSecureValueInsert, req)
	if err != nil {
		return nil, fmt.Errorf("insert template %q: %w", q, err)
	}

	fmt.Printf("CREATE: %s\n", q)

	_, err = s.db.GetSqlxSession().Exec(ctx, q, req.GetArgs()...)
	if err != nil {
		return nil, err
	}

	return row.toK8s()
}

// Get implements SecureValueStore.
func (s *secureStore) Read(ctx context.Context, ns string, name string) (*secret.SecureValue, error) {
	v, err := s.get(ctx, ns, name)
	if err != nil {
		return nil, err
	}
	return v.toK8s()
}

// Update implements SecureValueStore.
func (*secureStore) Update(ctx context.Context, s *secret.SecureValue) (*secret.SecureValue, error) {
	panic("unimplemented")
}

// Delete implements SecureValueStore.
func (s *secureStore) Delete(ctx context.Context, ns string, name string) (*secret.SecureValue, bool, error) {
	panic("unimplemented")
}

// List implements SecureValueStore.
func (s *secureStore) List(ctx context.Context, ns string, options *internalversion.ListOptions) (*secret.SecureValueList, error) {
	return &secret.SecureValueList{
		Items: []secret.SecureValue{},
	}, nil // nothing
}

// Decrypt implements SecureValueStore.
func (s *secureStore) Decrypt(ctx context.Context, ns string, name string) (string, error) {
	v, err := s.get(ctx, ns, name)
	if err != nil {
		return "", err
	}

	// TODO!!!
	if v.APIs != "" {
		fmt.Printf("MAKE SURE ctx is an app that can read: %s\n", v.APIs)
	}

	return s.keeper.Decode(ctx, SaltyValue{
		Value:  v.Value,
		Salt:   v.Salt,
		Keeper: v.Keeper,
		Addr:   v.Addr,
	})
}

// History implements SecureValueStore.
func (s *secureStore) History(ctx context.Context, ns string, name string, continueToken string) (*secret.SecureValueActivity, error) {
	panic("unimplemented")
}

func (s *secureStore) get(ctx context.Context, ns string, name string) (*secretValueRow, error) {
	return nil, fmt.Errorf("TODO")
}
