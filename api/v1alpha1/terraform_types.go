/*
Copyright 2021.

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
	"fmt"
	"strings"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/fluxcd/pkg/apis/meta"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CACertSecretName = "tf-controller.tls"
	// RunnerTLSSecretName is the name of the secret containing a TLS cert that will be written to
	// the namespace in which a terraform runner is created
	RunnerTLSSecretName   = "terraform-runner.tls"
	RunnerLabel           = "infra.contrib.fluxcd.io/terraform"
	GitRepositoryIndexKey = ".metadata.gitRepository"
	BucketIndexKey        = ".metadata.bucket"
	OCIRepositoryIndexKey = ".metadata.ociRepository"
)

type ReadInputsFromSecretSpec struct {
	// +required
	Name string `json:"name"`

	// +required
	As string `json:"as"`
}

// WriteOutputsToSecretSpec defines where to store outputs, and which outputs to be stored.
type WriteOutputsToSecretSpec struct {
	// Name is the name of the Secret to be written
	// +required
	Name string `json:"name"`

	// Outputs contain the selected names of outputs to be written
	// to the secret. Empty array means writing all outputs, which is default.
	// +optional
	Outputs []string `json:"outputs,omitempty"`
}

type Variable struct {
	// Name is the name of the variable
	// +required
	Name string `json:"name"`

	// +optional
	Value *apiextensionsv1.JSON `json:"value,omitempty"`

	// +optional
	ValueFrom *corev1.EnvVarSource `json:"valueFrom,omitempty"`
}

// TerraformSpec defines the desired state of Terraform
type TerraformSpec struct {

	// ApprovePlan specifies name of a plan wanted to approve.
	// If its value is "auto", the controller will automatically approve every plan.
	// +optional
	ApprovePlan string `json:"approvePlan,omitempty"`

	// Destroy produces a destroy plan. Applying the plan will destroy all resources.
	// +optional
	Destroy bool `json:"destroy,omitempty"`

	// +optional
	BackendConfig *BackendConfigSpec `json:"backendConfig,omitempty"`

	// +optional
	BackendConfigsFrom []BackendConfigsReference `json:"backendConfigsFrom,omitempty"`

	// +optional
	// +kubebuilder:default:=default
	Workspace string `json:"workspace,omitempty"`

	// List of input variables to set for the Terraform program.
	// +optional
	Vars []Variable `json:"vars,omitempty"`

	// List of references to a Secret or a ConfigMap to generate variables for
	// Terraform resources based on its data, selectively by varsKey. Values of the later
	// Secret / ConfigMap with the same keys will override those of the former.
	// +optional
	VarsFrom []VarsReference `json:"varsFrom,omitempty"`

	// Values map to the Terraform variable "values", which is an object of arbitrary values.
	// It is a convenient way to pass values to Terraform resources without having to define
	// a variable for each value. To use this feature, your Terraform file must define the variable "values".
	// +optional
	Values *apiextensionsv1.JSON `json:"values,omitempty"`

	// List of all configuration files to be created in initialization.
	// +optional
	FileMappings []FileMapping `json:"fileMappings,omitempty"`

	// The interval at which to reconcile the Terraform.
	// +required
	Interval metav1.Duration `json:"interval"`

	// The interval at which to retry a previously failed reconciliation.
	// When not specified, the controller uses the TerraformSpec.Interval
	// value to retry failures.
	// +optional
	RetryInterval *metav1.Duration `json:"retryInterval,omitempty"`

	// Path to the directory containing Terraform (.tf) files.
	// Defaults to 'None', which translates to the root path of the SourceRef.
	// +optional
	Path string `json:"path,omitempty"`

	// SourceRef is the reference of the source where the Terraform files are stored.
	// +required
	SourceRef CrossNamespaceSourceReference `json:"sourceRef"`

	// Suspend is to tell the controller to suspend subsequent TF executions,
	// it does not apply to already started executions. Defaults to false.
	// +optional
	Suspend bool `json:"suspend,omitempty"`

	// Force instructs the controller to unconditionally
	// re-plan and re-apply TF resources. Defaults to false.
	// +kubebuilder:default:=false
	// +optional
	Force bool `json:"force,omitempty"`

	// +optional
	ReadInputsFromSecrets []ReadInputsFromSecretSpec `json:"readInputsFromSecrets,omitempty"`

	// A list of target secrets for the outputs to be written as.
	// +optional
	WriteOutputsToSecret *WriteOutputsToSecretSpec `json:"writeOutputsToSecret,omitempty"`

	// Disable automatic drift detection. Drift detection may be resource intensive in
	// the context of a large cluster or complex Terraform statefile. Defaults to false.
	// +kubebuilder:default:=false
	// +optional
	DisableDriftDetection bool `json:"disableDriftDetection,omitempty"`

	// +optional
	// PushSpec *PushSpec `json:"pushSpec,omitempty"`

	// +optional
	CliConfigSecretRef *corev1.SecretReference `json:"cliConfigSecretRef,omitempty"`

	// List of health checks to be performed.
	// +optional
	HealthChecks []HealthCheck `json:"healthChecks,omitempty"`

	// Create destroy plan and apply it to destroy terraform resources
	// upon deletion of this object. Defaults to false.
	// +kubebuilder:default:=false
	// +optional
	DestroyResourcesOnDeletion bool `json:"destroyResourcesOnDeletion,omitempty"`

	// Name of a ServiceAccount for the runner Pod to provision Terraform resources.
	// Default to tf-runner.
	// +kubebuilder:default:=tf-runner
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Clean the runner pod up after each reconciliation cycle
	// +kubebuilder:default:=true
	// +optional
	AlwaysCleanupRunnerPod *bool `json:"alwaysCleanupRunnerPod,omitempty"`

	// Configure the termination grace period for the runner pod. Use this parameter
	// to allow the Terraform process to gracefully shutdown. Consider increasing for
	// large, complex or slow-moving Terraform managed resources.
	// +kubebuilder:default:=30
	// +optional
	RunnerTerminationGracePeriodSeconds *int64 `json:"runnerTerminationGracePeriodSeconds,omitempty"`

	// RefreshBeforeApply forces refreshing of the state before the apply step.
	// +kubebuilder:default:=false
	// +optional
	RefreshBeforeApply bool `json:"refreshBeforeApply,omitempty"`

	// +optional
	RunnerPodTemplate RunnerPodTemplate `json:"runnerPodTemplate,omitempty"`

	// EnableInventory enables the object to store resource entries as the inventory for external use.
	// +optional
	EnableInventory bool `json:"enableInventory,omitempty"`

	// +optional
	TFState *TFStateSpec `json:"tfstate,omitempty"`

	// Targets specify the resource, module or collection of resources to target.
	// +optional
	Targets []string `json:"targets,omitempty"`

	// StoreReadablePlan enables storing the plan in a readable format.
	// +kubebuilder:validation:Enum=none;json;human
	// +kubebuilder:default:=none
	// +optional
	StoreReadablePlan string `json:"storeReadablePlan,omitempty"`

	// +optional
	Webhooks []Webhook `json:"webhooks,omitempty"`

	// +optional
	DependsOn []meta.NamespacedObjectReference `json:"dependsOn,omitempty"`
}

type Webhook struct {
	// +kubebuilder:validation:Enum=post-planning
	// +kubebuilder:default:=post-planning
	// +required
	Stage string `json:"stage"`

	// +kubebuilder:default:=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +required
	URL string `json:"url"`

	// +kubebuilder:value:Enum=SpecAndPlan,SpecOnly,PlanOnly
	// +kubebuilder:default:=SpecAndPlan
	// +optional
	PayloadType string `json:"payloadType,omitempty"`

	// +optional
	ErrorMessageTemplate string `json:"errorMessageTemplate,omitempty"`

	// +required
	TestExpression string `json:"testExpression,omitempty"`
}

func (w Webhook) IsEnabled() bool {
	return w.Enabled == nil || *w.Enabled
}

type PlanStatus struct {
	// +optional
	LastApplied string `json:"lastApplied,omitempty"`

	// +optional
	Pending string `json:"pending,omitempty"`

	// +optional
	IsDestroyPlan bool `json:"isDestroyPlan,omitempty"`

	// +optional
	IsDriftDetectionPlan bool `json:"isDriftDetectionPlan,omitempty"`
}

// TerraformStatus defines the observed state of Terraform
type TerraformStatus struct {
	meta.ReconcileRequestStatus `json:",inline"`

	// ObservedGeneration is the last reconciled generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// The last successfully applied revision.
	// The revision format for Git sources is <branch|tag>/<commit-sha>.
	// +optional
	LastAppliedRevision string `json:"lastAppliedRevision,omitempty"`

	// LastAttemptedRevision is the revision of the last reconciliation attempt.
	// +optional
	LastAttemptedRevision string `json:"lastAttemptedRevision,omitempty"`

	// LastPlannedRevision is the revision used by the last planning process.
	// The result could be either no plan change or a new plan generated.
	// +optional
	LastPlannedRevision string `json:"lastPlannedRevision,omitempty"`

	// LastDriftDetectedAt is the time when the last drift was detected
	// +optional
	LastDriftDetectedAt *metav1.Time `json:"lastDriftDetectedAt,omitempty"`

	// LastAppliedByDriftDetectionAt is the time when the last drift was detected and
	// terraform apply was performed as a result
	// +optional
	LastAppliedByDriftDetectionAt *metav1.Time `json:"lastAppliedByDriftDetectionAt,omitempty"`

	// +optional
	AvailableOutputs []string `json:"availableOutputs,omitempty"`

	// +optional
	Plan PlanStatus `json:"plan,omitempty"`

	// Inventory contains the list of Terraform resource object references that have been successfully applied.
	// +optional
	Inventory *ResourceInventory `json:"inventory,omitempty"`

	// +optional
	Lock LockStatus `json:"lock,omitempty"`
}

// LockStatus defines the observed state of a Terraform State Lock
type LockStatus struct {
	// +optional
	LastApplied string `json:"lastApplied,omitempty"`

	// Pending holds the identifier of the Lock Holder to be used with Force Unlock
	// +optional
	Pending string `json:"pending,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=tf
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// Terraform is the Schema for the terraforms API
type Terraform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TerraformSpec   `json:"spec,omitempty"`
	Status TerraformStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TerraformList contains a list of Terraform
type TerraformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Terraform `json:"items"`
}

// BackendConfigSpec is for specifying configuration for Terraform's Kubernetes backend
type BackendConfigSpec struct {

	// Disable is to completely disable the backend configuration.
	// +optional
	Disable bool `json:"disable"`

	// +optional
	SecretSuffix string `json:"secretSuffix,omitempty"`

	// +optional
	InClusterConfig bool `json:"inClusterConfig,omitempty"`

	// +optional
	CustomConfiguration string `json:"customConfiguration,omitempty"`

	// +optional
	ConfigPath string `json:"configPath,omitempty"`

	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// TFStateSpec allows the user to set ForceUnlock
type TFStateSpec struct {
	// ForceUnlock a Terraform state if it has become locked for any reason. Defaults to `no`.
	//
	// This is an Enum and has the expected values of:
	//
	// - auto
	// - yes
	// - no
	//
	// WARNING: Only use `auto` in the cases where you are absolutely certain that
	// no other system is using this state, you could otherwise end up in a bad place
	// See https://www.terraform.io/language/state/locking#force-unlock for more
	// information on the terraform state lock and force unlock.
	//
	// +optional
	// +kubebuilder:validation:Enum:=yes;no;auto
	// +kubebuilder:default:string=no
	ForceUnlock ForceUnlockEnum `json:"forceUnlock,omitempty"`

	// LockIdentifier holds the Identifier required by Terraform to unlock the state
	// if it ever gets into a locked state.
	//
	// You'll need to put the Lock Identifier in here while setting ForceUnlock to
	// either `yes` or `auto`.
	//
	// Leave this empty to do nothing, set this to the value of the `Lock Info: ID: [value]`,
	// e.g. `f2ab685b-f84d-ac0b-a125-378a22877e8d`, to force unlock the state.
	//
	// +optional
	LockIdentifier string `json:"lockIdentifier,omitempty"`
}

type ForceUnlockEnum string

const (
	ForceUnlockEnumAuto ForceUnlockEnum = "auto"
	ForceUnlockEnumYes  ForceUnlockEnum = "yes"
	ForceUnlockEnumNo   ForceUnlockEnum = "no"
)

const (
	TerraformKind             = "Terraform"
	TerraformFinalizer        = "finalizers.tf.contrib.fluxcd.io"
	MaxConditionMessageLength = 20000
	DisabledValue             = "disabled"
	ApprovePlanAutoValue      = "auto"
	ApprovePlanDisableValue   = "disable"
	DefaultWorkspaceName      = "default"
)

// The potential reasons that are associated with condition types
const (
	ArtifactFailedReason            = "ArtifactFailed"
	DeletionBlockedByDependants     = "DeletionBlockedByDependantsReason"
	DependencyNotReadyReason        = "DependencyNotReady"
	TFExecNewFailedReason           = "TFExecNewFailed"
	TFExecInitFailedReason          = "TFExecInitFailed"
	VarsGenerationFailedReason      = "VarsGenerationFailed"
	TemplateGenerationFailedReason  = "TemplateGenerationFailed"
	WorkspaceSelectFailedReason     = "SelectWorkspaceFailed"
	DriftDetectionFailedReason      = "DriftDetectionFailed"
	DriftDetectedReason             = "DriftDetected"
	NoDriftReason                   = "NoDrift"
	TFExecPlanFailedReason          = "TFExecPlanFailed"
	PostPlanningWebhookFailedReason = "PostPlanningWebhookFailed"
	TFExecApplyFailedReason         = "TFExecApplyFailed"
	TFExecOutputFailedReason        = "TFExecOutputFailed"
	OutputsWritingFailedReason      = "OutputsWritingFailed"
	HealthChecksFailedReason        = "HealthChecksFailed"
	TFExecApplySucceedReason        = "TerraformAppliedSucceed"
	TFExecLockHeldReason            = "LockHeld"
	TFExecForceUnlockReason         = "ForceUnlock"
)

// These constants are the Condition Types that the Terraform Resource works with
const (
	ConditionTypeApply       = "Apply"
	ConditionTypeHealthCheck = "HealthCheck"
	ConditionTypeOutput      = "Output"
	ConditionTypePlan        = "Plan"
	ConditionTypeStateLocked = "StateLocked"
)

// Webhook stages
const (
	PostPlanningWebhook = "post-planning"
)

const (
	TFDependencyOfPrefix = "tf.dependency.of."
)

// SetTerraformReadiness sets the ReadyCondition, ObservedGeneration, and LastAttemptedRevision, on the Terraform.
func SetTerraformReadiness(terraform *Terraform, status metav1.ConditionStatus, reason, message string, revision string) {
	newCondition := metav1.Condition{
		Type:    meta.ReadyCondition,
		Status:  status,
		Reason:  reason,
		Message: trimString(message, MaxConditionMessageLength),
	}

	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	terraform.Status.ObservedGeneration = terraform.Generation
	terraform.Status.LastAttemptedRevision = revision
}

func TerraformApplying(terraform Terraform, revision string, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypeApply,
		Status:  metav1.ConditionUnknown,
		Reason:  meta.ProgressingReason,
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	if revision != "" {
		(&terraform).Status.LastAttemptedRevision = revision
	}
	return terraform
}

func TerraformOutputsAvailable(terraform Terraform, availableOutputs []string, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypeOutput,
		Status:  metav1.ConditionTrue,
		Reason:  "TerraformOutputsAvailable",
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	(&terraform).Status.AvailableOutputs = availableOutputs
	return terraform
}

func TerraformOutputsWritten(terraform Terraform, revision string, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypeOutput,
		Status:  metav1.ConditionTrue,
		Reason:  "TerraformOutputsWritten",
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)

	SetTerraformReadiness(&terraform, metav1.ConditionTrue, "TerraformOutputsWritten", message+": "+revision, revision)
	return terraform
}

func TerraformApplied(terraform Terraform, revision string, message string, isDestroyApply bool, entries []ResourceRef) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypeApply,
		Status:  metav1.ConditionTrue,
		Reason:  TFExecApplySucceedReason,
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)

	if terraform.Status.Plan.IsDriftDetectionPlan {
		(&terraform).Status.LastAppliedByDriftDetectionAt = &metav1.Time{Time: time.Now()}
	}

	(&terraform).Status.Plan = PlanStatus{
		LastApplied:   terraform.Status.Plan.Pending,
		Pending:       "",
		IsDestroyPlan: isDestroyApply,
	}
	if revision != "" {
		(&terraform).Status.LastAppliedRevision = revision
	}

	if len(entries) > 0 {
		(&terraform).Status.Inventory = &ResourceInventory{Entries: entries}
	}

	SetTerraformReadiness(&terraform, metav1.ConditionUnknown, TFExecApplySucceedReason, message+": "+revision, revision)
	return terraform
}

func GetPlanIdAndApproveMessage(revision string, message string) (string, string) {
	planRev := strings.Replace(revision, "/", "-", 1)
	planId := fmt.Sprintf("plan-%s", planRev)
	shortPlanId := planId
	parts := strings.SplitN(revision, "/", 2)
	if len(parts) == 2 {
		if len(parts[1]) >= 10 {
			shortPlanId = fmt.Sprintf("plan-%s-%s", parts[0], parts[1][0:10])
		}
	}
	approveMessage := fmt.Sprintf("%s: set approvePlan: \"%s\" to approve this plan.", message, shortPlanId)
	return planId, approveMessage
}

func TerraformPostPlanningWebhookFailed(terraform Terraform, revision string, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypePlan,
		Status:  metav1.ConditionFalse,
		Reason:  PostPlanningWebhookFailedReason,
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	(&terraform).Status.Plan = PlanStatus{
		LastApplied:   terraform.Status.Plan.LastApplied,
		Pending:       "",
		IsDestroyPlan: terraform.Spec.Destroy,
	}
	if revision != "" {
		(&terraform).Status.LastAttemptedRevision = revision
		(&terraform).Status.LastPlannedRevision = revision
	}

	return terraform
}

func TerraformPlannedWithChanges(terraform Terraform, revision string, forceOrAutoApply bool, message string) Terraform {
	planId, approveMessage := GetPlanIdAndApproveMessage(revision, message)
	newCondition := metav1.Condition{
		Type:    ConditionTypePlan,
		Status:  metav1.ConditionTrue,
		Reason:  "TerraformPlannedWithChanges",
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	(&terraform).Status.Plan = PlanStatus{
		LastApplied:          terraform.Status.Plan.LastApplied,
		Pending:              planId,
		IsDestroyPlan:        terraform.Spec.Destroy,
		IsDriftDetectionPlan: terraform.HasDrift(),
	}
	if revision != "" {
		(&terraform).Status.LastAttemptedRevision = revision
		(&terraform).Status.LastPlannedRevision = revision
	}

	if forceOrAutoApply {
		SetTerraformReadiness(&terraform, metav1.ConditionUnknown, "TerraformPlannedWithChanges", message, revision)
	} else {
		// this is the manual mode, where we don't want to apply the plan
		SetTerraformReadiness(&terraform, metav1.ConditionUnknown, "TerraformPlannedWithChanges", approveMessage, revision)
	}
	return terraform
}

func TerraformPlannedNoChanges(terraform Terraform, revision string, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypePlan,
		Status:  metav1.ConditionFalse,
		Reason:  "TerraformPlannedNoChanges",
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	(&terraform).Status.Plan = PlanStatus{
		LastApplied:   terraform.Status.Plan.LastApplied,
		Pending:       "",
		IsDestroyPlan: terraform.Spec.Destroy,
	}
	if revision != "" {
		(&terraform).Status.LastAttemptedRevision = revision
		(&terraform).Status.LastPlannedRevision = revision
	}

	SetTerraformReadiness(&terraform, metav1.ConditionTrue, "TerraformPlannedNoChanges", message+": "+revision, revision)
	return terraform
}

// TerraformProgressing resets the conditions of the given Terraform to a single
// ReadyCondition with status ConditionUnknown.
func TerraformProgressing(terraform Terraform, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    meta.ReadyCondition,
		Status:  metav1.ConditionUnknown,
		Reason:  meta.ProgressingReason,
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	return terraform
}

// TerraformNotReady registers a failed apply attempt of the given Terraform.
func TerraformNotReady(terraform Terraform, revision, reason, message string) Terraform {
	SetTerraformReadiness(&terraform, metav1.ConditionFalse, reason, trimString(message, MaxConditionMessageLength), revision)
	if revision != "" {
		terraform.Status.LastAttemptedRevision = revision
	}
	return terraform
}

func TerraformAppliedFailResetPlanAndNotReady(terraform Terraform, revision, reason, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypeApply,
		Status:  metav1.ConditionFalse,
		Reason:  "TerraformAppliedFail",
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	terraform = TerraformNotReady(terraform, revision, reason, message)
	terraform.Status.Plan.Pending = ""
	return terraform
}

func TerraformDriftDetected(terraform Terraform, revision, reason, message string) Terraform {
	(&terraform).Status.LastDriftDetectedAt = &metav1.Time{Time: time.Now()}

	SetTerraformReadiness(&terraform, metav1.ConditionFalse, reason, trimString(message, MaxConditionMessageLength), revision)
	return terraform
}

func TerraformNoDrift(terraform Terraform, revision, reason, message string) Terraform {
	SetTerraformReadiness(&terraform, metav1.ConditionTrue, reason, message+": "+revision, revision)
	return terraform
}

func TerraformHealthCheckFailed(terraform Terraform, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypeHealthCheck,
		Status:  metav1.ConditionFalse,
		Reason:  HealthChecksFailedReason,
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	return terraform
}

func TerraformHealthCheckSucceeded(terraform Terraform, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypeHealthCheck,
		Status:  metav1.ConditionTrue,
		Reason:  "HealthChecksSucceed",
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	return terraform
}

// TerraformForceUnlock will set a new condition on the Terraform resource indicating
// that we are attempting to force unlock it.
func TerraformForceUnlock(terraform Terraform, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypeStateLocked,
		Status:  metav1.ConditionFalse,
		Reason:  TFExecForceUnlockReason,
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)

	if terraform.Status.Lock.Pending != "" && terraform.Status.Lock.LastApplied != terraform.Status.Lock.Pending {
		terraform.Status.Lock.LastApplied = terraform.Status.Lock.Pending
	}

	terraform.Status.Lock.Pending = ""
	return terraform
}

// TerraformStateLocked will set a new condition on the Terraform resource indicating
// that the resource has been locked.
func TerraformStateLocked(terraform Terraform, lockID, message string) Terraform {
	newCondition := metav1.Condition{
		Type:    ConditionTypeStateLocked,
		Status:  metav1.ConditionTrue,
		Reason:  TFExecLockHeldReason,
		Message: trimString(message, MaxConditionMessageLength),
	}
	apimeta.SetStatusCondition(terraform.GetStatusConditions(), newCondition)
	SetTerraformReadiness(&terraform, metav1.ConditionFalse, newCondition.Reason, newCondition.Message, "")

	if terraform.Status.Lock.Pending != "" && terraform.Status.Lock.LastApplied != terraform.Status.Lock.Pending {
		terraform.Status.Lock.LastApplied = terraform.Status.Lock.Pending
	}

	terraform.Status.Lock.Pending = lockID
	return terraform
}

// HasDrift returns true if drift has been detected since the last successful apply
func (in Terraform) HasDrift() bool {
	for _, condition := range in.Status.Conditions {
		if condition.Type == ConditionTypeApply &&
			condition.Status == metav1.ConditionTrue &&
			in.Status.LastDriftDetectedAt != nil &&
			(*in.Status.LastDriftDetectedAt).After(condition.LastTransitionTime.Time) {
			return true
		}
	}
	return false
}

// GetDependsOn returns the list of dependencies, namespace scoped.
func (in Terraform) GetDependsOn() []meta.NamespacedObjectReference {
	return in.Spec.DependsOn
}

// GetRetryInterval returns the retry interval
func (in Terraform) GetRetryInterval() time.Duration {
	if in.Spec.RetryInterval != nil {
		return in.Spec.RetryInterval.Duration
	}
	return in.Spec.Interval.Duration
}

// GetStatusConditions returns a pointer to the Status.Conditions slice.
func (in *Terraform) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *Terraform) WorkspaceName() string {
	if in.Spec.Workspace != "" {
		return in.Spec.Workspace
	}
	return DefaultWorkspaceName
}

func (in Terraform) ToBytes(scheme *runtime.Scheme) ([]byte, error) {
	return runtime.Encode(
		serializer.NewCodecFactory(scheme).LegacyCodec(
			corev1.SchemeGroupVersion,
			GroupVersion,
			sourcev1.GroupVersion,
		), &in)
}

func (in *Terraform) FromBytes(b []byte, scheme *runtime.Scheme) error {
	return runtime.DecodeInto(
		serializer.NewCodecFactory(scheme).LegacyCodec(
			corev1.SchemeGroupVersion,
			GroupVersion,
			sourcev1.GroupVersion,
		), b, in)
}

func (in *Terraform) GetRunnerHostname(ip string) string {
	prefix := strings.ReplaceAll(ip, ".", "-")
	return fmt.Sprintf("%s.%s.pod.cluster.local", prefix, in.Namespace)
}

func (in *TerraformSpec) GetAlwaysCleanupRunnerPod() bool {
	if in.AlwaysCleanupRunnerPod == nil {
		return true
	}

	return *in.AlwaysCleanupRunnerPod
}

func trimString(str string, limit int) string {
	if len(str) <= limit {
		return str
	}

	return str[0:limit] + "..."
}

func init() {
	SchemeBuilder.Register(&Terraform{}, &TerraformList{})
}
