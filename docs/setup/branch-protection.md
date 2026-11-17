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

## Status: DEFERRED (accepted override)

> **The ruleset is NOT yet applied — still plan-gated, re-verified 2026-11-17.**
> The GitHub rulesets API returns
> `403 Upgrade to GitHub Pro or make this repository public` for this private org
> repository (research-verified live 2026-10-31; re-verified by read-side probe at
> execution, same day). Rulesets on private repositories require a paid org plan
> (Team tier or equivalent). Until the org plan changes (or the repo flips public —
> itself a deferred decision), the payload below sits recorded and the protection
> sub-item of CI-03 is **plan-gated** — explicitly flagged, not silently skipped.
>
> **Re-apply attempted 2026-11-17 at v1.0 close-out (D-11):** the Apply command
> verbatim from below returned
> `403 Upgrade to GitHub Pro or make this repository public to enable this feature.`
> and the read-side verify returns the same 403 — the org plan has not changed.
> The enforcement probe (trivial direct push) is deliberately NOT run on the 403
> path: with no ruleset live it would actually land on `master` and violate the
> PR-only contract it exists to prove.
>
> **Deferral decision 2026-10-31: `defer`** — keep everything else, do not upgrade the
> org plan now; the moment the plan changes, run the Apply command above.
>
> **Ratified 2026-10-31: accepted override.** The maintainer confirmed the deferral as a
> deliberate, accepted limitation (org-plan gated; no plan upgrade, no public flip) —
> recorded in 01-VERIFICATION.md overrides and REQUIREMENTS.md CI-03; the re-apply path
> below stays current for whenever that decision changes.

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
