import { test, expect, type Page } from '@playwright/test';

type Call = { path: string; method: string; body: any; query: URLSearchParams };
async function mockAPI(
  page: Page,
  base = '',
  options: {
    role?: string;
    authenticated?: boolean;
    setup?: boolean;
    oidcEnabled?: boolean;
    oidcButtonText?: string;
    oidcButtonIconURL?: string;
  } = {}
) {
  const calls: Call[] = [];
  let authenticated = options.authenticated ?? true;
  let setup = options.setup ?? false;
  const role = options.role || 'admin';
  const user = {
    id: 'owner-1',
    username: 'joseph',
    email: 'joseph@example.com',
    role,
    buckets: ['bucket-1'],
    createdAt: '2026-01-01T00:00:00Z'
  };
  const buckets = [
    {
      id: 'bucket-1',
      globalAliases: ['photos'],
      browseAvailable: true,
      localAliases: [],
      keys: [
        {
          accessKeyId: 'GK123',
          name: 'App key',
          permissions: { read: true, write: true, owner: false }
        }
      ],
      objects: 3,
      bytes: 4096,
      websiteAccess: true,
      websiteConfig: { indexDocument: 'index.html', errorDocument: '404.html' },
      quotas: { maxSize: null, maxObjects: null }
    }
  ];
  let layout = {
    version: 1,
    roles: [
      { id: 'node-1', zone: 'dc1', capacity: 1073741824, tags: ['storage'] }
    ],
    stagedRoleChanges: [] as any[]
  };
  await page.route('**/api/**', async (route) => {
    const request = route.request();
    const requestUrl = new URL(request.url());
    const path = requestUrl.pathname.slice((base + '/api').length);
    let body: any = null;
    try {
      body = request.postDataJSON();
    } catch {
      /* Upload bodies are multipart. */
    }
    calls.push({
      path,
      method: request.method(),
      body,
      query: requestUrl.searchParams
    });
    let result: any = {};
    if (!requestUrl.pathname.startsWith(base + '/api/'))
      return route.fulfill({
        status: 400,
        json: { message: 'Wrong API base' }
      });
    if (path === '/auth/status')
      result = {
        enabled: true,
        authenticated,
        needsSetup: setup,
        oidcEnabled: options.oidcEnabled ?? true,
        oidcButtonText: options.oidcButtonText ?? 'Sign in with Pocket ID',
        oidcButtonIconURL: options.oidcButtonIconURL ?? '/favicon-32x32.png',
        user: authenticated ? user : null
      };
    else if (path === '/auth/login') {
      if (body.password === 'wrong')
        return route.fulfill({
          status: 401,
          json: { message: 'Invalid credentials' }
        });
      authenticated = true;
    } else if (path === '/auth/register') {
      setup = false;
      authenticated = true;
    } else if (path === '/auth/logout') authenticated = false;
    else if (path === '/auth/change-password') result = true;
    else if (path === '/buckets') result = buckets;
    else if (path === '/v2/GetBucketInfo') result = buckets[0];
    else if (path === '/v2/GetClusterHealth')
      result = {
        status: 'healthy',
        knownNodes: 1,
        connectedNodes: 1,
        storageNodes: 1,
        storageNodesOk: 1
      };
    else if (path === '/users')
      result =
        request.method() === 'GET'
          ? [user, { ...user, id: 'dev-1', username: 'member', role: 'user' }]
          : { id: 'new-user', ...body };
    else if (path.startsWith('/users/')) result = true;
    else if (path === '/v2/ListKeys')
      result = [{ id: 'GK123', name: 'App key' }];
    else if (path === '/v2/GetKeyInfo')
      result = {
        accessKeyId: 'GK123',
        secretAccessKey: 'test-secret',
        name: 'App key'
      };
    else if (path === '/v2/CreateKey' || path === '/v2/ImportKey')
      result = {
        accessKeyId: body.accessKeyId || 'GKNEW',
        secretAccessKey: body.secretAccessKey || 'new-secret',
        name: body.name
      };
    else if (path === '/v2/GetClusterStatus')
      result = {
        layoutVersion: layout.version,
        nodes: [
          {
            id: 'node-1',
            hostname: 'storage-01',
            addr: '127.0.0.1:3901',
            isUp: true
          }
        ]
      };
    else if (path === '/v2/GetNodeInfo')
      result = {
        success: {
          'node-1': {
            garageVersion: '2.1.0',
            dbEngine: 'lmdb',
            rustVersion: '1.90.0'
          }
        }
      };
    else if (path === '/v2/GetClusterLayout') result = layout;
    else if (path === '/v2/UpdateClusterLayout')
      layout.stagedRoleChanges = body.roles;
    else if (path === '/v2/RevertClusterLayout') layout.stagedRoleChanges = [];
    else if (path === '/v2/ApplyClusterLayout') {
      layout = { ...layout, version: body.version, stagedRoleChanges: [] };
      result = { message: ['Layout applied'], layout };
    } else if (path === '/v2/ConnectClusterNodes') result = [{ success: true }];
    else if (path === '/config')
      result = {
        s3_web: {
          root_domain: '.storage.example.com',
          bind_addr: '0.0.0.0:3902'
        }
      };
    else if (path === '/logs')
      result = {
        entries: [
          {
            timestamp: '2026-09-17T00:00:00Z',
            level: 'INFO',
            message: 'User signed in',
            context: { ip: '127.0.0.1', user: 'joseph' }
          }
        ],
        total: 75,
        counts: { INFO: 75 },
        file: '/data/logs/app.log',
        size: 1000
      };
    else if (path === '/browse/photos' && request.method() === 'GET') {
      const prefix = requestUrl.searchParams.get('prefix') || '';
      result = {
        prefix,
        prefixes: prefix ? [] : ['archive/', 'nested/'],
        objects: [
          {
            objectKey: 'hello #?.txt',
            size: 42,
            lastModified: '2026-09-17T00:00:00Z',
            url: `/browse/photos/${prefix}hello #?.txt`
          }
        ],
        nextToken: null
      };
    } else if (path === '/browse/photos' && request.method() === 'POST')
      result = { moved: body.items.length };
    else if (path.startsWith('/browse/photos/')) {
      if (request.method() === 'GET')
        return route.fulfill({
          contentType: 'text/plain',
          body: 'test object'
        });
      result = true;
    } else if (!path.startsWith('/v2/'))
      return route.fulfill({
        status: 404,
        json: { message: `Unhandled mock: ${path}` }
      });
    await route.fulfill({ json: result });
  });
  return { calls, buckets };
}

