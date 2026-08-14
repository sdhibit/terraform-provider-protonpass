---
page_title: "pass-cli Compatibility"
description: |-
  Tested and recommended versions of pass-cli for use with the protonpass
  provider, plus environment variables for CI/CD authentication.
---

# pass-cli Compatibility

The `protonpass` provider delegates all Proton Pass operations to the
`pass-cli` binary. This guide documents which versions have been tested,
which are recommended, and how to authenticate in CI environments.

## Version Matrix

| Version | Status | Notes |
|---|---|---|
| **v1.5.2** | Minimum supported | Oldest version confirmed to work with this provider |
| **v1.6.1 – v1.10.0** | Supported, untested | No breaking changes expected but not verified |
| **v2.0.0 – v2.0.2** | Supported, untested | First stable 2.x series; introduces PAT authentication |
| **v2.0.3 – v2.2.3** | Supported, untested | `item list` output changed here — see below |
| **v2.2.4 – v2.3.2** | Recommended | `pass-cli test` removed here — see below |
| **v2.3.2** | Tested locally | Verified against a live session, including item create round-trips |

> **Tested** means a real `pass-cli` session was used to exercise the
> provider. **Supported, untested** means the provider handles that
> version's behaviour but no live session was run against it.

Provider **v1.3.0 and later** are required for `pass-cli` v2.0.3 and later.
Earlier provider releases break on those CLIs; see the two sections below.

### Breaking change: `pass-cli test` removed in v2.2.4

`pass-cli test` was removed in CLI v2.2.4. Provider versions before v1.3.0
used it as their session health check, so on v2.2.4+ every Terraform
operation failed at provider configuration time with:

```
Proton Pass CLI Not Ready
Could not verify pass-cli session.
… error: unrecognized subcommand 'test'
```

The provider now probes with `pass-cli info` instead. `info` performs the
same authenticated round trip (user info for password sessions, token name
for PAT and agent sessions), exits non-zero when no session is active, and
predates every CLI version this provider supports. If the installed CLI is
old enough not to recognise `info`, the provider falls back to `test`
automatically, so no configuration change is needed either way.

### Breaking change: `item list` output changed in v2.0.3

As of CLI v2.0.3, `pass-cli item list --output=json` no longer includes the
`content` object. This is deliberate: listing must never return secret
material. Item titles and types moved to the top level of each entry:

```json
{
  "items": [
    {
      "id": "…", "share_id": "…", "vault_id": "…",
      "state": "Active", "flags": [],
      "create_time": "…", "modify_time": "…",
      "title": "Database Credentials",
      "item_type": "login"
    }
  ]
}
```

Provider versions before v1.3.0 only understood the older nested shape, so on
CLI v2.0.3+ every listed item parsed with an empty title and was misreported
as a note. That broke item creation (the new item could not be found on
readback, failing the apply while leaving the item in the vault), the
`protonpass_items` data source, and trashed-item detection.

The provider now accepts both shapes, so no configuration change is needed.

Note that `item list` returns metadata only on v2.0.3+. The
`protonpass_items` data source exposes exactly that metadata (`item_id`,
`share_id`, `title`, `type`, `create_time`, `modify_time`). To read secret
values, use the `protonpass_item` data source, which fetches the full item
with `pass-cli item view`.

To check your installed version:

```shell
pass-cli --version
```

## Verifying Your Session

The provider calls `pass-cli info` as a health check at the start of each
Terraform operation. It considers the session valid if the command exits
with code 0; stdout is not inspected.

```shell
pass-cli info
```

A successful response exits 0 and prints the account or token the session
belongs to. A non-zero exit code indicates the session is not active.

## Authentication in CI/CD

### Interactive Session (v1.x and v2.x)

For local development and simple CI setups, authenticate once with:

```shell
pass-cli login
pass-cli info
```

The session is stored locally and reused by subsequent `pass-cli` calls.

### Personal Access Token (v1.10.0+)

For CI/CD pipelines, use a Personal Access Token (PAT) to avoid
interactive login. PATs grant scoped access to specific vaults and items.

Generate a token in your Proton Pass account settings, then either pass it
to `login` as a flag:

```shell
pass-cli login --pat "$PROTON_PASS_PERSONAL_ACCESS_TOKEN"
```

Or set the environment variable and let `pass-cli` pick it up when the flag
is omitted:

```shell
export PROTON_PASS_PERSONAL_ACCESS_TOKEN="<your-token>"
pass-cli login
pass-cli info
```

The token has the format `pst_<token>::<key>`.

> Do not hardcode the token value. Inject it via your CI secret store
> (GitHub Actions secrets, HashiCorp Vault, AWS Secrets Manager, etc.).

### Session locks

`pass-cli` v2.2.0+ can lock a session (`pass-cli session create-lock`). A
locked session fails the provider health check. Unlock it with `pass-cli
session unlock` before running Terraform, or leave CI sessions unlocked.
PAT sessions cannot enable a session lock as of v2.2.6.

## Environment Variables Reference

The following environment variables are documented by `pass-cli` for
authentication. They apply to the `pass-cli login` command.

| Variable | Purpose | Version |
|---|---|---|
| `PROTON_PASS_PERSONAL_ACCESS_TOKEN` | Token for PAT-based login | v1.10.0+ |
| `PROTON_PASS_USERNAME` | Account username for interactive login | v1.x+ |
| `PROTON_PASS_USERNAME_FILE` | Path to a file containing the username | v1.x+ |
| `PROTON_PASS_PASSWORD` | Account password for interactive login | v1.x+ |
| `PROTON_PASS_PASSWORD_FILE` | Path to a file containing the password | v1.x+ |
| `PROTON_PASS_SECOND_PASSWORD` | Proton second password | v2.3.0+ |
| `PROTON_PASS_SECOND_PASSWORD_FILE` | Path to a file containing the second password | v2.3.0+ |
| `PROTON_PASS_TOTP` | TOTP code for two-factor authentication | v1.x+ |
| `PROTON_PASS_TOTP_FILE` | Path to a file containing the TOTP code | v1.x+ |
| `PROTON_PASS_EXTRA_PASSWORD` | Pass-specific extra password | v1.x+ |
| `PROTON_PASS_EXTRA_PASSWORD_FILE` | Path to a file containing the extra password | v1.x+ |

> **Security note**: never set `PROTON_PASS_PASSWORD` or
> `PROTON_PASS_PERSONAL_ACCESS_TOKEN` in files committed to source control.
> Always inject them from a secret store at runtime.

The `protonpass` Terraform provider itself reads none of these variables
directly, apart from `PROTON_PASS_AGENT_REASON` (see the `agent_reason`
provider attribute). The rest are consumed by `pass-cli` during session
setup, before Terraform runs.

## Example: GitHub Actions

```yaml
- name: Set up pass-cli session
  run: |
    pass-cli login --pat "$PROTON_PASS_PAT"
    pass-cli info
  env:
    PROTON_PASS_PAT: ${{ secrets.PROTON_PASS_PAT }}

- name: Terraform apply
  run: terraform apply -auto-approve
```

## Upstream Resources

- [pass-cli releases](https://github.com/protonpass/pass-cli/tags)
- [pass-cli changelog](https://github.com/protonpass/pass-cli/blob/main/CHANGELOG.md)
- [pass-cli documentation](https://protonpass.github.io/pass-cli/)
