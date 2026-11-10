import { expect, test, type Page } from "@playwright/test";
import { FakeBackend, message, tokens } from "./fake-backend";

function messages(page: Page) {
  return page.getByRole("log", { name: "Messages" });
}

async function openChat(page: Page, name: string) {
  await page.locator(".list").getByRole("button", { name: new RegExp(name) }).click();
}

test.describe("authentication", () => {
  test("signs in and signs out with token revocation", async ({ page }) => {
    const backend = new FakeBackend();
    await backend.install(page, false);
    await page.goto("/");

    await page.getByLabel("Email").fill("me@example.com");
    await page.getByLabel("Password").fill("password123");
    await page.getByRole("button", { name: "Sign in", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Chats" })).toBeVisible();
    expect(backend.callsTo("POST", "/auth/login")[0].body).toEqual({ email: "me@example.com", password: "password123" });

    await page.getByRole("button", { name: "Sign out" }).click();
    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
    await expect.poll(() => backend.callsTo("POST", "/auth/logout")[0]?.body).toEqual({ refresh_token: tokens.refresh_token });
  });

  test("validates registration and shows server errors", async ({ page }) => {
    const backend = new FakeBackend();
    backend.overrides.set("POST /auth/register", (route) => route.fulfill({ status: 409, json: { error: "email already taken" } }));
    await backend.install(page, false);
    await page.goto("/");

    await page.getByRole("button", { name: "Create one" }).click();
    await page.getByLabel("Email").fill("me@example.com");
    await page.getByLabel("Password").fill("short");
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(page.getByRole("alert")).toHaveText("Use a password between 8 and 72 bytes.");

    await page.getByLabel("Password").fill("long-enough-password");
    await page.getByRole("button", { name: "Create account" }).click();
    await expect(page.getByRole("alert")).toHaveText("email already taken");
  });

  test("refreshes an expired token once for concurrent requests", async ({ page }) => {
    const backend = new FakeBackend();
    await page.addInitScript(
      (value) => localStorage.setItem("messenger.session.v1", JSON.stringify(value)),
      { ...tokens, expires_at: "2000-01-01T00:00:00Z" },
    );
    await backend.install(page, false);
    await page.goto("/");

    await expect(page.getByRole("heading", { name: "Chats" })).toBeVisible();
    expect(backend.callsTo("POST", "/auth/refresh")).toHaveLength(1);
  });

  test("returns to sign in when the refresh token is rejected", async ({ page }) => {
    const backend = new FakeBackend();
    backend.overrides.set("POST /auth/refresh", (route) => route.fulfill({ status: 401, json: { error: "invalid or expired refresh token" } }));
    await page.addInitScript(
      (value) => localStorage.setItem("messenger.session.v1", JSON.stringify(value)),
      { ...tokens, expires_at: "2000-01-01T00:00:00Z" },
    );
    await backend.install(page, false);
    await page.goto("/");

    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
  });
});

test.describe("chats", () => {
  test("lists chats with previews and unread counts, and marks them read", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    backend.groupChat("g1", "Weekend", "kim", ["me", "sam"]);
    backend.messages.push(message("m1", "c1", "sam", "Are you there?", 1), message("m2", "g1", "kim", "Plans?", 2));
    backend.unread = { c1: 1, g1: 3 };
    await backend.install(page);
    await page.goto("/");

    const list = page.locator(".list");
    await expect(list.getByRole("button").first()).toContainText("Weekend");
    await expect(list.getByRole("button", { name: /Weekend/ })).toContainText("Kim Lee: Plans?");
    await expect(list.getByLabel("3 unread")).toBeVisible();
    await expect(list.getByLabel("1 unread")).toBeVisible();

    await openChat(page, "Sam Rivera");
    await expect(messages(page).getByText("Are you there?")).toBeVisible();
    await expect(list.getByLabel("1 unread")).toHaveCount(0);
    await expect.poll(() => backend.callsTo("POST", "/messages/chats/c1/read")[0]?.body).toEqual({ message_id: "m1" });
  });

  test("loads more chats with the cursor", async ({ page }) => {
    const backend = new FakeBackend();
    for (let i = 0; i < 52; i++) backend.groupChat(`g${i}`, `Group ${i}`, "me", ["sam"]);
    await backend.install(page);
    await page.goto("/");

    await expect(page.locator(".list > li > .list-item")).toHaveCount(50);
    await page.getByRole("button", { name: "Load more" }).click();
    await expect(page.locator(".list > li > .list-item")).toHaveCount(52);
    expect(backend.calls.some((call) => call.path === "/chat/" && call.method === "GET")).toBe(true);
  });

  test("starts a private chat from search and adds a contact", async ({ page }) => {
    const backend = new FakeBackend();
    await backend.install(page);
    await page.goto("/");

    await page.getByRole("button", { name: "New conversation" }).first().click();
    await page.getByLabel("Search people").fill("lee");
    const row = page.locator(".person").filter({ hasText: "Lee Park" });
    await row.getByRole("button", { name: "Add" }).click();
    await expect(row.getByRole("button", { name: "Add" })).toHaveCount(0);
    expect(backend.contacts).toContain("lee");

    await row.getByRole("button", { name: "Message" }).click();
    await expect(page.getByRole("heading", { name: "Lee Park" })).toBeVisible();
  });

  test("deletes a private chat", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    await backend.install(page);
    await page.goto("/");

    await openChat(page, "Sam Rivera");
    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Delete chat" }).click();
    await expect(page.locator(".list")).toHaveCount(0);
    await expect(page.getByText("No conversations yet.")).toBeVisible();
  });
});

