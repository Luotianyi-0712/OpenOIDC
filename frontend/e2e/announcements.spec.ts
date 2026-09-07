import { test, expect, type Page } from '@playwright/test'
import type { Announcement } from '../src/types/announcement'

const content = '# Release details\n\n**Important update** with [documentation](https://example.test/docs).\n\n- First change\n- Second change\n\n| Item | State |\n| --- | --- |\n| Login | Ready |\n\n```js\nconst ready = true\n```\n\n<script>window.announcementXss = true</script>\n<img src=x onerror="window.announcementXss=true">\n\n[unsafe](javascript:alert(1))'

function announcement(overrides: Partial<Announcement> = {}): Announcement {
  return {
    id: 'announcement-1', title: 'September release', content,
    display_mode: 'both', dismissible: true, scrolling: false, is_published: true,
    revision: 'revision-1', created_at: '2026-09-08T08:00:00Z', updated_at: '2026-09-08T08:00:00Z', ...overrides,
  }
}

async function mockAPI(page: Page, initial: Announcement[] = [], role = 'admin') {
  let items = [...initial]
  let userID = 'account-1'
  let sequence = 10
  let failPublic = false
  await page.addInitScript(() => localStorage.setItem('locale', 'en'))
  await page.route('**/api/v1/**', async route => {
    const path = new URL(route.request().url()).pathname.replace('/api/v1', '')
    const method = route.request().method()
    let data: unknown = {}
    if (path === '/announcements') {
      if (failPublic) return route.fulfill({ status: 503, json: { success: false, error: { message: 'Test outage' } } })
      data = items.filter(item => item.is_published)
    } else if (path.startsWith('/admin/announcements')) {
      if (method === 'GET') data = items
      else if (method === 'POST') {
        data = announcement({ ...route.request().postDataJSON(), id: `announcement-${++sequence}`, revision: `revision-${sequence}` })
        items.unshift(data as Announcement)
      } else if (method === 'PUT') {
        const id = path.split('/').pop()
        items = items.map(item => item.id === id ? { ...item, ...route.request().postDataJSON(), revision: `revision-${++sequence}` } : item)
        data = items.find(item => item.id === id)
      } else if (method === 'DELETE') {
        items = items.filter(item => item.id !== path.split('/').pop())
        data = { deleted: true }
      }
    } else if (path === '/auth/status') data = { authenticated: true }
    else if (path === '/me') data = { id: userID, uid: 1, role, display_name: 'Test account', email: 'test@example.test', email_verified: true, security_level: 1 }
    else if (path === '/developer/status') data = { can_access: false, can_create: false, current_trust_level: 1, min_trust_level: 1 }
    else if (path === '/settings/public') data = { captcha_enabled: 'false', social_login_enabled: 'false', password_login_enabled: 'true', registration_enabled: 'true' }
    else if (path === '/social/providers') data = []
    else if (path === '/settings/password-policy') data = { min_length: 8 }
    await route.fulfill({ json: { success: true, data } })
  })
  return {
    update: (next: Announcement[]) => { items = next },
    switchAccount: () => { userID = 'account-2' },
    fail: (value: boolean) => { failPublic = value },
  }
}

test('read state, dismissal, Markdown safety and updated revisions', async ({ page }, testInfo) => {
  const api = await mockAPI(page, [announcement()])
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.goto('/admin/announcements')
  await expect(page.getByTestId('announcement-banner')).toBeVisible()
  await expect(page.getByTestId('announcement-unread')).toBeVisible()
  await page.getByTestId('announcement-bell').click()
  await page.getByRole('dialog').getByRole('button', { name: /September release/ }).click()
  await expect(page.getByRole('dialog').getByRole('heading', { name: 'Release details' })).toBeVisible()
  await expect(page.getByRole('dialog').locator('strong')).toHaveText('Important update')
  await expect(page.getByRole('dialog').locator('table')).toBeVisible()
  await expect(page.getByRole('dialog').locator('pre')).toContainText('const ready = true')
  await expect(page.getByRole('dialog').locator('script, [onerror], a[href^="javascript:"]')).toHaveCount(0)
  expect(await page.evaluate(() => (window as unknown as Record<string, unknown>).announcementXss)).toBeUndefined()
  await page.screenshot({ path: testInfo.outputPath('notification-window.png'), fullPage: true })
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.getByTestId('announcement-bell')).toBeFocused()
  await expect(page.getByTestId('announcement-unread')).toHaveCount(0)
  await page.getByRole('button', { name: 'Dismiss banner', exact: true }).click()
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Announcements', exact: true })).toBeVisible()
  await expect(page.getByTestId('announcement-banner')).toHaveCount(0)
  await expect(page.getByTestId('announcement-unread')).toHaveCount(0)
  api.update([announcement({ revision: 'revision-2' })])
  await page.reload()
  await expect(page.getByTestId('announcement-banner')).toBeVisible()
  await expect(page.getByTestId('announcement-unread')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  expect(errors).toEqual([])
})

