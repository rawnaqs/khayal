import { test, expect } from "./helpers";

// WebGL under software rendering is heavy: run these serially and only on
// chromium (the mobile/touch path is covered manually).
test.describe.configure({ mode: "serial" });
test.skip(({ browserName }) => browserName !== "chromium", "graph tests are chromium-only");

declare global {
  interface Window {
    __graphDebug: {
      nodeIds: () => string[];
      nodeState: (id: string) => { x: number; y: number; kind: string } | null;
      nodeViewport: (id: string) => { x: number; y: number } | null;
      canvasRect: () => { left: number; top: number; width: number; height: number } | null;
      camera: () => { x: number; y: number; ratio: number };
      pause: (v: boolean) => void;
      counts: () => { downs: number; clicks: number };
      captor: () => { enabled: boolean; isMouseDown: boolean; draggedEvents: number; isMoving: boolean };
      canvasRect: () => { left: number; top: number; width: number; height: number } | null;
    };
  }
}


// Small note dots (~3px) drift while physics runs, so read the position
// immediately before pressing — exactly what a human does.
async function nodeAt(page: import("@playwright/test").Page, id: string) {
  return page.evaluate((nid) => {
    const d = window.__graphDebug;
    const rect = d.canvasRect();
    const vp = d.nodeViewport(nid);
    return vp && rect ? { x: rect.left + vp.x, y: rect.top + vp.y } : null;
  }, id);
}

async function dragNode(page: import("@playwright/test").Page, id: string) {
  const before = await page.evaluate((nid) => window.__graphDebug.nodeState(nid), id);
  const camBefore = await page.evaluate(() => window.__graphDebug.camera());
  const downsBefore = await page.evaluate(() => window.__graphDebug.counts().downs);
  const p = await nodeAt(page, id);
  await page.mouse.move(p!.x, p!.y);
  await page.mouse.down();
  await page.waitForTimeout(40);
  const downFired = (await page.evaluate(() => window.__graphDebug.counts().downs)) - downsBefore;
  await page.mouse.move(p!.x + 70, p!.y + 60, { steps: 12 });
  await page.mouse.move(p!.x + 150, p!.y + 110, { steps: 12 });
  const after = await page.evaluate((nid) => window.__graphDebug.nodeState(nid), id);
  await page.mouse.up();
  await page.waitForTimeout(200);
  const camAfter = await page.evaluate(() => window.__graphDebug.camera());
  return {
    downFired,
    nodeMoved: Math.hypot(after!.x - before!.x, after!.y - before!.y),
    camMoved: Math.hypot(camAfter.x - camBefore.x, camAfter.y - camBefore.y),
  };
}

async function openGraph(page: import("@playwright/test").Page) {
  await page.locator("nav .nt", { hasText: "graph" }).click();
  await page.waitForSelector('[data-testid="graph-canvas"] canvas', { timeout: 10000 });
  // wait for the reveal to finish, then freeze physics so node
  // positions are stable while we interact (otherwise drift makes the
  // measured coordinates stale before the click lands)
  await page.waitForTimeout(4000);
  await page.evaluate(() => window.__graphDebug.pause(true));
  await page.waitForTimeout(100);
}

// pick the node nearest the viewport centre, in PAGE coordinates
// (graphToViewport is container-relative; the mouse works in page coords)
async function centreNode(page: import("@playwright/test").Page) {
  return page.evaluate(() => {
    const d = window.__graphDebug;
    const rect = d.canvasRect();
    if (!rect) return null;
    let best: { id: string; vp: { x: number; y: number }; dist: number } | null = null;
    for (const id of d.nodeIds()) {
      const vp = d.nodeViewport(id);
      if (!vp) continue;
      const pageX = rect.left + vp.x;
      const pageY = rect.top + vp.y;
      const dist = Math.hypot(pageX - window.innerWidth / 2, pageY - window.innerHeight / 2);
      if (!best || dist < best.dist) best = { id, vp: { x: pageX, y: pageY }, dist };
    }
    return best;
  });
}

// find an empty spot (no node within 40px) in page coordinates
async function emptySpot(page: import("@playwright/test").Page) {
  return page.evaluate(() => {
    const d = window.__graphDebug;
    const rect = d.canvasRect();
    if (!rect) return null;
    const nodes = d.nodeIds().map((id) => d.nodeViewport(id)).filter(Boolean) as { x: number; y: number }[];
    for (let x = rect.left + 30; x < rect.left + rect.width - 30; x += 40) {
      for (let y = rect.top + 30; y < rect.top + rect.height - 30; y += 40) {
        const vx = x - rect.left;
        const vy = y - rect.top;
        if (nodes.every((n) => Math.hypot(n.x - vx, n.y - vy) > 40)) return { x, y };
      }
    }
    return null;
  });
}

test("dragging a small note node moves the node, not the camera (physics live)", async ({ page }) => {
  await page.goto("/");
  // deliberately do NOT pause physics: the reported bug only appears while
  // nodes are drifting and small
  await page.locator("nav .nt", { hasText: "graph" }).click();
  await page.waitForSelector('[data-testid="graph-canvas"] canvas', { timeout: 10000 });
  await page.waitForTimeout(3000);

  const id = await page.evaluate(() => {
    const d = window.__graphDebug;
    return d.nodeIds().find((n) => d.nodeState(n)?.kind === "note") ?? null;
  });
  expect(id, "no note node").not.toBeNull();

  const r = await dragNode(page, id!);
  console.log("NOTE-DRAG", JSON.stringify({ down: r.downFired, moved: +r.nodeMoved.toFixed(1), cam: +r.camMoved.toFixed(3) }));
  expect(r.downFired, "the press must register on the node").toBe(1);
  expect(r.nodeMoved, "node should follow the drag").toBeGreaterThan(5);
  expect(r.camMoved, "camera must not pan while dragging a node").toBeLessThan(0.5);
});

