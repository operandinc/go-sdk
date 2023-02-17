// Package operand is the SDK for the Operand API.
package operand

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/bufbuild/connect-go"
	filev1 "github.com/operandinc/go-sdk/file/v1"
	"github.com/operandinc/go-sdk/file/v1/filev1connect"
	"github.com/operandinc/go-sdk/operand/v1/operandv1connect"
	"github.com/operandinc/go-sdk/tenant/v1/tenantv1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

// Client is the client for the Operand API.
type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
}

// NewClient creates a new client for the Operand API.
func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: http.DefaultClient,
		endpoint:   "https://mcp.operand.ai",
		apiKey:     apiKey,
	}
}

// WithEndpoint sets the endpoint for the client.
func (c *Client) WithEndpoint(endpoint string) *Client {
	c.endpoint = endpoint
	return c
}

// WithHTTPClient sets the HTTP client for the client.
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	c.httpClient = httpClient
	return c
}

// FileService returns a client for the Operand File Service.
func (c *Client) FileService() filev1connect.FileServiceClient {
	return filev1connect.NewFileServiceClient(c.httpClient, c.endpoint, c.clientOpts()...)
}

// TenantService returns a client for the Operand Tenant Service.
func (c *Client) TenantService() tenantv1connect.TenantServiceClient {
	return tenantv1connect.NewTenantServiceClient(c.httpClient, c.endpoint, c.clientOpts()...)
}

// OperandService returns a client for the Operand Operand Service.
func (c *Client) OperandService() operandv1connect.OperandServiceClient {
	return operandv1connect.NewOperandServiceClient(c.httpClient, c.endpoint, c.clientOpts()...)
}

// CreateFile is a utility method for creating files. Since this is a common operation
// and is a little more involved, we provide a helper method for it.
func (c *Client) CreateFile(
	ctx context.Context,
	name string,
	parent *string,
	data io.Reader, // Nullable, if nil, we'll create a folder (i.e. a file with no data).
	properties *filev1.Properties,
) (*filev1.CreateFileResponse, error) {
	var buf bytes.Buffer

	mw := multipart.NewWriter(&buf)
	mw.WriteField("name", name)
	if parent != nil {
		mw.WriteField("parent_id", *parent)
	}
	if properties != nil {
		marshaled, err := protojson.Marshal(properties)
		if err != nil {
			return nil, err
		}
		mw.WriteField("properties", string(marshaled))
	}
	if data != nil {
		part, err := mw.CreateFormFile("file", name)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(part, data)
		if err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/upload", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Key "+c.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	createFileResponse := &filev1.CreateFileResponse{}
	if err := protojson.Unmarshal(body, createFileResponse); err != nil {
		return nil, err
	}

	return createFileResponse, nil
}

func (c *Client) clientOpts() []connect.ClientOption {
	return []connect.ClientOption{
		connect.WithInterceptors(&headerInterceptor{apiKey: c.apiKey}),
	}
}

type headerInterceptor struct {
	apiKey string
}

var _ connect.Interceptor = (*headerInterceptor)(nil)

func (hi *headerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, ar connect.AnyRequest) (connect.AnyResponse, error) {
		if ar.Spec().IsClient {
			ar.Header().Set("Authorization", "Key "+hi.apiKey)
		}
		return next(ctx, ar)
	}
}

func (hi *headerInterceptor) WrapStreamingClient(
	next connect.StreamingClientFunc,
) connect.StreamingClientFunc {
	return func(ctx context.Context, s connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, s)
		if s.IsClient {
			conn.RequestHeader().Set("Authorization", "Key "+hi.apiKey)
		}
		return conn
	}
}

func (hi *headerInterceptor) WrapStreamingHandler(
	next connect.StreamingHandlerFunc,
) connect.StreamingHandlerFunc {
	return next // No-op (client-only interceptor).
}
