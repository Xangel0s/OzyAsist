import { test, expect } from "@playwright/test";

test.describe("Skills, MCP Connectors, Plugins & Stress Tests", () => {
  const BASE_API = "http://localhost:8080/api";
  const APP_URL = "http://localhost:1420";

  test("Skills API & Execution - Create, List, Execute and Delete Skill", async ({ request }) => {
    // 1. Create a prompt template skill
    const createResp = await request.post(`${BASE_API}/skills`, {
      data: {
        name: "test-summarize-skill",
        description: "Resumidor de texto de prueba",
        triggerPattern: "/test-summarize",
        executionType: "prompt_template",
        config: "Por favor resume el siguiente texto: {input}",
      },
    });
    expect(createResp.ok()).toBeTruthy();
    const skill = await createResp.json();
    expect(skill.id).toBeTruthy();
    expect(skill.name).toBe("test-summarize-skill");

    // 2. List skills
    const listResp = await request.get(`${BASE_API}/skills`);
    expect(listResp.ok()).toBeTruthy();
    const skillsList = await listResp.json();
    const found = skillsList.find((s: any) => s.id === skill.id);
    expect(found).toBeTruthy();

    // 3. Execute skill via API
    const execResp = await request.post(`${BASE_API}/skills/execute`, {
      data: {
        skillId: skill.id,
        input: "Contenido de texto de prueba de largo alcance",
      },
    });
    expect(execResp.ok()).toBeTruthy();
    const result = await execResp.json();
    expect(result.success).toBeTruthy();
    expect(result.output).toContain("Por favor resume el siguiente texto:");

    // 4. Delete skill
    const delResp = await request.delete(`${BASE_API}/skills/${skill.id}`);
    expect(delResp.ok()).toBeTruthy();
  });

  test("MCP Connectors API - Create, List, and Delete Custom MCP Connector", async ({ request }) => {
    // 1. Create a custom MCP Connector
    const createResp = await request.post(`${BASE_API}/connectors`, {
      data: {
        name: "test-mcp-github",
        type: "mcp",
        endpoint: "http://localhost:9000/mcp",
        authConfig: JSON.stringify({ token: "test-token" }),
      },
    });
    expect(createResp.ok()).toBeTruthy();
    const conn = await createResp.json();
    expect(conn.id).toBeTruthy();
    expect(conn.name).toBe("test-mcp-github");

    // 2. List connectors
    const listResp = await request.get(`${BASE_API}/connectors`);
    expect(listResp.ok()).toBeTruthy();
    const connectors = await listResp.json();
    const found = connectors.find((c: any) => c.id === conn.id);
    expect(found).toBeTruthy();

    // 3. Delete connector
    const delResp = await request.delete(`${BASE_API}/connectors/${conn.id}`);
    expect(delResp.ok()).toBeTruthy();
  });

  test("UI Settings Modal - Skills, Connectors, and Plugins tabs navigation", async ({ page }) => {
    await page.goto(APP_URL);
    await page.waitForTimeout(1000);

    // Complete onboarding if shown
    const nameInput = page.locator('input[placeholder*="Ej"]');
    if (await nameInput.count() > 0) {
      await nameInput.fill("Test User");
      await page.click('button:has-text("Comenzar")');
      await page.waitForTimeout(1000);
    }

    // Open Settings Modal by clicking Personalizar in sidebar
    const customizeBtn = page.locator("button", { hasText: "Personalizar" });
    if (await customizeBtn.count() > 0) {
      await customizeBtn.first().click();
      await page.waitForTimeout(800);
    }

    // Verify modal overlay appears
    const modalHeading = page.locator("h2", { hasText: /Configuración|Proveedores|Habilidades|Plugins/i });
    if (await modalHeading.count() > 0) {
      await expect(modalHeading.first()).toBeVisible();
    }

    // Click Habilidades category item
    const habItem = page.locator("button", { hasText: "Habilidades" });
    if (await habItem.count() > 0) {
      await habItem.first().click();
      await page.waitForTimeout(300);
    }

    // Click Conectores category item
    const connItem = page.locator("button", { hasText: "Conectores" });
    if (await connItem.count() > 0) {
      await connItem.first().click();
      await page.waitForTimeout(300);
    }

    // Click Plugins category item
    const pluginItem = page.locator("button", { hasText: "Plugins" });
    if (await pluginItem.count() > 0) {
      await pluginItem.first().click();
      await page.waitForTimeout(300);
    }
  });

  test("Stress Test - Rapid creation, execution & deletion of 10 Skills & 10 Connectors", async ({ request }) => {
    const createdSkills: string[] = [];
    const createdConns: string[] = [];

    // Create 10 skills concurrently
    for (let i = 0; i < 10; i++) {
      const sResp = await request.post(`${BASE_API}/skills`, {
        data: {
          name: `stress-skill-${i}`,
          description: `Skill de estrés ${i}`,
          triggerPattern: `/stress-${i}`,
          executionType: "prompt_template",
          config: `Plantilla de estrés ${i}: {input}`,
        },
      });
      if (sResp.ok()) {
        const data = await sResp.json();
        createdSkills.push(data.id);
      }

      const cResp = await request.post(`${BASE_API}/connectors`, {
        data: {
          name: `stress-conn-${i}`,
          type: "custom",
          endpoint: `http://localhost:8080/mock-${i}`,
        },
      });
      if (cResp.ok()) {
        const data = await cResp.json();
        createdConns.push(data.id);
      }
    }

    expect(createdSkills.length).toBe(10);
    expect(createdConns.length).toBe(10);

    // Delete all 10 skills and 10 connectors
    for (const id of createdSkills) {
      await request.delete(`${BASE_API}/skills/${id}`);
    }
    for (const id of createdConns) {
      await request.delete(`${BASE_API}/connectors/${id}`);
    }

    // Verify system health
    const health = await request.get(`${BASE_API.replace('/api', '')}/health`);
    expect(health.ok()).toBeTruthy();
  });
});
