package handler

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/ai"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/model"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/queue"
	"github.com/alifiandzaki131103-rgb/threads-affiliate-automation/internal/repository"
)

type PostHandler struct {
	pool        *pgxpool.Pool
	rdb         *redis.Client // kept for worker/API dependency symmetry
	aiClient    *ai.Client    // kept for future synchronous generation paths
	queueClient *queue.Client
}

func NewPostHandler(pool *pgxpool.Pool, rdb *redis.Client, aiClient *ai.Client, queueClient *queue.Client) *PostHandler {
	return &PostHandler{pool: pool, rdb: rdb, aiClient: aiClient, queueClient: queueClient}
}

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

	if _, err := repository.GetAccountByIDForUser(c.Context(), h.pool, accountID, userID); err != nil {
		if repository.IsNoRows(err) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "account not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch account"})
	}

	links, err := repository.GetLinksWithProductByUserID(c.Context(), h.pool, userID)
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

func (h *PostHandler) ApprovePost(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	postID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	if err := repository.UpdatePostStatus(c.Context(), h.pool, postID, userID, "approved"); err != nil {
		if repository.IsNoRows(err) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to approve post"})
	}

	return c.JSON(fiber.Map{"message": "post approved", "post_id": postID})
}

func (h *PostHandler) CreateABTest(c *fiber.Ctx) error {
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

	// Verify account belongs to user
	if _, err := repository.GetAccountByIDForUser(c.Context(), h.pool, accountID, userID); err != nil {
		if repository.IsNoRows(err) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "account not found"})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch account"})
	}

	// Find the link and product info
	links, err := repository.GetLinksWithProductByUserID(c.Context(), h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch links"})
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

	// Generate variant A with "honest_friend" persona
	resultA, err := h.aiClient.Generate(c.Context(), &ai.GenerateRequest{
		ProductName:   productName,
		Price:         price,
		Category:      category,
		Platform:      platform,
		Persona:       "honest_friend",
		Format:        "single",
		LinkPlacement: "direct",
		ShortURL:      shortURL,
	})
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate variant A"})
	}

	// Generate variant B with "hot_take" persona
	resultB, err := h.aiClient.Generate(c.Context(), &ai.GenerateRequest{
		ProductName:   productName,
		Price:         price,
		Category:      category,
		Platform:      platform,
		Persona:       "hot_take",
		Format:        "single",
		LinkPlacement: "direct",
		ShortURL:      shortURL,
	})
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate variant B"})
	}

	// Create post A
	scheduledAt := time.Now().Add(1 * time.Hour)
	postAID, err := repository.CreatePostReturningID(c.Context(), h.pool, accountID, linkID,
		resultA.Content, resultA.LinkPlacement, resultA.Persona, resultA.Format, "draft", scheduledAt)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create variant A post"})
	}

	// Create post B
	postBID, err := repository.CreatePostReturningID(c.Context(), h.pool, accountID, linkID,
		resultB.Content, resultB.LinkPlacement, resultB.Persona, resultB.Format, "draft", scheduledAt)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create variant B post"})
	}

	// Create A/B test record
	abTest := &model.ABTest{
		LinkID:         linkID,
		VariantAPostID: &postAID,
		VariantBPostID: &postBID,
		Status:         "running",
	}
	if err := repository.CreateABTest(c.Context(), h.pool, abTest); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create A/B test"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"ab_test":   abTest,
		"variant_a": fiber.Map{"post_id": postAID, "content": resultA.Content, "persona": resultA.Persona},
		"variant_b": fiber.Map{"post_id": postBID, "content": resultB.Content, "persona": resultB.Persona},
	})
}

func (h *PostHandler) ListABTests(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	tests, err := repository.GetABTestsByUserID(c.Context(), h.pool, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch A/B tests"})
	}

	return c.JSON(fiber.Map{"count": len(tests), "ab_tests": tests})
}

func (h *PostHandler) PublishNow(c *fiber.Ctx) error {
	userID := GetUserID(c)
	if userID == uuid.Nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	postID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid post id"})
	}

	post, err := repository.GetPostByIDForUser(c.Context(), h.pool, postID, userID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "post not found"})
	}

	account, err := repository.GetAccountByID(c.Context(), h.pool, post.AccountID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "account not found"})
	}

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
