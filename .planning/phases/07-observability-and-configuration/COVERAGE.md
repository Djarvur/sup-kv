# API Coverage — Phase 7 (Observability and Configuration)

No external API integration: this phase adds no third-party API, SDK, REST/GraphQL surface or external service. Its whole surface is in-repo operator observability and configuration: `log/slog` lifecycle records over the standard library, the project's own Prometheus text renderer (`internal/metrics`), configuration from the standard `flag` package and `SUPKV_*` environment variables, and the already-public project protobuf (`api/supkvapi/v1`) for the existing gRPC API. The only network endpoints touched are the existing UDP replication socket, the existing gRPC listener and the opt-in metrics/debug listener the binary already serves; no vendor endpoint, credential or SDK is introduced. No plan contains a package-install task, and the shipped runtime dependency graph stays empty (the `tool` directives and their indirect requires are pre-existing).

No external API rows are therefore claimed or fabricated; the API-coverage table is intentionally empty for this phase.

<assumption_delta_decision>
Decision: **no-change**.

Rationale: the two candidate structural moves are decision-backed, not new planner assumptions. The `internal/config` extraction implements D-03's "one config package and one validation function" and stays a composition-level package (standard library plus own-module constants, deliberately outside the `internal/app` no-I/O rule); the `internal/udprepl` split implements D-08's same-package four-unit decomposition with tests unchanged. Neither introduces a new library, a new architecture tier, a new external dependency or a new project convention that AGENTS does not already need to state. The only documentation ripple is the `## Layout` entry for `internal/config/` and the project-deviation rule names, both delivered by plan 07-10; no ADR is warranted because the decisions they implement are already recorded in `07-CONTEXT.md` (D-03, D-07, D-08) and the phase's research.

`detected` was not consulted as a gate: the `gsd_run query assumption-delta scan 7 --json` command was unavailable in this planning environment (`gsd_run: command not found`); the decision above is made by inspection of the phase's decision set and the plan file list. If the orchestrator can run the scan, treat this record as the planner's decision and reconcile only if the scan reports a category outside config extraction/decomposition.
</assumption_delta_decision>
