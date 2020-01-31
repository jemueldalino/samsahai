/*
Copyright 2019 Agoda DevOps Container.

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
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TeamSpec defines the desired state of Team
type TeamSpec struct {
	// Description represents description for this team
	// +optional
	Description string `json:"desc,omitempty"`

	// Owners represents contact point of this team
	// +optional
	Owners []string `json:"owners,omitempty"`

	// Resources represents how many resources per namespace for the team
	// +optional
	Resources corev1.ResourceList `json:"resources,omitempty"`

	// GitStorage represents git as storage for retrieving configuration, etc.
	GitStorage GitStorage `json:"gitStorage"`

	// StagingCtrl represents configuration about the staging controller.
	// For easier for developing, debugging and testing purposes
	// +optional
	StagingCtrl *StagingCtrl `json:"stagingCtrl,omitempty"`

	// Credential
	// +optional
	Credential Credential `json:"credential,omitempty"`
}

type StagingCtrl struct {
	// Image represents image for run staging controller.
	Image string `json:"image,omitempty"`

	// Endpoint represents the staging endpoint endpoint.
	Endpoint string `json:"endpoint,omitempty"`

	// IsDeploy represents flag to deploy staging controller or not.
	IsDeploy bool `json:"isDeploy"`

	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type GitStorage struct {
	// URL represents git url
	URL string `json:"url"`

	// Ref represents git ref
	// +optional
	Ref string `json:"ref,omitempty"`

	// Path represents a directory path in git repo
	// +optional
	Path string `json:"path,omitempty"`

	// CloneDepth limits fetching to the specified number of commits
	// +optional
	CloneDepth int `json:"cloneDepth,omitempty"`

	// CloneTimeout represents timeout duration of git clone
	// +optional
	CloneTimeout *metav1.Duration `json:"cloneTimeout,omitempty"`

	// PullTimeout represents timeout duration of git pull
	// +optional
	PullTimeout *metav1.Duration `json:"pullTimeout,omitempty"`

	// PushTimeout represents timeout duration of git push
	// +optional
	PushTimeout *metav1.Duration `json:"pushTimeout,omitempty"`
}

type Credential struct {
	// SecretName
	SecretName string `json:"secretName,omitempty"`

	// Git
	Git *UsernamePasswordCredential `json:"git,omitempty"`

	// Teamcity
	// +optional
	Teamcity *UsernamePasswordCredential `json:"teamcity,omitempty"`
}

type UsernamePasswordCredential struct {
	UsernameRef *corev1.SecretKeySelector `json:"username"`
	PasswordRef *corev1.SecretKeySelector `json:"password"`
	Username    string                    `json:"-"`
	Password    string                    `json:"-"`
}

type TokenCredential struct {
	TokenRef *corev1.SecretKeySelector `json:"token"`
	Token    string                    `json:"-"`
}

// TeamStatus defines the observed state of Team
type TeamStatus struct {
	// +optional
	Namespace TeamNamespace `json:"namespace,omitempty"`

	// StableComponentList represents a list of stable components
	// +optional
	StableComponents []StableComponent `json:"stableComponents,omitempty"`

	// CurrentActiveComponents represents a list of stable components in current active namespace
	// +optional
	CurrentActiveComponents []StableComponent `json:"currentActiveComponents,omitempty"`

	// Conditions contains observations of the resource's state e.g.,
	// Team namespace is created, destroyed
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []TeamCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// DesiredComponentImageCreatedTime represents mapping of desired component image and created time
	// map[componentName][repository:tag] = image and createdTime
	// +optional
	DesiredComponentImageCreatedTime map[string]map[string]DesiredImageTime `json:"desiredComponentImageCreatedTime,omitempty"`
}

// SetStableComponents sets stable components
func (ts *TeamStatus) SetStableComponents(stableComps []StableComponent) {
	ts.StableComponents = make([]StableComponent, 0)
	for _, stableComp := range stableComps {
		ts.StableComponents = append(ts.StableComponents, StableComponent{
			Spec:   stableComp.Spec,
			Status: stableComp.Status,
		})
	}
}

// SetCurrentActiveComponents sets current active components
func (ts *TeamStatus) SetCurrentActiveComponents(currentComps []StableComponent) {
	ts.CurrentActiveComponents = make([]StableComponent, 0)
	for _, currentComp := range currentComps {
		ts.CurrentActiveComponents = append(ts.CurrentActiveComponents, StableComponent{
			Spec:   currentComp.Spec,
			Status: currentComp.Status,
		})
	}
}

// UpdateDesiredComponentImageCreatedTime updates desired component version and created time mapping
func (ts *TeamStatus) UpdateDesiredComponentImageCreatedTime(compName, image string, desiredImageTime DesiredImageTime) {
	if ts.DesiredComponentImageCreatedTime == nil {
		ts.DesiredComponentImageCreatedTime = make(map[string]map[string]DesiredImageTime)
	}

	if _, ok := ts.DesiredComponentImageCreatedTime[compName]; !ok {
		ts.DesiredComponentImageCreatedTime[compName] = map[string]DesiredImageTime{
			image: desiredImageTime,
		}
		return
	}

	descCreatedTime := SortByCreatedTimeDESC(ts.DesiredComponentImageCreatedTime[compName])
	if strings.EqualFold(descCreatedTime[0].Image, image) {
		return
	}

	ts.DesiredComponentImageCreatedTime[compName][image] = desiredImageTime
}

type DesiredImageTime struct {
	*Image      `json:"image"`
	CreatedTime metav1.Time `json:"createdTime"`
}

type TeamNamespace struct {
	// +optional
	Staging string `json:"staging,omitempty"`

	// +optional
	PreviousActive string `json:"previousActive,omitempty"`

	// +optional
	PreActive string `json:"preActive,omitempty"`

	// +optional
	Active string `json:"active,omitempty"`
}

type TeamCondition struct {
	Type   TeamConditionType      `json:"type"`
	Status corev1.ConditionStatus `json:"status"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

type TeamConditionType string

const (
	TeamNamespaceStagingCreated        TeamConditionType = "TeamNamespaceStagingCreated"
	TeamNamespacePreActiveCreated      TeamConditionType = "TeamNamespacePreActiveCreated"
	TeamNamespacePreviousActiveCreated TeamConditionType = "TeamNamespacePreviousActiveCreated"
	TeamNamespaceActiveCreated         TeamConditionType = "TeamNamespaceActiveCreated"
	TeamGitCheckoutUpToDate            TeamConditionType = "TeamGitCheckoutUpToDate"
)

func (ts *TeamStatus) IsConditionTrue(cond TeamConditionType) bool {
	for i, c := range ts.Conditions {
		if c.Type == cond {
			return ts.Conditions[i].Status == corev1.ConditionTrue
		}
	}

	return false
}

func (ts *TeamStatus) SetCondition(cond TeamConditionType, status corev1.ConditionStatus, message string) {
	for i, c := range ts.Conditions {
		if c.Type == cond {
			ts.Conditions[i].Status = status
			ts.Conditions[i].LastTransitionTime = metav1.Now()
			ts.Conditions[i].Message = message
			return
		}
	}

	ts.Conditions = append(ts.Conditions, TeamCondition{
		Type:               cond,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Message:            message,
	})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// Team is the Schema for the teams API
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec,omitempty"`
	Status TeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

func SortByCreatedTimeDESC(desiredCreatedTime map[string]DesiredImageTime) TeamDesiredImageTimeList {
	var d TeamDesiredImageTimeList
	for k, v := range desiredCreatedTime {
		d = append(d, TeamDesiredImageTime{k, v})
	}

	sort.Sort(sort.Reverse(d))
	return d
}

type TeamDesiredImageTime struct {
	Image     string
	ImageTime DesiredImageTime
}

type TeamDesiredImageTimeList []TeamDesiredImageTime

func (p TeamDesiredImageTimeList) Len() int {
	return len(p)
}

func (p TeamDesiredImageTimeList) Less(i, j int) bool {
	return p[i].ImageTime.CreatedTime.Time.Before(p[j].ImageTime.CreatedTime.Time)
}

func (p TeamDesiredImageTimeList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}