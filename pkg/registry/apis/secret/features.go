package secret

import (
	"context"
	"fmt"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	secret "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Storage              = (*featuresStorage)(nil)
	_ rest.Scoper               = (*featuresStorage)(nil)
	_ rest.SingularNameProvider = (*featuresStorage)(nil)
	_ rest.Lister               = (*featuresStorage)(nil)
	_ rest.Getter               = (*featuresStorage)(nil)
)

type featuresStorage struct {
	resource       *utils.ResourceInfo
	tableConverter rest.TableConvertor
}

// NOTE! this does not depend on config or any system state!
// In the future, the existence of features (and their properties) can be defined dynamically
func NewFeaturesStorage() *featuresStorage {
	resourceInfo := secret.SecureValuesResourceInfo
	return &featuresStorage{
		resource:       &resourceInfo,
		tableConverter: resourceInfo.TableConverter(),
	}
}

func (s *featuresStorage) New() runtime.Object {
	return s.resource.NewFunc()
}

func (s *featuresStorage) Destroy() {}

func (s *featuresStorage) NamespaceScoped() bool {
	return false
}

func (s *featuresStorage) GetSingularName() string {
	return s.resource.GetSingularName()
}

func (s *featuresStorage) NewList() runtime.Object {
	return s.resource.NewListFunc()
}

func (s *featuresStorage) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return s.tableConverter.ConvertToTable(ctx, object, tableOptions)
}

func (s *featuresStorage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	return s.resource.NewListFunc(), nil
}

func (s *featuresStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return nil, fmt.Errorf("not found")
}
