package secret

import (
	"context"
	"fmt"

	"github.com/grafana/authlib/claims"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	secret "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
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
	resource       utils.ResourceInfo
	namespacer     claims.NamespaceFormatter
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
	list := &secret.SecureValueList{}
	return list, nil
}

func (s *secretStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns := request.NamespaceValue(ctx)
	fmt.Printf("GET: %s/%s\n", ns, name)
	return nil, fmt.Errorf("TODO...")
}

func (s *secretStorage) Create(ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions,
) (runtime.Object, error) {
	ns := request.NamespaceValue(ctx)
	fmt.Printf("CREATE: %s/%+v\n", ns, obj)
	return nil, fmt.Errorf("TODO Create...")
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

	obj, err := objInfo.UpdatedObject(ctx, old)
	if err != nil {
		return old, created, err
	}

	p, ok := obj.(*secret.SecureValue)
	if !ok {
		return nil, created, fmt.Errorf("expected playlist after update")
	}

	fmt.Printf("UPDATED: %+v\n", p)

	r, err := s.Get(ctx, name, nil)
	return r, created, err
}

// GracefulDeleter
func (s *secretStorage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	v, err := s.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return v, false, err // includes the not-found error
	}
	fmt.Printf("UPDATED: %+v\n", v)
	return nil, true, fmt.Errorf("TODO")
}

// CollectionDeleter
func (s *secretStorage) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *internalversion.ListOptions) (runtime.Object, error) {
	return nil, fmt.Errorf("DeleteCollection for secrets not implemented")
}
