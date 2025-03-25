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

// checks if the EmailContactChannel type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &EmailContactChannel{}

// EmailContactChannel Route for contacting a user via email
type EmailContactChannel struct {
	Address                         string           `json:"address"`
	ContextAboutUser                NullableString   `json:"context_about_user,omitempty"`
	AdditionalRecipients            []EmailRecipient `json:"additional_recipients,omitempty"`
	ExperimentalSubjectLine         NullableString   `json:"experimental_subject_line,omitempty"`
	ExperimentalReferencesMessageId NullableString   `json:"experimental_references_message_id,omitempty"`
	ExperimentalInReplyToMessageId  NullableString   `json:"experimental_in_reply_to_message_id,omitempty"`
	Template                        NullableString   `json:"template,omitempty"`
}

type _EmailContactChannel EmailContactChannel

// NewEmailContactChannel instantiates a new EmailContactChannel object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewEmailContactChannel(address string) *EmailContactChannel {
	this := EmailContactChannel{}
	this.Address = address
	return &this
}

// NewEmailContactChannelWithDefaults instantiates a new EmailContactChannel object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewEmailContactChannelWithDefaults() *EmailContactChannel {
	this := EmailContactChannel{}
	return &this
}

// GetAddress returns the Address field value
func (o *EmailContactChannel) GetAddress() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Address
}

// GetAddressOk returns a tuple with the Address field value
// and a boolean to check if the value has been set.
func (o *EmailContactChannel) GetAddressOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Address, true
}

// SetAddress sets field value
func (o *EmailContactChannel) SetAddress(v string) {
	o.Address = v
}

// GetContextAboutUser returns the ContextAboutUser field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EmailContactChannel) GetContextAboutUser() string {
	if o == nil || IsNil(o.ContextAboutUser.Get()) {
		var ret string
		return ret
	}
	return *o.ContextAboutUser.Get()
}

// GetContextAboutUserOk returns a tuple with the ContextAboutUser field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *EmailContactChannel) GetContextAboutUserOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.ContextAboutUser.Get(), o.ContextAboutUser.IsSet()
}

// HasContextAboutUser returns a boolean if a field has been set.
func (o *EmailContactChannel) HasContextAboutUser() bool {
	if o != nil && o.ContextAboutUser.IsSet() {
		return true
	}

	return false
}

// SetContextAboutUser gets a reference to the given NullableString and assigns it to the ContextAboutUser field.
func (o *EmailContactChannel) SetContextAboutUser(v string) {
	o.ContextAboutUser.Set(&v)
}

// SetContextAboutUserNil sets the value for ContextAboutUser to be an explicit nil
func (o *EmailContactChannel) SetContextAboutUserNil() {
	o.ContextAboutUser.Set(nil)
}

// UnsetContextAboutUser ensures that no value is present for ContextAboutUser, not even an explicit nil
func (o *EmailContactChannel) UnsetContextAboutUser() {
	o.ContextAboutUser.Unset()
}

// GetAdditionalRecipients returns the AdditionalRecipients field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EmailContactChannel) GetAdditionalRecipients() []EmailRecipient {
	if o == nil {
		var ret []EmailRecipient
		return ret
	}
	return o.AdditionalRecipients
}

// GetAdditionalRecipientsOk returns a tuple with the AdditionalRecipients field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *EmailContactChannel) GetAdditionalRecipientsOk() ([]EmailRecipient, bool) {
	if o == nil || IsNil(o.AdditionalRecipients) {
		return nil, false
	}
	return o.AdditionalRecipients, true
}

// HasAdditionalRecipients returns a boolean if a field has been set.
func (o *EmailContactChannel) HasAdditionalRecipients() bool {
	if o != nil && !IsNil(o.AdditionalRecipients) {
		return true
	}

	return false
}

// SetAdditionalRecipients gets a reference to the given []EmailRecipient and assigns it to the AdditionalRecipients field.
func (o *EmailContactChannel) SetAdditionalRecipients(v []EmailRecipient) {
	o.AdditionalRecipients = v
}

