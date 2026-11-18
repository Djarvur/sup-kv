# Branch Protection: `master`

Server-side protection of `refs/heads/master` via a GitHub **repository ruleset**, applied
and verified exclusively through `gh api` — never by UI clicking. This document is the
audit trail (decision D-08): every command below is re-runnable verbatim.

- Payload: [`ruleset.json`](./ruleset.json) — ruleset `protect master`
- Rule set: `pull_request` (PR required before merging), `required_status_checks`
  (the five ci.yml checks), `non_fast_forward` (no force pushes), `deletion`
  (no master deletion). No path-based exemptions exist or may be added — docs-only,
  workflow-only, and planning-only changes go through pull requests exactly like code
  (CI-03: the PR-only contract admits no side doors).

## Status: ENFORCED

> **The ruleset is applied and active — re-applied 2026-11-18.**
> The repository flipped PUBLIC on 2026-11-18 (maintainer decision), removing the free-plan
> gate that had blocked the rulesets API for the private repo (`403 Upgrade to GitHub Pro or
> make this repository public`, research-verified 2026-10-31, re-verified at every close-out
> through v1.0). The first Apply attempt on the public repo surfaced a latent payload bug —
> GitHub requires `conditions.ref_name.exclude` to be present (empty array) — fixed in the
> same change; the payload had never been executable before (403 masked it).
>
> CI-03's protection sub-item is no longer plan-gated: the override recorded at v1.0 close
> (ratified 2026-10-31) is discharged by this application. The enforcement probe below was
> run and passed: a direct push to `master` is rejected by GitHub.

## Apply

```bash
gh api --method POST repos/Djarvur/sup-kv/rulesets --input docs/setup/ruleset.json
# 201 = created; 422 = payload validation error (fix the payload, never the command shape)
```

## Verify (read side)

```bash
gh api repos/Djarvur/sup-kv/rulesets \
  --jq '.[] | select(.name == "protect master") | {name, enforcement, rules: [.rules[].type]}'
# Must show: enforcement "active" and rule types
# ["pull_request", "required_status_checks", "non_fast_forward", "deletion"]
```

Prove enforcement after applying: attempt a trivial direct push to `master` from a
scratch branch — GitHub must reject it.

## Delete branch on merge

An ordinary repository setting, not plan-gated (already enabled):

```bash
gh api --method PATCH repos/Djarvur/sup-kv -f delete_branch_on_merge=true
gh api repos/Djarvur/sup-kv --jq '.delete_branch_on_merge'   # must print true
```

## Re-application and audit (context-coupling rule)

- **Every gate-matrix change is a ruleset change.** The five `required_status_checks`
  contexts must equal the exact check-run display names produced by
  `.github/workflows/ci.yml`: `lint`, `gate (ubuntu-latest)`, `gate (ubuntu-24.04-arm)`,
  `gate (macos-15)`, `gate (darwin-amd64-cross)`. Renaming a matrix cell, adding a cell,
  or renaming the lint job without updating `ruleset.json` in the same commit produces
  stuck "Expected" checks or silently requires nothing (Pitfall 6). Update the contexts
  in `ruleset.json` in the same commit as the workflow change.
- **Bump = re-run this doc.** To change protection, edit `ruleset.json`, land it via a
  PR, then re-apply (for an update: `gh api --method PUT
  repos/Djarvur/sup-kv/rulesets/<id> --input docs/setup/ruleset.json` using the ruleset
  id from the verify command) and re-run the verification above. The payload in git is
  the source of truth; GitHub state follows it.
- **Re-apply from scratch:** delete the ruleset
  (`gh api --method DELETE repos/Djarvur/sup-kv/rulesets/<id>`) and run the Apply
  command again.

## Zero-approving-reviews rationale

`required_approving_review_count` is **0**. This is a solo-maintainer repository: GitHub
does not allow PR authors to approve their own pull requests, so any count ≥ 1 would make
every PR unmergeable by construction — a self-approval deadlock. The PR-only contract
itself is still enforced by the `pull_request` rule: no direct pushes to `master`, every
change lands through a pull request that runs the required checks. `require_last_push_approval`
and `required_review_thread_resolution` are likewise `false` for the same reason — each
would hand the solo developer an approval requirement nobody else can satisfy. If a second
maintainer ever joins, revisit all three values in the same commit that grants write access.
