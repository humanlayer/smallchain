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

// checks if the SMSContactChannel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SMSContactChannel{}

// SMSContactChannel Route for contacting a user via SMS
type SMSContactChannel struct {
	PhoneNumber      string         `json:"phone_number"`
	ContextAboutUser NullableString `json:"context_about_user,omitempty"`
}

type _SMSContactChannel SMSContactChannel

// NewSMSContactChannel instantiates a new SMSContactChannel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSMSContactChannel(phoneNumber string) *SMSContactChannel {
	this := SMSContactChannel{}
	this.PhoneNumber = phoneNumber
	return &this
}

// NewSMSContactChannelWithDefaults instantiates a new SMSContactChannel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSMSContactChannelWithDefaults() *SMSContactChannel {
	this := SMSContactChannel{}
	return &this
}

// GetPhoneNumber returns the PhoneNumber field value
func (o *SMSContactChannel) GetPhoneNumber() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.PhoneNumber
}

// GetPhoneNumberOk returns a tuple with the PhoneNumber field value
// and a boolean to check if the value has been set.
func (o *SMSContactChannel) GetPhoneNumberOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.PhoneNumber, true
}

// SetPhoneNumber sets field value
func (o *SMSContactChannel) SetPhoneNumber(v string) {
	o.PhoneNumber = v
}

// GetContextAboutUser returns the ContextAboutUser field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *SMSContactChannel) GetContextAboutUser() string {
	if o == nil || IsNil(o.ContextAboutUser.Get()) {
		var ret string
		return ret
	}
	return *o.ContextAboutUser.Get()
}

// GetContextAboutUserOk returns a tuple with the ContextAboutUser field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *SMSContactChannel) GetContextAboutUserOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.ContextAboutUser.Get(), o.ContextAboutUser.IsSet()
}

// HasContextAboutUser returns a boolean if a field has been set.
func (o *SMSContactChannel) HasContextAboutUser() bool {
	if o != nil && o.ContextAboutUser.IsSet() {
		return true
	}

	return false
}

// SetContextAboutUser gets a reference to the given NullableString and assigns it to the ContextAboutUser field.
func (o *SMSContactChannel) SetContextAboutUser(v string) {
	o.ContextAboutUser.Set(&v)
}

// SetContextAboutUserNil sets the value for ContextAboutUser to be an explicit nil
func (o *SMSContactChannel) SetContextAboutUserNil() {
	o.ContextAboutUser.Set(nil)
}

// UnsetContextAboutUser ensures that no value is present for ContextAboutUser, not even an explicit nil
func (o *SMSContactChannel) UnsetContextAboutUser() {
	o.ContextAboutUser.Unset()
}

func (o SMSContactChannel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SMSContactChannel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["phone_number"] = o.PhoneNumber
	if o.ContextAboutUser.IsSet() {
		toSerialize["context_about_user"] = o.ContextAboutUser.Get()
	}
	return toSerialize, nil
}

func (o *SMSContactChannel) UnmarshalJSON(data []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"phone_number",
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

	varSMSContactChannel := _SMSContactChannel{}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&varSMSContactChannel)

	if err != nil {
		return err
	}

	*o = SMSContactChannel(varSMSContactChannel)

	return err
}

type NullableSMSContactChannel struct {
	value *SMSContactChannel
	isSet bool
}

func (v NullableSMSContactChannel) Get() *SMSContactChannel {
	return v.value
}

func (v *NullableSMSContactChannel) Set(val *SMSContactChannel) {
	v.value = val
	v.isSet = true
}

func (v NullableSMSContactChannel) IsSet() bool {
	return v.isSet
}

func (v *NullableSMSContactChannel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSMSContactChannel(val *SMSContactChannel) *NullableSMSContactChannel {
	return &NullableSMSContactChannel{value: val, isSet: true}
}

func (v NullableSMSContactChannel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSMSContactChannel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