test('administrator can preview, publish, withdraw and delete', async ({ page }, testInfo) => {
  await mockAPI(page)
  await page.goto('/admin/announcements')
  await page.getByRole('button', { name: 'New announcement' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Title', { exact: true }).fill('Maintenance notice')
  await dialog.getByRole('textbox', { name: 'Content (Markdown)', exact: true }).fill(content)
  await dialog.getByRole('button', { name: 'Both', exact: true }).click()
  await dialog.getByLabel('Scrolling banner').check()
  await dialog.getByRole('tab', { name: 'Preview' }).click()
  await expect(dialog.getByRole('heading', { name: 'Release details' })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('announcement-editor.png'), fullPage: true })
  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.getByRole('cell', { name: 'Draft', exact: true })).toBeVisible()
  await expect(page.getByTestId('announcement-banner')).toHaveCount(0)
  await page.getByRole('button', { name: 'Publish', exact: true }).click()
  await expect(page.getByTestId('announcement-banner')).toContainText('Maintenance notice')
  await expect(page.getByTestId('announcement-unread')).toBeVisible()
  await page.getByRole('button', { name: 'Unpublish', exact: true }).click()
  await expect(page.getByTestId('announcement-banner')).toHaveCount(0)
  page.once('dialog', dialog => dialog.accept())
  await page.getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(page.getByText('No announcements yet', { exact: true })).toBeVisible()
})

test('banner options, reduced motion, account isolation and load errors', async ({ page }, testInfo) => {
  const title = 'Service update: '.repeat(18)
  const api = await mockAPI(page, [announcement({ title, dismissible: false, scrolling: true })])
  await page.goto('/login')
  const banner = page.getByTestId('announcement-banner')
  await expect(banner).toBeVisible()
  await expect(banner.getByRole('button', { name: 'Dismiss banner' })).toHaveCount(0)
  await expect(banner.locator('.banner-title')).toHaveClass(/is-scrolling/)
  await page.mouse.move(0, 400)
  const before = await banner.locator('.banner-text').evaluate(el => getComputedStyle(el).transform)
  expect(await banner.locator('.banner-text').evaluate(el => getComputedStyle(el).animationName)).toMatch(/^announcement-scroll/)
  await banner.locator('.banner-text').evaluate(el => {
    const animation = el.getAnimations()[0]
    animation.currentTime = Number(animation.effect!.getTiming().duration) / 2
  })
  await expect.poll(() => banner.locator('.banner-text').evaluate(el => getComputedStyle(el).transform)).not.toBe(before)
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await expect(banner.locator('.banner-text')).toHaveCSS('animation-name', 'none')
  await page.screenshot({ path: testInfo.outputPath('announcement-banner.png'), fullPage: true })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.getByTestId('announcement-bell').click()
  await page.getByRole('button', { name: 'Mark all as read' }).click()
  await page.keyboard.press('Escape')
  api.switchAccount()
  await page.reload()
  await expect(page.getByTestId('announcement-unread')).toBeVisible()
  api.fail(true)
  await page.getByTestId('announcement-bell').click()
  await expect(page.getByRole('alert')).toContainText('Could not load announcements')
  api.fail(false)
  await page.getByRole('button', { name: 'Retry', exact: true }).click()
  await expect(page.getByRole('alert')).toHaveCount(0)
})

test('display modes, multiple banners and responsive navigation', async ({ page }, testInfo) => {
  await mockAPI(page, [
    announcement({ display_mode: 'banner', title: 'First banner' }),
    announcement({ id: 'announcement-2', display_mode: 'banner', title: 'Second banner' }),
    announcement({ id: 'announcement-3', display_mode: 'modal', title: 'Window only' }),
  ])
  await page.goto('/privacy')
  const banner = page.getByTestId('announcement-banner')
  await expect(banner).toContainText('First banner')
  await banner.getByRole('button', { name: 'Next announcement' }).click()
  await expect(banner).toContainText('Second banner')
  await banner.getByRole('button', { name: 'Previous announcement' }).click()
  await expect(banner).toContainText('First banner')
  await page.getByTestId('announcement-bell').click()
  await expect(page.getByRole('dialog').getByRole('button', { name: /Window only/ })).toBeVisible()
  await expect(page.getByRole('dialog').getByRole('button', { name: /First banner|Second banner/ })).toHaveCount(0)
  await page.keyboard.press('Escape')

  for (const width of [320, 768, 1024, 1440]) {
    await page.setViewportSize({ width, height: 800 })
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    const overlaps = await page.getByRole('navigation').first().evaluate(nav => {
      const controls = Array.from(nav.querySelectorAll('a, button')).map(el => ({
        name: el.textContent?.trim() || el.getAttribute('aria-label'), rect: el.getBoundingClientRect(),
      })).filter(({ rect }) => rect.width && rect.height)
      return controls.flatMap((a, index) => controls.slice(index + 1).filter(b =>
        Math.min(a.rect.right, b.rect.right) - Math.max(a.rect.left, b.rect.left) > 1 &&
        Math.min(a.rect.bottom, b.rect.bottom) - Math.max(a.rect.top, b.rect.top) > 1,
      ).map(b => `${a.name} / ${b.name}`))
    })
    expect(overlaps, `navigation controls at ${width}px`).toEqual([])
  }
  await page.screenshot({ path: testInfo.outputPath('multiple-banners.png'), fullPage: true })
})
