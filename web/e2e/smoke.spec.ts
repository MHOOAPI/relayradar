import { expect, test } from '@playwright/test';

const routes = ['/', '/stations', '/models', '/status', '/model-status', '/changes'];

for (const path of routes) {
  test(`renders ${path} without runtime errors`, async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', (err) => errors.push(err.message));
    page.on('response', (res) => {
      if (res.status() >= 500) errors.push(`${res.status()} ${res.url()}`);
    });

    const res = await page.goto(path, { waitUntil: 'networkidle' });
    expect(res?.status(), `HTTP status for ${path}`).toBeLessThan(500);

    const h1 = page.locator('main h1').first();
    await expect(h1).toBeVisible();
    await expect(h1).not.toHaveText(/^\s*$/);

    expect(errors, `runtime errors on ${path}:\n${errors.join('\n')}`).toEqual([]);
  });
}