for (const base of ['', '/console', '/tools/garage']) {
  test(`production SPA boots and deep links work at ${base || '/'}`, async ({
    page
  }) => {
    const errors: string[] = [];
    page.on('pageerror', (error) => errors.push(error.message));
    await mockAPI(page, base);
    await page.goto(base + '/');
    await expect(
      page.getByRole('heading', { name: 'Dashboard' })
    ).toBeVisible();
    await page
      .getByRole('navigation', { name: 'Main navigation' })
      .getByRole('link', { name: 'Buckets', exact: true })
      .click();
    await page.getByRole('link', { name: /photos/ }).click();
    await expect(
      page.getByRole('region', { name: 'Object browser' })
    ).toBeVisible();
    await page.getByRole('button', { name: 'nested', exact: true }).click();
    await expect(page).toHaveURL(
      new RegExp(`${base}/buckets/bucket-1\\?prefix=nested%2F`)
    );
    await page.reload();
    await expect(
      page
        .getByRole('navigation', { name: 'Object folders' })
        .getByRole('button', { name: 'nested' })
    ).toBeVisible();
    await page.goBack();
    await expect(
      page
        .getByRole('navigation', { name: 'Object folders' })
        .getByRole('button', { name: 'nested' })
    ).toHaveCount(0);
    expect(errors).toEqual([]);
  });
}