test.describe("messages", () => {
  test("sends over the socket and confirms delivery", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    backend.autoConfirm = false;
    await backend.install(page);
    await page.goto("/");
    await openChat(page, "Sam Rivera");

    await page.getByLabel("Message", { exact: true }).fill("Hello Sam");
    await page.keyboard.press("Enter");
    const bubble = messages(page).locator(".message.mine");
    await expect(bubble.getByLabel("Sending")).toBeVisible();
    expect(backend.sent[0]).toMatchObject({ chat_id: "c1", type: "text", payload: { text: "Hello Sam" } });

    const id = String(backend.sent[0].message_id);
    backend.push({ type: "message.created", message: { message_id: id, chat_id: "c1", from_user: "me", type: "text", payload: { text: "Hello Sam" }, created_at: new Date().toISOString() } });
    await expect(bubble.getByLabel("Sent")).toBeVisible();
    await expect(page.getByLabel("Message", { exact: true })).toHaveValue("");
  });

  test("shows server rejections on the failed message", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    backend.autoConfirm = false;
    await backend.install(page);
    await page.goto("/");
    await openChat(page, "Sam Rivera");

    await page.getByLabel("Message", { exact: true }).fill("Hi");
    await page.keyboard.press("Enter");
    backend.push({ type: "error", message_id: backend.sent[0].message_id, error: "user is blocked" });
    await expect(messages(page).getByText("user is blocked")).toBeVisible();
    await expect(messages(page).getByLabel("Failed")).toBeVisible();
  });

  test("applies live edits, deletions and read receipts", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    backend.messages.push(message("m1", "c1", "sam", "Original", 1), message("m2", "c1", "me", "Mine", 2));
    await backend.install(page);
    await page.goto("/");
    await openChat(page, "Sam Rivera");

    const mine = messages(page).locator('[data-message-id="m2"]');
    await expect(mine.getByLabel("Sent")).toBeVisible();

    backend.push({ type: "message.edited", message: { message_id: "m1", chat_id: "c1", type: "text", payload: { text: "Changed" }, edited_at: "2026-11-10T10:05:00Z" } });
    await expect(messages(page).getByText("Changed")).toBeVisible();
    await expect(messages(page).locator('[data-message-id="m1"]')).toContainText("edited");

    backend.push({ type: "message.deleted", message: { message_id: "m1", chat_id: "c1", deleted_at: "2026-11-10T10:06:00Z" } });
    await expect(messages(page).locator('[data-message-id="m1"]')).toContainText("Message deleted");

    backend.push({ type: "message.read", message: { chat_id: "c1", user_id: "sam", message_id: "m2", read_at: "2026-11-10T10:02:00Z" } });
    await expect(mine.getByLabel("Read")).toBeVisible();
  });

  test("edits and deletes own messages", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    backend.messages.push(message("m1", "c1", "me", "Typo mesage", 1));
    await backend.install(page);
    await page.goto("/");
    await openChat(page, "Sam Rivera");

    const bubble = messages(page).locator('[data-message-id="m1"]');
    await bubble.hover();
    await bubble.getByRole("button", { name: "Edit message" }).click();
    await expect(page.getByText("Editing message")).toBeVisible();
    await page.getByLabel("Message", { exact: true }).fill("Typo message");
    await page.getByRole("button", { name: "Save message" }).click();
    await expect(bubble).toContainText("Typo message");
    expect(backend.callsTo("PATCH", "/messages/chats/c1/messages/m1")[0].body).toEqual({ payload: { text: "Typo message" } });

    page.once("dialog", (dialog) => dialog.accept());
    await bubble.hover();
    await bubble.getByRole("button", { name: "Delete message" }).click();
    await expect(bubble).toContainText("Message deleted");
  });

  test("counts unread messages from other chats and discovers new ones", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    await backend.install(page);
    await page.goto("/");
    await expect(page.locator(".list")).toContainText("Sam Rivera");

    backend.push({ type: "message.created", message: { message_id: "x1", chat_id: "c1", from_user: "sam", type: "text", payload: { text: "Ping" }, created_at: new Date().toISOString() } });
    await expect(page.locator(".list").getByLabel("1 unread")).toBeVisible();

    backend.privateChat("c2", "kim");
    backend.push({ type: "message.created", message: { message_id: "x2", chat_id: "c2", from_user: "kim", type: "text", payload: { text: "New chat" }, created_at: new Date().toISOString() } });
    await expect(page.locator(".list").getByRole("button", { name: /Kim Lee/ })).toContainText("New chat");
  });

  test("loads earlier history", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    for (let i = 0; i < 55; i++) backend.messages.push(message(`m${i}`, "c1", "sam", `Message ${i}`, i));
    await backend.install(page);
    await page.goto("/");
    await openChat(page, "Sam Rivera");

    await expect(messages(page).locator(".message")).toHaveCount(50);
    await page.getByRole("button", { name: "Load earlier messages" }).click();
    await expect(messages(page).locator(".message")).toHaveCount(55);
    await expect(messages(page).getByText("Message 0", { exact: true })).toBeVisible();
  });
});

