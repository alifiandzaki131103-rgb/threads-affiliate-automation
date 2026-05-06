# Phase 2: Full Posting Flow Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Make the platform fully functional — users can connect Threads accounts, generate AI content, review/approve posts, and auto-publish to Threads.

**Architecture:** Add account handler (manual token input, no OAuth for MVP), post handler (CRUD + generate + approve + publish), and update frontend with Account Settings and functional Posts page.

**Tech Stack:** Go (Fiber), PostgreSQL, Asynq, React + TailwindCSS

---

## Task 1: Account Handler — Backend

**Objective:** Create CRUD endpoints for Threads accounts (manual access token input for MVP — Meta OAuth is complex and requires app review)

**Files:**
- Create: `internal/handler/account_handler.go`
- Create: `internal/repository/account_repo.go`

### internal/repository/account_repo.go

```go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
)

func CreateAccount(ctx context.Context, pool *pgxpool.Pool, account *model.ThreadsAccount) error {
	account.ID = uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO threads_accounts (id, user_id, threads_user_id, access_token, persona, niche, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		account.ID, account.UserID, account.ThreadsUserID, account.AccessToken,
		account.Persona, account.Niche, account.Status,
	)
	return err
}

func GetAccountsByUserID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]model.ThreadsAccount, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, user_id, threads_user_id, persona, niche, status, created_at
		FROM threads_accounts WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []model.ThreadsAccount
	for rows.Next() {
		var a model.ThreadsAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.ThreadsUserID, &a.Persona, &a.Niche, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func GetAccountByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*model.ThreadsAccount, error) {
	var a model.ThreadsAccount
	err := pool.QueryRow(ctx, `
		SELECT id, user_id, threads_user_id, access_token, persona, niche, status, created_at
		FROM threads_accounts WHERE id = $1`, id).Scan(
		&a.ID, &a.UserID, &a.ThreadsUserID, &a.AccessToken, &a.Persona, &a.Niche, &a.Status, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func UpdateAccount(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, persona, niche, status string) error {
	_, err := pool.Exec(ctx, `
		UPDATE threads_accounts SET persona = $1, niche = $2, status = $3 WHERE id = $4`,
		persona, niche, status, id)
	return err
}

func DeleteAccount(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM threads_accounts WHERE id = $1`, id)
	return err
}
```

### internal/handler/account_handler.go

```go
package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
)

type AccountHandler struct {
	pool *pgxpool.Pool
}

func NewAccountHandler(pool *pgxpool.Pool) *AccountHandler {
	return &AccountHandler{pool: pool}
}