test('login errors, custom OIDC button, password changes, and logout', async ({
  page
}) => {
  const { calls } = await mockAPI(page, '/console', { authenticated: false });
  await page.goto('/console/buckets');
  await expect(page).toHaveURL(/\/console\/auth\/login$/);
  await expect(
    page.getByRole('link', { name: 'Sign in with Pocket ID' })
  ).toHaveAttribute('href', '/console/api/v1/auth/oidc/login');
  await expect(
    page.getByRole('link', { name: 'Sign in with Pocket ID' }).locator('img')
  ).toHaveAttribute('src', '/favicon-32x32.png');
  await page.getByLabel('Username').fill('joseph');
  await page.getByLabel('Password', { exact: true }).fill('wrong');
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByRole('alert')).toHaveText('Invalid credentials');
  await page.getByLabel('Password', { exact: true }).fill('correct');
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await page.getByRole('button', { name: 'Change password' }).click();
  await page.getByLabel('Current password', { exact: true }).fill('correct');
  await page.getByLabel('New password', { exact: true }).fill('updated');
  await page.getByLabel('Confirm password', { exact: true }).fill('updated');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(
    calls.find((call) => call.path === '/auth/change-password')?.body
  ).toEqual({ currentPassword: 'correct', newPassword: 'updated' });
  await page.getByRole('button', { name: 'Sign out' }).click();
  await expect(
    page.getByRole('heading', { name: 'Welcome back' })
  ).toBeVisible();
});

test('first-run registration validates password confirmation', async ({
  page
}) => {
  const { calls } = await mockAPI(page, '', {
    authenticated: false,
    setup: true
  });
  await page.goto('/');
  await expect(page).toHaveURL(/\/auth\/register$/);
  await page.getByLabel('Username').fill('first-owner');
  await page.getByLabel('Password', { exact: true }).fill('secret123');
  await page.getByLabel('Confirm password').fill('different');
  await page.getByRole('button', { name: 'Create admin account' }).click();
  await expect(page.getByRole('alert')).toHaveText('Passwords do not match');
  expect(calls.filter((call) => call.path === '/auth/register')).toHaveLength(
    0
  );
  await page.getByLabel('Confirm password').fill('secret123');
  await page.getByRole('button', { name: 'Create admin account' }).click();
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
});

test('user routes and requests remain scoped', async ({ page }) => {
  const { calls } = await mockAPI(page, '', { role: 'user' });
  await page.goto('/users');
  await expect(
    page.getByRole('heading', { name: 'Buckets', exact: true })
  ).toBeVisible();
  await expect(page.getByRole('button', { name: 'Create bucket' })).toHaveCount(
    0
  );
  const nav = page.getByRole('navigation', { name: 'Main navigation' });
  for (const name of ['Users', 'Logs', 'Cluster', 'Access Keys'])
    await expect(nav.getByRole('link', { name, exact: true })).toHaveCount(0);
  await page.goto('/buckets/bucket-1');
  await expect(
    page.getByRole('region', { name: 'Object browser' })
  ).toBeVisible();
  await expect(
    page.getByRole('button', { name: 'permissions', exact: true })
  ).toHaveCount(0);
  expect(
    calls.some((call) =>
      ['/users', '/v2/ListKeys', '/v2/GetClusterHealth', '/config'].includes(
        call.path
      )
    )
  ).toBe(false);
});

