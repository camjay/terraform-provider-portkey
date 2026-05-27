terraform {
  required_providers {
    portkey = {
      source = "portkey-ai/portkey"
    }
  }
}

# ----------------------------------------------------------------------------
# Classic integration: provider API key stored directly in Terraform state.
# ----------------------------------------------------------------------------
resource "portkey_integration" "openai" {
  name           = "openai-production"
  ai_provider_id = "openai"
  key            = var.openai_api_key
  description    = "Production OpenAI integration"
}

# ----------------------------------------------------------------------------
# Secret-reference-backed integration.
#
# Instead of sending the provider key inline, resolve it at request time from
# a portkey_secret_reference that points at AWS Secrets Manager / Azure Key
# Vault / HashiCorp Vault. Nothing sensitive ever lives in Terraform state.
# ----------------------------------------------------------------------------
resource "portkey_secret_reference" "openai_key" {
  name         = "openai-prod-key"
  manager_type = "aws_sm"
  secret_path  = "portkey/openai/prod"
  secret_key   = "OPENAI_API_KEY"

  aws_access_key_auth = {
    aws_access_key_id     = var.aws_access_key_id
    aws_secret_access_key = var.aws_secret_access_key
    aws_region            = "us-east-1"
  }
}

resource "portkey_integration" "openai_from_secret_ref" {
  name           = "openai-production-sr"
  ai_provider_id = "openai"
  # `key` / `key_wo` intentionally omitted - the mapping below supplies it.

  secret_mappings = [
    {
      target_field        = "key"
      secret_reference_id = portkey_secret_reference.openai_key.slug
    },
  ]
}

# ----------------------------------------------------------------------------
# Composite example: some configuration fields inline, a sensitive field
# resolved from a secret reference. Example target is AWS Bedrock, where
# aws_secret_access_key is the sensitive field we want to keep out of state.
# ----------------------------------------------------------------------------
resource "portkey_integration" "bedrock_hybrid" {
  name           = "bedrock-production"
  ai_provider_id = "bedrock"

  configurations = jsonencode({
    aws_auth_type     = "accessKey"
    aws_region        = "us-east-1"
    aws_access_key_id = var.aws_access_key_id
    # aws_secret_access_key is supplied by the mapping below.
  })

  secret_mappings = [
    {
      target_field        = "configurations.aws_secret_access_key"
      secret_reference_id = portkey_secret_reference.openai_key.slug
      secret_key          = "AWS_SECRET_ACCESS_KEY" # override for multi-value secrets
    },
  ]
}

variable "openai_api_key" {
  type      = string
  sensitive = true
}

variable "aws_access_key_id" {
  type      = string
  sensitive = true
}

variable "aws_secret_access_key" {
  type      = string
  sensitive = true
}

# ----------------------------------------------------------------------------
# Write-only configurations.
#
# When the caller's pipeline already has access to a secret manager (e.g.
# HashiCorp Vault via TFC dynamic credentials, Doppler/Infisical via their
# Terraform integrations), `configurations_wo` lets the resolved values flow
# into Portkey without ever landing in Terraform state. This is complementary
# to `portkey_secret_reference` — use that when Portkey itself should resolve
# the secret at request time; use `configurations_wo` when the secret is
# resolved at apply time and you just want to keep it out of state.
#
# Mutually exclusive with `configurations`. Increment `configurations_version`
# to trigger an update.
# ----------------------------------------------------------------------------

# Example: ephemeral Vault read via TFC dynamic credentials, fed into a Vertex
# AI integration's `vertex_service_account_json` field without state storage.
ephemeral "vault_kv_secret_v2" "vertex" {
  mount = "secret"
  name  = "portkey/integrations/vertex-prod"
}

resource "portkey_integration" "vertex_write_only_configurations" {
  name           = "vertex-production-wo"
  ai_provider_id = "vertex-ai"
  key_wo         = "unused-placeholder" # vertex auths via configurations
  key_version    = 1

  configurations_wo = jsonencode({
    vertex_auth_type            = "serviceAccount"
    vertex_region               = "us-central1"
    vertex_project_id           = ephemeral.vault_kv_secret_v2.vertex.data["vertex_project_id"]
    vertex_service_account_json = ephemeral.vault_kv_secret_v2.vertex.data["vertex_service_account_json"]
  })

  # Bump configurations_version to push a refreshed value through to Portkey.
  configurations_version = 1
}
