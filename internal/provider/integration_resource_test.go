package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIntegrationResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccIntegrationResourceConfig(rName, "openai", "Initial description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttrSet("portkey_integration.test", "slug"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "description", "Initial description"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
					resource.TestCheckResourceAttr("portkey_integration.test", "allow_all_models", "true"),
					resource.TestCheckResourceAttrSet("portkey_integration.test", "created_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "portkey_integration.test",
				ImportState:       true,
				ImportStateVerify: true,
				// key, key_wo, configurations_wo: write-only, never returned by API
				// key_version, configurations_version: not stored on server, local triggers only
				// configurations: not returned by API
				// slug: BUG - API returns UUID instead of original slug on GET (needs investigation)
				// updated_at: timestamp may change between operations
				ImportStateVerifyIgnore: []string{"key", "key_wo", "key_version", "configurations", "configurations_wo", "configurations_version", "slug", "updated_at"},
			},
			// Update testing
			{
				Config: testAccIntegrationResourceConfig(rName+"-updated", "openai", "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_integration.test", "description", "Updated description"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccIntegrationResource_withSlug(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	slug := acctest.RandomWithPrefix("tf-slug")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithSlug(rName, slug, "openai"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "slug", slug),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "openai"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_updateName(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-rename")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfig(rName, "openai", "Initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
				),
			},
			{
				Config: testAccIntegrationResourceConfig(rName+"-renamed", "openai", "Initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-renamed"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_withConfigurations(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-config")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithConfigurations(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "bedrock"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfig(name, aiProviderID, description string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = %[2]q
  description    = %[3]q
  key            = "sk-test-fake-key-12345"
}
`, name, aiProviderID, description)
}

func testAccIntegrationResourceConfigWithSlug(name, slug, aiProviderID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  slug           = %[2]q
  ai_provider_id = %[3]q
  key            = "sk-test-fake-key-12345"
}
`, name, slug, aiProviderID)
}

func testAccIntegrationResourceConfigWithConfigurations(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "bedrock"

  configurations = jsonencode({
    aws_auth_type = "assumedRole"
    aws_role_arn  = "arn:aws:iam::123456789012:role/TestRole"
    aws_region    = "us-east-1"
  })
}
`, name)
}

// Tests for write-only key (key_wo) and key_version trigger

func TestAccIntegrationResource_withWriteOnlyKey(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wo-key")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with key_wo and key_version
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKey(rName, "sk-test-key-1", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "key_version", "1"),
				),
			},
			// Update key_version to trigger key update
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKey(rName, "sk-test-key-2", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "key_version", "2"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_keyVersionNoChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-no-key-change")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKey(rName, "sk-test-key", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "key_version", "1"),
				),
			},
			// Update name but not key_version - key should NOT be sent
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKey(rName+"-updated", "sk-test-key", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_integration.test", "key_version", "1"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_withOpenAIConfigurations(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-openai-config")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithOpenAI(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "openai"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithWriteOnlyKey(name, key string, keyVersion int) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key_wo         = %[2]q
  key_version    = %[3]d
}
`, name, key, keyVersion)
}

func testAccIntegrationResourceConfigWithOpenAI(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key_wo         = "sk-test-fake-key-12345"
  key_version    = 1

  configurations = jsonencode({
    openai_organization = "org-test123"
    openai_project      = "proj-test456"
  })
}
`, name)
}

func TestAccIntegrationResource_withAzureOpenAIConfigurations(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithAzureOpenAI(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithAzureOpenAI(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"
  key            = "test-azure-api-key-12345"

  configurations = jsonencode({
    azure_auth_mode     = "default"
    azure_resource_name = "test-azure-resource"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
  })
}
`, name)
}

