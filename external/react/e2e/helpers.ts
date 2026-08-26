import { test as base, expect } from "@playwright/test";

// Extend base test with custom fixtures
export const test = base.extend({
  page: async ({ page }, use) => {
    // Setup: configure localStorage before each test
    await page.goto("/");
    await page.evaluate(() => {
      localStorage.setItem("khayal_token", "abc");
      // same-origin host: vite dev proxy forwards /v1 to the test server
      localStorage.setItem("khayal_host", window.location.origin);
    });
    await page.reload();

    // Use the page
    await use(page);
  },
});

export { expect };
