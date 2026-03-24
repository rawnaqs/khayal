import { test, expect } from './helpers'

test.describe('Search Flow', () => {
  test('should navigate to search tab', async ({ page }) => {
    await page.locator('.nt-l').filter({ hasText: 'search' }).click()

    // Should show search view
    await expect(page.getByPlaceholder('Search your vault...')).toBeVisible()
  })

  test('should show search mode chips', async ({ page }) => {
    await page.locator('.nt-l').filter({ hasText: 'search' }).click()

    await expect(page.locator('.mc').filter({ hasText: 'hybrid' })).toBeVisible()
    await expect(page.locator('.mc').filter({ hasText: 'keyword' })).toBeVisible()
    await expect(page.locator('.mc').filter({ hasText: 'semantic' })).toBeVisible()
  })

  test('should show suggestion chips in idle state', async ({ page }) => {
    await page.locator('.nt-l').filter({ hasText: 'search' }).click()

    await expect(page.locator('.sc').filter({ hasText: 'people' })).toBeVisible()
    await expect(page.locator('.sc').filter({ hasText: 'payments' })).toBeVisible()
    await expect(page.locator('.sc').filter({ hasText: 'this week' })).toBeVisible()
  })

  test('should perform search on enter', async ({ page }) => {
    // Mock the search API
    await page.route('**/v1/search**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          query: 'test',
          mode: 'hybrid',
          results: [
            {
              id: '1',
              note_path: 'khayal/2024-01-01/test.md',
              title: 'Test Note',
              excerpt: 'This is a test note with some content',
              score: 0.95,
              type: 'text',
              created_at: '2024-01-01T10:00:00Z',
              tags: ['test', 'example'],
            },
          ],
          total: 1,
          took_ms: 100,
        }),
      })
    })

    await page.locator('.nt-l').filter({ hasText: 'search' }).click()
    await page.fill('input[placeholder="Search your vault..."]', 'test')
    await page.press('input[placeholder="Search your vault..."]', 'Enter')

    // Should show results
    await expect(page.locator('.r1-title')).toBeVisible()
    await expect(page.getByText('1 results')).toBeVisible()
  })

  test('should show loading state during search', async ({ page }) => {
    // Delay the API response
    await page.route('**/v1/search**', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 500))
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          query: 'test',
          mode: 'hybrid',
          results: [],
          total: 0,
          took_ms: 100,
        }),
      })
    })

    await page.locator('.nt-l').filter({ hasText: 'search' }).click()
    await page.fill('input[placeholder="Search your vault..."]', 'test')
    await page.press('input[placeholder="Search your vault..."]', 'Enter')

    // Should show loading state - check for shimmer or loading indicator
    // The SearchView uses animate-shimmer class for loading skeletons
    await expect(page.locator('.animate-shimmer').first()).toBeVisible({ timeout: 2000 })
  })

  test('should show no results state', async ({ page }) => {
    await page.route('**/v1/search**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          query: 'nonexistent',
          mode: 'hybrid',
          results: [],
          total: 0,
          took_ms: 50,
        }),
      })
    })

    await page.locator('.nt-l').filter({ hasText: 'search' }).click()
    await page.fill('input[placeholder="Search your vault..."]', 'nonexistent')
    await page.press('input[placeholder="Search your vault..."]', 'Enter')

    await expect(page.getByText('nothing found')).toBeVisible()
  })

  test('should show filter chips after search', async ({ page }) => {
    await page.route('**/v1/search**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          query: 'test',
          mode: 'hybrid',
          results: [
            { id: '1', title: 'Note 1', type: 'text', score: 0.9, tags: [], created_at: '2024-01-01', note_path: 'test.md', excerpt: 'test' },
            { id: '2', title: 'Note 2', type: 'article', score: 0.8, tags: [], created_at: '2024-01-01', note_path: 'test.md', excerpt: 'test' },
          ],
          total: 2,
          took_ms: 100,
        }),
      })
    })

    await page.locator('.nt-l').filter({ hasText: 'search' }).click()
    await page.fill('input[placeholder="Search your vault..."]', 'test')
    await page.press('input[placeholder="Search your vault..."]', 'Enter')

    // Should show filter chips - use .fc class to target only filter chips
    await expect(page.locator('.fc').filter({ hasText: 'all' })).toBeVisible()
    await expect(page.locator('.fc').filter({ hasText: 'text' })).toBeVisible()
    await expect(page.locator('.fc').filter({ hasText: 'article' })).toBeVisible()
    await expect(page.locator('.fc').filter({ hasText: 'image' })).toBeVisible()
  })

  test('should clear search', async ({ page }) => {
    await page.locator('.nt-l').filter({ hasText: 'search' }).click()
    await page.fill('input[placeholder="Search your vault..."]', 'test')

    // Should show clear button
    const clearButton = page.locator('.srch-clear')
    await expect(clearButton).toBeVisible()

    await clearButton.click()

    // Should clear input
    await expect(page.locator('input[placeholder="Search your vault..."]')).toHaveValue('')
  })

  test('should show recent searches after performing search', async ({ page }) => {
    await page.route('**/v1/search**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          query: 'test',
          mode: 'hybrid',
          results: [],
          total: 0,
          took_ms: 50,
        }),
      })
    })

    await page.locator('.nt-l').filter({ hasText: 'search' }).click()
    await page.fill('input[placeholder="Search your vault..."]', 'test')
    await page.press('input[placeholder="Search your vault..."]', 'Enter')

    // Clear search to go back to idle state
    await page.locator('.srch-clear').click()

    // Should show recent searches
    await expect(page.getByText('recent searches')).toBeVisible()
    await expect(page.locator('.recent-item').filter({ hasText: 'test' })).toBeVisible()
  })

  test('should change search mode', async ({ page }) => {
    await page.locator('.nt-l').filter({ hasText: 'search' }).click()

    // Click on keyword mode
    await page.locator('.mc').filter({ hasText: 'keyword' }).click()

    // Should show keyword as active
    const keywordChip = page.locator('.mc').filter({ hasText: 'keyword' })
    await expect(keywordChip).toHaveClass(/on/)
  })
})