func (h *AccountHandler) CreateAccount(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req struct {
		ThreadsUserID string `json:"threads_user_id"`
		AccessToken   string `json:"access_token"`
		Persona       string `json:"persona"`
		Niche         string `json:"niche"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.ThreadsUserID == "" || req.AccessToken == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "threads_user_id and access_token are required"})
	}

	account := &model.ThreadsAccount{
		UserID:        userID,
		ThreadsUserID: req.ThreadsUserID,
		AccessToken:   req.AccessToken,
		Persona:       req.Persona,
		Niche:         req.Niche,
		Status:        "active",
	}

	if err := repository.CreateAccount(c.Context(), h.pool, account); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create account"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":              account.ID,
		"threads_user_id": account.ThreadsUserID,
		"persona":         account.Persona,
		"niche":           account.Niche,
		"status":          account.Status,
	})
}

func (h *AccountHandler) ListAccounts(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	accounts, err := repository.GetAccountsByUserID(c.Context(), h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch accounts"})
	}

	return c.JSON(fiber.Map{"count": len(accounts), "accounts": accounts})
}

func (h *AccountHandler) UpdateAccount(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid account id"})
	}

	var req struct {
		Persona string `json:"persona"`
		Niche   string `json:"niche"`
		Status  string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := repository.UpdateAccount(c.Context(), h.pool, accountID, req.Persona, req.Niche, req.Status); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update account"})
	}

	return c.JSON(fiber.Map{"message": "account updated"})
}

func (h *AccountHandler) DeleteAccount(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid account id"})
	}
	_ = userID // TODO: verify ownership

	if err := repository.DeleteAccount(c.Context(), h.pool, accountID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete account"})
	}

	return c.SendStatus(http.StatusNoContent)
}
```

---

## Task 2: Post Handler — Backend

**Objective:** Create endpoints for posts: list, generate content (trigger AI), approve, and publish

**Files:**
- Create: `internal/handler/post_handler.go`
- Create: `internal/repository/post_repo.go`

### internal/repository/post_repo.go

```go
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
)

func GetPostsByUserID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]model.Post, error) {
	rows, err := pool.Query(ctx, `
		SELECT p.id, p.account_id, p.link_id, p.content, p.link_placement, p.persona, p.format,
		       p.scheduled_at, p.published_at, p.thread_id, p.status, p.created_at
		FROM posts p
		JOIN threads_accounts ta ON p.account_id = ta.id
		WHERE ta.user_id = $1
		ORDER BY p.created_at DESC
		LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.AccountID, &p.LinkID, &p.Content, &p.LinkPlacement,
			&p.Persona, &p.Format, &p.ScheduledAt, &p.PublishedAt, &p.ThreadID, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func UpdatePostStatus(ctx context.Context, pool *pgxpool.Pool, postID uuid.UUID, status string) error {
	_, err := pool.Exec(ctx, `UPDATE posts SET status = $1 WHERE id = $2`, status, postID)
	return err
}

func GetPostByID(ctx context.Context, pool *pgxpool.Pool, postID uuid.UUID) (*model.Post, error) {
	var p model.Post
	err := pool.QueryRow(ctx, `
		SELECT id, account_id, link_id, content, link_placement, persona, format,
		       scheduled_at, published_at, thread_id, status, created_at
		FROM posts WHERE id = $1`, postID).Scan(
		&p.ID, &p.AccountID, &p.LinkID, &p.Content, &p.LinkPlacement,
		&p.Persona, &p.Format, &p.ScheduledAt, &p.PublishedAt, &p.ThreadID, &p.Status, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetScheduledPostsDue(ctx context.Context, pool *pgxpool.Pool) ([]model.Post, error) {
	rows, err := pool.Query(ctx, `
		SELECT p.id, p.account_id, p.content, p.scheduled_at, p.status
		FROM posts p
		WHERE p.status = 'approved' AND p.scheduled_at <= $1
		ORDER BY p.scheduled_at ASC
		LIMIT 25`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.AccountID, &p.Content, &p.ScheduledAt, &p.Status); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}
```

### internal/handler/post_handler.go

```go
package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/ai"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/queue"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
)

type PostHandler struct {
	pool        *pgxpool.Pool
	rdb         *redis.Client
	aiClient    *ai.Client
	queueClient *queue.Client
}

func NewPostHandler(pool *pgxpool.Pool, rdb *redis.Client, aiClient *ai.Client, queueClient *queue.Client) *PostHandler {
	return &PostHandler{pool: pool, rdb: rdb, aiClient: aiClient, queueClient: queueClient}
}

// ListPosts returns all posts for the authenticated user
func (h *PostHandler) ListPosts(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	posts, err := repository.GetPostsByUserID(c.Context(), h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch posts"})
	}

	return c.JSON(fiber.Map{"count": len(posts), "posts": posts})
}

// GenerateContent triggers AI content generation for a specific link
func (h *PostHandler) GenerateContent(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req struct {
		LinkID    string `json:"link_id"`
		AccountID string `json:"account_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	linkID, err := uuid.Parse(req.LinkID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid link_id"})
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid account_id"})
	}

	// Get link details
	links, err := repository.GetLinksByUserID(c.Context(), h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch link"})
	}

	var productName, category, platform, shortURL string
	var price float64
	found := false
	for _, l := range links {
		if l.ID == linkID {
			productName = l.ProductName
			category = l.Category
			platform = l.Platform
			price = l.Price
			shortURL = "https://affiliate.billingku.online/s/" + l.ShortSlug
			found = true
			break
		}
	}

	if !found {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "link not found"})
	}

	// Enqueue content generation task
	payload := &queue.GenerateContentPayload{
		LinkID:      linkID,
		ProductName: productName,
		Price:       price,
		Category:    category,
		Platform:    platform,
		ShortURL:    shortURL,
		UserID:      userID,
		AccountID:   accountID,
	}

	task, err := queue.NewGenerateContentTask(payload)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create task"})
	}

	_, err = h.queueClient.Enqueue(task)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to enqueue task"})
	}

	return c.Status(http.StatusAccepted).JSON(fiber.Map{
		"message": "Content generation queued",
		"link_id": linkID,
	})
}

// ApprovePost approves a post for publishing
func (h *PostHandler) ApprovePost(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	postID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	if err := repository.UpdatePostStatus(c.Context(), h.pool, postID, "approved"); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to approve post"})
	}

	return c.JSON(fiber.Map{"message": "post approved", "post_id": postID})
}

// PublishNow immediately publishes a post
func (h *PostHandler) PublishNow(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	postID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	// Get post
	post, err := repository.GetPostByID(c.Context(), h.pool, postID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	// Get account for access token
	account, err := repository.GetAccountByID(c.Context(), h.pool, post.AccountID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "account not found"})
	}

	// Enqueue publish task (immediate)
	payload := &queue.PublishPostPayload{
		PostID:        post.ID,
		AccountID:     post.AccountID,
		Content:       post.Content,
		ThreadsUserID: account.ThreadsUserID,
		AccessToken:   account.AccessToken,
	}

	task, err := queue.NewPublishPostTask(payload, time.Now())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create publish task"})
	}

	_, err = h.queueClient.Enqueue(task)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to enqueue publish task"})
	}

	return c.Status(http.StatusAccepted).JSON(fiber.Map{
		"message": "Post queued for immediate publishing",
		"post_id": postID,
	})
}
```

**Note:** The `time` import is needed in post_handler.go. Also need to update `repository/link_repo.go` to return product info with links.

---

## Task 3: Update Link Repository to Include Product Info

**Objective:** Modify GetLinksByUserID to JOIN with products table and return product name, price, category

**Files:**
- Modify: `internal/repository/link_repo.go`
- Modify: `internal/model/requests.go` (add LinkWithProduct struct)

### Add to internal/model/requests.go

```go
// LinkWithProduct combines affiliate link data with product info
type LinkWithProduct struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	OriginalURL string    `json:"original_url"`
	ShortSlug   string    `json:"short_slug"`
	Platform    string    `json:"platform"`
	Status      string    `json:"status"`
	ClickCount  int       `json:"click_count"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
}
```

### Update internal/repository/link_repo.go — GetLinksByUserID

Change the query to JOIN products:

```go
func GetLinksByUserID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]model.LinkWithProduct, error) {
	rows, err := pool.Query(ctx, `
		SELECT al.id, al.product_id, al.original_url, al.short_slug, al.platform, al.status, al.click_count,
		       COALESCE(p.name, 'Unknown'), COALESCE(p.price, 0), COALESCE(p.category, ''), al.created_at
		FROM affiliate_links al
		JOIN products p ON al.product_id = p.id
		WHERE p.user_id = $1
		ORDER BY al.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.LinkWithProduct
	for rows.Next() {
		var l model.LinkWithProduct
		if err := rows.Scan(&l.ID, &l.ProductID, &l.OriginalURL, &l.ShortSlug, &l.Platform,
			&l.Status, &l.ClickCount, &l.ProductName, &l.Price, &l.Category, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}
```

---

## Task 4: Register New Routes in main.go

**Objective:** Wire up account and post handlers in the API router

**Files:**
- Modify: `cmd/api/main.go`

### Add to imports:

```go
"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/ai"
"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/queue"
```

### Add after linkHandler initialization:

```go
// AI client
aiClient := ai.NewClient(cfg.AI.APIURL)

// Queue client
queueClient := queue.NewClient(fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port))

// Handlers
accountHandler := handler.NewAccountHandler(pool)
postHandler := handler.NewPostHandler(pool, rdb, aiClient, queueClient)
```

### Add routes in protected group:

```go
// Account routes
protected.Post("/accounts", accountHandler.CreateAccount)
protected.Get("/accounts", accountHandler.ListAccounts)
protected.Put("/accounts/:id", accountHandler.UpdateAccount)
protected.Delete("/accounts/:id", accountHandler.DeleteAccount)

// Post routes
protected.Get("/posts", postHandler.ListPosts)
protected.Post("/posts/generate", postHandler.GenerateContent)
protected.Post("/posts/:id/approve", postHandler.ApprovePost)
protected.Post("/posts/:id/publish", postHandler.PublishNow)
```

---

## Task 5: Update Config to Include AI URL

**Objective:** Ensure config.go has AI section

**Files:**
- Modify: `internal/config/config.go` (add AI struct if missing)

### Add to Config struct:

```go
type Config struct {
	App   AppConfig
	DB    DBConfig
	Redis RedisConfig
	JWT   JWTConfig
	AI    AIConfig
}

type AIConfig struct {
	APIURL string
}
```

### In Load() function, add:

```go
cfg.AI.APIURL = getEnv("AI_API_URL", "http://ai-service:8081")
```

---

## Task 6: Frontend — Account Settings Page

**Objective:** Create page to add/manage Threads accounts

**Files:**
- Create: `web/frontend/src/pages/Accounts.jsx`

```jsx
import { useState, useEffect } from 'react';
import { UserCircle, Plus, Trash2, CheckCircle } from 'lucide-react';
import api from '../api';

export default function Accounts() {
  const [accounts, setAccounts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ threads_user_id: '', access_token: '', persona: 'honest_friend', niche: '' });
  const [saving, setSaving] = useState(false);

  useEffect(() => { loadAccounts(); }, []);

  const loadAccounts = async () => {
    try {
      const { data } = await api.get('/accounts');
      setAccounts(data.accounts || []);
    } catch (err) {
      console.error('Failed to load accounts:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      await api.post('/accounts', form);
      setShowForm(false);
      setForm({ threads_user_id: '', access_token: '', persona: 'honest_friend', niche: '' });
      loadAccounts();
    } catch (err) {
      alert('Failed to add account: ' + (err.response?.data?.error || err.message));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this account?')) return;
    try {
      await api.delete(`/accounts/${id}`);
      loadAccounts();
    } catch (err) {
      alert('Failed to delete account');
    }
  };

  const personas = ['honest_friend', 'hot_take', 'problem_solver', 'curious_explorer', 'lifestyle_sharer', 'comparison_nerd'];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-white">Threads Accounts</h2>
        <button onClick={() => setShowForm(!showForm)} className="flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg transition text-sm">
          <Plus size={16} /> Add Account
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="bg-gray-900 rounded-xl border border-gray-800 p-5 mb-6 space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Threads User ID</label>
            <input type="text" value={form.threads_user_id} onChange={e => setForm({...form, threads_user_id: e.target.value})}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm" placeholder="Your Threads numeric user ID" required />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Access Token</label>
            <input type="password" value={form.access_token} onChange={e => setForm({...form, access_token: e.target.value})}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm" placeholder="From Meta Developer Portal" required />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Persona</label>
            <select value={form.persona} onChange={e => setForm({...form, persona: e.target.value})}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm">
              {personas.map(p => <option key={p} value={p}>{p.replace('_', ' ')}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Niche</label>
            <input type="text" value={form.niche} onChange={e => setForm({...form, niche: e.target.value})}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm" placeholder="e.g. skincare, tech, fashion" />
          </div>
          <button type="submit" disabled={saving} className="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg text-sm disabled:opacity-50">
            {saving ? 'Saving...' : 'Connect Account'}
          </button>
        </form>
      )}

      {loading ? (
        <div className="text-center text-gray-400 py-8">Loading...</div>
      ) : accounts.length === 0 ? (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center">
          <UserCircle className="mx-auto text-gray-600 mb-3" size={32} />
          <p className="text-gray-400">Belum ada akun Threads terhubung.</p>
          <p className="text-gray-500 text-sm mt-1">Tambahkan akun untuk mulai auto-posting.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {accounts.map(account => (
            <div key={account.id} className="bg-gray-900 rounded-xl border border-gray-800 p-4 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <CheckCircle className="text-green-400" size={20} />
                <div>
                  <p className="text-white text-sm font-medium">ID: {account.threads_user_id}</p>
                  <p className="text-gray-500 text-xs">{account.persona?.replace('_', ' ')} • {account.niche || 'No niche'}</p>
                </div>
              </div>
              <button onClick={() => handleDelete(account.id)} className="text-red-400 hover:text-red-300 p-2">
                <Trash2 size={16} />
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="mt-6 bg-gray-900/50 rounded-xl border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-2">📋 Cara Mendapatkan Access Token</h3>
        <ol className="text-xs text-gray-500 space-y-1 list-decimal list-inside">
          <li>Buka <a href="https://developers.facebook.com" target="_blank" className="text-indigo-400 hover:underline">Meta Developer Portal</a></li>
          <li>Buat App → pilih "Business" type</li>
          <li>Tambahkan product "Threads API"</li>
          <li>Generate User Token dengan scope: threads_basic, threads_content_publish</li>
          <li>Copy User ID dan Access Token ke form di atas</li>
        </ol>
      </div>
    </div>
  );
}
```

---

## Task 7: Frontend — Update Posts Page (Functional)

**Objective:** Make Posts page fetch real data and support generate/approve/publish actions

**Files:**
- Rewrite: `web/frontend/src/pages/Posts.jsx`

The Posts page should:
1. Fetch posts from `/api/posts`
2. Show "Generate Content" button that calls `/api/posts/generate` with selected link + account
3. Show "Approve" button for pending_review posts
4. Show "Publish Now" button for approved posts
5. Real-time status updates

---

## Task 8: Frontend — Update App.jsx Routes

**Objective:** Add Accounts route and navigation

**Files:**
- Modify: `web/frontend/src/App.jsx` — add Accounts import and route
- Modify: `web/frontend/src/components/Layout.jsx` — add Accounts nav item

### App.jsx changes:

```jsx
import Accounts from './pages/Accounts';
// Add route:
<Route path="accounts" element={<Accounts />} />
```

### Layout.jsx — add nav item:

```jsx
{ path: '/accounts', label: 'Accounts', icon: <UserCircle size={18} /> }
```

---

## Task 9: Build & Deploy

**Objective:** Rebuild Docker containers and verify everything works

**Steps:**
```bash
cd /root/threads-affiliate
docker compose build api worker frontend
docker compose up -d
# Test endpoints
curl -s http://localhost:8080/api/accounts -H "Authorization: Bearer <token>" | jq
curl -s http://localhost:8080/api/posts -H "Authorization: Bearer <token>" | jq
```

---

## Summary of New Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/accounts | Connect Threads account |
| GET | /api/accounts | List connected accounts |
| PUT | /api/accounts/:id | Update account settings |
| DELETE | /api/accounts/:id | Remove account |
| GET | /api/posts | List all posts |
| POST | /api/posts/generate | Trigger AI content generation |
| POST | /api/posts/:id/approve | Approve post for publishing |
| POST | /api/posts/:id/publish | Publish post immediately |
