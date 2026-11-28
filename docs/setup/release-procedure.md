# Release procedure: v1.1.0

The tag → snapshot → verify → tag-push → release-verify path for a sup-kv release.
This document is the executed audit trail: every command below is re-runnable verbatim,
and each step records its expected output beside it. The recorded outputs come from the
v1.1.0 run — the release plan fills the final transcript; the snapshot-shaped outputs
recorded today are from the 2026-11-28 pre-release snapshot at commit `fbda859`.

- Pipeline: the `go.mod`-pinned goreleaser tool directive (`go tool goreleaser`,
  v2.20.2) driven by [`.goreleaser.yaml`](../../.goreleaser.yaml); the tag-triggered
  [`.github/workflows/release.yml`](../../.github/workflows/release.yml) is what
  publishes on GitHub. Nothing here invents new machinery — it drives the existing
  Makefile targets and workflow.
- One machine, serial steps: snapshot → inspection → tag, in this document's order.
  The inspection gates the tag; never tag an uninspected snapshot.
- The snapshot is re-runnable: `make release-snapshot` (`--snapshot --clean`) rebuilds
  `dist/` from scratch and the inspection below re-runs on any re-snapshot. A failed
  snapshot is left in `dist/` for diagnosis — nothing cleans up after itself; read the
  log, fix, and re-snapshot.
- The tag is the one-way act: pushing a `v*` tag fires the release workflow and
  publishes the release with the assets the config defined. Everything before it is
  reversible; it is not. The tag step is reserved for the final plan after every gate
  is green on merged `master`.

## 1. Prerequisites

The phase branch is merged to `master` (PR-only — direct pushes are rejected by the
[branch-protection ruleset](./branch-protection.md)), the tree is clean, and `master`
is current:

```bash
git status --porcelain   # must print nothing
git fetch origin && git pull --ff-only
# expected: "Already up to date." (or the fast-forward to the release commit)
```

The snapshot's `sboms` pipe shells out to `syft` — the same version-pinned
provisioning the release workflow gets from
`anchore/sbom-action/download-syft@v0.24.3` (which resolves syft v1.54.0). Assert it
on PATH before snapshotting:

```bash
syft version | grep -E '^Version:'
# expected: Version:       1.54.0
```

If it is missing or unpinned, provision it as a checksum-verified GitHub-release
download — never a package-manager install, and `--skip=sbom` is NEVER the workaround
(`checksums.txt` hashes the SBOM files and the archive contract counts them as
assets):

```bash
DL=$(mktemp -d /tmp/syft-dl.XXXXXX) && EX=$(mktemp -d /tmp/syft-ex.XXXXXX)
curl -fsSL -o "$DL/syft_1.54.0_linux_amd64.tar.gz" \
  https://github.com/anchore/syft/releases/download/v1.54.0/syft_1.54.0_linux_amd64.tar.gz
curl -fsSL -o "$DL/syft_1.54.0_checksums.txt" \
  https://github.com/anchore/syft/releases/download/v1.54.0/syft_1.54.0_checksums.txt
grep 'syft_1.54.0_linux_amd64.tar.gz' "$DL/syft_1.54.0_checksums.txt" | shasum -a 256 -c -
# expected: syft_1.54.0_linux_amd64.tar.gz: OK
tar -xzf "$DL/syft_1.54.0_linux_amd64.tar.gz" -C "$EX" syft
install -m 0755 "$EX/syft" "$HOME/.local/bin/syft"   # or /usr/local/bin with sudo
rm -rf "$DL" "$EX"
```

Executed 2026-11-28: the checksum line printed `OK` and `syft version` reported
`Version:       1.54.0`.

## 2. Gates on master

```bash
make lint
# expected: exit 0 — config verify passes, then the strict golangci-lint run
#           reports no issues (~1–2 min)

make test
# expected: exit 0 — the race+coverage suite over all packages is green
#           (~10 min; the verbatim ok-line transcript is recorded at the v1.1.0 run)
```

Both must pass on the exact commit the tag will name. A red gate stops here — nothing
publishes until `master` is green.

## 3. The release snapshot

```bash
make release-snapshot
# expected: exit 0; dist/ populated. Executed 2026-11-28 at commit fbda859, the run
#           ended:
#     • calculating checksums
#     • writing artifacts metadata
#     • release succeeded after 25s
#     • thanks for using GoReleaser!
```

`--snapshot` builds every configured target plus checksums into `dist/` with no tag
and no publishing. Re-running it rebuilds `dist/` from scratch (`--clean`); the
inspection in step 4 re-runs unchanged after any re-snapshot.

## 4. Archive inspection

The asset set the config defines becomes the shipped contract at tag time — inspect
it here, before the tag exists.

