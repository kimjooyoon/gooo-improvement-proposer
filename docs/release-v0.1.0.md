# v0.1.0 release evidence

The v0.1.0 release is published from an annotated tag after the first feature
pull request is merged and the post-main conformance run is green.

The release workflow binds the tag to its commit and publishes:

- the source archive;
- the conformance evidence archive;
- `release-manifest.json` with `immutable=true`, tag target, asset names, and
  SHA-256 digests;
- `SHA256SUMS` covering every published asset.

The evidence archive contains the fixed six artifacts for every controlled
case, the 12-cell semantic IR and generated evaluator, conformance index, CI
report, runtime receipt, and summary. Failed workflow attempts remain
counterexamples in the Actions history; they are not replaced by a later
successful run.

The release manifest also carries exact zero authority fields:

```text
repository_writes=0
pull_request_creations=0
merge_operations=0
local_test_executions=0
cross_project_required_gates=0
```