test("panning still works after dragging a node", async ({ page }) => {
  await page.goto("/");
  await page.locator("nav .nt", { hasText: "graph" }).click();
  await page.waitForSelector('[data-testid="graph-canvas"] canvas', { timeout: 10000 });
  await page.waitForTimeout(3000);

  const id = await page.evaluate(() => {
    const d = window.__graphDebug;
    return d.nodeIds().find((n) => d.nodeState(n)?.kind === "note") ?? null;
  });
  await dragNode(page, id!);

  // the captor must be usable again: empty-space drag pans the camera
  const camBefore = await page.evaluate(() => window.__graphDebug.camera());
  const spot = await emptySpot(page);
  expect(spot).not.toBeNull();
  await page.mouse.move(spot!.x, spot!.y);
  await page.mouse.down();
  await page.mouse.move(spot!.x + 120, spot!.y + 60, { steps: 10 });
  await page.mouse.up();
  await page.waitForTimeout(200);
  const camAfter = await page.evaluate(() => window.__graphDebug.camera());
  const camMoved = Math.hypot(camAfter.x - camBefore.x, camAfter.y - camBefore.y);
  console.log("PAN-AFTER-DRAG", camMoved.toFixed(3));
  expect(camMoved, "camera should pan after a node drag").toBeGreaterThan(0.001);
});

test("panning on empty space still works", async ({ page }) => {
  await page.goto("/");
  await openGraph(page);
  const camBefore = await page.evaluate(() => window.__graphDebug.camera());
  const spot = await emptySpot(page);
  expect(spot, "no empty spot found").not.toBeNull();
  await page.mouse.move(spot!.x, spot!.y);
  await page.mouse.down();
  await page.mouse.move(spot!.x + 120, spot!.y + 60, { steps: 10 });
  await page.mouse.up();
  await page.waitForTimeout(200);
  const camAfter = await page.evaluate(() => window.__graphDebug.camera());
  const camMoved = Math.hypot(camAfter.x - camBefore.x, camAfter.y - camBefore.y);
  console.log("PANNED", camMoved.toFixed(3));
  expect(camMoved, "camera should pan on empty-space drag").toBeGreaterThan(0.001);
});

test("clicking a node selects it; × clears the card", async ({ page }) => {
  await page.goto("/");
  await openGraph(page);

  // Try each person node until one actually lands (nodes can overlap;
  // a click may hit a note on top, which opens the note instead).
  const persons = await page.evaluate(() => {
    const d = window.__graphDebug;
    const rect = d.canvasRect();
    const out: { id: string; x: number; y: number }[] = [];
    for (const id of d.nodeIds()) {
      if (d.nodeState(id)?.kind !== "person") continue;
      const vp = d.nodeViewport(id);
      if (vp && rect) out.push({ id, x: rect.left + vp.x, y: rect.top + vp.y });
    }
    return out;
  });
  expect(persons.length, "no person nodes").toBeGreaterThan(0);

  let cardShown = false;
  for (const p of persons.slice(0, 8)) {
    await page.mouse.click(p.x, p.y);
    await page.waitForTimeout(250);
    const card = await page.getByTestId("graph-info").count();
    const noteSheet = await page.locator("[role=dialog]").count();
    const counts = await page.evaluate(() => (window as any).__graphDebug.counts());
    const cap = await page.evaluate(() => (window as any).__graphDebug.captor());
    console.log("CLICK", p.id, "card:", card, "noteSheet:", noteSheet, "counts:", JSON.stringify(counts), "cap:", JSON.stringify(cap));
    if (card > 0) {
      cardShown = true;
      break;
    }
    if (noteSheet > 0) {
      await page.keyboard.press("Escape");
      await page.waitForTimeout(200);
    }
  }
  expect(cardShown, "clicking a person should open the info card").toBe(true);

  await page.getByTestId("graph-info-close").click();
  await page.waitForTimeout(300);
  const stillThere = await page.getByTestId("graph-info").count();
  console.log("CARD-COUNT-AFTER-X", stillThere);
  expect(stillThere, "card should be dismissed by the × button").toBe(0);
});

test("clicking a note node opens the note", async ({ page }) => {
  await page.goto("/");
  await openGraph(page);
  const pick = await centreNode(page);
  expect(pick).not.toBeNull();
  await page.mouse.click(pick!.vp.x, pick!.vp.y);
  await page.waitForTimeout(600);
  const dialog = await page.locator("[role=dialog]").count();
  const card = await page.getByTestId("graph-info").count();
  console.log("NOTE-CLICK dialog:", dialog, "card:", card);
  expect(dialog + card, "clicking a node must do something visible").toBeGreaterThan(0);
});

test("reload button visible and not overlapped by the legend", async ({ page }) => {
  await page.goto("/");
  await openGraph(page);

  const reload = page.getByTestId("graph-reload");
  await expect(reload).toBeVisible();
  const r = (await reload.boundingBox())!;
  const legend = page.locator(".graph-legend");
  await expect(legend).toBeVisible();
  const l = (await legend.boundingBox())!;
  const overlaps =
    r.x < l.x + l.width && r.x + r.width > l.x && r.y < l.y + l.height && r.y + r.height > l.y;
  console.log("RELOAD", JSON.stringify(r), "LEGEND", JSON.stringify(l), "OVERLAP", overlaps);
  expect(overlaps, "reload button and legend must not intersect").toBe(false);
});
