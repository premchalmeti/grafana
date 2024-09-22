package secret

import (
	"context"
	"fmt"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	secret "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	secretstore "github.com/grafana/grafana/pkg/storage/secret"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Scoper               = (*secretStorage)(nil)
	_ rest.SingularNameProvider = (*secretStorage)(nil)
	_ rest.Getter               = (*secretStorage)(nil)
	_ rest.Lister               = (*secretStorage)(nil)
	_ rest.Storage              = (*secretStorage)(nil)
	_ rest.Creater              = (*secretStorage)(nil)
	_ rest.Updater              = (*secretStorage)(nil)
	_ rest.GracefulDeleter      = (*secretStorage)(nil)
)

type secretStorage struct {
	store          secretstore.SecureValueStore
	resource       utils.ResourceInfo
	tableConverter rest.TableConvertor
}

func (s *secretStorage) New() runtime.Object {
	return s.resource.NewFunc()
}

func (s *secretStorage) Destroy() {}

func (s *secretStorage) NamespaceScoped() bool {
	return true // namespace == org
}

func (s *secretStorage) GetSingularName() string {
	return s.resource.GetSingularName()
}

func (s *secretStorage) NewList() runtime.Object {
	return s.resource.NewListFunc()
}

func (s *secretStorage) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return s.tableConverter.ConvertToTable(ctx, object, tableOptions)
}

func (s *secretStorage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	ns := request.NamespaceValue(ctx)
	return s.store.List(ctx, ns, options)
}

func (s *secretStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns := request.NamespaceValue(ctx)
	return s.store.Read(ctx, ns, name)
}

func (s *secretStorage) Create(ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions,
) (runtime.Object, error) {
	sv, ok := obj.(*secret.SecureValue)
	if !ok {
		return nil, fmt.Errorf("expected SecureValue for create")
	}
	return s.store.Create(ctx, sv)
}

func (s *secretStorage) Update(ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool,
	options *metav1.UpdateOptions,
) (runtime.Object, bool, error) {
	created := false
	old, err := s.Get(ctx, name, nil)
	if err != nil {
		return old, created, err
	}

	// makes sure the UID and RV are OK
	obj, err := objInfo.UpdatedObject(ctx, old)
	if err != nil {
		return old, created, err
	}

	sv, ok := obj.(*secret.SecureValue)
	if !ok {
		return nil, created, fmt.Errorf("expected SecureValue for update")
	}

	// Is this really a create request
	if sv.UID == "" {
		n, err := s.Create(ctx, sv, nil, &metav1.CreateOptions{})
		return n, true, err
	}

	sv, err = s.store.Update(ctx, sv)
	return sv, created, err
}

// GracefulDeleter
func (s *secretStorage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	ns := request.NamespaceValue(ctx)
	return s.store.Delete(ctx, ns, name)
}

// CollectionDeleter
func (s *secretStorage) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *internalversion.ListOptions) (runtime.Object, error) {
	return nil, fmt.Errorf("DeleteCollection for secrets not implemented")
}