func TestAccIntegrationResource_withAzureOpenAIMultipleDeployments(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure-multi")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithAzureOpenAIMultiple(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithAzureOpenAIMultiple(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"
  key            = "test-azure-api-key-12345"

  configurations = jsonencode({
    azure_auth_mode     = "default"
    azure_resource_name = "test-azure-resource"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      },
      {
        alias                 = "gpt35"
        azure_deployment_name = "gpt-35-turbo-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-35-turbo"
      }
    ]
  })
}
`, name)
}

func TestAccIntegrationResource_withAzureOpenAIEntraAuth(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure-entra")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithAzureOpenAIEntra(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithAzureOpenAIEntra(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"

  configurations = jsonencode({
    azure_auth_mode           = "entra"
    azure_resource_name       = "test-azure-resource"
    azure_entra_tenant_id     = "test-tenant-id-12345"
    azure_entra_client_id     = "test-client-id-12345"
    azure_entra_client_secret = "test-client-secret-12345"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
  })
}
`, name)
}

func TestAccIntegrationResource_withAzureOpenAIManagedAuth(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure-managed")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithAzureOpenAIManaged(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "status", "active"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithAzureOpenAIManaged(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"

  configurations = jsonencode({
    azure_auth_mode         = "managed"
    azure_resource_name     = "test-azure-resource"
    azure_managed_client_id = "test-managed-client-id-12345"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
  })
}
`, name)
}

// Test updating Azure OpenAI configuration (add a deployment)
func TestAccIntegrationResource_updateAzureOpenAIConfig(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-azure-update")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with single deployment
			{
				Config: testAccIntegrationResourceConfigAzureOpenAISingle(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
				),
			},
			// Update: add another deployment
			{
				Config: testAccIntegrationResourceConfigAzureOpenAIUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "azure-openai"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigAzureOpenAISingle(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "azure-openai"
  key            = "test-azure-api-key-12345"

  configurations = jsonencode({
    azure_auth_mode     = "default"
    azure_resource_name = "test-azure-resource"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      }
    ]
  })
}
`, name)
}

func testAccIntegrationResourceConfigAzureOpenAIUpdated(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = "%[1]s-updated"
  ai_provider_id = "azure-openai"
  key            = "test-azure-api-key-12345"

  configurations = jsonencode({
    azure_auth_mode     = "default"
    azure_resource_name = "test-azure-resource-updated"
    azure_deployment_config = [
      {
        azure_deployment_name = "gpt-4-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-4"
        is_default            = true
      },
      {
        alias                 = "gpt35"
        azure_deployment_name = "gpt-35-turbo-deployment"
        azure_api_version     = "2024-02-15-preview"
        azure_model_slug      = "gpt-35-turbo"
      }
    ]
  })
}
`, name)
}

func TestAccIntegrationResource_conflictKeyAndKeyWO(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-conflict")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccIntegrationResourceConfigConflict(rName),
				ExpectError: regexp.MustCompile(`Conflicting API Key Attributes`),
			},
		},
	})
}

func testAccIntegrationResourceConfigConflict(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key            = "sk-deprecated-key"
  key_wo         = "sk-write-only-key"
  key_version    = 1
}
`, name)
}

// Test key_wo without key_version - should create successfully with warning
func TestAccIntegrationResource_withWriteOnlyKeyNoVersion(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wo-no-ver")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with key_wo but NO key_version - should work (with warning)
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKeyNoVersion(rName, "sk-test-key-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckNoResourceAttr("portkey_integration.test", "key_version"),
				),
			},
			// Update name only - key should NOT be sent (key_version is null and unchanged)
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyKeyNoVersion(rName+"-updated", "sk-test-key-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-updated"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithWriteOnlyKeyNoVersion(name, key string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key_wo         = %[2]q
}
`, name, key)
}

func TestAccIntegrationResource_allowAllModelsFalse(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-no-models")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with allow_all_models = false
			{
				Config: testAccIntegrationResourceConfigAllowAllModels(rName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "allow_all_models", "false"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_allowAllModelsUpdate(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-models-update")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with allow_all_models = true (default)
			{
				Config: testAccIntegrationResourceConfigAllowAllModels(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "allow_all_models", "true"),
				),
			},
			// Update to allow_all_models = false
			{
				Config: testAccIntegrationResourceConfigAllowAllModels(rName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "allow_all_models", "false"),
				),
			},
			// Update back to allow_all_models = true
			{
				Config: testAccIntegrationResourceConfigAllowAllModels(rName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "allow_all_models", "true"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigAllowAllModels(name string, allowAllModels bool) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name             = %[1]q
  ai_provider_id   = "openai"
  key              = "sk-test-fake-key-12345"
  allow_all_models = %[2]t
}
`, name, allowAllModels)
}

