/*


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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	helmerv1beta1 "github.com/openshift-psap/special-resource-operator/pkg/helmer/api/v1beta1"
)

// SpecialResourceImages
type SpecialResourceImages struct {
	// Name of the DriverContainer image containing artifacts
	Name       string                 `json:"name"`
	// Kind of hte DriverContainer image containing artifacts (e.g. ImageStreamTag)
	Kind       string                 `json:"kind"`
	// Namespace of the DriverContainer image containing artifacts
	Namespace  string                 `json:"namespace"`
	// Pullsecret required to retrieve the DriverContainer image containing artifacts
	PullSecret string                 `json:"pullsecret,omitempty"`
	Paths      []SpecialResourcePaths `json:"path"`
}

// SpecialResourceClaims
type SpecialResourceClaims struct {
	// Name is the PVC name to mount
	Name      string `json:"name"`
	// MountPath is the location to mount the PV on
	MountPath string `json:"mountPath"`
}

// SpecialResourcePaths
type SpecialResourcePaths struct {
	// SourcePath gives the path to copy build artifacts from
	SourcePath     string `json:"sourcePath"`
	// DestinationDir gives the path to copy build artifacts to
	DestinationDir string `json:"destinationDir"`
}

// SpecialResourceArtifacts
type SpecialResourceArtifacts struct {
	// HostPath containing required build artifacts
	// +kubebuilder:validation:Optional
	HostPaths []SpecialResourcePaths `json:"hostPaths,omitempty"`
	//Images describes a pre-built DriverContainer image contianing the needed artifacts
	// +kubebuilder:validation:Optional
	Images []SpecialResourceImages `json:"images,omitempty"`
	// Claims describes a PVC containing required build artifacts
	// +kubebuilder:validation:Optional
	Claims []SpecialResourceClaims `json:"claims,omitempty"`
}

// SpecialResourceBuildArgs is not used.
type SpecialResourceBuildArgs struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SpecialResourceConfiguration is not used.
type SpecialResourceConfiguration struct {
	Name  string   `json:"name"`
	Value []string `json:"value"`
}

// SpecialResourceGit is the git repo to pull source from
type SpecialResourceGit struct {
	// Ref describes the refspec to pull (e.g. main)
	Ref string `json:"ref"`
	// Uri describes the uri of the git repo to pull from (e.g. http://github.com/my/repo.git)
	Uri string `json:"uri"`
}

// SpecialResourceSource is the source to pull sourcecode from
type SpecialResourceSource struct {
	// Git describes the git repo to build the drivercontainer from
	Git SpecialResourceGit `json:"git,omitempty"`
}

// SpecialResourceDriverContainer
type SpecialResourceDriverContainer struct {
	// Source describes the source to build the drivercontainer from
	// +kubebuilder:validation:Optional
	Source SpecialResourceSource `json:"source,omitempty"`

	// Artifacts describes the location of any build artifacts required
	// +kubebuilder:validation:Optional
	Artifacts SpecialResourceArtifacts `json:"artifacts,omitempty"`
}

// SpecialResourceSpec describes the desired state of the resource, such as the chart to be used and a selector
// on which nodes it should be installed.
// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
// +kubebuilder:validation:Required
type SpecialResourceSpec struct {
	// Chart describes the Helm chart that needs to be installed.
	// +kubebuilder:validation:Required
	Chart helmerv1beta1.HelmChart `json:"chart"`

	// Namespace describes in which namespace the chart will be installed.
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// ForceUpgrade is not used.
	// +kubebuilder:validation:Optional
	ForceUpgrade bool `json:"forceUpgrade"`

	// Debug enables additional logging.
	// +kubebuilder:validation:Optional
	Debug bool `json:"debug"`

	// Set is a user-defined hierarchical value tree from where the chart takes its parameters.
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Set unstructured.Unstructured `json:"set,omitempty"`

	// DriverContainer describes how to build the driver-container
	// +kubebuilder:validation:Optional
	DriverContainer SpecialResourceDriverContainer `json:"driverContainer,omitempty"`

	// NodeSelector is used to determine on which nodes the software stack should be installed.
	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Dependencies is a list of dependencies required by this SpecialReosurce.
	// +kubebuilder:validation:Optional
	Dependencies []SpecialResourceDependency `json:"dependencies,omitempty"`
}

// SpecialResourceDependency is a Helm chart the SpecialResource depends on.
type SpecialResourceDependency struct {
	helmerv1beta1.HelmChart `json:"chart,omitempty"`

	// Set are Helm hierarchical values for this chart installation.
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:EmbeddedResource
	Set unstructured.Unstructured `json:"set,omitempty"`
}

// SpecialResourceStatus is the most recently observed status of the SpecialResource.
// It is populated by the system and is read-only.
// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
type SpecialResourceStatus struct {
	// State describes at which step the chart installation is.
	State string `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SpecialResource describes a software stack for hardware accelerators on an existing Kubernetes cluster.
// +kubebuilder:resource:path=specialresources,scope=Cluster
// +kubebuilder:resource:path=specialresources,scope=Cluster,shortName=sr
type SpecialResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:Required

	Spec   SpecialResourceSpec   `json:"spec,omitempty"`
	Status SpecialResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpecialResourceList is a list of SpecialResource objects.
type SpecialResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of SpecialResources. More info:
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md
	Items []SpecialResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpecialResource{}, &SpecialResourceList{})
}
