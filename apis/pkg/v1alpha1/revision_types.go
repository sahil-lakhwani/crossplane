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

package v1alpha1

import (
	rbac "k8s.io/api/rbac/v1"

	runtimev1alpha1 "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
)

// PackageRevisionDesiredState is the desired state of the package revision.
type PackageRevisionDesiredState string

const (
	// PackageRevisionActive is an active package revision.
	PackageRevisionActive PackageRevisionDesiredState = "Active"

	// PackageRevisionInactive is an inactive package revision.
	PackageRevisionInactive PackageRevisionDesiredState = "Inactive"
)

// PackageRevisionSpec specifies the desired state of a PackageRevision.
type PackageRevisionSpec struct {
	// Reference to install Pod. PackageRevision reads logs of this Pod to
	// create resources owned by the PackageRevision.
	InstallPodRef runtimev1alpha1.Reference `json:"installPodRef"`

	// DesiredState of the PackageRevision. Can be either Active or Inactive.
	DesiredState PackageRevisionDesiredState `json:"desiredState"`

	// Image used for install Pod to extract package contents.
	Image string `json:"image"`

	// Revision number. Indicates when the revision will be garbage collected
	// based on the parent's RevisionHistoryLimit.
	Revision int64 `json:"revision"`
}

// Dependency specifies the dependency of a package.
type Dependency struct {
	// Package is the name of the depended upon package image .
	Package string `json:"package"`

	// Version is the semantic version range for the dependency.
	Version string `json:"version"`
}

// PackageRevisionStatus represents the observed state of a PackageRevision.
type PackageRevisionStatus struct {
	runtimev1alpha1.ConditionedStatus `json:",inline"`
	ControllerRef                     runtimev1alpha1.Reference `json:"controllerRef,omitempty"`

	// Crossplane is a semantic version for supported Crossplane version for the
	// package.
	Crossplane string `json:"crossplane,omitempty"`

	// DependsOn is the list of packages and CRDs that this package depends on.
	DependsOn []Dependency `json:"dependsOn,omitempty"`

	// References to objects owned by PackageRevision.
	ObjectRefs []runtimev1alpha1.TypedReference `json:"objectRefs,omitempty"`

	// PermissionRequests are additional permissions that should be added to a
	// packaged controller's service account.
	PermissionRequests []rbac.PolicyRule `json:"permissionRequests,omitempty"`
}
