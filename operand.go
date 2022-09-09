// Package operand exposes the Operand API.
package operand

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client implements the Operand API for a given API Key / endpoint.
type Client struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

// DefaultEndpoint is the default endpoint used for the Operand API.
const DefaultEndpoint = "https://prod.operand.ai"

// NewClient creates a new Client object.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:   apiKey,
		endpoint: DefaultEndpoint,
		client:   http.DefaultClient,
	}
}

// WithEndpoint attaches a non-default endpoint to the Client.
// This is generally used with dedicated, or non-serverless deployments.
func (c *Client) WithEndpoint(endpoint string) *Client {
	// Ensure that the endpoint doesn't end with a trailing slash.
	c.endpoint = strings.TrimSuffix(endpoint, "/")
	return c
}

// WithHTTPClient attaches a non-default HTTP client to the Client.
func (c *Client) WithHTTPClient(client *http.Client) *Client {
	c.client = client
	return c
}

func (c *Client) doRequest(ctx context.Context, method, path string, body, dst any) error {
	var reqBody io.Reader
	if body != nil {
		marshalled, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(marshalled)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reqBody)
	if err != nil {
		return err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", c.apiKey)
	}

	client := c.client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, buf)
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return err
		}
	}

	return nil
}

// ObjectType is an enumeration over the various supported object types.
type ObjectType string

// Supported object types.
const (
	ObjectTypeCollection       ObjectType = "collection"
	ObjectTypeText             ObjectType = "text"
	ObjectTypeHTML             ObjectType = "html"
	ObjectTypeMarkdown         ObjectType = "markdown"
	ObjectTypePDF              ObjectType = "pdf"
	ObjectTypeImage            ObjectType = "image"
	ObjectTypeGitHubRepository ObjectType = "github_repository"
	ObjectTypeEPUB             ObjectType = "epub"
	ObjectTypeAudio            ObjectType = "audio"
	ObjectTypeRSS              ObjectType = "rss"
	ObjectTypeNotion           ObjectType = "notion"
	ObjectTypeMbox             ObjectType = "mbox"
	ObjectEmail                ObjectType = "email"
	ObjectTypeNotionPage       ObjectType = "notion_page"
)

// Metadata defintitions for objects (dependent on type).
type (
	// CollectionMetadata is the metadata for a collection object.
	CollectionMetadata struct{}

	// TextMetadata is the metadata for a text object.
	TextMetadata struct {
		Text string `json:"text"`
	}

	// HTMLMetadata is the metadata for an HTML object.
	HTMLMetadata struct {
		HTML  string  `json:"html,omitempty"`
		Title *string `json:"title"`
		URL   *string `json:"url"`
	}

	// MarkdownMetadata is the metadata for a markdown object.
	MarkdownMetadata struct {
		Markdown string  `json:"markdown"`
		Title    *string `json:"title"`
	}

	// PDFMetadata is the metadata for a PDF object.
	PDFMetadata struct {
		URL string `json:"pdfUrl"`
	}

	// ImageMetadata is the metadata for an image object.
	ImageMetadata struct {
		URL string `json:"imageUrl"`
	}

	// GitHubRepositoryMetadata is the metadata for a GitHub repository object.
	GitHubRepositoryMetadata struct {
		AccessToken string  `json:"accessToken"`
		RepoOwner   string  `json:"repoOwner"`
		RepoName    string  `json:"repoName"`
		RootPath    *string `json:"rootPath"`
		RootURL     *string `json:"rootUrl"`
		Ref         *string `json:"ref"`
	}

	// EPUBMetadata is the metadata for an EPUB object.
	EPUBMetadata struct {
		URL      string  `json:"epubUrl"`
		Title    *string `json:"title"`
		Language *string `json:"language"`
	}

	// AudioMetadata is the metadata for an audio object.
	AudioMetadata struct {
		URL    string  `json:"audioUrl"`
		GCSUri *string `json:"gcsUri"`
	}
	// RSSMetadata is the metadata for an RSS object.
	RSSMetadata struct {
		URL string `json:"rssUrl"`
	}
	// NotionMetadata is the metadata for a Notion object.
	NotionMetadata struct {
		AccessToken string `json:"accessToken"`
	}
	// MBOXMetadata is the metadata for an MBOX object.
	MboxMetadata struct {
		URL string `json:"mboxUrl"`
	}
	// EmailMetadata is the metadata for an Email object.
	EmailMetadata struct {
		Email   string     `json:"email"`
		Sent    *time.Time `json:"sent"`
		From    *string    `json:"from"`
		Subject *string    `json:"subject"`
		To      []string   `json:"to"`
	}
	NotionPageMetadata struct {
		PageID string  `json:"pageId"`
		URL    string  `json:"url"`
		Title  *string `json:"title"`
	}
)