```bash
ls dist/
# expected: EXACTLY two archives (Linux x86_64 + arm64) with one .sbom.json per
#           archive, checksums.txt, artifacts.json, metadata.json, config.yaml and
#           the two unpacked build directories — and no darwin artifact anywhere.
#           Snapshot run 2026-11-28 (tagged release names swap the
#           1.0.0-SNAPSHOT-<sha> form for 1.1.0):
#     artifacts.json
#     checksums.txt
#     config.yaml
#     metadata.json
#     sup-kv_1.0.0-SNAPSHOT-fbda859_Linux_arm64.tar.gz
#     sup-kv_1.0.0-SNAPSHOT-fbda859_Linux_arm64.tar.gz.sbom.json
#     sup-kv_1.0.0-SNAPSHOT-fbda859_Linux_x86_64.tar.gz
#     sup-kv_1.0.0-SNAPSHOT-fbda859_Linux_x86_64.tar.gz.sbom.json
#     sup-kv_linux_amd64_v1
#     sup-kv_linux_arm64_v8.0
```

Each archive carries the binary and the deployment artifacts (D-05's archive
contract):

```bash
tar tzf dist/sup-kv_*Linux_x86_64.tar.gz
tar tzf dist/sup-kv_*Linux_arm64.tar.gz
# expected, for BOTH archives:
#     CHANGELOG.md
#     LICENSE
#     README.md
#     deploy/sup-kv.service
#     docs/DEPLOYMENT.md
#     sup-kv
```

Checksum and SBOM integrity, verified from a copy of `dist/` (so the verification
never touches the working tree's own artifacts and a concurrent re-snapshot cannot
race the check):

```bash
rm -rf /tmp/sup-kv-release-check && cp -r dist /tmp/sup-kv-release-check
cd /tmp/sup-kv-release-check && shasum -a 256 -c checksums.txt
# expected: every line prints OK — the two archives and their two SBOMs
cd -
```

Executed 2026-11-28: all four lines (`...Linux_arm64.tar.gz`,
`...Linux_arm64.tar.gz.sbom.json`, `...Linux_x86_64.tar.gz`,
`...Linux_x86_64.tar.gz.sbom.json`) printed `OK`, and the SBOM headers read
`"spdxVersion":"SPDX-2.3"` with `"Tool: syft-1.54.0"`.

## 5. The tag — THE ONE-WAY STEP

Publishing starts here. Everything above is re-runnable; this step is not. Only after
steps 1–4 are green on the commit being tagged:

```bash
git tag -a v1.1.0 -m "sup-kv v1.1.0" && git push origin v1.1.0
# expected: the tag lands; the release workflow fires on the push
#           (.github/workflows/release.yml runs go tool goreleaser release --clean)
```

Watch the run (`gh run watch` or the Actions tab). A failure here is diagnosed from
the workflow log; the snapshot gate of step 3 is the reason this step has no config
surprises left.

## 6. Release verification

```bash
gh release view v1.1.0
# expected — the body:
#   - starts with the sliced CHANGELOG section: "## [1.1.0] - <date>" followed by
#     the milestone paragraph and the Added/Changed/Fixed bullets;
#   - contains neither "[Unreleased]" nor any "(Phase " text.
# expected — the assets: exactly the two Linux archives, their two .sbom.json
#   files and checksums.txt — the same set step 4 inspected.
gh release view v1.1.0 | grep -c 'Unreleased'   # expected: 0
gh release view v1.1.0 | grep -c '(Phase '      # expected: 0
```

The verbatim body and asset list are recorded at the v1.1.0 run. The body is the
sliced `[1.1.0]` CHANGELOG section by construction (the release-header template), so
these checks verify the slice plus the scrub survived the real release.

## 7. Post-release

The two links a reader hits first must resolve against the new release:

```bash
grep -n 'releases' README.md   # the Install-from-release section's link
curl -fsSI https://github.com/Djarvur/sup-kv/releases/latest | grep -i '^location'
# expected: the location header resolves to .../releases/tag/v1.1.0
grep -n '^\[1\.1\.0\]:' CHANGELOG.md
# expected: [1.1.0]: https://github.com/Djarvur/sup-kv/releases/tag/v1.1.0
curl -fsSI https://github.com/Djarvur/sup-kv/releases/tag/v1.1.0 | head -1
# expected: HTTP/2 200 (or HTTP/1.1 200 OK)
```

The shipped binary identity closes the loop: download an archive from the release,
unpack it, and run `./sup-kv --version` — it must report `sup-kv v1.1.0 <commit>
<goversion>` (the ldflags stamp matches the tag per `.goreleaser.yaml`'s version
identity contract).

## Re-verification after a re-snapshot

Steps 3–4 are the only snapshot-coupled steps, and both re-run verbatim after any
`make release-snapshot`. Step 5 is the only state the procedure cannot redo: a pushed
tag is published history. If step 4 fails, do not tag — fix the config, re-snapshot,
re-inspect.
