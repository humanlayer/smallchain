/*
FastAPI

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 0.1.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package humanlayerapi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// checks if the HumanContactOutput type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &HumanContactOutput{}

// HumanContactOutput struct for HumanContactOutput
type HumanContactOutput struct {
	RunId  string                     `json:"run_id"`
	CallId string                     `json:"call_id"`
	Spec   HumanContactSpecOutput     `json:"spec"`
	Status NullableHumanContactStatus `json:"status,omitempty"`
}

type _HumanContactOutput HumanContactOutput

// NewHumanContactOutput instantiates a new HumanContactOutput object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewHumanContactOutput(runId string, callId string, spec HumanContactSpecOutput) *HumanContactOutput {
	this := HumanContactOutput{}
	this.RunId = runId
	this.CallId = callId
	this.Spec = spec
	return &this
}

// NewHumanContactOutputWithDefaults instantiates a new HumanContactOutput object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewHumanContactOutputWithDefaults() *HumanContactOutput {
	this := HumanContactOutput{}
	return &this
}

// GetRunId returns the RunId field value
func (o *HumanContactOutput) GetRunId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.RunId
}

// GetRunIdOk returns a tuple with the RunId field value
// and a boolean to check if the value has been set.
func (o *HumanContactOutput) GetRunIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.RunId, true
}

// SetRunId sets field value
func (o *HumanContactOutput) SetRunId(v string) {
	o.RunId = v
}

// GetCallId returns the CallId field value
func (o *HumanContactOutput) GetCallId() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.CallId
}

// GetCallIdOk returns a tuple with the CallId field value
// and a boolean to check if the value has been set.
func (o *HumanContactOutput) GetCallIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.CallId, true
}

// SetCallId sets field value
func (o *HumanContactOutput) SetCallId(v string) {
	o.CallId = v
}

// GetSpec returns the Spec field value
func (o *HumanContactOutput) GetSpec() HumanContactSpecOutput {
	if o == nil {
		var ret HumanContactSpecOutput
		return ret
	}

	return o.Spec
}

// GetSpecOk returns a tuple with the Spec field value
// and a boolean to check if the value has been set.
func (o *HumanContactOutput) GetSpecOk() (*HumanContactSpecOutput, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Spec, true
}

// SetSpec sets field value
func (o *HumanContactOutput) SetSpec(v HumanContactSpecOutput) {
	o.Spec = v
}

// GetStatus returns the Status field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *HumanContactOutput) GetStatus() HumanContactStatus {
	if o == nil || IsNil(o.Status.Get()) {
		var ret HumanContactStatus
		return ret
	}
	return *o.Status.Get()
}

// GetStatusOk returns a tuple with the Status field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *HumanContactOutput) GetStatusOk() (*HumanContactStatus, bool) {
	if o == nil {
		return nil, false
	}
	return o.Status.Get(), o.Status.IsSet()
}

// HasStatus returns a boolean if a field has been set.
func (o *HumanContactOutput) HasStatus() bool {
	if o != nil && o.Status.IsSet() {
		return true
	}

	return false
}

// SetStatus gets a reference to the given NullableHumanContactStatus and assigns it to the Status field.
func (o *HumanContactOutput) SetStatus(v HumanContactStatus) {
	o.Status.Set(&v)
}

// SetStatusNil sets the value for Status to be an explicit nil
func (o *HumanContactOutput) SetStatusNil() {
	o.Status.Set(nil)
}

// UnsetStatus ensures that no value is present for Status, not even an explicit nil
func (o *HumanContactOutput) UnsetStatus() {
	o.Status.Unset()
}

func (o HumanContactOutput) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o HumanContactOutput) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["run_id"] = o.RunId
	toSerialize["call_id"] = o.CallId
	toSerialize["spec"] = o.Spec
	if o.Status.IsSet() {
		toSerialize["status"] = o.Status.Get()
	}
	return toSerialize, nil
}

func (o *HumanContactOutput) UnmarshalJSON(data []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"run_id",
		"call_id",
		"spec",
	}

	allProperties := make(map[string]interface{})

	err = json.Unmarshal(data, &allProperties)

	if err != nil {
		return err
	}

	for _, requiredProperty := range requiredProperties {
		if _, exists := allProperties[requiredProperty]; !exists {
			return fmt.Errorf("no value given for required property %v", requiredProperty)
		}
	}

	varHumanContactOutput := _HumanContactOutput{}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&varHumanContactOutput)

	if err != nil {
		return err
	}

	*o = HumanContactOutput(varHumanContactOutput)

	return err
}

type NullableHumanContactOutput struct {
	value *HumanContactOutput
	isSet bool
}

func (v NullableHumanContactOutput) Get() *HumanContactOutput {
	return v.value
}

func (v *NullableHumanContactOutput) Set(val *HumanContactOutput) {
	v.value = val
	v.isSet = true
}

func (v NullableHumanContactOutput) IsSet() bool {
	return v.isSet
}

func (v *NullableHumanContactOutput) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableHumanContactOutput(val *HumanContactOutput) *NullableHumanContactOutput {
	return &NullableHumanContactOutput{value: val, isSet: true}
}

func (v NullableHumanContactOutput) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableHumanContactOutput) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
