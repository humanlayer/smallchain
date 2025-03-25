# LLM Providers Guide

This document provides detailed information about configuring different LLM providers in Kubechain.

## Supported Providers

Kubechain supports the following LLM providers:

- OpenAI
- Anthropic
- Vertex AI (Google Cloud)
- Bedrock (AWS)
- Mistral
- Cohere
- Google AI
- Cloudflare Workers AI

## LLM Configuration Structure

The LLM custom resource has been designed to support multiple providers with a flexible configuration structure:

```yaml
apiVersion: kubechain.humanlayer.dev/v1alpha1
kind: LLM
metadata:
  name: my-llm
spec:
  # Required: The LLM provider name
  provider: openai  # One of: openai, anthropic, vertex, bedrock, mistral, cohere, google, cloudflare

  # Optional for Bedrock (which uses AWS SDK credentials)
  # Required for all other providers
  apiKeyFrom:
    secretKeyRef:
      name: my-secret
      key: API_KEY

  # Common configuration options shared across providers
  baseConfig:
    model: "gpt-4o"         # Model name/id
    baseUrl: "https://..."  # Optional API endpoint URL
    temperature: "0.7"      # Temperature (0.0-1.0)
    maxTokens: 1000         # Maximum tokens to generate

  # Provider-specific configuration
  providerConfig:
    # Only one of these should be specified, matching the provider field above
    openaiConfig:
      organization: "org-123456"
    
    vertexConfig:
      cloudProject: "my-gcp-project"
      cloudLocation: "us-central1"
    
    bedrockConfig:
      awsRegion: "us-west-2"
    
    cloudflareConfig:
      accountId: "abcdef123456"
    
    # Other provider configs (anthropicConfig, mistralConfig, cohereConfig, googleConfig)
```

## Provider-Specific Requirements

### OpenAI

```yaml
spec:
  provider: openai
  apiKeyFrom:
    secretKeyRef:
      name: openai
      key: OPENAI_API_KEY
  baseConfig:
    model: "gpt-4o"
    temperature: "0.7"
  providerConfig:
    openaiConfig:
      organization: "org-123456"  # Optional: Your OpenAI organization ID
```

### Anthropic

```yaml
spec:
  provider: anthropic
  apiKeyFrom:
    secretKeyRef:
      name: anthropic
      key: ANTHROPIC_API_KEY
  baseConfig:
    model: "claude-3-5-sonnet-20240620"
    temperature: "0.5"
```

### Vertex AI

```yaml
spec:
  provider: vertex
  apiKeyFrom:
    secretKeyRef:
      name: vertex-credentials
      key: service-account-json  # Contains GCP service account JSON
  baseConfig:
    model: "gemini-pro"
  providerConfig:
    vertexConfig:
      cloudProject: "my-gcp-project"  # Required: GCP project ID
      cloudLocation: "us-central1"    # Required: GCP region
```

### AWS Bedrock

```yaml
spec:
  provider: bedrock
  # No apiKeyFrom - uses AWS SDK credentials from environment/IAM
  baseConfig:
    model: "anthropic.claude-instant-v1"
  providerConfig:
    bedrockConfig:
      awsRegion: "us-west-2"  # Required: AWS region where Bedrock is available
```

### Mistral

```yaml
spec:
  provider: mistral
  apiKeyFrom:
    secretKeyRef:
      name: mistral
      key: MISTRAL_API_KEY
  baseConfig:
    model: "mistral-large-latest"
    temperature: "0.7"
    maxTokens: 1000
    topP: "0.95"
  providerConfig:
    mistralConfig:
      maxRetries: 3       # Optional: Number of retries for API calls
      timeout: 60         # Optional: Timeout in seconds
      randomSeed: 42      # Optional: Seed for deterministic sampling
```

### Cohere

```yaml
spec:
  provider: cohere
  apiKeyFrom:
    secretKeyRef:
      name: cohere
      key: COHERE_API_KEY
  baseConfig:
    model: "command"
    temperature: "0.7"
```

### Google AI

```yaml
spec:
  provider: google
  apiKeyFrom:
    secretKeyRef:
      name: google
      key: GOOGLE_API_KEY
  baseConfig:
    model: "gemini-pro"
  providerConfig:
    googleConfig:
      cloudProject: "my-gcp-project"  # Optional 
      cloudLocation: "us-central1"    # Optional
```

### Cloudflare

```yaml
spec:
  provider: cloudflare
  apiKeyFrom:
    secretKeyRef:
      name: cloudflare
      key: CLOUDFLARE_API_TOKEN
  baseConfig:
    model: "@cf/meta/llama-3-8b-instruct"
  providerConfig:
    cloudflareConfig:
      accountId: "abcdef123456"  # Required: Your Cloudflare account ID
```

## Credential Handling

Each provider has different credential requirements:

| Provider   | Credential Type      | Secret Key Reference                         |
|------------|----------------------|---------------------------------------------|
| OpenAI     | API Key              | `apiKeyFrom.secretKeyRef`                   |
| Anthropic  | API Key              | `apiKeyFrom.secretKeyRef`                   |
| Vertex     | Service Account JSON | `apiKeyFrom.secretKeyRef`                   |
| Bedrock    | AWS SDK Credentials  | Not required (uses AWS SDK/environment/IAM) |
| Mistral    | API Key              | `apiKeyFrom.secretKeyRef`                   |
| Cohere     | API Key              | `apiKeyFrom.secretKeyRef`                   |
| Google     | API Key              | `apiKeyFrom.secretKeyRef`                   |
| Cloudflare | API Token            | `apiKeyFrom.secretKeyRef`                   |

### Secret Examples

OpenAI/Anthropic/Mistral/Cohere/Google/Cloudflare:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: openai
type: Opaque
data:
  OPENAI_API_KEY: base64-encoded-api-key
```

Vertex AI:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: vertex-credentials
type: Opaque
data:
  service-account-json: base64-encoded-service-account-json
```

AWS Bedrock doesn't require a secret, as it uses AWS SDK credentials.