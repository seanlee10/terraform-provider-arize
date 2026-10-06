# Arize Terraform Provider Examples

This directory contains practical examples demonstrating how to use the Arize Terraform provider to manage your Arize account infrastructure as code.

## Examples Overview

### 1. Basic Setup (`01_basic_setup.tf`)
**What it does:** Creates a simple user and space, then adds the user to the space.

**Use case:** Getting started with the provider, learning the basics.

**Key concepts:**
- Creating a user
- Creating a space
- Adding users to spaces
- Outputs

**To run:**
```bash
cd examples
terraform init
terraform plan
terraform apply
```

---

### 2. Team Management (`02_team_management.tf`)
**What it does:** Manages a team with multiple members, different roles, and multiple spaces.

**Use case:** Setting up a collaborative environment for your ML team with role-based access control.

**Key concepts:**
- Creating multiple users (using `for_each`)
- Different space roles: admin, member, readOnly, annotator
- Managing team access to multiple spaces
- Using locals for team configuration

**To run:**
```bash
terraform plan -var-file=02_team_management.tfvars
terraform apply -var-file=02_team_management.tfvars
```

**Key output:** Summary of all team members, their roles, and assigned spaces.

---

### 3. CI/CD & Automation (`03_cicd_automation.tf`)
**What it does:** Creates API keys for different services with appropriate permissions.

**Use case:** Setting up service accounts for automated workflows (data pipelines, monitoring, exports).

**Key concepts:**
- Creating API keys with specific permissions
- Managing sensitive outputs
- Querying existing API keys
- Identifying keys that need rotation
- Integration with secret managers

**To run:**
```bash
terraform plan
terraform apply
```

**Important:** The actual API keys are marked as sensitive. Capture them from the output and store in your secret manager (AWS Secrets Manager, HashiCorp Vault, etc.).

**Key output:**
- Service API keys (IDs only)
- Keys that need rotation
- All API keys in the account

---

### 4. Multi-Environment Setup (`04_multi_environment.tf`)
**What it does:** Manages development, staging, and production environments with consistent, environment-specific configuration.

**Use case:** Managing Arize resources across multiple environments with different security levels and team sizes.

**Key concepts:**
- Environment-specific variables
- Conditional resource creation
- Safety checks (prevent accidental production changes)
- Reusable configuration with locals
- Remote state backend setup

**To run:**

Development:
```bash
terraform plan -var="environment=dev"
terraform apply -var="environment=dev"
```

Staging:
```bash
terraform plan -var="environment=staging" -var="team_size=4"
terraform apply -var="environment=staging" -var="team_size=4"
```

Production (requires explicit flag):
```bash
terraform plan -var="environment=prod" -var="enable_production_space=true"
terraform apply -var="environment=prod" -var="enable_production_space=true"
```

**Key output:** Environment-specific information (space IDs, team members, service accounts).

---

### 5. Importing Existing Resources (`05_importing_existing_resources.tf`)
**What it does:** Demonstrates how to import existing Arize resources into Terraform state and then manage them alongside new resources.

**Use case:** Adopting Terraform for an account that already has Arize resources created manually.

---

### 6. SAML/SSO Setup (`06_saml_sso_setup.tf`)
**What it does:** Configures SAML Identity Provider and sets up automatic user provisioning with role mappings.

**Use case:** Enterprise organizations using Okta, Azure AD, Google Workspace, or other SAML providers for centralized identity management.

**Key concepts:**
- Configuring SAML metadata URL or raw XML
- Setting allowed email domains
- Enabling automatic user provisioning
- Default role assignment
- Team space assignments based on SAML groups
- Centralized identity management

**To run:**
```bash
terraform plan
terraform apply
```

**Key output:** SAML IdP ID and space IDs for team collaboration

---

### 7. SAML with Custom Roles (`07_saml_custom_roles.tf`)
**What it does:** Combines custom roles with SAML IdP configuration to implement fine-grained permission control.

