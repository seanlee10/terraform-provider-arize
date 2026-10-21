# Publishing to the Terraform Registry

Releases are built with GoReleaser in the layout the Terraform Registry requires: per-platform `.zip` archives, a `SHA256SUMS` file, its detached GPG signature (`SHA256SUMS.sig`), and `terraform-registry-manifest.json` (plugin protocol 6.0).

## Namespace

The registry namespace is the GitHub owner of the repository. `github.com/seanlee10/terraform-provider-arize` publishes as `seanlee10/arize`; publishing as `arize-ai/arize` (the address in `main.go` and the docs) requires the repository to live in the `arize-ai` GitHub organization. Either transfer the repository, or change the provider address in `main.go` and every `source = "arize-ai/arize"` to match.

## One-time setup

1. **Signing key.** Export the public half of the release signing key:

   ```bash
   gpg --armor --export <GPG_FINGERPRINT>
   ```

   Add it at https://registry.terraform.io → *Settings* → *Signing Keys* for the namespace.

2. **Publish the provider.** Go to https://registry.terraform.io/publish/provider, sign in with GitHub, and select the repository. The registry installs a webhook and ingests existing signed releases.

## Releasing a version

```bash
git tag v0.2.0
git push origin v0.2.0

export GITHUB_TOKEN=$(gh auth token)
export GPG_FINGERPRINT=<fingerprint of the signing key>
goreleaser release --clean
```

The registry picks up the new GitHub release through the webhook within a few minutes. Tags with a pre-release suffix (e.g. `v0.2.0-rc1`) are marked as pre-releases.

To check the release layout without publishing:

```bash
GPG_FINGERPRINT=<fingerprint> goreleaser release --snapshot --clean
```

## Notes

- v0.0.1 was released before the registry layout (tar.gz archives, no signature), so the registry won't ingest it; v0.1.0 is the first registry-compatible release.
- `bin/` and `dist/` are build output and are git-ignored.