test('encoded downloads, share URLs, bulk moves, deletion, and uploads', async ({
  page
}) => {
  const { calls } = await mockAPI(page);
  await page.goto('/buckets/bucket-1?prefix=nested%2F');
  const object = page.getByRole('checkbox', { name: 'Select hello #?.txt' });
  await expect(
    page.getByRole('link', { name: 'Download hello #?.txt' })
  ).toHaveAttribute(
    'href',
    '/api/browse/photos/nested/hello%20%23%3F.txt?dl=1'
  );
  await object.check();
  await page.getByRole('button', { name: 'Share files', exact: true }).click();
  await expect(page.getByLabel('Share URL')).toHaveValue(
    'https://photos/nested/hello%20%23%3F.txt'
  );
  await page.getByRole('button', { name: 'Close dialog' }).click();
  await page.getByRole('button', { name: 'Move', exact: true }).first().click();
  await page
    .getByRole('navigation', { name: 'Destination folders' })
    .getByRole('button', { name: 'Root', exact: true })
    .click();
  await page
    .getByRole('dialog')
    .getByRole('button', { name: 'archive', exact: true })
    .click();
  await page.getByLabel('New destination folder').fill('2026/september');
  await page.getByRole('button', { name: 'Create', exact: true }).click();
  await expect(
    page.getByText('Destination: /archive/2026/september/')
  ).toBeVisible();
  await page.getByRole('button', { name: 'Move here' }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(
    calls.find(
      (call) => call.path === '/browse/photos' && call.method === 'POST'
    )?.body
  ).toEqual({
    items: ['nested/hello #?.txt'],
    destination: 'archive/2026/september/'
  });
  await object.check();
  await page
    .getByRole('button', { name: 'Delete', exact: true })
    .first()
    .click();
  await page.getByRole('button', { name: 'Delete permanently' }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(
    calls.some(
      (call) =>
        call.path === '/browse/photos/nested/hello%20%23%3F.txt' &&
        call.method === 'DELETE'
    )
  ).toBe(true);
  await page.getByLabel('Upload files', { exact: true }).setInputFiles({
    name: 'new #?.txt',
    mimeType: 'text/plain',
    buffer: Buffer.from('hello')
  });
  await expect(
    page.getByRole('complementary', { name: 'Uploads' })
  ).toContainText('success');
  expect(
    calls.some(
      (call) =>
        call.path === '/browse/photos/nested/new%20%23%3F.txt' &&
        call.method === 'PUT'
    )
  ).toBe(true);
  await page
    .getByRole('navigation', { name: 'Main navigation' })
    .getByRole('link', { name: 'Dashboard' })
    .click();
  await expect(
    page.getByRole('complementary', { name: 'Uploads' })
  ).toBeVisible();
});

test('user editing preserves bucket IDs and sends PATCH', async ({ page }) => {
  const { calls } = await mockAPI(page);
  await page.goto('/users');
  await page
    .getByRole('row')
    .filter({ hasText: 'member' })
    .getByRole('button', { name: 'Edit', exact: true })
    .click();
  await page.getByLabel('Username', { exact: true }).fill('updated-dev');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(
    calls.find(
      (call) => call.path === '/users/dev-1' && call.method === 'PATCH'
    )?.body
  ).toEqual({
    username: 'updated-dev',
    email: 'joseph@example.com',
    role: 'user',
    buckets: ['bucket-1']
  });
});

test('cluster assignment is staged and applied with the next layout version', async ({
  page
}) => {
  const { calls } = await mockAPI(page);
  await page.goto('/cluster');
  await page.getByRole('button', { name: 'Edit assignment' }).click();
  await page.getByLabel('Zone', { exact: true }).fill('dc2');
  await page.getByLabel('Storage capacity (GiB)').fill('2');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: 'Staged changes' })
  ).toBeVisible();
  await page.getByRole('button', { name: 'Apply changes' }).click();
  await page.getByRole('button', { name: 'Confirm', exact: true }).click();
  await expect(page.getByRole('dialog')).toContainText('Layout applied');
  expect(
    calls.find((call) => call.path === '/v2/UpdateClusterLayout')?.body.roles[0]
  ).toEqual({
    id: 'node-1',
    zone: 'dc2',
    capacity: 2147483648,
    tags: ['storage']
  });
  expect(
    calls.find((call) => call.path === '/v2/ApplyClusterLayout')?.body
  ).toEqual({ version: 2 });
});

test('log filters reset pagination and details expand', async ({ page }) => {
  const { calls } = await mockAPI(page);
  await page.goto('/logs');
  await page.getByText('User signed in').click();
  await expect(
    page.getByText('"ip": "127.0.0.1"', { exact: false })
  ).toBeVisible();
  await page.getByRole('button', { name: 'Next', exact: true }).click();
  await expect(page.getByText('Page 2 of 2', { exact: false })).toBeVisible();
  await page.getByLabel('Log level').selectOption('ERROR');
  await expect(page.getByText('Page 1 of 2', { exact: false })).toBeVisible();
  await page.getByLabel('Search logs').fill('login');
  await expect
    .poll(() =>
      calls
        .filter((call) => call.path === '/logs')
        .at(-1)
        ?.query.get('search')
    )
    .toBe('login');
});