// GetExperimentalSubjectLine returns the ExperimentalSubjectLine field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EmailContactChannel) GetExperimentalSubjectLine() string {
	if o == nil || IsNil(o.ExperimentalSubjectLine.Get()) {
		var ret string
		return ret
	}
	return *o.ExperimentalSubjectLine.Get()
}

// GetExperimentalSubjectLineOk returns a tuple with the ExperimentalSubjectLine field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *EmailContactChannel) GetExperimentalSubjectLineOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.ExperimentalSubjectLine.Get(), o.ExperimentalSubjectLine.IsSet()
}

// HasExperimentalSubjectLine returns a boolean if a field has been set.
func (o *EmailContactChannel) HasExperimentalSubjectLine() bool {
	if o != nil && o.ExperimentalSubjectLine.IsSet() {
		return true
	}

	return false
}

// SetExperimentalSubjectLine gets a reference to the given NullableString and assigns it to the ExperimentalSubjectLine field.
func (o *EmailContactChannel) SetExperimentalSubjectLine(v string) {
	o.ExperimentalSubjectLine.Set(&v)
}

// SetExperimentalSubjectLineNil sets the value for ExperimentalSubjectLine to be an explicit nil
func (o *EmailContactChannel) SetExperimentalSubjectLineNil() {
	o.ExperimentalSubjectLine.Set(nil)
}

// UnsetExperimentalSubjectLine ensures that no value is present for ExperimentalSubjectLine, not even an explicit nil
func (o *EmailContactChannel) UnsetExperimentalSubjectLine() {
	o.ExperimentalSubjectLine.Unset()
}

// GetExperimentalReferencesMessageId returns the ExperimentalReferencesMessageId field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EmailContactChannel) GetExperimentalReferencesMessageId() string {
	if o == nil || IsNil(o.ExperimentalReferencesMessageId.Get()) {
		var ret string
		return ret
	}
	return *o.ExperimentalReferencesMessageId.Get()
}

// GetExperimentalReferencesMessageIdOk returns a tuple with the ExperimentalReferencesMessageId field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *EmailContactChannel) GetExperimentalReferencesMessageIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.ExperimentalReferencesMessageId.Get(), o.ExperimentalReferencesMessageId.IsSet()
}

// HasExperimentalReferencesMessageId returns a boolean if a field has been set.
func (o *EmailContactChannel) HasExperimentalReferencesMessageId() bool {
	if o != nil && o.ExperimentalReferencesMessageId.IsSet() {
		return true
	}

	return false
}

// SetExperimentalReferencesMessageId gets a reference to the given NullableString and assigns it to the ExperimentalReferencesMessageId field.
func (o *EmailContactChannel) SetExperimentalReferencesMessageId(v string) {
	o.ExperimentalReferencesMessageId.Set(&v)
}

// SetExperimentalReferencesMessageIdNil sets the value for ExperimentalReferencesMessageId to be an explicit nil
func (o *EmailContactChannel) SetExperimentalReferencesMessageIdNil() {
	o.ExperimentalReferencesMessageId.Set(nil)
}

// UnsetExperimentalReferencesMessageId ensures that no value is present for ExperimentalReferencesMessageId, not even an explicit nil
func (o *EmailContactChannel) UnsetExperimentalReferencesMessageId() {
	o.ExperimentalReferencesMessageId.Unset()
}

// GetExperimentalInReplyToMessageId returns the ExperimentalInReplyToMessageId field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EmailContactChannel) GetExperimentalInReplyToMessageId() string {
	if o == nil || IsNil(o.ExperimentalInReplyToMessageId.Get()) {
		var ret string
		return ret
	}
	return *o.ExperimentalInReplyToMessageId.Get()
}

// GetExperimentalInReplyToMessageIdOk returns a tuple with the ExperimentalInReplyToMessageId field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *EmailContactChannel) GetExperimentalInReplyToMessageIdOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.ExperimentalInReplyToMessageId.Get(), o.ExperimentalInReplyToMessageId.IsSet()
}

// HasExperimentalInReplyToMessageId returns a boolean if a field has been set.
func (o *EmailContactChannel) HasExperimentalInReplyToMessageId() bool {
	if o != nil && o.ExperimentalInReplyToMessageId.IsSet() {
		return true
	}

	return false
}

