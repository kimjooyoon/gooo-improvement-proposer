# Release evidence history

## v0.1.0 failed release trigger

The first annotated tag was deliberately retained after its release workflow
failed before publishing assets. The authoritative GitHub record is:

```text
tag: v0.1.0
tag object: 9acc18c4f021a42fbd41f2c67f22bb1df1152187
tag target commit: 2007b49fdf60765b1868636da75b980f0c16db28
failed workflow run: 33396465907
release: absent
```

This tag/object/target and failed run are a `FAILED_RELEASE_TRIGGER`
counterexample. They are not used as successful release evidence and are not
deleted, force-moved, or recreated.

## v0.1.1 initial failed attempt

The first v0.1.1 tag-triggered run also remains in Actions history while the
release workflow is corrected. It failed before publishing assets because the
checkout action temporarily represented the annotated tag ref as its commit
target. The tag itself remains intact:

```text
tag: v0.1.1
tag target commit: 6757651d5b6abae7dfb7c7a3ec7a0cab103e3279
failed workflow run: 33397076212
release: absent at the time of the failed run
```

The workflow now checks out the explicit annotated tag ref and supports a
manual retry for that existing tag.

## v0.1.1 success contract

The successful release is published from a new annotated tag after the
workflow-fix pull request and its post-main conformance run are green.

The release workflow binds the v0.1.1 tag to its commit and publishes:

- the source archive;
- the conformance evidence archive;
- `release-manifest.json` with `immutable=true`, tag target, asset names, and
  SHA-256 digests;
- `SHA256SUMS` covering every published asset.

The v0.1.1 evidence archive contains the fixed six artifacts for every controlled
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
