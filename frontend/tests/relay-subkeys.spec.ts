import { test, expect } from '@playwright/test'
import type { Page } from '@playwright/test'
import { gotoAndEnsureAuth } from './auth.utils'

declare const process: {
  env: Record<string, string | undefined>
}

async function seedSelectedProject(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('axonhub_selected_project_id', 'project-alpha')
  })
}

test.describe('Relay Sub-Key E2E', () => {
  test('operator relay subkeys page is accessible', async ({ page }) => {
    await gotoAndEnsureAuth(page, '/relay-subkeys')

    expect(page.url()).not.toContain('/500')
    expect(page.url()).not.toContain('/sign-in')
    await expect(page.getByRole('heading', { name: 'Relay Sub-Key Operations' })).toBeVisible()
    await expect(page.getByText(/Operator overview|Active products|Failed traces/i).first()).toBeVisible()
  })

  test('project relay subkeys page is accessible with a selected project', async ({ page }) => {
    await seedSelectedProject(page)
    await gotoAndEnsureAuth(page, '/project/relay-subkeys')

    expect(page.url()).not.toContain('/500')
    expect(page.url()).not.toContain('/sign-in')
    await expect(page.getByRole('heading', { name: 'Project Relay Sub-Keys' })).toBeVisible()
    await expect(page.getByText(/Project view|Available products/i).first()).toBeVisible()
  })

  test('project missing key detail shows not found instead of the first mock key', async ({ page }) => {
    await seedSelectedProject(page)
    await gotoAndEnsureAuth(page, '/project/relay-subkeys/keys/pw-e2e-missing-key')

    await expect(page.getByText('Sub-key not found')).toBeVisible()
    await expect(page.getByText('The requested key is unavailable for the selected project.')).toBeVisible()
    await expect(
      page.getByRole('heading', {
        name: /Production checkout relay|Production gateway relay|Support assistant relay|Old staging relay|Codex lab preview/i,
      })
    ).toHaveCount(0)
  })

  test('rest mode product failures do not fall back to mock products', async ({ page }) => {
    test.skip(process.env.VITE_RELAY_SUBKEYS_API_MODE !== 'rest', 'Relay REST fallback behavior only runs in REST mode')

    await page.route('**/admin/relay-subkeys/products', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Relay REST E2E forced failure' }),
      })
    })

    await gotoAndEnsureAuth(page, '/relay-subkeys')

    await expect(page.getByText('Unable to load relay data')).toBeVisible()
    await expect(page.getByText('Relay REST E2E forced failure')).toBeVisible()
    await expect(page.getByText('GPT Shared Pro')).toHaveCount(0)
  })
})