// SetExperimentalInReplyToMessageId gets a reference to the given NullableString and assigns it to the ExperimentalInReplyToMessageId field.
func (o *EmailContactChannel) SetExperimentalInReplyToMessageId(v string) {
	o.ExperimentalInReplyToMessageId.Set(&v)
}

// SetExperimentalInReplyToMessageIdNil sets the value for ExperimentalInReplyToMessageId to be an explicit nil
func (o *EmailContactChannel) SetExperimentalInReplyToMessageIdNil() {
	o.ExperimentalInReplyToMessageId.Set(nil)
}

// UnsetExperimentalInReplyToMessageId ensures that no value is present for ExperimentalInReplyToMessageId, not even an explicit nil
func (o *EmailContactChannel) UnsetExperimentalInReplyToMessageId() {
	o.ExperimentalInReplyToMessageId.Unset()
}

// GetTemplate returns the Template field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *EmailContactChannel) GetTemplate() string {
	if o == nil || IsNil(o.Template.Get()) {
		var ret string
		return ret
	}
	return *o.Template.Get()
}

// GetTemplateOk returns a tuple with the Template field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *EmailContactChannel) GetTemplateOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return o.Template.Get(), o.Template.IsSet()
}

// HasTemplate returns a boolean if a field has been set.
func (o *EmailContactChannel) HasTemplate() bool {
	if o != nil && o.Template.IsSet() {
		return true
	}

	return false
}

// SetTemplate gets a reference to the given NullableString and assigns it to the Template field.
func (o *EmailContactChannel) SetTemplate(v string) {
	o.Template.Set(&v)
}

// SetTemplateNil sets the value for Template to be an explicit nil
func (o *EmailContactChannel) SetTemplateNil() {
	o.Template.Set(nil)
}

// UnsetTemplate ensures that no value is present for Template, not even an explicit nil
func (o *EmailContactChannel) UnsetTemplate() {
	o.Template.Unset()
}

func (o EmailContactChannel) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o EmailContactChannel) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["address"] = o.Address
	if o.ContextAboutUser.IsSet() {
		toSerialize["context_about_user"] = o.ContextAboutUser.Get()
	}
	if o.AdditionalRecipients != nil {
		toSerialize["additional_recipients"] = o.AdditionalRecipients
	}
	if o.ExperimentalSubjectLine.IsSet() {
		toSerialize["experimental_subject_line"] = o.ExperimentalSubjectLine.Get()
	}
	if o.ExperimentalReferencesMessageId.IsSet() {
		toSerialize["experimental_references_message_id"] = o.ExperimentalReferencesMessageId.Get()
	}
	if o.ExperimentalInReplyToMessageId.IsSet() {
		toSerialize["experimental_in_reply_to_message_id"] = o.ExperimentalInReplyToMessageId.Get()
	}
	if o.Template.IsSet() {
		toSerialize["template"] = o.Template.Get()
	}
	return toSerialize, nil
}

func (o *EmailContactChannel) UnmarshalJSON(data []byte) (err error) {
	// This validates that all required properties are included in the JSON object
	// by unmarshalling the object into a generic map with string keys and checking
	// that every required field exists as a key in the generic map.
	requiredProperties := []string{
		"address",
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

	varEmailContactChannel := _EmailContactChannel{}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&varEmailContactChannel)

	if err != nil {
		return err
	}

	*o = EmailContactChannel(varEmailContactChannel)

	return err
}

type NullableEmailContactChannel struct {
	value *EmailContactChannel
	isSet bool
}

func (v NullableEmailContactChannel) Get() *EmailContactChannel {
	return v.value
}

func (v *NullableEmailContactChannel) Set(val *EmailContactChannel) {
	v.value = val
	v.isSet = true
}

func (v NullableEmailContactChannel) IsSet() bool {
	return v.isSet
}

func (v *NullableEmailContactChannel) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableEmailContactChannel(val *EmailContactChannel) *NullableEmailContactChannel {
	return &NullableEmailContactChannel{value: val, isSet: true}
}

func (v NullableEmailContactChannel) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableEmailContactChannel) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
