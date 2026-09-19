# Documentation

Public documents describe current contracts. Release notes belong in GitHub
Releases. Completed task lists, dated experiments, and development chronology
do not belong in maintained guides.

## Find information

| Need | Guide |
|---|---|
| Run Go programs | [Gart](gart.md) |
| Build Flutter applications | [Glutter](glutter.md) |
| Understand limitations | [Compatibility](parity.md) |
| Inspect compiler structure | [Architecture](design.md) |
| Understand suspension | [Block model](block-model.md) |
| Understand web integers | [JavaScript integers](js-integers.md) |
| Use Go UI state and interop | [Flutter architecture](flutter-design.md) |
| Generate library bindings | [Binding guide](../flutter/nodegen/README.md) |
| Validate upstream behavior | [Upstream parity](upstream-parity.md) and [blockers](upstream-blockers.md) |
| Run repository validation | [CI](ci.md) |
| Measure compiler phases | [Compilation time](compilation-time.md) |
| Upgrade toolchains | [SDK procedure](sdk-upgrades.md) |
| Preserve upstream terms | [Licenses](licenses.md) |

## Maintenance dependencies

| Change | Update or validate |
|---|---|
| CLI flag or output layout | Command help and the corresponding command guide |
| Supported language or runtime behavior | Compatibility, verifier tests, and upstream parity coverage |
| Compiler representation or suspension | Architecture or block model, plus the affected semantic tests |
| Widget, callback, state, or service contract | Flutter architecture, binding guide, and maintained fixture |
| SDK, dependency, generator, or reused material | SDK manifest, input inventories, licenses, and applicable notice tests |
| CI selection, mode, or resource budget | CI guide, matrix, and local runner tests |
| Security or contribution policy | Root policy documents |

Prefer executable inventories over copied API counts. Keep detailed invariants
beside their code. A public contract change requires its related guide in the
same coherent commit. Add a document only when an existing guide cannot serve
its audience without unrelated material.