// IndexingStatus is an enumeration over the different states an object can be in.
type IndexingStatus string

// Supported indexing statuses.
const (
	IndexingStatusIndexing IndexingStatus = "indexing"
	IndexingStatusReady    IndexingStatus = "ready"
	IndexingStatusError    IndexingStatus = "error"
)

// Object is the fundamental type of the Operand API. Objects can
// be of many types, i.e. HTML, Images, PDF, etc. There is a special
// object type, named "collection", which is analogous to a folder.
type Object struct {
	ID             string          `json:"id"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	Type           ObjectType      `json:"type"`
	Metadata       json.RawMessage `json:"metadata"`
	Properties     map[string]any  `json:"properties"`
	IndexingStatus IndexingStatus  `json:"indexingStatus"`
	ParentID       *string         `json:"parentId"`
	Label          *string         `json:"label"`

	// Optionally included in a GetObject response if the atom count is requested.
	// Zero otherwise (in all other responses).
	Objects int `json:"objects"`
}

// UnmarshalMetadata unmarshals the metadata field of an object, depending
// on the object type. The return value of this function must be cast to the appropriate type.
func (o *Object) UnmarshalMetadata() (any, error) {
	var rval any
	switch o.Type {
	case ObjectTypeCollection:
		rval = new(CollectionMetadata)
	case ObjectTypeText:
		rval = new(TextMetadata)
	case ObjectTypeHTML:
		rval = new(HTMLMetadata)
	case ObjectTypeMarkdown:
		rval = new(MarkdownMetadata)
	case ObjectTypePDF:
		rval = new(PDFMetadata)
	case ObjectTypeImage:
		rval = new(ImageMetadata)
	case ObjectTypeGitHubRepository:
		rval = new(GitHubRepositoryMetadata)
	case ObjectTypeEPUB:
		rval = new(EPUBMetadata)
	case ObjectTypeAudio:
		rval = new(AudioMetadata)
	case ObjectTypeRSS:
		rval = new(RSSMetadata)
	case ObjectTypeNotion:
		rval = new(NotionMetadata)
	case ObjectTypeMbox:
		rval = new(MboxMetadata)
	case ObjectEmail:
		rval = new(EmailMetadata)
	case ObjectTypeNotionPage:
		rval = new(NotionPageMetadata)
	default:
		return nil, fmt.Errorf("unsupported object type: %s", o.Type)
	}

	if err := json.Unmarshal(o.Metadata, &rval); err != nil {
		return nil, err
	}

	return rval, nil
}

// Wait waits for an object to be indexed before returning. A context can
// (and should) be passed into this function with a timeout to ensure that
// this doesn't block indefinitely.
func (o *Object) Wait(ctx context.Context, client *Client) error {
	// If we're already ready, we're done and we don't need to wait for anything.
	if o.IndexingStatus != IndexingStatusIndexing {
		return nil
	}

	// If the context is already cancelled for some reason, return early.
	if err := ctx.Err(); err != nil {
		return err
	}

	// Keep track of the number of iterations.
	var iterations int

	// Periodically poll the object until it's ready.
	for o.IndexingStatus == IndexingStatusIndexing {
		// We sleep for progressively longer periods of time.
		var sleepDuration time.Duration
		if iterations == 0 {
			// If this is the first iteration, don't sleep.
		} else if iterations < 10 {
			// Sleep for a small amount of time for the first 10 iterations.
			sleepDuration = time.Millisecond * 300
		} else {
			// For remaining iterations, sleep for a longer amount of time.
			// This is likely a larger object.
			sleepDuration = time.Second
		}

		// Sleep for the duration.
		if sleepDuration > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleepDuration):
			}
		}

		// Re-fetch the object.
		obj, err := client.GetObject(ctx, o.ID, nil)
		if err != nil {
			return err
		}
		*o = *obj

		// Increment the number of iterations.
		iterations++
	}

	// At this point, the indexing status has been updated and we can return.
	return nil
}

// CreateObjectArgs contains the arguments for the CreateObject function.
type CreateObjectArgs struct {
	ParentID   *string        `json:"parentId,omitempty"`
	Type       ObjectType     `json:"type"`
	Metadata   any            `json:"metadata"`
	Properties map[string]any `json:"properties,omitempty"`
	Label      *string        `json:"label,omitempty"`
}

// CreateObject creates a new object in the Operand API.
func (c *Client) CreateObject(ctx context.Context, args CreateObjectArgs) (*Object, error) {
	obj := new(Object)
	if err := c.doRequest(ctx, "POST", "/v3/objects", args, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// ListObjectsArgs contains the arguments for the ListObjects function.
type ListObjectsArgs struct {
	ParentID      *string `json:"parentId,omitempty"`
	Limit         int     `json:"limit,omitempty"`
	EndingBefore  *string `json:"endingBefore,omitempty"`
	StartingAfter *string `json:"startingAfter,omitempty"`
}

// ListObjectsResponse contains the response from the ListObjects function.
type ListObjectsResponse struct {
	Objects []Object `json:"objects"`
	HasMore bool     `json:"hasMore"`
}

// ListObjects lists objects in the Operand API.
func (c *Client) ListObjects(
	ctx context.Context,
	args ListObjectsArgs,
) (*ListObjectsResponse, error) {
	resp := new(ListObjectsResponse)
	if err := c.doRequest(ctx, "GET", "/v3/objects", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetObjectExtraArgs contains the (optional) arguments for the GetObject function.
type GetObjectExtraArgs struct {
	// Optionally include the number of objects underneath this object in the response.
	Count bool
}

// GetObject returns a singular object from the Operand API.
func (c *Client) GetObject(
	ctx context.Context,
	id string,
	extra *GetObjectExtraArgs,
) (*Object, error) {
	obj := new(Object)

	params := url.Values{}
	if extra != nil && extra.Count {
		params.Set("count", "true")
	}

	endpoint := fmt.Sprintf("/v3/objects/%s", id)
	if encoded := params.Encode(); encoded != "" {
		endpoint = endpoint + "?" + encoded
	}

	if err := c.doRequest(ctx, "GET", endpoint, nil, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// UpdateObjectArgs contains the arguments for the UpdateObject function.
// This endpoint allows for partial updates, meaning only the fields you
// specify here will be updated. For example, if you don't specify a label,
// the existing label will be preserved.
type UpdateObjectArgs struct {
	Type       ObjectType     `json:"type"`
	Metadata   any            `json:"metadata"`
	Properties map[string]any `json:"properties,omitempty"`
	Label      *string        `json:"label,omitempty"`
}

// UpdateObject updates an existing object in the Operand API.
func (c *Client) UpdateObject(
	ctx context.Context,
	id string,
	args UpdateObjectArgs,
) (*Object, error) {
	obj := new(Object)
	if err := c.doRequest(ctx, "PUT", "/v3/objects/"+id, args, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// DeleteResponse is a generic response to delete operations, indicating success or failure.
type DeleteResponse struct {
	Deleted bool `json:"deleted"`
}

// DeleteObjectExtraArgs contains the (optional) arguments for the DeleteObject function.
type DeleteObjectExtraArgs struct {
	// Nothing here yet, but we might want to add additional arguments in the future.
}

// DeleteObject deletes an object from the Operand API.
func (c *Client) DeleteObject(
	ctx context.Context,
	id string,
	extra *DeleteObjectExtraArgs,
) (*DeleteResponse, error) {
	resp := new(DeleteResponse)
	if err := c.doRequest(ctx, "DELETE", "/v3/objects/"+id, nil, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ContentType is an enumeration over the various content types.
type ContentType string

// Supported content types.
const (
	ContentTypeTitle    ContentType = "title"
	ContentTypeContent  ContentType = "content"
	ContentTypeLink     ContentType = "link"
	ContentTypeImage    ContentType = "image"
	ContentTypeCode     ContentType = "code"
	ContentTypeListItem ContentType = "list_item"
)

// Content is an individual piece of content.
type Content struct {
	ObjectID string      `json:"objectId"`
	Content  string      `json:"content"`
	Type     ContentType `json:"type"`
	Score    float32     `json:"score"`
}

// SearchContentsArgs contains the arguments for the SearchContents function.
type SearchContentsArgs struct {
	ParentIDs []string       `json:"parentIds,omitempty"` // Can be omitted, in which case all objects are searched.
	Query     string         `json:"query"`               // Must not be empty.
	Max       int            `json:"max,omitempty"`
	Filter    map[string]any `json:"filter,omitempty"`
}

// SearchContentsResponse contains the response from the SearchContents function.
type SearchContentsResponse struct {
	ID        string            `json:"id"`
	LatencyMS int64             `json:"latencyMs"`
	Contents  []Content         `json:"contents"`
	Objects   map[string]Object `json:"objects"`
}

// SearchContents searches for content in the Operand API.
func (c *Client) SearchContents(
	ctx context.Context,
	args SearchContentsArgs,
) (*SearchContentsResponse, error) {
	resp := new(SearchContentsResponse)
	if err := c.doRequest(ctx, "POST", "/v3/search/contents", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SearchObjectsArgs contains the arguments for the SearchObjects function.
type SearchObjectsArgs struct {
	ParentIDs []string       `json:"parentIds,omitempty"` // Can be omitted, in which case all objects are searched.
	Query     string         `json:"query"`               // Must not be empty.
	Max       int            `json:"max,omitempty"`
	Filter    map[string]any `json:"filter,omitempty"`
}

// SnippetObject is a snippet and an object.
type SnippetObject struct {
	Snippet string `json:"snippet"`
	Object  Object `json:"object"`
}

// SearchObjectsResponse contains the response from the SearchObjects function.
type SearchObjectsResponse struct {
	ID        string          `json:"id"`
	LatencyMS int64           `json:"latencyMs"`
	Results   []SnippetObject `json:"results"`
}

// SearchObjects searches for objects in the Operand API.
func (c *Client) SearchObjects(
	ctx context.Context,
	args SearchObjectsArgs,
) (*SearchObjectsResponse, error) {
	resp := new(SearchObjectsResponse)
	if err := c.doRequest(ctx, "POST", "/v3/search/objects", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SearchRelatedArgs contains the arguments for the SearchRelated function.
type SearchRelatedArgs struct {
	ParentIDs []string       `json:"parentIds,omitempty"` // Can be omitted, in which case all objects are searched.
	ObjectID  string         `json:"objectId"`            // Required.
	Max       int            `json:"max,omitempty"`
	Filter    map[string]any `json:"filter,omitempty"`
}

// SearchRelatedResponse contains the response from the SearchRelated function.
type SearchRelatedResponse struct {
	ID        string   `json:"id"`
	LatencyMS int64    `json:"latencyMs"`
	Objects   []Object `json:"objects"`
}

// SearchRelated searches for related objects in the Operand API.
func (c *Client) SearchRelated(
	ctx context.Context,
	args SearchRelatedArgs,
) (*SearchRelatedResponse, error) {
	resp := new(SearchRelatedResponse)
	if err := c.doRequest(ctx, "POST", "/v3/search/related", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// AnswerStyle is an enumeration over the various answer styles.
type AnswerStyle string

// Supported answer styles.
const (
	AnswerStyleDirect  ContentType = "direct"
	AnswerStyleOperand ContentType = "operand"
)

// CompletionAnswerArgs contains the arguments for the CompletionAnswer function.
type CompletionAnswerArgs struct {
	ParentIDs []string       `json:"parentIds,omitempty"` // Can be omitted, in which case all objects are searched.
	Question  string         `json:"question"`            // Must not be empty.
	Style     AnswerStyle    `json:"style,omitempty"`
	Filter    map[string]any `json:"filter,omitempty"`
}

// CompletionAnswerResponse contains the response from the CompletionAnswer function.
type CompletionAnswerResponse struct {
	ID        string   `json:"id"`
	LatencyMS int64    `json:"latencyMs"`
	Answer    string   `json:"answer"`
	Sources   []Object `json:"sources"`
}

// CompletionAnswer searches for answers in the Operand API.
func (c *Client) CompletionAnswer(
	ctx context.Context,
	args CompletionAnswerArgs,
) (*CompletionAnswerResponse, error) {
	resp := new(CompletionAnswerResponse)
	if err := c.doRequest(ctx, "POST", "/v3/completion/answer", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CompletionTypeAheadArgs contains the arguments for the CompletionTypeAhead function.
type CompletionTypeAheadArgs struct {
	ParentIDs []string       `json:"parentIds,omitempty"` // Can be omitted, in which case all objects are searched.
	Text      string         `json:"text"`                // Must not be empty.
	Count     int            `json:"count,omitempty"`     // The number of generations to perform. Defaults to 3.
	Filter    map[string]any `json:"filter,omitempty"`
}

// CompletionTypeAheadResponse contains the response from the CompletionTypeAhead function.
type CompletionTypeAheadResponse struct {
	ID          string   `json:"id"`
	LatencyMS   int64    `json:"latencyMs"`
	Completions []string `json:"completions"`
	Sources     []Object `json:"sources"`
}

// CompletionTypeAhead completes a text string using data from the Operand API.
func (c *Client) CompletionTypeAhead(
	ctx context.Context,
	args CompletionTypeAheadArgs,
) (*CompletionTypeAheadResponse, error) {
	resp := new(CompletionTypeAheadResponse)
	if err := c.doRequest(ctx, "POST", "/v3/completion/typeahead", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Trigger

type CallbackKind string

const (
	CallbackKindWebhook CallbackKind = "webhook"
)

type (
	WebhookCallbackMetadata struct {
		URL string `json:"url"`
	}
)

type Trigger struct {
	ID                string          `json:"id"`
	CreatedAt         time.Time       `json:"createdAt"`
	Query             string          `json:"query"`
	Filter            map[string]any  `json:"filter,omitempty"`
	MatchingThreshold float32         `json:"matchingThreshold,omitempty"`
	CallbackKind      CallbackKind    `json:"callbackKind"`
	CallbackMetadata  json.RawMessage `json:"callbackMetadata"`

	// Optional fields.
	LastFired *time.Time `json:"lastFired,omitempty"`
}

func (t *Trigger) UnmarshalMetadata() (any, error) {
	var rval any
	switch t.CallbackKind {
	case CallbackKindWebhook:
		rval = new(WebhookCallbackMetadata)
	default:
		return nil, fmt.Errorf("unknown callback kind: %s", t.CallbackKind)
	}

	if err := json.Unmarshal(t.CallbackMetadata, rval); err != nil {
		return nil, err
	}

	return rval, nil
}

type CreateTriggerArgs struct {
	Query             string         `json:"query"`
	Filter            map[string]any `json:"filter,omitempty"`
	MatchingThreshold *float32       `json:"matchingThreshold,omitempty"`
	CallbackKind      CallbackKind   `json:"callbackKind"`
	CallbackMetadata  any            `json:"callbackMetadata"`
}

func (c *Client) CreateTrigger(
	ctx context.Context,
	args CreateTriggerArgs,
) (*Trigger, error) {
	trig := new(Trigger)
	if err := c.doRequest(ctx, "POST", "/v3/triggers", args, trig); err != nil {
		return nil, err
	}
	return trig, nil
}

type ListTriggersArgs struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type ListTriggersResponse struct {
	Triggers []Trigger `json:"triggers"`
	HasMore  bool      `json:"hasMore"`
}

func (c *Client) ListTriggers(
	ctx context.Context,
	args ListTriggersArgs,
) (*ListTriggersResponse, error) {
	resp := new(ListTriggersResponse)
	if err := c.doRequest(ctx, "GET", "/v3/triggers", args, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type GetTriggerArgs struct {
	ID string `json:"id"`
}

func (c *Client) GetTrigger(
	ctx context.Context,
	args GetTriggerArgs,
) (*Trigger, error) {
	trig := new(Trigger)
	if err := c.doRequest(ctx, "GET", "/v3/triggers/"+args.ID, nil, trig); err != nil {
		return nil, err
	}
	return trig, nil
}

type DeleteTriggerArgs struct {
	ID string `json:"id"`
}

type DeleteTriggerResponse struct {
	Deleted bool `json:"deleted"`
}

func (c *Client) DeleteTrigger(
	ctx context.Context,
	args DeleteTriggerArgs,
) (*DeleteTriggerResponse, error) {
	resp := new(DeleteTriggerResponse)
	if err := c.doRequest(ctx, "DELETE", "/v3/triggers/"+args.ID, nil, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// FeedbackArgs contains the arguments for the Feedback function.
type FeedbackArgs struct {
	SearchID string `json:"searchId"`
	ObjectID string `json:"objectId"`
}

// Feedback sends feedback to the Operand API.
// This should be used when a user clicks on a result after an object-based search.
func (c *Client) Feedback(ctx context.Context, args FeedbackArgs) error {
	return c.doRequest(ctx, "POST", "/v3/feedback", args, nil)
}

/* Utility Functions */

// AsRef returns a reference to the value passed in.
func AsRef[T any](v T) *T {
	return &v
}
