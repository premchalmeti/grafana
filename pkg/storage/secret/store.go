package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	secret "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	"github.com/grafana/grafana/pkg/infra/db"
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

func ProvideSecureValueStore(db db.DB, keeper SecretKeeper) (SecureValueStore, error) {
	// TODO... read config and actually use key
	return &secureStore{
		keeper: keeper,
		db:     db,
	}, nil
}

var (
	_ SecureValueStore = (*secureStore)(nil)
)

type secureStore struct {
	keeper SecretKeeper
	db     db.DB
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

// Create implements SecureValueStore.
func (*secureStore) Create(ctx context.Context, s *secret.SecureValue) (*secret.SecureValue, error) {
	panic("unimplemented")
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
	panic("unimplemented")
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