test.describe("groups", () => {
  test("creates a group with selected members", async ({ page }) => {
    const backend = new FakeBackend();
    await backend.install(page);
    await page.goto("/");

    await page.getByRole("button", { name: "New conversation" }).first().click();
    await page.getByRole("tab", { name: "Group" }).click();
    await page.getByLabel("Group name").fill("Team");
    await page.locator(".person").filter({ hasText: "Sam Rivera" }).getByRole("button", { name: "Add" }).click();
    await page.locator(".person").filter({ hasText: "Kim Lee" }).getByRole("button", { name: "Add" }).click();
    await page.getByRole("button", { name: "Create group (3)" }).click();

    await expect(page.getByRole("heading", { name: "Team" })).toBeVisible();
    expect(backend.callsTo("POST", "/chats/groups")[0].body).toEqual({ title: "Team", member_ids: ["sam", "kim"] });
  });

  test("owner renames the group and manages members", async ({ page }) => {
    const backend = new FakeBackend();
    backend.groupChat("g1", "Team", "me", ["sam", "kim"]);
    await backend.install(page);
    await page.goto("/");
    await openChat(page, "Team");

    await page.getByRole("button", { name: "Group info" }).click();
    const dialog = page.getByRole("dialog", { name: "Group" });
    await dialog.getByLabel("Group name").fill("Core team");
    await dialog.getByRole("button", { name: "Rename" }).click();
    await expect(page.getByRole("heading", { name: "Core team" })).toBeVisible();

    await dialog.getByRole("button", { name: "Remove Kim Lee" }).click();
    await expect(dialog.getByText("2 members")).toBeVisible();

    await dialog.getByRole("button", { name: "Add members" }).click();
    await dialog.getByLabel("Search people").fill("lee");
    await dialog.locator(".person").filter({ hasText: "Lee Park" }).getByRole("button", { name: "Add" }).click();
    await expect(dialog.getByText("3 members")).toBeVisible();
  });

  test("members see read-only info and can leave", async ({ page }) => {
    const backend = new FakeBackend();
    backend.groupChat("g1", "Team", "sam", ["me"]);
    await backend.install(page);
    await page.goto("/");
    await openChat(page, "Team");

    await page.getByRole("button", { name: "Group info" }).click();
    const dialog = page.getByRole("dialog", { name: "Group" });
    await expect(dialog.getByRole("button", { name: "Rename" })).toHaveCount(0);
    await expect(dialog.getByRole("button", { name: /Remove/ })).toHaveCount(0);
    await dialog.getByRole("button", { name: "Close" }).click();

    page.once("dialog", (confirm) => confirm.accept());
    await page.getByRole("button", { name: "Leave group" }).click();
    await expect(page.getByText("No conversations yet.")).toBeVisible();
    expect(backend.callsTo("DELETE", "/chats/g1")).toHaveLength(1);
  });
});

