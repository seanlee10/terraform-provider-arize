# Publishing to Terraform Registry

Your Arize Terraform provider has been built and released to GitHub! Here's the final step to make it publicly available on the Terraform Registry.

## Status

✅ **Complete:**
- Provider code written and tested
- Binaries built for all platforms (Linux, macOS, Windows)
- Release published to GitHub: https://github.com/seanlee10/terraform-provider-arize/releases/tag/v0.0.1
- Binaries available for download: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64, windows/arm64

## Next Step: Register with Terraform Registry

The Terraform Registry automatically discovers providers published on GitHub. To register your provider:

### 1. Verify Repository Configuration

Ensure your GitHub repository has:

- ✅ **Repository Description**: "Terraform provider for Arize"
- ✅ **Repository Topics**: Add `terraform`, `terraform-provider`, `provider`
- ✅ **README.md**: Already present with documentation
- ✅ **License**: (Optional but recommended) Add MIT or Apache 2.0 license
- ✅ **GitHub Releases**: v0.0.1 published with binaries

### 2. Visit Terraform Registry Publisher Dashboard

Go to: https://registry.terraform.io/publish/provider

You'll be asked to:
1. Sign in with your GitHub account
2. Authorize Terraform Registry to access your repositories

### 3. Claim Your Namespace

Search for your repository: `seanlee10/terraform-provider-arize`

Select it to claim the namespace `arize-ai/arize`

Terraform will automatically:
- Crawl your GitHub releases
- Download and verify binaries
- Create provider pages for each version
- Make it available at: `registry.terraform.io/providers/arize-ai/arize/latest`

### 4. Verify Publication (takes 5-10 minutes)

Once registered, users can use:

```hcl
terraform {
  required_providers {
    arize = {
      source  = "arize-ai/arize"
      version = "0.0.1"
    }
  }
}

provider "arize" {
  api_key = var.arize_api_key
}
```

And `terraform init` will automatically download your provider!

## What Happens Automatically

Once you register, Terraform Registry will:

1. **Monitor** your GitHub releases for new versions
2. **Extract** provider binaries from releases
3. **Verify** checksums match
4. **Index** the provider for search
5. **Generate** documentation from your code
6. **Host** downloadable packages

## Current Release

**Version:** 0.0.1  
**GitHub Release:** https://github.com/seanlee10/terraform-provider-arize/releases/tag/v0.0.1

**Available Binaries:**
- `terraform-provider-arize_0.0.1_linux_amd64.tar.gz`
- `terraform-provider-arize_0.0.1_linux_arm64.tar.gz`
- `terraform-provider-arize_0.0.1_darwin_amd64.tar.gz`
- `terraform-provider-arize_0.0.1_darwin_arm64.tar.gz`
- `terraform-provider-arize_0.0.1_windows_amd64.tar.gz`
- `terraform-provider-arize_0.0.1_windows_arm64.tar.gz`
- `terraform-provider-arize_0.0.1_SHA256SUMS`

## Future Releases

To release a new version:

```bash
# Update version in your code (if needed)
# Commit changes
git add .
git commit -m "version: bump to 0.0.2"

# Create and push tag
git tag v0.0.2
git push origin v0.0.2

# GoReleaser will automatically create the release
# (With GITHUB_TOKEN set)
export GITHUB_TOKEN=$(gh auth token)
goreleaser release --clean
```

The Terraform Registry will automatically detect and publish the new version within minutes.

## Testing Your Published Provider

Once registered, test it locally:

```bash
# Create a test directory
mkdir test-provider
cd test-provider

# Create main.tf
cat > main.tf << 'EOF'
terraform {
  required_providers {
    arize = {
      source  = "arize-ai/arize"
      version = "0.0.1"
    }
  }
}

provider "arize" {}
EOF

# Initialize - Terraform will download from registry
terraform init

# Should see:
# - Downloading registry.terraform.io/arize-ai/arize v0.0.1
# - Installed arize-ai/arize v0.0.1
```

## Documentation

Your provider documentation will be automatically generated from:
- **Code comments** in resource schemas
- **Examples** in your repository
- **README.md** documentation

You can view it at: https://registry.terraform.io/providers/arize-ai/arize/latest

## Summary

Your provider is ready for public use! The only remaining step is to claim your namespace on the Terraform Registry. After that, users worldwide can use:

```hcl
terraform {
  required_providers {
    arize = {
      source = "arize-ai/arize"
    }
  }
}
```

And it will work automatically via `terraform init`! 🚀
