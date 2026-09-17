import { test, expect, devices } from "@playwright/test";

// Frontend stress: inject a synthetic graph at the API layer and measure
// frame times while the physics loop runs. Keeps the backend out of it so
// we isolate rendering cost.
test.describe.configure({ mode: "serial" });
test.beforeEach(({}, testInfo) => {
  test.skip(testInfo.project.name !== "chromium", "stress sim runs on the chromium project");
  // heavy (software-rendered WebGL): opt in with KHAYAL_STRESS=1
  test.skip(!process.env.KHAYAL_STRESS, "set KHAYAL_STRESS=1 to run graph stress");
});

const BASE = "http://localhost:5173";

function syntheticGraph(notes: number, people: number, edges: number) {
  const nodes: any[] = [];
  for (let i = 0; i < notes; i++) {
    nodes.push({
      id: `khayal/note-${i}.md`,
      kind: "note",
      name: `Note ${i}`,
      type: "text",
      created: new Date(2026, 0, 1 + (i % 300)).toISOString(),
    });
  }
  for (let p = 0; p < people; p++) {
    nodes.push({ id: `person:person-${p}`, kind: "person", name: `Person ${p}` });
  }
  const seen = new Set<string>();
  const out: any[] = [];
  for (let e = 0; e < edges; e++) {
    const s = `khayal/note-${e % notes}.md`;
    const t = `khayal/note-${(e * 7 + 1) % notes}.md`;
    if (s === t) continue;
    const key = s + "\u0000" + t;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push({ source: s, target: t, types: ["similar"] });
  }
  return { nodes, edges: out };
}

async function setup(page: import("@playwright/test").Page, graph: unknown) {
  await page.route("**/v1/graph", (route) => route.fulfill({ json: graph }));
  await page.addInitScript(() => {
    localStorage.setItem("khayal_token", "abc");
    localStorage.setItem("khayal_host", location.origin);
  });
  await page.goto(BASE);
}

function stats(frames: number[]) {
  const sorted = [...frames].sort((a, b) => a - b);
  const avg = frames.reduce((a, b) => a + b, 0) / frames.length;
  const p = (q: number) => sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * q))];
  return { avg, p50: p(0.5), p95: p(0.95), max: sorted[sorted.length - 1], fps: 1000 / avg };
}

async function startSampling(page: import("@playwright/test").Page) {
  await page.evaluate(() => {
    (window as any).__frames = [];
    (window as any).__stopFrames = false;
    let last = performance.now();
    const tick = (t: number) => {
      (window as any).__frames.push(t - last);
      last = t;
      if (!(window as any).__stopFrames) requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  });
}

async function stopSampling(page: import("@playwright/test").Page) {
  return page.evaluate(() => {
    (window as any).__stopFrames = true;
    return (window as any).__frames.slice(1);
  });
}

async function measureFrames(page: import("@playwright/test").Page, ms: number) {
  await startSampling(page);
  await page.waitForTimeout(ms);
  return stopSampling(page);
}

async function loadGraph(page: import("@playwright/test").Page) {
  const t0 = Date.now();
  await page.locator("nav .nt", { hasText: "graph" }).click();
  await page.waitForSelector('[data-testid="graph-canvas"] canvas', { timeout: 30000 });
  const ttfm = Date.now() - t0;
  await page.waitForTimeout(2500);
  return ttfm;
}

for (const shape of [
  { name: "moderate", notes: 480, people: 20, edges: 2000 },
  { name: "heavy", notes: 480, people: 20, edges: 8000 },
  { name: "extreme", notes: 480, people: 20, edges: 22000 },
]) {
  test(`stress ${shape.name}: ${shape.notes + shape.people} nodes / ${shape.edges} edges`, async ({ browser }) => {
    const context = await browser.newContext({ ...devices["Desktop Chrome"], baseURL: BASE });
    const page = await context.newPage();
    await setup(page, syntheticGraph(shape.notes, shape.people, shape.edges));

    const ttfm = await loadGraph(page);
    const frames = await measureFrames(page, 3000);
    const s = stats(frames);

    // hover a node: this changes settings, forcing the reducers to run over
    // every edge/node — where an O(E²) reducer shows up as a frame spike
    const node = await page.evaluate(() => {
      const d = (window as any).__graphDebug;
      const rect = d.canvasRect();
      const id = d.nodeIds()[0];
      const vp = d.nodeViewport(id);
      return { x: rect.left + vp.x, y: rect.top + vp.y };
    });
    await startSampling(page);
    await page.mouse.move(node.x, node.y, { steps: 2 });
    await page.waitForTimeout(1500);
    const hoverFrames = await stopSampling(page);
    const h = stats(hoverFrames);

    console.log(
      `STRESS ${shape.name} nodes=${shape.notes + shape.people} edges=${shape.edges}`,
      `ttfm=${ttfm}ms idle[avg=${s.avg.toFixed(1)} p95=${s.p95.toFixed(1)} max=${s.max.toFixed(0)} fps=${s.fps.toFixed(1)}]`,
      `hover[avg=${h.avg.toFixed(1)} p95=${h.p95.toFixed(1)} max=${h.max.toFixed(0)}ms]`,
    );
    expect(frames.length).toBeGreaterThan(5);
    await context.close();
  });
}