**Use case:** Organizations that need custom permission sets beyond the default roles (admin, member, readOnly, annotator).

**Key concepts:**
- Creating custom roles with `arize_role`
- Using custom roles as default for SAML users
- Combining organization roles with space roles
- Multi-team setup with different custom roles
- Permission inheritance and role precedence

**To run:**
```bash
terraform plan
terraform apply
```

**Key output:** Custom role IDs and team structure with role assignments

**Key concepts:**
- Using data sources to query existing resources
- Filtering existing data
- Importing resources into Terraform state
- Managing mix of imported and new resources

**To run:**

First, list existing resources:
```bash
terraform plan
```

Then import specific resources:
```bash
# Import a user
terraform import arize_user.imported_user "user-id-from-arize"

# Import a space
terraform import arize_space.imported_space "space-id-from-arize"
```

Finally, add configuration and manage:
```bash
terraform plan  # Should show no changes for imported resources
terraform apply
```

**Key output:** Analysis of existing users, spaces, and API keys.

---

## Quick Start

1. **Set up your Arize API key:**
   ```bash
   export ARIZE_API_KEY="your-api-key-here"
   ```

2. **Choose an example to run:**
   ```bash
   cd examples
   terraform init
   ```

3. **Preview the changes:**
   ```bash
   terraform plan
   ```

4. **Apply the configuration:**
   ```bash
   terraform apply
   ```

---

## Common Workflows

### Add a team member to production space

Edit `02_team_management.tf` and add to the `team` local:
```hcl
locals {
  team = {
    # ... existing members ...
    new_person = {
      email = "newperson@company.com"
      name  = "New Person"
      role  = "member"
    }
  }
}
```

Then apply:
```bash
terraform apply
```

### Rotate API keys

1. Create a new API key in `03_cicd_automation.tf`
2. Capture the new key from outputs
3. Update your services with the new key
4. Delete the old key resource from config
5. Run `terraform apply`

### Move a user between spaces

Edit space member roles or use `terraform state mv` to reorganize.

---

## Debugging

### View Terraform state for a resource

```bash
terraform state show arize_user.example
```

### List all resources in state

```bash
terraform state list
```

### See detailed output

```bash
terraform plan -var-file=01_basic_setup.tfvars -out=plan.tfplan
terraform show plan.tfplan
```

### Enable debug logging

```bash
TF_LOG=DEBUG terraform plan
```

---

## Cleanup

Remove resources created by an example:

```bash
terraform destroy
```

Or remove specific resources:

```bash
terraform destroy -target=arize_user.example_user
```

---

## Next Steps

- Read the [comprehensive guide](../docs/GUIDE.md) for more advanced patterns
- Check [CLAUDE.md](../CLAUDE.md) for architecture details
- Review the provider [README](../README.md) for complete resource documentation
- Explore [Terraform best practices](https://www.terraform.io/docs/cloud/guides/recommended-practices)

---

## Tips & Tricks

### Use tfvars files for reusable configurations

Create `prod.tfvars`:
```hcl
environment = "prod"
team_size = 5
enable_production_space = true
```

Then apply with:
```bash
terraform apply -var-file=prod.tfvars
```

### Store API keys securely

Never commit actual API keys. Use:
```bash
terraform output api_keys_secret_export | jq . | \
  aws secretsmanager create-secret --name arize/prod/api-keys --secret-string file:///dev/stdin
```

### Export Terraform state for backup

```bash
terraform state pull > terraform.tfstate.backup
```

### Validate configuration before applying

```bash
terraform validate
terraform fmt -check
```

---

## Troubleshooting

**"Space already exists in account"**
- Use a unique space name or import the existing space

**"User not found" when importing**
- Verify the user ID is correct
- Check that the user exists in your Arize account

**"API key is sensitive - cannot output"**
- The key is hidden for security. Capture it from the Terraform output immediately after creation.

---

For more help, see the [comprehensive guide](../docs/GUIDE.md) or [CLAUDE.md](../CLAUDE.md).
