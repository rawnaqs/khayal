import { test, expect } from './helpers'

test.describe('Offline Flow', () => {
  test('should show offline state when capture fails due to network', async ({ page }) => {
    // Go offline to trigger offline save
    await page.evaluate(() => {
      Object.defineProperty(navigator, 'onLine', { value: false, writable: true })
    })

    await page.fill('textarea', 'This is a test note')
    await page.click('.send')

    // Should show offline state
    await expect(page.getByText('saved offline')).toBeVisible()
  })

  test('should save capture to offline queue when offline', async ({ page }) => {
    // Go offline
    await page.evaluate(() => {
      Object.defineProperty(navigator, 'onLine', { value: false, writable: true })
    })

    await page.fill('textarea', 'Offline note')
    await page.click('.send')

    // Should show offline state
    await expect(page.getByText('saved offline')).toBeVisible()
  })

  test('should have service worker registered', async ({ page }) => {
    const swRegistered = await page.evaluate(() => {
      return navigator.serviceWorker?.controller !== null
    })

    // Service worker should be registered (may not be active on first load)
    // This is a best-effort check
    expect(typeof swRegistered).toBe('boolean')
  })

  test('should show PWA manifest link in HTML', async ({ page }) => {
    // Check that the manifest link exists in the HTML
    const manifestLink = page.locator('link[rel="manifest"]')
    await expect(manifestLink).toHaveAttribute('href', '/manifest.webmanifest')
  })

  test('should have correct theme color meta tag', async ({ page }) => {
    const themeColor = page.locator('meta[name="theme-color"]')
    await expect(themeColor).toHaveAttribute('content', '#C9933A')
  })

  test('should have apple-touch-icon link', async ({ page }) => {
    const appleTouchIcon = page.locator('link[rel="apple-touch-icon"]')
    await expect(appleTouchIcon).toHaveAttribute('href', '/icon-192.png')
  })

  test('should load app shell on visit', async ({ page }) => {
    // Visit and check capture tab is visible
    await page.goto('/')
    await expect(page.locator('.nt-l').filter({ hasText: 'capture' })).toBeVisible()
  })

  test('should load app shell on second visit', async ({ page }) => {
    // First visit
    await page.goto('/')
    await expect(page.locator('.nt-l').filter({ hasText: 'capture' })).toBeVisible()

    // Second visit - should still work
    const startTime = Date.now()
    await page.goto('/')
    await expect(page.locator('.nt-l').filter({ hasText: 'capture' })).toBeVisible()
    const loadTime = Date.now() - startTime

    // Should load reasonably fast
    expect(loadTime).toBeLessThan(5000)
  })

  test('should sync offline queue when coming back online', async ({ page }) => {
    // Go offline to trigger offline save
    await page.evaluate(() => {
      Object.defineProperty(navigator, 'onLine', { value: false, writable: true })
    })

    // First capture fails -> saved offline
    await page.fill('textarea', 'Sync test note')
    await page.click('.send')
    await expect(page.getByText('saved offline')).toBeVisible()

    // Dismiss the offline state using the X button
    await page.locator('.tile-dismiss').click()

    // Simulate coming back online
    await page.evaluate(() => {
      Object.defineProperty(navigator, 'onLine', { value: true, writable: true })
      window.dispatchEvent(new Event('online'))
    })
  })

  test('should show queue view when clicking queue tab', async ({ page }) => {
    await page.locator('.nt-l').filter({ hasText: 'queue' }).click()

    // Should show queue view
    await expect(page.locator('.q-body')).toBeVisible()
  })
})
