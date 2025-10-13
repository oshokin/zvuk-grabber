package zvuk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// fetchJSON fetches JSON from the specified URI.
//
//nolint:revive // Has no sense, it's cause Go doesn't allow struct methods to be generic.
func fetchJSON[T any](c *ClientImpl, ctx context.Context, uri string) (*FetchJSONResult[T], error) {
	return fetchJSONWithQuery[T](c, ctx, uri, nil)
}

// fetchJSONWithQuery fetches JSON from the specified URI with the specified query.
//
//nolint:revive // Has no sense, it's cause Go doesn't allow struct methods to be generic.
func fetchJSONWithQuery[T any](
	c *ClientImpl,
	ctx context.Context,
	uri string,
	query url.Values,
) (*FetchJSONResult[T], error) {
	route, err := url.JoinPath(c.baseURL, uri)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, route, http.NoBody)
	if err != nil {
		return nil, err
	}

	if query != nil {
		request.URL.RawQuery = query.Encode()
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return &FetchJSONResult[T]{
			Data:       nil,
			StatusCode: response.StatusCode,
		}, fmt.Errorf("%w: %d", ErrUnexpectedHTTPStatus, response.StatusCode)
	}

	var result T
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return &FetchJSONResult[T]{
			Data:       nil,
			StatusCode: response.StatusCode,
		}, err
	}

	return &FetchJSONResult[T]{
		Data:       &result,
		StatusCode: response.StatusCode,
	}, nil
}
