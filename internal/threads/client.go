package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const baseURL = "https://graph.threads.net/v1.0"

// ThreadsClient calls the Meta Threads API.
type ThreadsClient struct {
	accessToken string
}

// ThreadInfo contains basic information returned for a Threads post.
type ThreadInfo struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	Permalink string `json:"permalink"`
}

// MetaAPIError contains error details returned by the Meta API.
type MetaAPIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type metaErrorResponse struct {
	Error MetaAPIError `json:"error"`
}

type createContainerResponse struct {
	ID string `json:"id"`
}

type publishContainerResponse struct {
	ID string `json:"id"`
}

// NewClient creates a Threads API client using the provided access token.
func NewClient(accessToken string) *ThreadsClient {
	return &ThreadsClient{accessToken: accessToken}
}

// CreateContainer creates a text media container for the given Threads user.
func (c *ThreadsClient) CreateContainer(ctx context.Context, userID string, text string) (containerID string, err error) {
	form := url.Values{}
	form.Set("media_type", "TEXT")
	form.Set("text", text)
	form.Set("access_token", c.accessToken)

	var response createContainerResponse
	if err := c.postForm(ctx, fmt.Sprintf("%s/%s/threads", baseURL, url.PathEscape(userID)), form, &response); err != nil {
		return "", err
	}

	return response.ID, nil
}

// PublishContainer publishes a previously created Threads media container.
func (c *ThreadsClient) PublishContainer(ctx context.Context, userID string, containerID string) (threadID string, err error) {
	form := url.Values{}
	form.Set("creation_id", containerID)
	form.Set("access_token", c.accessToken)

	var response publishContainerResponse
	if err := c.postForm(ctx, fmt.Sprintf("%s/%s/threads_publish", baseURL, url.PathEscape(userID)), form, &response); err != nil {
		return "", err
	}

	return response.ID, nil
}

// GetThread returns basic information for a Threads post.
func (c *ThreadsClient) GetThread(ctx context.Context, threadID string) (threadInfo *ThreadInfo, err error) {
	query := url.Values{}
	query.Set("fields", "id,text,timestamp,permalink")
	query.Set("access_token", c.accessToken)

	requestURL := fmt.Sprintf("%s/%s?%s", baseURL, url.PathEscape(threadID), query.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get thread request: %w", err)
	}

	var thread ThreadInfo
	if err := c.do(req, &thread); err != nil {
		return nil, err
	}

	return &thread, nil
}

// ReplyToThread creates and publishes a text reply to an existing Threads post.
func (c *ThreadsClient) ReplyToThread(ctx context.Context, userID string, replyToID string, text string) (threadID string, err error) {
	form := url.Values{}
	form.Set("media_type", "TEXT")
	form.Set("text", text)
	form.Set("reply_to_id", replyToID)
	form.Set("access_token", c.accessToken)

	var response publishContainerResponse
	if err := c.postForm(ctx, fmt.Sprintf("%s/%s/threads", baseURL, url.PathEscape(userID)), form, &response); err != nil {
		return "", err
	}

	return response.ID, nil
}

// Reply represents a single reply to a Threads post.
type Reply struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Username string `json:"username"`
}

type repliesResponse struct {
	Data []Reply `json:"data"`
}

// GetReplies fetches replies for a given thread post.
func (c *ThreadsClient) GetReplies(ctx context.Context, threadID string) ([]Reply, error) {
	query := url.Values{}
	query.Set("fields", "id,text,username")
	query.Set("access_token", c.accessToken)

	requestURL := fmt.Sprintf("%s/%s/replies?%s", baseURL, url.PathEscape(threadID), query.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get replies request: %w", err)
	}

	var response repliesResponse
	if err := c.do(req, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (c *ThreadsClient) postForm(ctx context.Context, requestURL string, form url.Values, target interface{}) error {
	body := form.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create threads request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = int64(len(body))

	return c.do(req, target)
}

func (c *ThreadsClient) do(req *http.Request, target interface{}) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("threads API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read threads API response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return parseMetaError(resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("failed to parse threads API response: %w", err)
	}

	return nil
}

func parseMetaError(statusCode int, body []byte) error {
	var response metaErrorResponse
	if err := json.Unmarshal(body, &response); err == nil && response.Error.Message != "" {
		return fmt.Errorf("threads API error: status=%d code=%d message=%s", statusCode, response.Error.Code, response.Error.Message)
	}

	return fmt.Errorf("threads API error: status=%d body=%s", statusCode, string(body))
}
