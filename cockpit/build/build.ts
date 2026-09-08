import { createBuilder } from "vite-plus";

const builder = await createBuilder({ configFile: "vite.config.ts" });
await builder.buildApp();
