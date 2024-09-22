package v0alpha1

import (
	"fmt"

	"github.com/grafana/grafana/pkg/apimachinery/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	GROUP   = "secret.grafana.app"
	VERSION = "v0alpha1"
)

var SecureValuesResourceInfo = utils.NewResourceInfo(GROUP, VERSION,
	"securevalue", "securevalues", "SecureValue",
	func() runtime.Object { return &SecureValue{} },
	func() runtime.Object { return &SecureValueList{} },
	utils.TableColumns{
		Definition: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Title", Type: "string", Format: "string", Description: "The display name"},
		},
		Reader: func(obj any) ([]interface{}, error) {
			r, ok := obj.(*SecureValue)
			if ok {
				return []interface{}{
					r.Name,
					r.Spec.Title,
				}, nil
			}
			return nil, fmt.Errorf("expected folder")
		},
	},
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: GROUP, Version: VERSION}
)
