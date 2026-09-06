import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, it } from "vite-plus/test";
import { Markdown } from "./markdown";
afterEach(cleanup);
it("renders GFM without active HTML, image loading or unsafe links", () => {
  render(<MemoryRouter><Markdown text={'# Safe heading\n\n<script>alert(1)</script>\n\n![tracking](https://tracker.invalid/pixel)\n\n[unsafe](javascript:alert%281%29)\n\n| A | B |\n| - | - |\n| x | y |'} /></MemoryRouter>);
  expect(screen.getByRole("heading", { name: "Safe heading" })).toBeTruthy();
  expect(screen.getByRole("table")).toBeTruthy();
  expect(document.querySelector("script")).toBeNull(); expect(document.querySelector("img")).toBeNull();
  expect(screen.queryByRole("link", { name: "unsafe" })).toBeNull();
});
it("resolves relative Markdown links against the selected file and slash ref", () => {
  render(<MemoryRouter><Markdown text="[Guide](../guide/hello%20world.md)" repository={{ route: "/repositories/alice/demo", ref: "feature/one", path: "docs/README.md" }} /></MemoryRouter>);
  const link = screen.getByRole("link", { name: "Guide" });
  expect(link.getAttribute("href")).toBe("/repositories/alice/demo?ref=feature%2Fone&path=guide%2Fhello+world.md");
});
