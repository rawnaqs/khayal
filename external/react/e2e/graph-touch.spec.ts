import { test, expect, devices } from "@playwright/test";

// Touch simulation needs CDP multi-touch, which we drive on desktop
// chromium with a touch-enabled context.
test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.name !== "chromium", "CDP touch sim runs on the chromium project");
});

const BASE = "http://localhost:5173";

async function setup(page: import("@playwright/test").Page) {
  await page.addInitScript(() => {
    localStorage.setItem("khayal_token", "abc");
    localStorage.setItem("khayal_host", location.origin);
  });
  await page.goto(BASE);
  await page.locator("nav .nt", { hasText: "graph" }).click();
  await page.waitForSelector('[data-testid="graph-canvas"] canvas', { timeout: 15000 });
  await page.waitForTimeout(3000);
}

type Point = { x: number; y: number; id?: number };
async function touch(cdp: any, type: string, points: Point[]) {
  await cdp.send("Input.dispatchTouchEvent", {
    type,
    touchPoints: points.map((p, i) => ({
      x: p.x,
      y: p.y,
      id: p.id ?? i,
      radiusX: 8,
      radiusY: 8,
      force: 1,
    })),
  });
}

async function nodeSpot(page: import("@playwright/test").Page, kind: string) {
  return page.evaluate((k) => {
    const d = (window as any).__graphDebug;
    const rect = d.canvasRect();
    const id = d.nodeIds().find((n: string) => d.nodeState(n)?.kind === k);
    if (!id) return null;
    const vp = d.nodeViewport(id);
    return vp && rect ? { id, x: rect.left + vp.x, y: rect.top + vp.y } : null;
  }, kind);
}

async function emptySpot(page: import("@playwright/test").Page) {
  return page.evaluate(() => {
    const d = (window as any).__graphDebug;
    const rect = d.canvasRect();
    const nodes = d.nodeIds().map((id: string) => d.nodeViewport(id)).filter(Boolean) as { x: number; y: number }[];
    for (let x = rect.left + 30; x < rect.left + rect.width - 30; x += 40) {
      for (let y = rect.top + 30; y < rect.top + rect.height - 30; y += 40) {
        const vx = x - rect.left, vy = y - rect.top;
        if (nodes.every((n) => Math.hypot(n.x - vx, n.y - vy) > 45)) return { x, y };
      }
    }
    return null;
  });
}

const cam = (page: import("@playwright/test").Page) =>
  page.evaluate(() => (window as any).__graphDebug.camera());
const state = (page: import("@playwright/test").Page, id: string) =>
  page.evaluate((nid) => (window as any).__graphDebug.nodeState(nid), id);

test("one finger drags a node without panning the view", async ({ browser }) => {
  const context = await browser.newContext({ ...devices["iPhone 13"], baseURL: BASE });
  const page = await context.newPage();
  await setup(page);
  const cdp = await context.newCDPSession(page);

  const spot = (await nodeSpot(page, "note"))!;
  expect(spot).not.toBeNull();
  const before = await state(page, spot.id);
  const c0 = await cam(page);

  await touch(cdp, "touchStart", [{ x: spot.x, y: spot.y, id: 1 }]);
  await page.waitForTimeout(40);
  await touch(cdp, "touchMove", [{ x: spot.x + 40, y: spot.y + 30, id: 1 }]);
  await page.waitForTimeout(40);
  await touch(cdp, "touchMove", [{ x: spot.x + 90, y: spot.y + 70, id: 1 }]);
  await page.waitForTimeout(40);
  const mid = await state(page, spot.id);
  await touch(cdp, "touchEnd", []);
  await page.waitForTimeout(200);
  const c1 = await cam(page);

  const moved = Math.hypot(mid.x - before.x, mid.y - before.y);
  const camMoved = Math.hypot(c1.x - c0.x, c1.y - c0.y);
  console.log("TOUCH-1FINGER-NODE moved", moved.toFixed(1), "cam", camMoved.toFixed(3));
  expect(moved, "one finger should drag the node").toBeGreaterThan(5);
  expect(camMoved, "one finger must not pan the view").toBeLessThan(0.5);
  await context.close();
});

test("one finger on empty space does not pan the view", async ({ browser }) => {
  const context = await browser.newContext({ ...devices["iPhone 13"], baseURL: BASE });
  const page = await context.newPage();
  await setup(page);
  const cdp = await context.newCDPSession(page);

  const spot = (await emptySpot(page))!;
  expect(spot).not.toBeNull();
  const c0 = await cam(page);
  await touch(cdp, "touchStart", [{ x: spot.x, y: spot.y, id: 1 }]);
  await touch(cdp, "touchMove", [{ x: spot.x + 50, y: spot.y + 40, id: 1 }]);
  await page.waitForTimeout(40);
  await touch(cdp, "touchMove", [{ x: spot.x + 110, y: spot.y + 80, id: 1 }]);
  await page.waitForTimeout(40);
  await touch(cdp, "touchEnd", []);
  await page.waitForTimeout(200);
  const c1 = await cam(page);

  const camMoved = Math.hypot(c1.x - c0.x, c1.y - c0.y);
  console.log("TOUCH-1FINGER-EMPTY cam", camMoved.toFixed(3));
  expect(camMoved, "one finger must never pan").toBeLessThan(0.001);
  await context.close();
});

test("two fingers pan the view", async ({ browser }) => {
  const context = await browser.newContext({ ...devices["iPhone 13"], baseURL: BASE });
  const page = await context.newPage();
  await setup(page);
  const cdp = await context.newCDPSession(page);

  const spot = (await emptySpot(page))!;
  expect(spot).not.toBeNull();
  const c0 = await cam(page);
  await touch(cdp, "touchStart", [
    { x: spot.x, y: spot.y, id: 1 },
    { x: spot.x + 60, y: spot.y + 40, id: 2 },
  ]);
  await page.waitForTimeout(40);
  await touch(cdp, "touchMove", [
    { x: spot.x + 60, y: spot.y + 50, id: 1 },
    { x: spot.x + 120, y: spot.y + 90, id: 2 },
  ]);
  await page.waitForTimeout(40);
  await touch(cdp, "touchMove", [
    { x: spot.x + 110, y: spot.y + 90, id: 1 },
    { x: spot.x + 170, y: spot.y + 130, id: 2 },
  ]);
  await page.waitForTimeout(40);
  await touch(cdp, "touchEnd", []);
  await page.waitForTimeout(200);
  const c1 = await cam(page);

  const camMoved = Math.hypot(c1.x - c0.x, c1.y - c0.y);
  console.log("TOUCH-2FINGER cam", camMoved.toFixed(3));
  expect(camMoved, "two fingers should pan the view").toBeGreaterThan(0.001);
  await context.close();
});
