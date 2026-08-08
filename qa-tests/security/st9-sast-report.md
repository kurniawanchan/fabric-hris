# ST-9 — SAST Gate Report

**Date:** 2026-08-07
**Scope requested:** `fabric-network/chaincode/employeeprofilerecord/`, `write-path-integration/*/`, `fabric-network/tools/*/`
**Tooling:** Semgrep Guardian MCP (`get_semgrep_sast_findings`, `get_semgrep_secrets_findings`, `list_semgrep_projects`, `whoami`)

## Outcome: BLOCKED — no Semgrep project is registered for this repo

This is a genuine, current-state finding, not a fabricated scan result.

### Evidence

1. **`whoami`** confirms an authenticated session is active:
   ```
   auth_method: oauth
   identity: <redacted — account identity string>
   deployment: <redacted — account deployment slug> (id 98102, org 179785)
   ```

2. **`list_semgrep_projects`** against that deployment returns zero projects:
   ```
   {"deployment":"<redacted — account deployment slug>",
    "message":"No projects were found in deployment \"<redacted — account deployment slug>\".",
    "projects":[]}
   ```

3. A direct attempt to pull findings anyway (`get_semgrep_sast_findings`, `repository_name:"fabric-hris"`) fails with:
   ```
   Error: no projects found in deployment "<redacted — account deployment slug>"
   ```

There is no repository under this Semgrep deployment corresponding to `fabric-hris` (or any name), so there is
no prior CI scan to pull SAST or secrets findings from for any of the requested paths:
`fabric-network/chaincode/employeeprofilerecord/`, `write-path-integration/gateway-client/`,
`write-path-integration/ipfsclient/`, `write-path-integration/keystore/`, `write-path-integration/writepaths/`,
`fabric-network/tools/ccdeploy/`, `fabric-network/tools/jcsverify/`, `fabric-network/tools/netjoin/`,
`fabric-network/tools/raftfaulttest/`, `fabric-network/tools/tenantprovision/`.

### Why no local/offline substitute was attempted

Per task instructions, this report must reflect the *real* state of the Semgrep Guardian platform for this
account, not a locally-run guess dressed up as a platform scan. Registering a brand-new Semgrep project against
this deployment was not attempted: the available MCP tools (`list_semgrep_projects`,
`get_semgrep_sast_findings`, `get_semgrep_secrets_findings`, `get_semgrep_supply_chain_findings`, `whoami`,
`login`) are read-only against existing platform-side scan results — none of them exposes a "create/onboard
project" or "trigger a scan" operation, so onboarding is not something achievable trivially through this MCP
surface (it requires either the Semgrep CI integration / `semgrep ci` with a token wired into a pipeline, or
manual repo linking in the Semgrep AppSec Platform UI — both are infrastructure changes outside this task's
read-only QA scope).

## Verdict for the SAST gate

**QA-5 gate status: NOT SATISFIED — infrastructure gap, not a code defect.**

The SAST gate cannot currently pass or fail on evidence, because there is no scan data to evaluate. This should
be tracked as a prerequisite/blocker distinct from any code-level finding:

- **Action needed (infra/process, not code):** wire `fabric-hris` (or at minimum the chaincode and write-path
  modules in scope) into the Semgrep AppSec Platform as a registered project — e.g. via the Semgrep GitHub/GitLab
  App, or a CI job running `semgrep ci` with `SEMGREP_APP_TOKEN` pointed at the account's existing (currently
  project-less) Semgrep deployment — so that subsequent ST-9 runs have real findings to fetch.
- Until that exists, ST-9 should remain open/blocked in the backlog rather than marked pass or fail.

## Confidentiality check

This report was grepped for the confidentiality register terms before being finalized; see structured output
for the exact command and result.