test.describe("people", () => {
  test("removes contacts and blocks users", async ({ page }) => {
    const backend = new FakeBackend();
    await backend.install(page);
    await page.goto("/");
    await page.getByRole("button", { name: "Contacts", exact: true }).click();

    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Remove Kim Lee" }).click();
    await expect(page.locator(".list")).not.toContainText("Kim Lee");

    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Block Sam Rivera" }).click();
    await expect(page.getByText("No contacts yet.")).toBeVisible();
    expect(backend.blocked).toEqual(["sam"]);

    await page.getByRole("button", { name: "Settings" }).click();
    const blockedRow = page.locator(".settings-section").filter({ hasText: "Blocked users" }).locator(".person");
    await expect(blockedRow).toContainText("Sam Rivera");
    await blockedRow.getByRole("button", { name: "Unblock" }).click();
    await expect(page.getByText("Nobody is blocked.")).toBeVisible();
  });

  test("disables writing to deleted and blocked users", async ({ page }) => {
    const backend = new FakeBackend();
    backend.user("sam").deleted = true;
    backend.privateChat("c1", "sam");
    backend.privateChat("c2", "kim");
    backend.blocked = ["kim"];
    await backend.install(page);
    await page.goto("/");

    await openChat(page, "Deleted account");
    await expect(page.getByLabel("Message", { exact: true })).toBeDisabled();
    await expect(page.getByLabel("Message", { exact: true })).toHaveAttribute("placeholder", "This account was deleted");

    await openChat(page, "Kim Lee");
    await expect(page.getByLabel("Message", { exact: true })).toHaveAttribute("placeholder", "You blocked this user");
    await expect(page.getByRole("button", { name: "Block user" })).toHaveCount(0);
  });
});

test.describe("settings", () => {
  test("updates profile and changes password", async ({ page }) => {
    const backend = new FakeBackend();
    await backend.install(page);
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();

    await page.getByLabel("Display name").fill("Alex Updated");
    await page.getByRole("button", { name: "Save profile" }).click();
    await expect(page.getByText("Profile saved.")).toBeVisible();

    await page.getByLabel("Current password").fill("old-password");
    await page.getByLabel("New password").fill("new-password");
    await page.getByRole("button", { name: "Change password" }).click();
    await expect(page.getByText("Password changed. Other devices were signed out.")).toBeVisible();
    expect(backend.callsTo("PATCH", "/auth/password")[0].body).toEqual({ current_password: "old-password", new_password: "new-password" });
    expect(backend.callsTo("POST", "/auth/login")[0].body).toEqual({ email: "me@example.com", password: "new-password" });
  });

  test("reports a wrong current password", async ({ page }) => {
    const backend = new FakeBackend();
    backend.overrides.set("PATCH /auth/password", (route) => route.fulfill({ status: 403, json: { error: "current password is incorrect" } }));
    await backend.install(page);
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();

    await page.getByLabel("Current password").fill("wrong-password");
    await page.getByLabel("New password").fill("new-password");
    await page.getByRole("button", { name: "Change password" }).click();
    await expect(page.getByText("current password is incorrect")).toBeVisible();
  });

  test("signs out on all devices", async ({ page }) => {
    const backend = new FakeBackend();
    await backend.install(page);
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();
    await page.getByRole("button", { name: "Sign out on all devices" }).click();

    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
    expect(backend.callsTo("POST", "/auth/logout-all")).toHaveLength(1);
  });

  test("deletes the account after confirmation", async ({ page }) => {
    const backend = new FakeBackend();
    await backend.install(page);
    await page.goto("/");
    await page.getByRole("button", { name: "Settings" }).click();

    await page.getByLabel("Confirm password").fill("password123");
    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Delete", exact: true }).click();

    await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();
    expect(backend.callsTo("DELETE", "/auth/account")[0].body).toEqual({ password: "password123" });
  });
});

test.describe("layout", () => {
  test("mobile shows either the list or the chat", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    backend.messages.push(message("m1", "c1", "sam", "Hi on mobile", 1));
    await page.setViewportSize({ width: 390, height: 780 });
    await backend.install(page);
    await page.goto("/");

    await expect(page.getByRole("heading", { name: "Chats" })).toBeVisible();
    await openChat(page, "Sam Rivera");
    await expect(messages(page).getByText("Hi on mobile")).toBeVisible();
    await expect(page.getByRole("heading", { name: "Chats" })).toBeHidden();

    await page.getByRole("button", { name: "Back" }).click();
    await expect(page.getByRole("heading", { name: "Chats" })).toBeVisible();
  });

  test("reconnects after the socket drops", async ({ page }) => {
    const backend = new FakeBackend();
    backend.privateChat("c1", "sam");
    await backend.install(page);
    await page.goto("/");
    await openChat(page, "Sam Rivera");

    await expect(page.locator(".chat-title span")).toContainText("sam@example.com");
    await backend.socket?.close();
    await expect(page.getByText("Connection lost. Retrying…")).toBeVisible();
    await expect(page.getByRole("button", { name: "Send message" })).toBeDisabled();
    await page.getByRole("button", { name: "Reconnect now" }).click();
    await expect(page.getByText("Connection lost. Retrying…")).toBeHidden();
  });
});
