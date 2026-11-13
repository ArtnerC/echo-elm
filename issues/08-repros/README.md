# issues/08-repros — moved

The synthetic reproductions for issue 08 now live in the parity corpus:

- fixtures: `test/corpus/cqframework/measure-families/` (registered in
  `test/corpus/cqframework/corpus.yaml`, tag `measure-families`)
- CQF 5.0.0 goldens: `test/goldens/cqf/measure-bundle/measure-families/`
- provenance: `test/corpus/cqframework/measure-families/PROVENANCE.md`

They were moved in the change that fixed their families, as this directory's original README
asked. The goldens are the pinned CLI's output for the files in their new location; they are
semantically identical to the references this directory shipped with.

## Run

From the repository root:

```powershell
# Java-free, against the committed goldens
.\echo-elm.exe parity --tag measure-families --ref-dir test/goldens/cqf

# Fresh CQF 5.0.0 (needs Java and tools/cqframework/5.0.0)
.\echo-elm.exe parity --tag measure-families
```

Expected: `match=18 differ=0 divergent=1 error=0 total=21`.

- `FlattenOneInstance` is `divergent` by decision: see its `divergence` note in `corpus.yaml`
  and issue 08's status section.
- `FluentNonFluentCall` and `FlattenAmbiguity` are `upstream-error` (CQF rejects them), so the
  harness skips them. echo-elm now rejects `FluentNonFluentCall` with CQF's message, and
  deliberately compiles `FlattenAmbiguity`; both are pinned by tests in
  `internal/translator/overloads_test.go`.