test('bucket aliases, quotas, website settings, and permissions retain API contracts', async ({
  page
}) => {
  const { calls } = await mockAPI(page);
  await page.goto('/buckets/bucket-1?tab=overview');
  await page.getByRole('button', { name: 'Add alias' }).click();
  await page.getByLabel('Alias', { exact: true }).fill('gallery');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(
    calls.find((call) => call.path === '/v2/AddBucketAlias')?.body
  ).toEqual({ bucketId: 'bucket-1', globalAlias: 'gallery' });
  await page.getByRole('button', { name: 'Edit quotas' }).click();
  await page.getByLabel('Maximum objects').fill('500');
  await page.getByLabel('Maximum size (GiB)').fill('2');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(calls.find((call) => call.path === '/v2/UpdateBucket')?.body).toEqual({
    quotas: { maxObjects: 500, maxSize: 2147483648 }
  });
  await page.getByRole('button', { name: 'Configure', exact: true }).click();
  await page.getByLabel('Index document').fill('home.html');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(
    calls.filter((call) => call.path === '/v2/UpdateBucket').at(-1)?.body
      .websiteAccess.indexDocument
  ).toBe('home.html');
  await page.getByRole('button', { name: 'permissions', exact: true }).click();
  await page
    .getByRole('checkbox', {
      name: 'write permission for App key',
      exact: true
    })
    .uncheck();
  await expect
    .poll(() => calls.find((call) => call.path === '/v2/DenyBucketKey')?.body)
    .toEqual({
      bucketId: 'bucket-1',
      accessKeyId: 'GK123',
      permissions: { read: false, write: true, owner: false }
    });
  await page.getByRole('button', { name: 'Allow access key' }).click();
  await page
    .getByRole('listbox', { name: 'Access keys', exact: true })
    .selectOption('GK123');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(
    calls.find((call) => call.path === '/v2/AllowBucketKey')?.body
  ).toEqual({
    bucketId: 'bucket-1',
    accessKeyId: 'GK123',
    permissions: { read: true, write: true, owner: false }
  });
});

test('key import, secret reveal, and confirmed deletion', async ({ page }) => {
  const { calls } = await mockAPI(page);
  await page.goto('/keys');
  await page.getByRole('button', { name: 'Import key', exact: true }).click();
  await page.getByLabel('Name', { exact: true }).fill('Imported app');
  await page.getByLabel('Access key ID', { exact: true }).fill('GKIMPORTED');
  await page
    .getByLabel('Secret access key', { exact: true })
    .fill('import-secret');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(calls.find((call) => call.path === '/v2/ImportKey')?.body).toEqual({
    name: 'Imported app',
    accessKeyId: 'GKIMPORTED',
    secretAccessKey: 'import-secret'
  });
  await page.getByRole('button', { name: 'Reveal secret' }).click();
  await expect(page.getByText('test-secret', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Hide', exact: true }).click();
  await expect(page.getByText('test-secret', { exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: 'Delete', exact: true }).click();
  expect(calls.some((call) => call.path === '/v2/DeleteKey')).toBe(false);
  await page.getByRole('button', { name: 'Delete key', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(
    calls.find((call) => call.path === '/v2/DeleteKey')?.query.get('id')
  ).toBe('GK123');
});

test('desktop and mobile navigation, persisted themes, and dialog keyboard support', async ({
  page
}, testInfo) => {
  await mockAPI(page);
  await page.addInitScript(() =>
    localStorage.setItem(
      'appdata',
      JSON.stringify({ state: { mode: 'light' } })
    )
  );
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await expect(page.locator('html')).not.toHaveClass(/dark/);
  await page.screenshot({
    path: testInfo.outputPath('dashboard-light.png'),
    fullPage: true
  });
  await page.getByRole('button', { name: 'Toggle color theme' }).click();
  await page.reload();
  await expect(page.locator('html')).toHaveClass(/dark/);
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath('dashboard-dark.png'),
    fullPage: true
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole('button', { name: 'Toggle navigation' }).click();
  await page
    .getByRole('navigation', { name: 'Main navigation' })
    .getByRole('link', { name: 'Buckets', exact: true })
    .click();
  await page.getByRole('button', { name: 'Create bucket' }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath('mobile-dialog.png'),
    fullPage: true
  });
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.getByRole('button', { name: 'Toggle navigation' }).click();
  await page.getByRole('button', { name: 'Sign out' }).click();
  await expect(
    page.getByRole('heading', { name: 'Welcome back' })
  ).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath('login-mobile.png'),
    fullPage: true
  });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.screenshot({
    path: testInfo.outputPath('login-desktop.png'),
    fullPage: true
  });
});

test('OIDC button supports default branding and can be disabled', async ({
  page
}) => {
  await mockAPI(page, '', {
    authenticated: false,
    oidcButtonText: 'Continue with OpenID Connect',
    oidcButtonIconURL: ''
  });
  await page.goto('/auth/login');
  const button = page.getByRole('link', {
    name: 'Continue with OpenID Connect'
  });
  await expect(button).toBeVisible();
  await expect(button.locator('svg')).toBeVisible();
  await expect(button.locator('img')).toHaveCount(0);
  await page.unrouteAll();
  await mockAPI(page, '', { authenticated: false, oidcEnabled: false });
  await page.reload();
  await expect(
    page.getByRole('button', { name: 'Sign in', exact: true })
  ).toBeVisible();
  await expect(page.locator('a[href$="/auth/oidc/login"]')).toHaveCount(0);
});

test('login-05 respects both themes and fits mobile screens', async ({
  page
}, testInfo) => {
  await mockAPI(page, '', { authenticated: false });
  await page.addInitScript(() => localStorage.setItem('theme', 'light'));
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto('/auth/login');
  await expect(
    page.getByRole('heading', { name: 'Welcome back' })
  ).toBeVisible();
  await expect(page.locator('html')).not.toHaveClass(/dark/);
  await expect(page.getByAltText('Garage')).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath('login-light.png'),
    animations: 'disabled',
    fullPage: true
  });
  await page.getByRole('button', { name: 'Toggle color theme' }).click();
  await expect(page.locator('html')).toHaveClass(/dark/);
  await page.screenshot({
    path: testInfo.outputPath('login-dark.png'),
    animations: 'disabled',
    fullPage: true
  });
  await page.setViewportSize({ width: 375, height: 667 });
  await expect(
    page.getByRole('button', { name: 'Sign in', exact: true })
  ).toBeInViewport();
  expect(
    await page
      .locator('main')
      .evaluate((el) => el.scrollWidth <= el.clientWidth)
  ).toBe(true);
  await page.screenshot({
    path: testInfo.outputPath('login-mobile.png'),
    fullPage: true
  });
});

