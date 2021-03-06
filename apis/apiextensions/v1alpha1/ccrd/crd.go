/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package ccrd generates CustomResourceDefinitions from Crossplane definitions.
//
// v1.JSONSchemaProps is incompatible with controller-tools (as of 0.2.4)
// because it is missing JSON tags and uses float64, which is a disallowed type.
// We thus copy the entire struct as CRDSpecTemplate. See the below issue:
// https://github.com/kubernetes-sigs/controller-tools/issues/291
package ccrd

import (
	"encoding/json"

	"github.com/pkg/errors"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crossplane/crossplane-runtime/pkg/meta"

	"github.com/crossplane/crossplane/apis/apiextensions/v1alpha1"
)

// Category names for generated claim and composite CRDs.
const (
	CategoryClaim     = "claim"
	CategoryComposite = "composite"
)

const (
	errGetSpecProps            = "cannot get spec properties from validation schema"
	errParseValidation         = "cannot parse validation schema"
	errInvalidClaimNames       = "invalid resource claim names"
	errMissingClaimNames       = "missing names"
	errFmtConflictingClaimName = "%q conflicts with composite resource name"
)

// ForCompositeResource derives the CustomResourceDefinition for a composite
// resource from the supplied CompositeResourceDefinition.
func ForCompositeResource(xrd *v1alpha1.CompositeResourceDefinition) (*extv1.CustomResourceDefinition, error) {
	crd := &extv1.CustomResourceDefinition{
		Spec: extv1.CustomResourceDefinitionSpec{
			Scope:    extv1.ClusterScoped,
			Group:    xrd.Spec.Group,
			Names:    xrd.Spec.Names,
			Versions: make([]extv1.CustomResourceDefinitionVersion, len(xrd.Spec.Versions)),
		},
	}

	crd.SetName(xrd.GetName())
	crd.SetLabels(xrd.GetLabels())
	crd.SetAnnotations(xrd.GetAnnotations())
	crd.SetOwnerReferences([]metav1.OwnerReference{meta.AsController(
		meta.TypedReferenceTo(xrd, v1alpha1.CompositeResourceDefinitionGroupVersionKind),
	)})

	crd.Spec.Names.Categories = append(crd.Spec.Names.Categories, CategoryComposite)

	for i, vr := range xrd.Spec.Versions {
		crd.Spec.Versions[i] = extv1.CustomResourceDefinitionVersion{
			Name:                     vr.Name,
			Served:                   vr.Served,
			Storage:                  vr.Referenceable,
			AdditionalPrinterColumns: append(vr.AdditionalPrinterColumns, CompositeResourcePrinterColumns()...),
			Schema: &extv1.CustomResourceValidation{
				OpenAPIV3Schema: &extv1.JSONSchemaProps{
					Type:       "object",
					Properties: BaseProps(),
				},
			},
			Subresources: &extv1.CustomResourceSubresources{
				Status: &extv1.CustomResourceSubresourceStatus{},
			},
		}

		p, err := getSpecProps(vr.Schema)
		if err != nil {
			return nil, errors.Wrap(err, errGetSpecProps)
		}
		for k, v := range p {
			crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"].Properties[k] = v
		}
		for k, v := range CompositeResourceSpecProps() {
			crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"].Properties[k] = v
		}
		for k, v := range CompositeResourceStatusProps() {
			crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["status"].Properties[k] = v
		}
	}

	return crd, nil
}

