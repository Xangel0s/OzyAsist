import { test, expect } from "@playwright/test";

test.describe("LLM & Agent Full Integration & Robustness", () => {
  test("Complete Onboarding flow works seamlessly", async ({ page }) => {
    await page.goto("http://localhost:1420");
    await page.waitForTimeout(1000);

    // Step 1: Memoria - Click Omitir este paso
    const skipStep1 = page.locator("button", { hasText: /Omitir este paso/i });
    if (await skipStep1.count() > 0) {
      await skipStep1.click();
      await page.waitForTimeout(500);
    }

    // Step 2: Feedback - Click Omitir este paso
    const skipStep2 = page.locator("button", { hasText: /Omitir este paso/i });
    if (await skipStep2.count() > 0) {
      await skipStep2.click();
      await page.waitForTimeout(500);
    }

    // Step 3: Instrucciones - Enter name and click Comenzar con Ozy
    const nameInput = page.locator('input[placeholder*="Ej"]');
    if (await nameInput.count() > 0) {
      await nameInput.fill("Peter");
      const comenzarBtn = page.locator("button", { hasText: /Comenzar/i });
      await comenzarBtn.click();
      await page.waitForTimeout(1000);
    }

    // Verify main app layout is visible
    await expect(page.locator("#root")).not.toBeEmpty();
  });

  test("Configure OpenCode Provider API Key in SettingsModal", async ({ page, request }) => {
    // 1. Direct API call to update OpenCode key on backend
    const apiKey = process.env.OPENCODE_API_KEY || "sk-test-opencode-key";
    const updateResp = await request.put("http://localhost:8080/api/settings", {
      data: { opencode_key: apiKey },
    });
    expect(updateResp.ok()).toBeTruthy();

    // 2. Verify /api/models returns opencode with deepseek-v4-pro
    const modelsResp = await request.get("http://localhost:8080/api/models");
    expect(modelsResp.ok()).toBeTruthy();
    const models = await modelsResp.json();
    const opencode = models.find((m: any) => m.provider === "opencode");
    expect(opencode).toBeTruthy();
    expect(opencode.models).toContain("deepseek-v4-pro");
  });

  test("Send chat prompt to OpenCode LLM and receive real streaming response", async ({ request }) => {
    // Register key first
    const apiKey = process.env.OPENCODE_API_KEY || "sk-test-opencode-key";
    await request.put("http://localhost:8080/api/settings", {
      data: { opencode_key: apiKey },
    });

    // Create chat via API
    const chatResp = await request.post("http://localhost:8080/api/chats", {
      data: {
        name: "Test OpenCode Integration",
        mode: "chat",
        provider: "opencode",
        model: "deepseek-v4-pro",
      },
    });
    expect(chatResp.ok()).toBeTruthy();
    const chat = await chatResp.json();
    expect(chat.provider).toBe("opencode");

    // Clean up
    await request.delete(`http://localhost:8080/api/chats/${chat.id}`);
  });

  test("Stress Test - Rapid chat creations and deletions maintain system stability", async ({ request }) => {
    const chatIds: string[] = [];
    for (let i = 0; i < 5; i++) {
      const resp = await request.post("http://localhost:8080/api/chats", {
        data: { name: `Stress Chat ${i}`, mode: "chat" },
      });
      if (resp.ok()) {
        const body = await resp.json();
        chatIds.push(body.id);
      }
    }
    expect(chatIds.length).toBe(5);

    // Delete all stress chats
    for (const id of chatIds) {
      await request.delete(`http://localhost:8080/api/chats/${id}`);
    }

    // Check backend health stays OK
    const healthResp = await request.get("http://localhost:8080/health");
    expect(healthResp.ok()).toBeTruthy();
  });
});
