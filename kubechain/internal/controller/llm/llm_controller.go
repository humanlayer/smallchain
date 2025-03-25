/*
Copyright 2025.

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

package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/googleai/vertex"
	"github.com/tmc/langchaingo/llms/mistral"
	"github.com/tmc/langchaingo/llms/openai"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=llms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubechain.humanlayer.dev,resources=llms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// LLMReconciler reconciles a LLM object
type LLMReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

//
// llms.withTools can be used for passing in tools
// This is in options.go in langchaingo/llms/
// WithTools will add an option to set the tools to use.
// func WithTools(tools []Tool) CallOption {
// 	return func(o *CallOptions) {
// 		o.Tools = tools
// 	}
// }

// Some providers can have a base url. Here is an example of a base url	for OpenAI.
// This is in openaillm_option.go in langchaingo/llms/openai/
// WithBaseURL passes the OpenAI base url to the client. If not set, the base url
// is read from the OPENAI_BASE_URL environment variable. If still not set in ENV
// VAR OPENAI_BASE_URL, then the default value is https://api.openai.com/v1 is used.
//
//	func WithBaseURL(baseURL string) Option {
//		return func(opts *options) {
//			opts.baseURL = baseURL
//		}
//	}

func (r *LLMReconciler) validateProviderConfig(ctx context.Context, llm *kubechainv1alpha1.LLM, apiKey string) error {
	var err error
	var model llms.Model

	// Common options from BaseConfig
	commonOpts := []llms.CallOption{}
	if llm.Spec.BaseConfig.Model != "" {
		commonOpts = append(commonOpts, llms.WithModel(llm.Spec.BaseConfig.Model))
	}
	if llm.Spec.BaseConfig.MaxTokens != nil {
		commonOpts = append(commonOpts, llms.WithMaxTokens(*llm.Spec.BaseConfig.MaxTokens))
	}
	if llm.Spec.BaseConfig.Temperature != "" {
		// Parse temperature string to float64
		var temp float64
		_, err := fmt.Sscanf(llm.Spec.BaseConfig.Temperature, "%f", &temp)
		if err == nil && temp >= 0 && temp <= 1 {
			commonOpts = append(commonOpts, llms.WithTemperature(temp))
		}
	}
	// Add TopP if configured
	if llm.Spec.BaseConfig.TopP != "" {
		// Parse TopP string to float64
		var topP float64
		_, err := fmt.Sscanf(llm.Spec.BaseConfig.TopP, "%f", &topP)
		if err == nil && topP >= 0 && topP <= 1 {
			commonOpts = append(commonOpts, llms.WithTopP(topP))
		}
	}
	// Add TopK if configured
	if llm.Spec.BaseConfig.TopK != nil {
		commonOpts = append(commonOpts, llms.WithTopK(*llm.Spec.BaseConfig.TopK))
	}
	// Add FrequencyPenalty if configured
	if llm.Spec.BaseConfig.FrequencyPenalty != "" {
		// Parse FrequencyPenalty string to float64
		var freqPenalty float64
		_, err := fmt.Sscanf(llm.Spec.BaseConfig.FrequencyPenalty, "%f", &freqPenalty)
		if err == nil && freqPenalty >= -2 && freqPenalty <= 2 {
			commonOpts = append(commonOpts, llms.WithFrequencyPenalty(freqPenalty))
		}
	}
	// Add PresencePenalty if configured
	if llm.Spec.BaseConfig.PresencePenalty != "" {
		// Parse PresencePenalty string to float64
		var presPenalty float64
		_, err := fmt.Sscanf(llm.Spec.BaseConfig.PresencePenalty, "%f", &presPenalty)
		if err == nil && presPenalty >= -2 && presPenalty <= 2 {
			commonOpts = append(commonOpts, llms.WithPresencePenalty(presPenalty))
		}
	}

	switch llm.Spec.Provider {
	case "openai":
		if llm.Spec.APIKeyFrom == nil {
			return fmt.Errorf("apiKeyFrom is required for openai")
		}
		providerOpts := []openai.Option{openai.WithToken(apiKey)}

		// Configure BaseURL if provided
		if llm.Spec.BaseConfig.BaseURL != "" {
			providerOpts = append(providerOpts, openai.WithBaseURL(llm.Spec.BaseConfig.BaseURL))
		}

		// Configure OpenAI specific options if provided
		if llm.Spec.ProviderConfig.OpenAIConfig != nil {
			config := llm.Spec.ProviderConfig.OpenAIConfig

			// Set organization if provided
			if config.Organization != "" {
				providerOpts = append(providerOpts, openai.WithOrganization(config.Organization))
			}

			// Configure API type if provided
			if config.APIType != "" {
				var apiType openai.APIType
				switch config.APIType {
				case "AZURE":
					apiType = openai.APITypeAzure
				case "AZURE_AD":
					apiType = openai.APITypeAzureAD
				default:
					apiType = openai.APITypeOpenAI
				}
				providerOpts = append(providerOpts, openai.WithAPIType(apiType))

				// When using Azure APIs, configure API Version
				if (config.APIType == "AZURE" || config.APIType == "AZURE_AD") && config.APIVersion != "" {
					providerOpts = append(providerOpts, openai.WithAPIVersion(config.APIVersion))
				}
			}
		}

		model, err = openai.New(providerOpts...)

	case "anthropic":
		if llm.Spec.APIKeyFrom == nil {
			return fmt.Errorf("apiKeyFrom is required for anthropic")
		}
		providerOpts := []anthropic.Option{anthropic.WithToken(apiKey)}
		if llm.Spec.BaseConfig.BaseURL != "" {
			providerOpts = append(providerOpts, anthropic.WithBaseURL(llm.Spec.BaseConfig.BaseURL))
		}
		if llm.Spec.ProviderConfig.AnthropicConfig != nil && llm.Spec.ProviderConfig.AnthropicConfig.AnthropicBetaHeader != "" {
			providerOpts = append(providerOpts, anthropic.WithAnthropicBetaHeader(llm.Spec.ProviderConfig.AnthropicConfig.AnthropicBetaHeader))
		}
		model, err = anthropic.New(providerOpts...)

	case "mistral":
		if llm.Spec.APIKeyFrom == nil {
			return fmt.Errorf("apiKeyFrom is required for mistral")
		}
		providerOpts := []mistral.Option{mistral.WithAPIKey(apiKey)}

		// Configure BaseURL as endpoint
		if llm.Spec.BaseConfig.BaseURL != "" {
			providerOpts = append(providerOpts, mistral.WithEndpoint(llm.Spec.BaseConfig.BaseURL))
		}

		// Configure model
		if llm.Spec.BaseConfig.Model != "" {
			providerOpts = append(providerOpts, mistral.WithModel(llm.Spec.BaseConfig.Model))
		}

		// Configure Mistral-specific options if provided
		if llm.Spec.ProviderConfig.MistralConfig != nil {
			config := llm.Spec.ProviderConfig.MistralConfig

			// Set MaxRetries if provided
			if config.MaxRetries != nil {
				providerOpts = append(providerOpts, mistral.WithMaxRetries(*config.MaxRetries))
			}

			// Set Timeout if provided (converting seconds to time.Duration)
			if config.Timeout != nil {
				timeoutDuration := time.Duration(*config.Timeout) * time.Second
				providerOpts = append(providerOpts, mistral.WithTimeout(timeoutDuration))
			}

			// Set RandomSeed if provided
			if config.RandomSeed != nil {
				commonOpts = append(commonOpts, llms.WithSeed(*config.RandomSeed))
			}
		}

		// Create the Mistral model with the provider options
		model, err = mistral.New(providerOpts...)

		// Pass any common options to the model during generation test
		if len(commonOpts) > 0 {
			commonOpts = append(commonOpts, llms.WithMaxTokens(1), llms.WithTemperature(0))
			_, err = llms.GenerateFromSinglePrompt(ctx, model, "test", commonOpts...)
			if err != nil {
				return fmt.Errorf("mistral validation failed with options: %w", err)
			}
			return nil
		}

	case "google":
		if llm.Spec.APIKeyFrom == nil {
			return fmt.Errorf("apiKeyFrom is required for google")
		}
		providerOpts := []googleai.Option{googleai.WithAPIKey(apiKey)}
		if llm.Spec.ProviderConfig.GoogleConfig != nil {
			if llm.Spec.ProviderConfig.GoogleConfig.CloudProject != "" {
				providerOpts = append(providerOpts, googleai.WithCloudProject(llm.Spec.ProviderConfig.GoogleConfig.CloudProject))
			}
			if llm.Spec.ProviderConfig.GoogleConfig.CloudLocation != "" {
				providerOpts = append(providerOpts, googleai.WithCloudLocation(llm.Spec.ProviderConfig.GoogleConfig.CloudLocation))
			}
		}
		if llm.Spec.BaseConfig.Model != "" {
			providerOpts = append(providerOpts, googleai.WithDefaultModel(llm.Spec.BaseConfig.Model))
		}
		model, err = googleai.New(ctx, providerOpts...)

	case "vertex":
		if llm.Spec.ProviderConfig.VertexConfig == nil {
			return fmt.Errorf("vertexConfig is required for vertex")
		}
		config := llm.Spec.ProviderConfig.VertexConfig
		providerOpts := []googleai.Option{
			googleai.WithCloudProject(config.CloudProject),
			googleai.WithCloudLocation(config.CloudLocation),
		}
		if llm.Spec.APIKeyFrom != nil && apiKey != "" {
			providerOpts = append(providerOpts, googleai.WithCredentialsJSON([]byte(apiKey)))
		}
		if llm.Spec.BaseConfig.Model != "" {
			providerOpts = append(providerOpts, googleai.WithDefaultModel(llm.Spec.BaseConfig.Model))
		}
		model, err = vertex.New(ctx, providerOpts...)

	default:
		return fmt.Errorf("unsupported provider: %s", llm.Spec.Provider)
	}

	if err != nil {
		return fmt.Errorf("failed to initialize %s client: %w", llm.Spec.Provider, err)
	}

	// Validate with a test call
	_, err = llms.GenerateFromSinglePrompt(ctx, model, "test", llms.WithTemperature(0), llms.WithMaxTokens(1))
	if err != nil {
		return fmt.Errorf("%s API validation failed: %w", llm.Spec.Provider, err)
	}

	return nil
}

func (r *LLMReconciler) validateOpenAIKey(apiKey string) error {
	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid API key (status code: %d)", resp.StatusCode)
	}

	return nil
}

func (r *LLMReconciler) validateSecret(ctx context.Context, llm *kubechainv1alpha1.LLM) (string, error) {
	// All providers require API keys
	if llm.Spec.APIKeyFrom == nil {
		return "", fmt.Errorf("apiKeyFrom is required for provider %s", llm.Spec.Provider)
	}

	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      llm.Spec.APIKeyFrom.SecretKeyRef.Name,
		Namespace: llm.Namespace,
	}, secret)
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %w", err)
	}

	key := llm.Spec.APIKeyFrom.SecretKeyRef.Key
	apiKey, exists := secret.Data[key]
	if !exists {
		return "", fmt.Errorf("key %q not found in secret", key)
	}

	if llm.Spec.Provider == "openai" {
		if err := r.validateOpenAIKey(string(apiKey)); err != nil {
			return "", fmt.Errorf("OpenAI API key validation failed: %w", err)
		}
	}

	return string(apiKey), nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LLMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the LLM instance
	var llm kubechainv1alpha1.LLM
	if err := r.Get(ctx, req.NamespacedName, &llm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Starting reconciliation", "namespacedName", req.NamespacedName, "provider", llm.Spec.Provider)

	// Create a copy for status update
	statusUpdate := llm.DeepCopy()

	// Initialize status if not set
	if statusUpdate.Status.Status == "" {
		statusUpdate.Status.Status = "Pending"
		statusUpdate.Status.StatusDetail = "Validating configuration"
		r.recorder.Event(&llm, corev1.EventTypeNormal, "Initializing", "Starting validation")
	}

	// Validate secret and get API key (if applicable)
	// TODO: Will this work with amazon bedrock? Probably not?? If so we should look at adding tests for this specifically.
	apiKey, err := r.validateSecret(ctx, &llm)
	if err != nil {
		log.Error(err, "Secret validation failed")
		statusUpdate.Status.Ready = false
		statusUpdate.Status.Status = "Error"
		statusUpdate.Status.StatusDetail = err.Error()
		r.recorder.Event(&llm, corev1.EventTypeWarning, "SecretValidationFailed", err.Error())
	} else {
		// Validate provider with API key
		err := r.validateProviderConfig(ctx, &llm, apiKey)
		if err != nil {
			log.Error(err, "Provider validation failed")
			statusUpdate.Status.Ready = false
			statusUpdate.Status.Status = "Error"
			statusUpdate.Status.StatusDetail = err.Error()
			r.recorder.Event(&llm, corev1.EventTypeWarning, "ValidationFailed", err.Error())
		} else {
			statusUpdate.Status.Ready = true
			statusUpdate.Status.Status = "Ready"
			statusUpdate.Status.StatusDetail = fmt.Sprintf("%s provider validated successfully", llm.Spec.Provider)
			r.recorder.Event(&llm, corev1.EventTypeNormal, "ValidationSucceeded", statusUpdate.Status.StatusDetail)
		}
	}

	// Update status using SubResource client
	if err := r.Status().Patch(ctx, statusUpdate, client.MergeFrom(&llm)); err != nil {
		log.Error(err, "Unable to update LLM status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled LLM",
		"provider", llm.Spec.Provider,
		"ready", statusUpdate.Status.Ready,
		"status", statusUpdate.Status.Status,
		"statusDetail", statusUpdate.Status.StatusDetail)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("llm-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubechainv1alpha1.LLM{}).
		Named("llm").
		Complete(r)
}
