/*
Copyright 2023 The Kubernetes Authors.
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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha2

// JobSetSpecApplyConfiguration represents a declarative configuration of the JobSetSpec type for use
// with apply.
type JobSetSpecApplyConfiguration struct {
	ReplicatedJobs          []ReplicatedJobApplyConfiguration `json:"replicatedJobs,omitempty"`
	Network                 *NetworkApplyConfiguration        `json:"network,omitempty"`
	SuccessPolicy           *SuccessPolicyApplyConfiguration  `json:"successPolicy,omitempty"`
	FailurePolicy           *FailurePolicyApplyConfiguration  `json:"failurePolicy,omitempty"`
	StartupPolicy           *StartupPolicyApplyConfiguration  `json:"startupPolicy,omitempty"`
	Suspend                 *bool                             `json:"suspend,omitempty"`
	Coordinator             *CoordinatorApplyConfiguration    `json:"coordinator,omitempty"`
	ManagedBy               *string                           `json:"managedBy,omitempty"`
	TTLSecondsAfterFinished *int32                            `json:"ttlSecondsAfterFinished,omitempty"`
}

// JobSetSpecApplyConfiguration constructs a declarative configuration of the JobSetSpec type for use with
// apply.
func JobSetSpec() *JobSetSpecApplyConfiguration {
	return &JobSetSpecApplyConfiguration{}
}

// WithReplicatedJobs adds the given value to the ReplicatedJobs field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ReplicatedJobs field.
func (b *JobSetSpecApplyConfiguration) WithReplicatedJobs(values ...*ReplicatedJobApplyConfiguration) *JobSetSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithReplicatedJobs")
		}
		b.ReplicatedJobs = append(b.ReplicatedJobs, *values[i])
	}
	return b
}

// WithNetwork sets the Network field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Network field is set to the value of the last call.
func (b *JobSetSpecApplyConfiguration) WithNetwork(value *NetworkApplyConfiguration) *JobSetSpecApplyConfiguration {
	b.Network = value
	return b
}

// WithSuccessPolicy sets the SuccessPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SuccessPolicy field is set to the value of the last call.
func (b *JobSetSpecApplyConfiguration) WithSuccessPolicy(value *SuccessPolicyApplyConfiguration) *JobSetSpecApplyConfiguration {
	b.SuccessPolicy = value
	return b
}

// WithFailurePolicy sets the FailurePolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FailurePolicy field is set to the value of the last call.
func (b *JobSetSpecApplyConfiguration) WithFailurePolicy(value *FailurePolicyApplyConfiguration) *JobSetSpecApplyConfiguration {
	b.FailurePolicy = value
	return b
}

// WithStartupPolicy sets the StartupPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the StartupPolicy field is set to the value of the last call.
func (b *JobSetSpecApplyConfiguration) WithStartupPolicy(value *StartupPolicyApplyConfiguration) *JobSetSpecApplyConfiguration {
	b.StartupPolicy = value
	return b
}

// WithSuspend sets the Suspend field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Suspend field is set to the value of the last call.
func (b *JobSetSpecApplyConfiguration) WithSuspend(value bool) *JobSetSpecApplyConfiguration {
	b.Suspend = &value
	return b
}

// WithCoordinator sets the Coordinator field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Coordinator field is set to the value of the last call.
func (b *JobSetSpecApplyConfiguration) WithCoordinator(value *CoordinatorApplyConfiguration) *JobSetSpecApplyConfiguration {
	b.Coordinator = value
	return b
}

// WithManagedBy sets the ManagedBy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ManagedBy field is set to the value of the last call.
func (b *JobSetSpecApplyConfiguration) WithManagedBy(value string) *JobSetSpecApplyConfiguration {
	b.ManagedBy = &value
	return b
}

// WithTTLSecondsAfterFinished sets the TTLSecondsAfterFinished field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TTLSecondsAfterFinished field is set to the value of the last call.
func (b *JobSetSpecApplyConfiguration) WithTTLSecondsAfterFinished(value int32) *JobSetSpecApplyConfiguration {
	b.TTLSecondsAfterFinished = &value
	return b
}
