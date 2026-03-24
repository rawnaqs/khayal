import { test as base, expect } from "@playwright/test";

// Extend base test with custom fixtures
export const test = base.extend({
  page: async ({ page }, use) => {
    // Setup: configure localStorage before each test
    await page.goto("/");
    await page.evaluate(() => {
      localStorage.setItem("khayal_token", "test-token");
      localStorage.setItem("khayal_host", "http://localhost:1133");
    });
    await page.reload();

    // Use the page
    await use(page);
  },
});

export { expect };