func TestAccIntegrationResource_workspaceScoped(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	workspaceID := testAccGetEnvOrSkip(t, "TEST_WORKSPACE_ID")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIntegrationResourceConfigWithWorkspace(rName, workspaceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "ai_provider_id", "openai"),
					resource.TestCheckResourceAttr("portkey_integration.test", "type", "workspace"),
					resource.TestCheckResourceAttrSet("portkey_integration.test", "workspace_id"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithWorkspace(name, workspaceID string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key            = "sk-test-placeholder"
  workspace_id   = %[2]q
}
`, name, workspaceID)
}

// Tests for write-only configurations (configurations_wo) and configurations_version trigger
//
// configurations_wo mirrors key_wo: it accepts a JSON string that is written
// to Portkey but never stored in Terraform state. Use it when callers want
// to source sensitive configuration fields from external secret pipelines
// (e.g. ephemeral Vault reads via TFC dynamic credentials, Doppler/Infisical
// TF integrations) without those values landing in TF state.

func TestAccIntegrationResource_withWriteOnlyConfigurations(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wo-cfg")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with configurations_wo and configurations_version
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyConfigurations(rName, "org-test-1", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "configurations_version", "1"),
					// configurations_wo never appears in state
					resource.TestCheckNoResourceAttr("portkey_integration.test", "configurations_wo"),
				),
			},
			// Update configurations_version to trigger configurations update
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyConfigurations(rName, "org-test-2", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					resource.TestCheckResourceAttr("portkey_integration.test", "configurations_version", "2"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_configurationsVersionNoChange(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-no-cfg-change")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyConfigurations(rName, "org-test", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "configurations_version", "1"),
				),
			},
			// Update name but not configurations_version - configurations should NOT be re-sent
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyConfigurations(rName+"-updated", "org-test", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName+"-updated"),
					resource.TestCheckResourceAttr("portkey_integration.test", "configurations_version", "1"),
				),
			},
		},
	})
}

func TestAccIntegrationResource_conflictConfigurationsAndConfigurationsWO(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-conflict-cfg")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccIntegrationResourceConfigConflictConfigurations(rName),
				ExpectError: regexp.MustCompile(`Conflicting Configurations Attributes`),
			},
		},
	})
}

// Test configurations_wo without configurations_version - should create successfully with warning
func TestAccIntegrationResource_withWriteOnlyConfigurationsNoVersion(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-wo-cfg-no-ver")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with configurations_wo but NO configurations_version - should work (with warning)
			{
				Config: testAccIntegrationResourceConfigWithWriteOnlyConfigurationsNoVersion(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("portkey_integration.test", "id"),
					resource.TestCheckResourceAttr("portkey_integration.test", "name", rName),
					// configurations_version should remain null
					resource.TestCheckNoResourceAttr("portkey_integration.test", "configurations_version"),
				),
			},
		},
	})
}

func testAccIntegrationResourceConfigWithWriteOnlyConfigurations(name, org string, configVersion int) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key_wo         = "sk-test-fake-key-12345"
  key_version    = 1

  configurations_wo = jsonencode({
    openai_organization = %[2]q
    openai_project      = "proj-test"
  })
  configurations_version = %[3]d
}
`, name, org, configVersion)
}

func testAccIntegrationResourceConfigConflictConfigurations(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key_wo         = "sk-test-fake-key-12345"
  key_version    = 1

  configurations    = jsonencode({ openai_organization = "org-A" })
  configurations_wo = jsonencode({ openai_organization = "org-B" })
}
`, name)
}

func testAccIntegrationResourceConfigWithWriteOnlyConfigurationsNoVersion(name string) string {
	return fmt.Sprintf(`
provider "portkey" {}

resource "portkey_integration" "test" {
  name           = %[1]q
  ai_provider_id = "openai"
  key_wo         = "sk-test-fake-key-12345"
  key_version    = 1

  configurations_wo = jsonencode({
    openai_organization = "org-test"
    openai_project      = "proj-test"
  })
}
`, name)
}
