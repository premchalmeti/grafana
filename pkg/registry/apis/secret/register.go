package secret

import (
	secret "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	grafanarest "github.com/grafana/grafana/pkg/apiserver/rest"
	"github.com/grafana/grafana/pkg/services/apiserver/builder"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	common "k8s.io/kube-openapi/pkg/common"
)

var _ builder.APIGroupBuilder = (*SecretAPIBuilder)(nil)

// This is used just so wire has something unique to return
type SecretAPIBuilder struct {
	// TODO...
}

func NewSecretAPIBuilder() *SecretAPIBuilder {
	return &SecretAPIBuilder{}
}

func RegisterAPIService(features featuremgmt.FeatureToggles,
	apiregistration builder.APIRegistrar,
) *SecretAPIBuilder {
	if !features.IsEnabledGlobally(featuremgmt.FlagGrafanaAPIServerWithExperimentalAPIs) {
		return nil // skip registration unless opting into experimental apis
	}

	builder := NewSecretAPIBuilder()
	apiregistration.RegisterAPI(builder)
	return builder
}

func (b *SecretAPIBuilder) GetGroupVersion() schema.GroupVersion {
	return secret.SecureValuesResourceInfo.GroupVersion()
}

func addKnownTypes(scheme *runtime.Scheme, gv schema.GroupVersion) {
	scheme.AddKnownTypes(gv,
		&secret.SecureValue{},
		&secret.SecureValueList{},
	)
}

func (b *SecretAPIBuilder) InstallSchema(scheme *runtime.Scheme) error {
	gv := secret.SecureValuesResourceInfo.GroupVersion()
	addKnownTypes(scheme, gv)

	// // Link this version to the internal representation.
	// // This is used for server-side-apply (PATCH), and avoids the error:
	// //   "no kind is registered for the type"
	// addKnownTypes(scheme, schema.GroupVersion{
	// 	Group:   gv.Group,
	// 	Version: runtime.APIVersionInternal,
	// })

	// If multiple versions exist, then register conversions from zz_generated.conversion.go
	// if err := playlist.RegisterConversions(scheme); err != nil {
	//   return err
	// }
	metav1.AddToGroupVersion(scheme, gv)
	return scheme.SetVersionPriority(gv)
}

func (b *SecretAPIBuilder) GetAPIGroupInfo(
	scheme *runtime.Scheme,
	codecs serializer.CodecFactory, // pointer?
	_ generic.RESTOptionsGetter,
	_ grafanarest.DualWriteBuilder,
) (*genericapiserver.APIGroupInfo, error) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(secret.GROUP, scheme, metav1.ParameterCodec, codecs)

	featureStore := NewFeaturesStorage()

	storage := map[string]rest.Storage{}
	storage[featureStore.resource.StoragePath()] = featureStore

	apiGroupInfo.VersionedResourcesStorageMap[secret.VERSION] = storage
	return &apiGroupInfo, nil
}

func (b *SecretAPIBuilder) GetOpenAPIDefinitions() common.GetOpenAPIDefinitions {
	return secret.GetOpenAPIDefinitions
}

func (b *SecretAPIBuilder) GetAuthorizer() authorizer.Authorizer {
	return nil // default authorizer is fine
}

// Register additional routes with the server
func (b *SecretAPIBuilder) GetAPIRoutes() *builder.APIRoutes {
	return nil
}
