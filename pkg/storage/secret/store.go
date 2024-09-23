package secret

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

type SecureValueStore interface {
	Create(ctx context.Context, s *secret.SecureValue) (*secret.SecureValue, error)
	Update(ctx context.Context, s *secret.SecureValue) (*secret.SecureValue, error)
	Delete(ctx context.Context, ns string, name string) (*secret.SecureValue, bool, error)
	List(ctx context.Context, ns string, options *internalversion.ListOptions) (*secret.SecureValueList, error)

	// The value will not be included
	Read(ctx context.Context, ns string, name string) (*secret.SecureValue, error)

	// Return a version that has the secure value visible
	Decrypt(ctx context.Context, ns string, name string) (*secret.SecureValue, error)

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
	sqlSecureValueList   = mustTemplate("secure_value_list.sql")
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

type secureValueRow struct {
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

func toSecureValueRow(v *secret.SecureValue) (*secureValueRow, error) {
	meta, err := utils.MetaAccessor(v)
	if err != nil {
		return nil, err
	}
	row := &secureValueRow{
		UID:       uuid.NewString(),
		Namespace: v.Namespace,
		Name:      v.Name,
		Title:     v.Spec.Title,
		Value:     v.Spec.Value,
		Created:   meta.GetCreationTimestamp().UnixMilli(),
		CreatedBy: meta.GetCreatedBy(),
		UpdatedBy: meta.GetUpdatedBy(),
	}
	t, _ := meta.GetUpdatedTimestamp()
	if t != nil {
		row.Updated = t.UnixMilli()
	} else {
		row.Updated = row.Created
	}

	if len(v.Labels) > 0 {
		v, err := json.Marshal(v.Labels)
		if err != nil {
			return row, err
		}
		row.Labels = string(v)
	}
	if len(v.Spec.APIs) > 0 {
		v, err := json.Marshal(v.Spec.APIs)
		if err != nil {
			return row, err
		}
		row.APIs = string(v)
	}
	if len(v.Annotations) > 0 {
		anno := make(map[string]string)
		for k, v := range v.Annotations {
			if !strings.HasPrefix("grafana.app/", k) {
				anno[k] = v
			}
		}
		v, err := json.Marshal(anno)
		if err != nil {
			return row, err
		}
		row.Annotations = string(v)
	}
	return row, nil
}

// Create implements SecureValueStore.
func (v *secureValueRow) toK8s() (*secret.SecureValue, error) {
	val := &secret.SecureValue{
		ObjectMeta: metav1.ObjectMeta{
			Name:              v.Name,
			Namespace:         v.Namespace,
			UID:               types.UID(v.UID),
			CreationTimestamp: metav1.NewTime(time.UnixMilli(v.Created)),
			Labels:            make(map[string]string),
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
	Row *secureValueRow
}

func (r createSecureValue) Validate() error {
	return nil // TODO
}

type updateSecureValue struct {
	sqltemplate.SQLTemplate
	Row *secureValueRow
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
	if v.Name == "" {
		return nil, fmt.Errorf("missing name")
	}
	if v.Spec.Value == "" {
		return nil, fmt.Errorf("missing value")
	}

	v.CreationTimestamp = metav1.NewTime(time.Now())
	row, err := toSecureValueRow(v)
	if err != nil {
		return nil, err
	}
	row.CreatedBy = authInfo.GetUID()
	row.UpdatedBy = authInfo.GetUID()
	row.Salt, err = util.GetRandomString(10)
	if err != nil {
		return nil, err
	}
	row.Value, err = s.keeper.Encode(ctx, SaltyValue{
		Value: v.Spec.Value,
		Salt:  row.Salt,
	})
	if err != nil {
		return nil, err
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

type listSecureValues struct {
	sqltemplate.SQLTemplate
	Request secureValueRow
}

func (r listSecureValues) Validate() error {
	return nil // TODO
}

// List implements SecureValueStore.
func (s *secureStore) List(ctx context.Context, ns string, options *internalversion.ListOptions) (*secret.SecureValueList, error) {
	req := &listSecureValues{
		SQLTemplate: sqltemplate.New(s.dialect),
		Request: secureValueRow{
			Namespace: ns,
		},
	}
	q, err := sqltemplate.Execute(sqlSecureValueList, req)
	if err != nil {
		return nil, fmt.Errorf("list template %q: %w", q, err)
	}

	selector := options.LabelSelector
	if selector == nil {
		selector = labels.Everything()
	}

	row := &secureValueRow{}
	list := &secret.SecureValueList{}
	rows, err := s.db.GetSqlxSession().Query(ctx, q, req.GetArgs()...)
	if err != nil {
		return nil, fmt.Errorf("list template %q: %w", q, err)
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		err = rows.Scan(&row.UID,
			&row.Namespace, &row.Name, &row.Title,
			&row.Salt, &row.Value,
			&row.Keeper, &row.Addr,
			&row.Created, &row.CreatedBy,
			&row.Updated, &row.UpdatedBy,
			&row.Annotations, &row.Labels,
			&row.APIs,
		)
		if err != nil {
			return nil, err
		}
		obj, err := row.toK8s()
		if err != nil {
			return nil, err
		}
		if selector.Matches(labels.Set(obj.Labels)) {
			list.Items = append(list.Items, *obj)
		}
	}
	return list, nil // nothing
}

// Decrypt implements SecureValueStore.
func (s *secureStore) Decrypt(ctx context.Context, ns string, name string) (*secret.SecureValue, error) {
	row, err := s.get(ctx, ns, name)
	if err != nil {
		return nil, err
	}

	// TODO!!!
	if row.APIs != "" {
		fmt.Printf("MAKE SURE ctx is an app that can read: %s\n", row.APIs)
	}

	v, err := row.toK8s()
	if err != nil {
		return nil, err
	}
	v.Spec.Value, err = s.keeper.Decode(ctx, SaltyValue{
		Value:  row.Value,
		Salt:   row.Salt,
		Keeper: row.Keeper,
		Addr:   row.Addr,
	})
	return v, err
}

// History implements SecureValueStore.
func (s *secureStore) History(ctx context.Context, ns string, name string, continueToken string) (*secret.SecureValueActivity, error) {
	panic("unimplemented")
}

func (s *secureStore) get(ctx context.Context, ns string, name string) (*secureValueRow, error) {
	req := &listSecureValues{
		SQLTemplate: sqltemplate.New(s.dialect),
		Request: secureValueRow{
			Namespace: ns,
			Name:      name,
		},
	}
	q, err := sqltemplate.Execute(sqlSecureValueList, req)
	if err != nil {
		return nil, fmt.Errorf("list template %q: %w", q, err)
	}

	rows, err := s.db.GetSqlxSession().Query(ctx, q, req.GetArgs()...)
	if err != nil {
		return nil, fmt.Errorf("list template %q: %w", q, err)
	}
	defer func() {
		_ = rows.Close()
	}()
	if rows.Next() {
		row := &secureValueRow{}
		err = rows.Scan(&row.UID,
			&row.Namespace, &row.Name, &row.Title,
			&row.Salt, &row.Value,
			&row.Keeper, &row.Addr,
			&row.Created, &row.CreatedBy,
			&row.Updated, &row.UpdatedBy,
			&row.Annotations, &row.Labels,
			&row.APIs,
		)
		return row, err
	}
	return nil, fmt.Errorf("not found")
}
