import { test, expect, type Page } from "@playwright/test";
const me = {
  id: "me",
  email: "me@example.com",
  display_name: "Alex Morgan",
  avatar_url: "",
};
const friend = {
  id: "friend",
  email: "sam@example.com",
  display_name: "Sam Rivera",
  avatar_url: "",
};
const tokens = {
  access_token: "access",
  refresh_token: "refresh",
  expires_at: "2099-01-01T00:00:00Z",
};
async function backend(page: Page, signedIn = true) {
  if (signedIn)
    await page.addInitScript(
      (value) =>
        localStorage.setItem("messenger.session.v1", JSON.stringify(value)),
      tokens,
    );
  await page.route("**/api/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    let data: unknown;
    if (path.startsWith("/api/auth/")) data = tokens;
    else if (path === "/api/users/me")
      data =
        route.request().method() === "PATCH"
          ? { ...me, ...route.request().postDataJSON() }
          : me;
    else if (path === "/api/users/me/contacts")
      data =
        route.request().method() === "POST" ? { status: "success" } : [friend];
    else if (path === "/api/users/search") data = [friend];
    else if (path === "/api/chat/") data = route.request().method() === "POST" ? { id: "room" } : [];
    else if (path === "/api/messages/chats/room/messages")
      data = [
        {
          message_id: "first",
          chat_id: "room",
          from_user: "friend",
          type: "text",
          payload: { text: "Hello from the history API" },
          created_at: "2026-10-07T10:00:00Z",
        },
      ];
    else
      return route.fulfill({
        status: 404,
        json: { error: "Unexpected endpoint" },
      });
    return route.fulfill({ json: data });
  });
}
async function openChat(page: Page) {
  await page.getByRole("button", { name: "Contacts", exact: true }).click();
  await page
    .getByRole("button", { name: /Sam Rivera sam@example.com/ })
    .click();
}
test("login, profile editing and sign out", async ({ page }) => {
  await backend(page, false);
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Welcome back." }),
  ).toBeVisible();
  await page.getByLabel("Email address").fill("me@example.com");
  await page.getByLabel("Password", { exact: true }).fill("password123");
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await page.getByRole("button", { name: "Edit profile" }).click();
  await page.getByLabel("Display name").fill("Alex Updated");
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByRole("dialog")).not.toBeVisible();
  await page.getByRole("button", { name: "Edit profile" }).click();
  await expect(page.getByLabel("Display name")).toHaveValue("Alex Updated");
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(
    page.getByRole("heading", { name: "Welcome back." }),
  ).toBeVisible();
});
test("registration submits backend contract and reports failures", async ({
  page,
}) => {
  await backend(page, false);
  await page.route("**/api/auth/register", (route) =>
    route.fulfill({ status: 409, json: { error: "email already taken" } }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "Create an account" }).click();
  await page.getByLabel("Email address").fill("me@example.com");
  await page.getByLabel("Password", { exact: true }).fill("password123");
  await page
    .getByRole("button", { name: "Create account", exact: true })
    .click();
  await expect(page.getByRole("alert")).toContainText("email already taken");
});
test("history normalization, WebSocket confirmation, and draft persistence", async ({
  page,
}) => {
  await backend(page);
  await page.routeWebSocket("**/ws?*", (ws) => {
    ws.onMessage((data) => {
      const message = JSON.parse(String(data));
      ws.send(JSON.stringify({
        type: "message.created",
        message: {
          message_id: message.message_id,
          chat_id: "room",
          from_user: "me",
          type: "text",
          payload: { text: message.payload.text },
          created_at: "2026-10-08T10:00:00Z",
        },
      }));
    });
  });
  await page.goto("/");
  await openChat(page);
  await expect(page.getByText("Hello from the history API")).toBeVisible();
  await expect(page.getByText("Connected", { exact: true })).toBeVisible();
  await page
    .getByRole("textbox", { name: "Message", exact: true })
    .fill("A new message");
  await page.getByRole("button", { name: "Send message" }).click();
  await expect(page.getByText("A new message", { exact: true })).toHaveCount(1);
  await expect(page.getByText("Unconfirmed")).toHaveCount(0);
  await page
    .getByRole("textbox", { name: "Message", exact: true })
    .fill("Keep this draft");
  await page
    .getByRole("button", { name: "Conversations", exact: true })
    .click();
  await page
    .getByRole("button", { name: /Sam Rivera sam@example.com/ })
    .click();
  await expect(
    page.getByRole("textbox", { name: "Message", exact: true }),
  ).toHaveValue("Keep this draft");
});
test("failed WebSocket preserves drafts and prevents sending on mobile", async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await backend(page);
  await page.routeWebSocket("**/ws?*", (ws) => {
    setTimeout(() => ws.close(), 100);
  });
  await page.goto("/");
  await openChat(page);
  await expect(
    page.getByText("Live connection unavailable.", { exact: false }),
  ).toBeVisible();
  await page
    .getByRole("textbox", { name: "Message", exact: true })
    .fill("Offline draft");
  await expect(
    page.getByRole("button", { name: "Send message" }),
  ).toBeDisabled();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await page.getByRole("button", { name: "Back to conversations" }).click();
  await expect(page.getByRole("heading", { name: /Messages/ })).toBeVisible();
});
test("people search and adding a contact", async ({ page }) => {
  await backend(page);
  await page.route("**/api/users/me/contacts", (route) =>
    route.fulfill({
      json: route.request().method() === "GET" ? [] : { status: "success" },
    }),
  );
  await page.goto("/");
  await page.getByRole("button", { name: "New conversation" }).click();
  await page.getByLabel("Find people").fill("sam");
  await page
    .getByRole("button", { name: "Add Sam Rivera to contacts" })
    .click();
  await expect(
    page.getByRole("button", { name: "Add Sam Rivera to contacts" }),
  ).toHaveCount(0);
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Contacts", exact: true }).click();
  await expect(page.getByText("sam@example.com")).toBeVisible();
});
test("concurrent expired requests use one token refresh", async ({ page }) => {
  await backend(page, false);
  await page.addInitScript(
    (value) =>
      localStorage.setItem(
        "messenger.session.v1",
        JSON.stringify({ ...value, expires_at: "2000-01-01T00:00:00Z" }),
      ),
    tokens,
  );
  let refreshes = 0;
  await page.route("**/api/auth/refresh", async (route) => {
    refreshes++;
    await route.fulfill({ json: tokens });
  });
  await page.goto("/");
  await expect(
    page.getByRole("button", { name: "New conversation" }),
  ).toBeEnabled();
  expect(refreshes).toBe(1);
});
test("expired refresh returns to login", async ({ page }) => {
  await backend(page, false);
  await page.addInitScript(
    (value) =>
      localStorage.setItem(
        "messenger.session.v1",
        JSON.stringify({ ...value, expires_at: "2000-01-01T00:00:00Z" }),
      ),
    tokens,
  );
  await page.route("**/api/auth/refresh", (route) =>
    route.fulfill({
      status: 401,
      json: { error: "invalid or expired refresh token" },
    }),
  );
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Welcome back." }),
  ).toBeVisible();
});
test("history pagination and incoming live broadcasts", async ({ page }) => {
  await backend(page);
  const history = Array.from({ length: 50 }, (_, index) => ({
    message_id: `history-${index}`,
    chat_id: "room",
    from_user: "friend",
    type: "text",
    payload: { text: `History message ${index}` },
    created_at: new Date(Date.UTC(2026, 9, 7, 12, 59 - index)).toISOString(),
  }));
  await page.route("**/api/messages/chats/room/messages?**", (route) =>
    route.fulfill({
      json: new URL(route.request().url()).searchParams.has("before")
        ? [
            {
              ...history[0],
              message_id: "oldest",
              payload: { text: "Earlier message" },
              created_at: "2026-10-06T10:00:00Z",
            },
          ]
        : history,
    }),
  );
  await page.routeWebSocket("**/ws?*", (ws) => {
    setTimeout(
      () =>
        ws.send(
          JSON.stringify({
            type: "message.created",
            message: {
              message_id: "live-message",
              chat_id: "room",
              from_user: "friend",
              type: "text",
              payload: { text: "Live message before persistence" },
              created_at: "2026-10-08T10:00:00Z",
            },
          }),
        ),
      200,
    );
  });
  await page.goto("/");
  await openChat(page);
  await expect(page.getByText("Live message before persistence")).toBeVisible();
  await page.getByRole("button", { name: "Load earlier messages" }).click();
  await expect(page.getByText("Earlier message")).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Load earlier messages" }),
  ).toHaveCount(0);
});
test("desktop and mobile visual review", async ({ page }, testInfo) => {
  await backend(page, false);
  await page.setViewportSize({ width: 1440, height: 960 });
  await page.goto("/");
  await page.screenshot({ path: testInfo.outputPath("login-desktop.png") });
  await page.getByLabel("Email address").fill("me@example.com");
  await page.getByLabel("Password", { exact: true }).fill("password123");
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "New conversation" }),
  ).toBeEnabled();
  await page.screenshot({ path: testInfo.outputPath("workspace-desktop.png") });
  await page.routeWebSocket("**/ws?*", () => {});
  await openChat(page);
  await expect(page.getByText("Hello from the history API")).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath("chat-desktop.png") });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: testInfo.outputPath("chat-mobile.png") });
});
