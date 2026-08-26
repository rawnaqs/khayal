import { test, expect } from "./helpers";

test("clicking a done queue item opens the note sheet", async ({ page }) => {
  page.on("console", (m) => console.log("[console]", m.type(), m.text()));
  page.on("response", async (r) => {
    if (r.url().includes("/v1/")) console.log("[net]", r.status(), r.url());
  });
  await page.goto("/");
  console.log("[ls]", await page.evaluate(() => JSON.stringify(localStorage)));
  // go to queue tab
  const navItem = page.locator("nav .nt", { hasText: "queue" });
  await navItem.waitFor({ state: "visible", timeout: 10000 });
  await navItem.click();
  // wait for done section
  await expect(page.getByText(/done \(/)).toBeVisible({ timeout: 10000 });
  // click the first done item with a note path
  const item = page.locator('[data-testid="done-item"]').filter({ hasText: ".md" }).first();
  await expect(item).toBeVisible();
  await item.click();
  // NoteView sheet should appear with note content
  await expect(page.locator("[data-radix-popper-content-wrapper], [role=dialog], .note-detail")).toBeVisible({ timeout: 5000 });
});
