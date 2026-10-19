import assert from "node:assert/strict";
import test from "node:test";
import { canViewTask } from "./access.mjs";

test("active HR can view a task", () => {
  assert.equal(canViewTask({ id: "hr-1", role: "hr", active: true }, { assigneeId: "staff-1" }), true);
});

test("inactive HR cannot view a task", () => {
  assert.equal(canViewTask({ id: "hr-1", role: "hr", active: false }, { assigneeId: "staff-1" }), false);
});

test("an unrelated employee cannot view a task", () => {
  assert.equal(canViewTask({ id: "staff-2", role: "staff", active: true }, { assigneeId: "staff-1" }), false);
});
