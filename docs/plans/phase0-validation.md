# Phase 0: Threads API Validation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Validate that Threads API publishing works before investing in full implementation.

**Architecture:** Simple Go script that authenticates via Meta OAuth and publishes a test post.

**Tech Stack:** Go, Threads API (Meta Graph API v18+)

---

### Task 1: Initialize Go Module

**Objective:** Set up Go module with basic dependencies

**Files:**
- Create: `go.mod`
- Create: `go.sum`

**Steps:**
```bash
cd /root/threads-affiliate
go mod init github.com/alifiandzaki131103-rgb/threads-affiliate-automation
```

---

### Task 2: Create Threads API Test Script

**Objective:** Script to test Threads API publish flow

**Files:**
- Create: `scripts/test_threads_api.go`

**What to test:**
1. Create a thread container (POST /{user_id}/threads)
2. Publish the thread (POST /{container_id}/threads_publish)
3. Verify the thread exists (GET /{thread_id})

**Note:** Requires a valid Threads access token. Get from Meta Developer Portal.

---

### Task 3: Verify API Compliance

**Objective:** Confirm automated posting is allowed

**Checks:**
- [ ] Meta Developer Portal: create app with Threads API access
- [ ] Confirm scopes: threads_basic, threads_content_publish, threads_manage_insights
- [ ] Test: publish 1 post via API
- [ ] Test: publish 3 posts with 1-hour intervals
- [ ] Confirm: no warnings or restrictions after test posts
- [ ] Document: any rate limits encountered

---

### Task 4: Test URL Resolution (Best-Effort)

**Objective:** Test if we can extract product info from Shopee/TikTok URLs

**Files:**
- Create: `scripts/test_url_resolve.go`

**Test URLs:**
- Shopee short: `s.shopee.co.id/xxx`
- TikTok short: `vt.tiktok.com/xxx`

**Expected:** Some will work (OG tags), some won't (SPA). Document success rate.
