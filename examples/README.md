# Arize Terraform Provider Examples

Each example is a self-contained Terraform configuration in its own directory, with its own `terraform` block (`seanlee10/arize`, `~> 0.3.6`) and variables. Run one example at a time from its directory.

| Directory | What it shows |
|-----------|---------------|
| [`01_basic_setup/`](01_basic_setup/) | A space and an existing user added to it |
| [`02_team_management/`](02_team_management/) | Inviting a team and giving each member a space role |
| [`03_datasets/`](03_datasets/) | A dataset kept in sync with a CSV file, applied from CI |
| [`04_multi_environment/`](04_multi_environment/) | Dev/staging/prod from one configuration, with a production safety check |
| [`05_importing_existing_resources/`](05_importing_existing_resources/) | Data sources, `terraform import`, and mixing imported with new resources |
| [`06_saml_sso_setup/`](06_saml_sso_setup/) | SAML IdP with role mappings to org and space roles |
| [`07_saml_custom_roles/`](07_saml_custom_roles/) | Custom roles granted per space through SAML |
| [`08_using_existing_roles/`](08_using_existing_roles/) | Looking up roles created in the Arize UI by name |
| [`09_evaluators/`](09_evaluators/) | LLM-as-judge and Python code evaluators |

## Quick Start

```bash
export ARIZE_API_KEY="your-api-key-here"

cd examples/01_basic_setup
terraform init
terraform plan -var="user_email=alice@company.com"
terraform apply -var="user_email=alice@company.com"
```

EU accounts: add `endpoint = "https://app.eu-west-1a.arize.com/graphql"` to the example's `provider "arize"` block.

**Users and spaces:** Arize adds a user to a space only after they accept their invitation. Adding a user whose `arize_user` is still a pending invitation fails with "User hasn't accepted the invitation yet". The examples therefore add existing users to spaces, looked up with the `arize_users` data source, or (example 2) invite users first and add them to spaces on a later apply, once they have accepted.

---

### 1. Basic Setup (`01_basic_setup/`)
**What it does:** Creates a space and adds an existing, active user to it as `member`.

**Key concepts:**
- Creating a space
- Looking up a user by email with the `arize_users` data source
- A `precondition` that fails clearly when no active user has that email

**To run:**
```bash
cd 01_basic_setup
terraform init
terraform apply -var="user_email=alice@company.com"
```

---

### 2. Team Management (`02_team_management/`)
**What it does:** Invites a team (`arize_user`) and adds each member to spaces with their role.

**Key concepts:**
- Creating multiple users with `for_each`
- Space roles: `admin`, `member`, `readOnly`, `annotator`
- Adding only users who have accepted their invitation; the `pending_invitations` output lists the rest. Rerun `terraform apply` after they accept.
- Someone who already has an Arize account can't be invited again: import their `arize_user` instead

**To run:**
```bash
cd 02_team_management
terraform init
terraform apply      # sends the invitations
terraform apply      # later: adds members who have accepted to the spaces
```

---

### 3. Datasets from CI (`03_datasets/`)
**What it does:** Syncs the examples in `datasets/basic_quiz.csv` to an Arize dataset. Run it from CI on every merge.