// ForCompositeResourceClaim derives the CustomResourceDefinition for a
// composite resource claim from the supplied CompositeResourceDefinition.
func ForCompositeResourceClaim(xrd *v1alpha1.CompositeResourceDefinition) (*extv1.CustomResourceDefinition, error) {
	if err := validateClaimNames(xrd); err != nil {
		return nil, errors.Wrap(err, errInvalidClaimNames)
	}

	crd := &extv1.CustomResourceDefinition{
		Spec: extv1.CustomResourceDefinitionSpec{
			Scope:    extv1.NamespaceScoped,
			Group:    xrd.Spec.Group,
			Names:    *xrd.Spec.ClaimNames,
			Versions: make([]extv1.CustomResourceDefinitionVersion, len(xrd.Spec.Versions)),
		},
	}

	crd.SetName(xrd.Spec.ClaimNames.Plural + "." + xrd.Spec.Group)
	crd.SetLabels(xrd.GetLabels())
	crd.SetAnnotations(xrd.GetAnnotations())
	crd.SetOwnerReferences([]metav1.OwnerReference{meta.AsController(
		meta.TypedReferenceTo(xrd, v1alpha1.CompositeResourceDefinitionGroupVersionKind),
	)})

	crd.Spec.Names.Categories = append(crd.Spec.Names.Categories, CategoryClaim)

	for i, vr := range xrd.Spec.Versions {
		crd.Spec.Versions[i] = extv1.CustomResourceDefinitionVersion{
			Name:                     vr.Name,
			Served:                   vr.Served,
			Storage:                  vr.Referenceable,
			AdditionalPrinterColumns: append(vr.AdditionalPrinterColumns, CompositeResourceClaimPrinterColumns()...),
			Schema: &extv1.CustomResourceValidation{
				OpenAPIV3Schema: &extv1.JSONSchemaProps{
					Type:       "object",
					Properties: BaseProps(),
				},
			},
			Subresources: &extv1.CustomResourceSubresources{
				Status: &extv1.CustomResourceSubresourceStatus{},
			},
		}

		p, err := getSpecProps(vr.Schema)
		if err != nil {
			return nil, errors.Wrap(err, errGetSpecProps)
		}
		for k, v := range p {
			crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"].Properties[k] = v
		}
		for k, v := range CompositeResourceClaimSpecProps() {
			crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["spec"].Properties[k] = v
		}
		for k, v := range CompositeResourceStatusProps() {
			crd.Spec.Versions[i].Schema.OpenAPIV3Schema.Properties["status"].Properties[k] = v
		}
	}

	return crd, nil
}

func validateClaimNames(d *v1alpha1.CompositeResourceDefinition) error {
	if d.Spec.ClaimNames == nil {
		return errors.New(errMissingClaimNames)
	}

	if n := d.Spec.ClaimNames.Kind; n == d.Spec.Names.Kind {
		return errors.Errorf(errFmtConflictingClaimName, n)
	}

	if n := d.Spec.ClaimNames.Plural; n == d.Spec.Names.Plural {
		return errors.Errorf(errFmtConflictingClaimName, n)
	}

	if n := d.Spec.ClaimNames.Singular; n != "" && n == d.Spec.Names.Singular {
		return errors.Errorf(errFmtConflictingClaimName, n)
	}

	if n := d.Spec.ClaimNames.ListKind; n != "" && n == d.Spec.Names.ListKind {
		return errors.Errorf(errFmtConflictingClaimName, n)
	}

	return nil
}

func getSpecProps(v *v1alpha1.CompositeResourceValidation) (map[string]extv1.JSONSchemaProps, error) {
	if v == nil {
		return nil, nil
	}

	s := &extv1.JSONSchemaProps{}
	if err := json.Unmarshal(v.OpenAPIV3Schema.Raw, s); err != nil {
		return nil, errors.Wrap(err, errParseValidation)
	}

	spec, ok := s.Properties["spec"]
	if !ok {
		return nil, nil
	}

	return spec.Properties, nil
}

// IsEstablished is a helper function to check whether api-server is ready
// to accept the instances of registered CRD.
func IsEstablished(s extv1.CustomResourceDefinitionStatus) bool {
	for _, c := range s.Conditions {
		if c.Type == extv1.Established {
			return c.Status == extv1.ConditionTrue
		}
	}
	return false
}
