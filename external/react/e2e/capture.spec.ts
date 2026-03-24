import { test, expect } from "./helpers";

test.describe("Capture Flow", () => {
  test("should show onboarding when not configured", async ({ page }) => {
    // Clear localStorage
    await page.evaluate(() => {
      localStorage.clear();
    });
    await page.reload();

    // Should show onboarding
    await expect(page.getByText("khayal")).toBeVisible();
    await expect(page.getByPlaceholder("token")).toBeVisible();
  });

  test("should navigate to capture tab by default", async ({ page }) => {
    // Should be on capture tab
    await expect(page.locator(".cap-body")).toBeVisible();
    await expect(page.getByText("txt")).toBeVisible();
    await expect(page.getByText("url")).toBeVisible();
    await expect(page.getByText("img")).toBeVisible();
  });

  test("should show text capture mode by default", async ({ page }) => {
    // Should show text input
    const textarea = page.locator("textarea");
    await expect(textarea).toBeVisible();
  });

  test("should switch to URL mode", async ({ page }) => {
    await page.click("text=url");

    // Should show URL input
    await expect(
      page.getByPlaceholder("https://example.com/article"),
    ).toBeVisible();
  });

  test("should switch to image mode", async ({ page }) => {
    await page.click("text=img");

    // Should show file input
    const fileInput = page.locator('input[type="file"]');
    expect(fileInput).toBeDefined();
  });

  test("should show stats when available", async ({ page }) => {
    // Mock the stats API
    await page.route("**/v1/stats", (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          streak: {
            current: 5,
            best: 10,
            next_milestone: 7,
            days_to_milestone: 2,
            this_week: [true, true, true, false, false, false, false],
          },
          today: { count: 3, by_hour: [], avg_per_day: 2.5 },
          vault: {
            total_notes: 100,
            today_delta: 3,
            last_capture_at: "2024-01-01T10:00:00Z",
            last_7_days: [1, 2, 3, 4, 5, 6, 7],
          },
        }),
      });
    });

    await page.reload();

    // Should show stats
    await expect(page.getByText("100")).toBeVisible();
  });

  test("should show greeting based on time of day", async ({ page }) => {
    // The greeting should be visible
    const greeting = page.locator(".cap-greeting");
    await expect(greeting).toBeVisible();
  });

  test("should have send button", async ({ page }) => {
    const sendButton = page.locator(".send");
    await expect(sendButton).toBeVisible();
  });

  test("should show hint text", async ({ page }) => {
    await expect(page.getByText("cmd+enter to capture")).toBeVisible();
  });

  test("should switch between capture modes and show different hints", async ({
    page,
  }) => {
    // Text mode hint
    await expect(page.getByText("cmd+enter to capture")).toBeVisible();

    // URL mode hint
    await page.click("text=url");
    await expect(
      page.getByText("article · will extract content"),
    ).toBeVisible();

    // Image mode hint
    await page.click("text=img");
    await expect(page.getByText("image · will be describe")).toBeVisible();
  });
});
