# GLM-5.3 built-in model

## Objective

Keep the knowledge repository's built-in chat model and smart-reasoning Agent aligned with the full GLM-5.3 flagship model.

## Implementation steps

- [x] Confirm the existing Zhipu provider supports the official GLM endpoint.
- [x] Replace the built-in DeepSeek chat model with GLM-5.3.
- [x] Point the built-in smart-reasoning Agent to the new model ID.
- [x] Run configuration tests and `ai-code-check`.

## Affected areas

- Built-in knowledge model configuration.
- Built-in smart-reasoning Agent configuration.

## Verification

Run the built-in configuration tests, Go tests, formatting checks, and inspect the reconciled online model after deployment.

## Progress

Configuration is updated. Relevant Go tests, Go vet, exact built-in model-reference validation, and `ai-code-check` passed.

## Final outcome

In progress.
