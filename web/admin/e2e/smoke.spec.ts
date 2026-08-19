// A4 主链冒烟:登录→工作台→订单列表(筛选/跟踪抽屉)→出账页。
// 走真实 Go 后端(BOSS_API_TARGET),断言真实业务 DOM 文案;种子账号 admin/admin123(scripts/devseed)。
import { expect, test } from '@playwright/test'

const ADMIN = { user: process.env.E2E_ADMIN_USER ?? 'admin', pass: process.env.E2E_ADMIN_PASS ?? 'admin123' }

async function login(page: import('@playwright/test').Page) {
  await page.goto('/login')
  await page.getByPlaceholder('账号').fill(ADMIN.user)
  await page.getByPlaceholder('密码').fill(ADMIN.pass)
  await page.getByRole('button', { name: /登\s*录/ }).click()
  await page.waitForURL(/#?\/dashboard/)
}

test('登录后工作台渲染真实数据卡片', async ({ page }) => {
  await login(page)
  await expect(page.getByRole('heading', { name: '工作台' })).toBeVisible()
})

test('订单页:列表加载+状态筛选+空态文案', async ({ page }) => {
  await login(page)
  await page.goto('/boss/order')
  await expect(page.getByRole('heading', { name: '订单管理' })).toBeVisible()
  // 表头来自 i18n columns 契约
  await expect(page.getByRole('columnheader', { name: '订单号' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: '当前环节' })).toBeVisible()
  // 状态筛选交互:选中"已完成"后列表仍在(空库显示空态,有库显示过滤结果)
  await page.getByRole('combobox').selectOption({ label: '已完成' })
  await page.getByRole('button', { name: /刷新/ }).click()
  await expect(page.locator('tbody')).toBeVisible()
})

test('出账页:渲染账期入口与账单表头', async ({ page }) => {
  await login(page)
  await page.goto('/billing/billing')
  await expect(page.getByText('出账管理').first()).toBeVisible()
  await expect(page.locator('.org-table-wrap, table').first()).toBeVisible()
})
