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

// checks if the ResponseOption type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &ResponseOption{}

// ResponseOption struct for ResponseOption
type ResponseOption struct {
	Name        string         `json:"name"`
	Title       NullableString `json:"title,omitempty"`
	Description NullableString `json:"description,omitempty"`
	PromptFill  NullableString `json:"prompt_fill,omitempty"`
	Interactive *bool          `json:"interactive,omitempty"`
}

type _ResponseOption ResponseOption

// NewResponseOption instantiates a new ResponseOption object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewResponseOption(name string) *ResponseOption {
	this := ResponseOption{}
	this.Name = name
	var interactive bool = false
	this.Interactive = &interactive
	return &this
}

// NewResponseOptionWithDefaults instantiates a new ResponseOption object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewResponseOptionWithDefaults() *ResponseOption {
	this := ResponseOption{}
	var interactive bool = false
	this.Interactive = &interactive
	return &this
}

// GetName returns the Name field value
func (o *ResponseOption) GetName() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *ResponseOption) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value
func (o *ResponseOption) SetName(v string) {
	o.Name = v
}

// GetTitle returns the Title field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *ResponseOption) GetTitle() string {
	if o == nil || IsNil(o.Title.Get()) {
		var ret string
		return ret
	}
	return *o.Title.Get()
}

// GetTitleOk returns a tuple with the Title field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *ResponseOption) GetTitleOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Title.Get(), o.Title.IsSet()
}

// HasTitle returns a boolean if a field has been set.
func (o *ResponseOption) HasTitle() bool {
	if o != nil && o.Title.IsSet() {
		return true
	}

	return false
}

// SetTitle gets a reference to the given NullableString and assigns it to the Title field.
func (o *ResponseOption) SetTitle(v string) {
	o.Title.Set(&v)
}

// SetTitleNil sets the value for Title to be an explicit nil
func (o *ResponseOption) SetTitleNil() {
	o.Title.Set(nil)
}

// UnsetTitle ensures that no value is present for Title, not even an explicit nil
func (o *ResponseOption) UnsetTitle() {
	o.Title.Unset()
}

// GetDescription returns the Description field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *ResponseOption) GetDescription() string {
	if o == nil || IsNil(o.Description.Get()) {
		var ret string
		return ret
	}
	return *o.Description.Get()
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *ResponseOption) GetDescriptionOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Description.Get(), o.Description.IsSet()
}

// HasDescription returns a boolean if a field has been set.
func (o *ResponseOption) HasDescription() bool {
	if o != nil && o.Description.IsSet() {
		return true
	}

	return false
}

// SetDescription gets a reference to the given NullableString and assigns it to the Description field.
func (o *ResponseOption) SetDescription(v string) {
	o.Description.Set(&v)
}

// SetDescriptionNil sets the value for Description to be an explicit nil
func (o *ResponseOption) SetDescriptionNil() {
	o.Description.Set(nil)
}

// UnsetDescription ensures that no value is present for Description, not even an explicit nil
func (o *ResponseOption) UnsetDescription() {
	o.Description.Unset()
}

// GetPromptFill returns the PromptFill field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *ResponseOption) GetPromptFill() string {
	if o == nil || IsNil(o.PromptFill.Get()) {
		var ret string
		return ret
	}
	return *o.PromptFill.Get()
}

// GetPromptFillOk returns a tuple with the PromptFill field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *ResponseOption) GetPromptFillOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.PromptFill.Get(), o.PromptFill.IsSet()
}

// HasPromptFill returns a boolean if a field has been set.
func (o *ResponseOption) HasPromptFill() bool {
	if o != nil && o.PromptFill.IsSet() {
		return true
	}

	return false
}

// SetPromptFill gets a reference to the given NullableString and assigns it to the PromptFill field.
func (o *ResponseOption) SetPromptFill(v string) {
	o.PromptFill.Set(&v)
}

// SetPromptFillNil sets the value for PromptFill to be an explicit nil
func (o *ResponseOption) SetPromptFillNil() {
	o.PromptFill.Set(nil)
}

// UnsetPromptFill ensures that no value is present for PromptFill, not even an explicit nil
func (o *ResponseOption) UnsetPromptFill() {
	o.PromptFill.Unset()
}

// GetInteractive returns the Interactive field value if set, zero value otherwise.
func (o *ResponseOption) GetInteractive() bool {
	if o == nil || IsNil(o.Interactive) {
		var ret bool
		return ret
	}
	return *o.Interactive
}

// GetInteractiveOk returns a tuple with the Interactive field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ResponseOption) GetInteractiveOk() (*bool, bool) {
	if o == nil || IsNil(o.Interactive) {
		return nil, false
	}
	return o.Interactive, true
}

// HasInteractive returns a boolean if a field has been set.
func (o *ResponseOption) HasInteractive() bool {
	if o != nil && !IsNil(o.Interactive) {
		return true
	}

	return false
}

// SetInteractive gets a reference to the given bool and assigns it to the Interactive field.
func (o *ResponseOption) SetInteractive(v bool) {
	o.Interactive = &v
}

func (o ResponseOption) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o ResponseOption) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["name"] = o.Name
	if o.Title.IsSet() {
		toSerialize["title"] = o.Title.Get()
	}
	if o.Description.IsSet() {
		toSerialize["description"] = o.Description.Get()
	}
	if o.PromptFill.IsSet() {
		toSerialize["prompt_fill"] = o.PromptFill.Get()
	}
	if !IsNil(o.Interactive) {
		toSerialize["interactive"] = o.Interactive
	}
	return toSerialize, nil
}

func (o *ResponseOption) UnmarshalJSON(data []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"name",
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

	varResponseOption := _ResponseOption{}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&varResponseOption)

	if err != nil {
		return err
	}

	*o = ResponseOption(varResponseOption)

	return err
}

type NullableResponseOption struct {
	value *ResponseOption
	isSet bool
}

func (v NullableResponseOption) Get() *ResponseOption {
	return v.value
}

func (v *NullableResponseOption) Set(val *ResponseOption) {
	v.value = val
	v.isSet = true
}

func (v NullableResponseOption) IsSet() bool {
	return v.isSet
}

func (v *NullableResponseOption) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableResponseOption(val *ResponseOption) *NullableResponseOption {
	return &NullableResponseOption{value: val, isSet: true}
}

func (v NullableResponseOption) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableResponseOption) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