test('viewers can browse and download but cannot change buckets or objects', async ({
  page
}) => {
  const { calls } = await mockAPI(page, '', { role: 'viewer' });
  await page.goto('/buckets/bucket-1');
  await expect(
    page.getByRole('region', { name: 'Object browser' })
  ).toBeVisible();
  await expect(
    page.getByRole('link', { name: /^Download / }).first()
  ).toBeVisible();
  for (const name of [
    'New folder',
    'Upload files',
    'Upload folder',
    'Move',
    'Delete'
  ])
    await expect(page.getByRole('button', { name, exact: true })).toHaveCount(
      0
    );
  await page.getByLabel('Select all objects').check();
  await expect(
    page.getByRole('button', { name: 'Move', exact: true })
  ).toHaveCount(0);
  await expect(
    page.getByRole('button', { name: 'Delete', exact: true })
  ).toHaveCount(0);
  await page.getByRole('button', { name: 'overview', exact: true }).click();
  await expect(
    page.getByRole('heading', { name: 'Bucket information' })
  ).toBeVisible();
  for (const name of ['Add alias', 'Edit quotas', 'Configure', 'Delete bucket'])
    await expect(page.getByRole('button', { name, exact: true })).toHaveCount(
      0
    );
  await page.goto('/keys');
  await expect(
    page.getByRole('heading', { name: 'Buckets', exact: true })
  ).toBeVisible();
  expect(calls.some((call) => call.method !== 'GET')).toBe(false);
  expect(
    calls.some((call) =>
      ['/config', '/v2/ListKeys', '/v2/GetKeyInfo'].includes(call.path)
    )
  ).toBe(false);
});

test('users can edit assigned bucket settings while keys remain admin-only', async ({
  page
}) => {
  const { calls } = await mockAPI(page, '', { role: 'user' });
  await page.goto('/buckets/bucket-1?tab=overview');
  await page.getByRole('button', { name: 'Edit quotas' }).click();
  await page.getByLabel('Maximum objects').fill('100');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  expect(
    calls.some(
      (call) =>
        call.path === '/v2/UpdateBucket' &&
        call.query.get('id') === 'bucket-1' &&
        call.body.quotas.maxObjects === 100
    )
  ).toBe(true);
  await expect(
    page.getByRole('button', { name: 'permissions', exact: true })
  ).toHaveCount(0);
  await page.getByRole('button', { name: 'browse', exact: true }).click();
  await expect(
    page.getByRole('button', { name: 'Upload files', exact: true })
  ).toBeVisible();
});
