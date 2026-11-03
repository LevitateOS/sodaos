# Factory implementation queue

Factory development now lives in the public [LevitateOS/factory](https://github.com/LevitateOS/factory) repository. The canonical dependency-ordered queue and all earlier implementation evidence moved with it:

- [Implementation slices](https://github.com/LevitateOS/factory/blob/main/docs/development/factory-products/implementation-tasks.md)
- Local checkout: [/home/vince/Projects/factory/docs/development/factory-products/implementation-tasks.md](/home/vince/Projects/factory/docs/development/factory-products/implementation-tasks.md)
- [Full product scope and completion standard](https://github.com/LevitateOS/factory/blob/main/docs/development/factory-products/ideal-filetree-plan.md)

Continue exactly one ready slice there. Preserve the full Linux connected/installed issue-to-PR/review/correction objective. This redirect does not mark the product complete or remove any remaining gate.

Forge's Go consumers now use the pinned public Factory module rather than duplicate local engine/store/application/domain packages. Forge also consumes the defining Factory UI, Storybook fixtures, fonts and theme through its standard Bun source link. The remaining native Rust consumer cutover is the next independent slice. Appliance composition, Fountain adapters and private services remain in this repository. The original complete queue and working source are retained in Factory's initial import and integration recovery patch.
