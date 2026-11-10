import { chromium, expect } from "@playwright/test";

const origin = process.env.MESSENGER_URL || "http://localhost";
const run = Date.now();
const browser = await chromium.launch();

async function register(name) {
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto(origin);
  await page.getByRole("button", { name: "Create one" }).click();
  await page.getByLabel("Email").fill(`smoke-${run}-${name}@example.test`);
  await page.getByLabel("Password").fill(`Smoke-${run}-password`);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page.getByRole("heading", { name: "Chats" })).toBeVisible({ timeout: 20000 });
  return page;
}

const log = (page) => page.getByRole("log", { name: "Messages" });

async function waitForProfile(page, query) {
  await page.getByRole("button", { name: "New conversation" }).first().click();
  for (let attempt = 0; attempt < 20; attempt++) {
    await page.getByLabel("Search people").fill(`${query}${attempt % 2 ? " " : ""}`);
    if (await page.locator(".person").filter({ hasText: query }).count()) return;
    await page.waitForTimeout(500);
  }
  throw new Error(`profile ${query} was not provisioned`);
}

try {
  const ann = await register("ann");
  const ben = await register("ben");
  const cat = await register("cat");

  await waitForProfile(ann, `smoke-${run}-ben`);
  const benRow = ann.locator(".person").filter({ hasText: `smoke-${run}-ben` });
  await benRow.getByRole("button", { name: "Add" }).click();
  await benRow.getByRole("button", { name: "Message" }).click();
  await ann.getByLabel("Message", { exact: true }).fill("Hello Ben");
  await ann.keyboard.press("Enter");
  await expect(log(ann).locator(".message.mine").getByLabel("Sent")).toBeVisible({ timeout: 10000 });

  await expect(ben.locator(".list").getByLabel("1 unread")).toBeVisible({ timeout: 10000 });
  await ben.locator(".list").getByRole("button", { name: new RegExp(`^smoke-${run}-ann`) }).click();
  await expect(log(ben).getByText("Hello Ben")).toBeVisible();
  await expect(log(ann).locator(".message.mine").getByLabel("Read")).toBeVisible({ timeout: 10000 });

  await ben.getByLabel("Message", { exact: true }).fill("Hi Ann");
  await ben.keyboard.press("Enter");
  await expect(log(ann).getByText("Hi Ann")).toBeVisible({ timeout: 10000 });

  const first = log(ann).locator(".message.mine").first();
  await first.hover();
  await first.getByRole("button", { name: "Edit message" }).click();
  await ann.getByLabel("Message", { exact: true }).fill("Hello Ben, edited");
  await ann.getByRole("button", { name: "Save message" }).click();
  await expect(log(ben).getByText("Hello Ben, edited")).toBeVisible({ timeout: 10000 });

  ann.once("dialog", (dialog) => dialog.accept());
  await first.hover();
  await first.getByRole("button", { name: "Delete message" }).click();
  await expect(log(ben).getByText("Message deleted")).toBeVisible({ timeout: 10000 });

  await waitForProfile(ann, `smoke-${run}-cat`);
  await ann.getByRole("tab", { name: "Group" }).click();
  await ann.getByLabel("Group name").fill(`Smoke ${run}`);
  for (const member of ["cat", "ben"]) {
    await ann.getByLabel("Search people").fill(`smoke-${run}-${member}`);
    await ann.locator(".person").filter({ hasText: `smoke-${run}-${member}` }).getByRole("button", { name: "Add" }).click();
  }
  await ann.getByRole("button", { name: /Create group/ }).click();
  await expect(ann.getByRole("heading", { name: `Smoke ${run}` })).toBeVisible();
  await ann.getByLabel("Message", { exact: true }).fill("Group hello");
  await ann.keyboard.press("Enter");
  await expect(cat.locator(".list")).toContainText(`Smoke ${run}`, { timeout: 15000 });
  await cat.locator(".list").getByRole("button", { name: new RegExp(`Smoke ${run}`) }).click();
  await expect(log(cat).getByText("Group hello")).toBeVisible();

  await ben.locator(".list").getByRole("button", { name: new RegExp(`^smoke-${run}-ann`) }).click();
  ben.once("dialog", (dialog) => dialog.accept());
  await ben.getByRole("button", { name: "Block user" }).click();
  await expect(ben.getByLabel("Message", { exact: true })).toHaveAttribute("placeholder", "You blocked this user");
  await ann.locator(".list").getByRole("button", { name: new RegExp(`^smoke-${run}-ben`) }).click();
  await ann.waitForTimeout(3000);
  await ann.getByLabel("Message", { exact: true }).fill("Are you there?");
  await ann.keyboard.press("Enter");
  await expect(log(ann).getByText("user is blocked")).toBeVisible({ timeout: 10000 });

  console.log(`live smoke passed (accounts smoke-${run}-*)`);
} finally {
  await browser.close();
}
