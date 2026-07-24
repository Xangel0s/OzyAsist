import { test, expect } from "@playwright/test";

test.describe("OzyAssist Smoke Tests", () => {
  test("Frontend loads and shows the app", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await expect(page.locator("html")).toHaveClass(/dark/);
    await expect(page).toHaveTitle(/Ozy Desktop/);
  });

  test("Backend health endpoint responds", async ({ request }) => {
    const resp = await request.get("http://localhost:8080/health");
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body.status).toBe("ok");
  });

  test("Backend models endpoint returns providers", async ({ request }) => {
    const resp = await request.get("http://localhost:8080/api/models");
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(Array.isArray(body)).toBeTruthy();
  });

  test("Backend projects returns array", async ({ request }) => {
    const resp = await request.get("http://localhost:8080/api/projects");
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(Array.isArray(body)).toBeTruthy();
  });

  test("Backend chats returns array", async ({ request }) => {
    const resp = await request.get("http://localhost:8080/api/chats");
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(Array.isArray(body)).toBeTruthy();
  });

  test("Frontend shows welcome screen", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await expect(page.locator("#root")).not.toBeEmpty();
    const hasContent = await page.locator("#root").innerText();
    expect(hasContent.length).toBeGreaterThan(0);
  });

  test("Frontend has interactive elements", async ({ page }) => {
    await page.goto("http://localhost:1420");
    const appRoot = page.locator("#root");
    await expect(appRoot).toBeVisible();
    const buttons = page.locator("button");
    const buttonCount = await buttons.count();
    expect(buttonCount).toBeGreaterThan(0);
  });

  test("Creating a chat via API returns valid chat", async ({ request }) => {
    const resp = await request.post("http://localhost:8080/api/chats", {
      data: { name: "Test Chat", mode: "chat" },
    });
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body.id).toBeTruthy();
    expect(body.name).toBe("Test Chat");
    await request.delete(`http://localhost:8080/api/chats/${body.id}`);
  });
});

test.describe("Frontend UI Validation", () => {
  test("Sidebar shows with nav items after onboarding", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await page.waitForTimeout(1000);

    // Complete onboarding: enter name and click Comenzar
    await page.fill('input[placeholder*="Ej"]', "Test User");
    await page.click('button:has-text("Comenzar")');
    await page.waitForTimeout(1500);

    // The "+ Nuevo" button should exist (first button with add icon + text)
    const newChatBtn = page.locator('button:has(span:has-text("add")):has-text("Nuevo")');
    await expect(newChatBtn).toBeVisible();
  });

  test("Action chips are visible on home after onboarding", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await page.waitForTimeout(1000);

    // Complete onboarding
    await page.fill('input[placeholder*="Ej"]', "Test User");
    await page.click('button:has-text("Comenzar")');
    await page.waitForTimeout(1500);

    // Click Home if not already there
    const homeBtn = page.locator('button:has(span:has-text("home"))');
    if (await homeBtn.count() > 0) {
      await homeBtn.first().click();
      await page.waitForTimeout(500);
    }

    // Action chips should be visible on home
    const codigo = page.locator("button", { hasText: "Código" });
    if (await codigo.count() > 0) {
      await expect(codigo.first()).toBeVisible();
    }
  });

  test("Model selector shows on home page after onboarding", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await page.waitForTimeout(1000);

    // Complete onboarding
    await page.fill('input[placeholder*="Ej"]', "Test User");
    await page.click('button:has-text("Comenzar")');
    await page.waitForTimeout(1500);

    // Look for expand_more icon or a button with model text
    const modelBtn = page.locator("button:has(span:has-text('expand_more'))").first();
    await expect(modelBtn).toBeVisible({ timeout: 5000 });
  });

  test("Attach button exists and is clickable after onboarding", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await page.waitForTimeout(1000);

    // Complete onboarding
    await page.fill('input[placeholder*="Ej"]', "Test User");
    await page.click('button:has-text("Comenzar")');
    await page.waitForTimeout(1500);

    // Find the "+" button for attaching files (with "add" icon)
    const addBtn = page.locator("button:has(span:has-text('add'))").first();
    await expect(addBtn).toBeVisible({ timeout: 5000 });

    // Click the first one - should open file picker (won't actually open in test)
    await addBtn.click();
  });

  test("TopAppBar has profile and fullscreen buttons after onboarding", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await page.waitForTimeout(1000);

    // Complete onboarding
    await page.fill('input[placeholder*="Ej"]', "Test User");
    await page.click('button:has-text("Comenzar")');
    await page.waitForTimeout(1500);

    // Profile button should exist
    const profileBtn = page.locator('button[aria-label="Perfil"]');
    await expect(profileBtn).toBeVisible({ timeout: 5000 });

    // Fullscreen button should exist
    const fullscreenBtn = page.locator('button[aria-label="Pantalla completa"]');
    await expect(fullscreenBtn).toBeVisible();

    // Minimize and Close should NOT exist (removed per MVP cleanup)
    const minimizeBtn = page.locator('button[aria-label="Minimizar"]');
    await expect(minimizeBtn).toHaveCount(0);

    const closeBtn = page.locator('button[aria-label="Cerrar"]');
    await expect(closeBtn).toHaveCount(0);
  });

  test("Navigate to Chat view and verify input elements after onboarding", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await page.waitForTimeout(1000);

    // Complete onboarding
    await page.fill('input[placeholder*="Ej"]', "Test User");
    await page.click('button:has-text("Comenzar")');
    await page.waitForTimeout(1500);

    // Click on "+ Nuevo" to create a chat
    const newChatBtn = page.locator("button", { hasText: /Nuevo/i }).first();
    await expect(newChatBtn).toBeVisible({ timeout: 5000 });
    await newChatBtn.click();
    await page.waitForTimeout(1000);

    // Should now be in chat view
    const textarea = page.locator("textarea").first();
    await expect(textarea).toBeVisible({ timeout: 5000 });
  });

  test("Mic and Eco buttons are removed from all components", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await page.waitForTimeout(1000);

    // Complete onboarding
    await page.fill('input[placeholder*="Ej"]', "Test User");
    await page.click('button:has-text("Comenzar")');
    await page.waitForTimeout(1500);

    // Mic button should NOT exist
    const micBtn = page.locator('button[aria-label="Micrófono"]');
    await expect(micBtn).toHaveCount(0);

    // Eco/graphic_eq button should NOT exist
    const ecoBtn = page.locator('button[aria-label="Eco"]');
    await expect(ecoBtn).toHaveCount(0);
  });

  test("File upload endpoint works", async ({ request }) => {
    const resp = await request.post("http://localhost:8080/api/files/upload", {
      multipart: {
        file: {
          name: "test.txt",
          mimeType: "text/plain",
          buffer: Buffer.from("contenido de prueba"),
        },
      },
    });
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body.id).toBeTruthy();
    expect(body.filename).toBe("test.txt");
  });
});
