import { chromium } from "@playwright/test";
import assert from "node:assert/strict";

const origin = process.env.MESSENGER_URL || "http://localhost";
const run = Date.now();
async function request(path, token, method = "GET", body) {
  const response = await fetch(`${origin}/api${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    signal: AbortSignal.timeout(20000),
  });
  if (!response.ok) throw new Error(`${method} ${path}: ${response.status}`);
  return response.json();
}
async function account(index) {
  const email = `smoke-${run}-${index}@example.test`;
  const tokens = await request("/auth/register", null, "POST", {
    email,
    password: `Smoke-${run}-password`,
  });
  for (let attempt = 0; attempt < 30; attempt++) {
    const response = await fetch(`${origin}/api/users/me`, {
      headers: { Authorization: `Bearer ${tokens.access_token}` },
    });
    if (response.ok) return { tokens, user: await response.json() };
    if (response.status !== 404) throw new Error(`profile: ${response.status}`);
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error("Profile creation timed out");
}
const [alice, bob, outsider] = await Promise.all([
  account(1),
  account(2),
  account(3),
]);
const chat = await request("/chat/", alice.tokens.access_token, "POST", {
  contact_id: bob.user.id,
});
const bobChats = await request("/chat/", bob.tokens.access_token);
assert(bobChats.some((item) => item.id === chat.id));
const denied = await fetch(`${origin}/api/messages/chats/${chat.id}/messages`, {
  headers: { Authorization: `Bearer ${outsider.tokens.access_token}` },
});
assert.equal(denied.status, 403);
const browser = await chromium.launch();
try {
  const contexts = await Promise.all([
    browser.newContext(),
    browser.newContext(),
  ]);
  const pages = await Promise.all(
    contexts.map(async (context, index) => {
      await context.addInitScript(
        (tokens) =>
          localStorage.setItem("messenger.session.v1", JSON.stringify(tokens)),
        [alice, bob][index].tokens,
      );
      const page = await context.newPage();
      await page.goto(origin);
      await page
        .getByRole("button", {
          name: new RegExp(
            [bob, alice][index].user.email.replaceAll(".", "\\."),
          ),
        })
        .click();
      await page.getByText("Connected", { exact: true }).waitFor();
      return page;
    }),
  );
  const message = `Live persistence check ${run}`;
  await pages[0]
    .getByRole("textbox", { name: "Message", exact: true })
    .fill(message);
  await pages[0].getByRole("button", { name: "Send message" }).click();
  await pages[1]
    .getByText(message, { exact: true })
    .waitFor({ timeout: 30000 });
  await pages[1].reload();
  await pages[1]
    .getByRole("button", {
      name: new RegExp(alice.user.email.replaceAll(".", "\\.")),
    })
    .click();
  await pages[1]
    .getByText(message, { exact: true })
    .waitFor({ timeout: 30000 });
  const deniedMessage = await fetch(
    `${origin}/api/messages/chats/${chat.id}/messages/not-a-message`,
    { headers: { Authorization: `Bearer ${outsider.tokens.access_token}` } },
  );
  assert.equal(deniedMessage.status, 403);
  console.log(
    "PASS: two-browser delivery, history after reload, server inbox, outsider denial.",
  );
  console.log(
    `Test account prefix: smoke-${run}; three accounts and one chat retained.`,
  );
} finally {
  await browser.close();
}