**Key concepts:**
- `arize_dataset` with `key_column` and `examples` built from `csvdecode`
- Changed rows are updated, new rows added, removed rows deleted, all in the latest dataset version
- A sample GitHub Actions step (in the file's comments)

**To run:**
```bash
cd 03_datasets
terraform init
terraform apply -var="space_id=<space_id>"
```

---

### 4. Multi-Environment Setup (`04_multi_environment/`)
**What it does:** Creates an environment's space (and a backup space for staging and prod) and adds existing users to it with an environment-specific role.

**Key concepts:**
- Environment-specific settings in `locals`
- Conditional resources (`count`)
- A variable `validation` that refuses `environment=prod` unless `enable_production=true` (needs Terraform 1.9+)
- One state per environment (workspaces or a backend key per environment)

**To run:**
```bash
cd 04_multi_environment
terraform init
terraform apply -var="environment=dev"
terraform apply -var="environment=staging" -var='team_emails=["alice@company.com"]'
terraform apply -var="environment=prod" -var="enable_production=true"
```

Or copy `prod.tfvars.example` (or `dev.tfvars.example`) to `prod.tfvars`, edit it, and run `terraform apply -var-file=prod.tfvars`. Keep `*.tfvars` out of git if they contain anything sensitive.

---

### 5. Importing Existing Resources (`05_importing_existing_resources/`)
**What it does:** Lists existing users and roles, imports existing resources, and adds an existing user to a new space.

**Key concepts:**
- Data sources (`arize_users`, `arize_roles`) and filtering their results
- Import IDs for every resource
- `import` blocks (Terraform 1.5+)

**To run:**
```bash
cd 05_importing_existing_resources
terraform init

terraform import arize_user.imported_user "<user_id>"
terraform import arize_space.imported_space "<space_id>"
terraform import arize_space_member.imported_member "<space_id>:<user_id>"

terraform plan   # should show no changes for the imported resources
terraform apply
```

---

### 6. SAML/SSO Setup (`06_saml_sso_setup/`)
**What it does:** Configures a SAML IdP whose role mappings put IdP groups into spaces with built-in roles. Users are created on their first SAML login, so no `arize_user` resources are needed.

**Key concepts:**
- `metadata_url` or `metadata_xml`
- `email_domains_list`
- `role_mappings` to org and space roles; optional `default_*` roles for users matching no mapping
- Why `terraform destroy` leaves the IdP in Arize

**To run:**
```bash
cd 06_saml_sso_setup
terraform init
terraform apply -var="org_id=<org_id>"
```

---

### 7. SAML with Custom Roles (`07_saml_custom_roles/`)
**What it does:** Creates custom roles with `arize_role` and grants them per space to IdP groups.

**Key concepts:**
- Custom roles and permissions
- `space_custom_roles` in `role_mappings`
- Built-in organization roles combined with custom space roles

**To run:**
```bash
cd 07_saml_custom_roles
terraform init
terraform apply -var="org_id=<org_id>"
```

---

### 8. Using Existing Roles (`08_using_existing_roles/`)
**What it does:** Reads roles created in the Arize UI and references one by name in a SAML role mapping.

**Key concepts:**
- The `arize_roles` data source and a name-to-ID map
- `lookup(..., null)` with a `precondition`, so a missing role fails with a clear message
- Filtering roles by permission or `is_predefined`

**To run:**
```bash
cd 08_using_existing_roles
terraform init
terraform apply -var="org_id=<org_id>" -var="data_scientist_role_name=data-scientist"
```

---

### 9. Evaluators (`09_evaluators/`)
**What it does:** Creates LLM-as-judge and Python code evaluators and lists the evaluators already in the space.

**Key concepts:**
- `template` evaluators with classification choices and a judge model (`llm.integration_name` or `llm.integration_id`)
- `code` evaluators (Arize derives `span_attributes` from the `evaluate()` signature)
- Span, trace, or session granularity
- Config changes publish new evaluator versions with a commit message
- Importing with `<space_id>/<evaluator_id>`

**To run:**
```bash
cd 09_evaluators
terraform init
terraform apply -var="space_id=<space_id>" -var="llm_integration_name=<integration name>"
```

---

## Debugging

```bash
terraform state list                     # resources in state
terraform state show arize_space.main    # one resource
terraform plan -out=plan.tfplan && terraform show plan.tfplan
TF_LOG=DEBUG terraform plan              # provider debug logging
terraform validate && terraform fmt -check
```

## Cleanup

```bash
terraform destroy
```

Destroying an `arize_user` deactivates the user in Arize (or cancels a pending invitation). Destroying an `arize_saml_idp` only removes it from state.

## Troubleshooting

**"User hasn't accepted the invitation yet"**
- Arize adds users to spaces only after they accept. Wait for them to accept, then run `terraform apply` again.

**"Not Authorized" when creating a space**
- The API key's user can't create spaces in that organization. The error lists the user's role in each organization; account admins and organization `admin` or `member` users can create spaces.

**401 Unauthorized**
- Check `ARIZE_API_KEY`. EU accounts must also set `endpoint = "https://app.eu-west-1a.arize.com/graphql"`.

---

For more, see the [comprehensive guide](../docs/GUIDE.md) and the provider [README](../README.md).
