package forgejo

import (
	"context"
	"strconv"
)

type WorkLabel struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (c *Client) WorkLabels(ctx context.Context, token string, repo Repository, page int) ([]WorkLabel, error) {
	path, err := workPath(repo)
	if err != nil || page < 1 {
		return nil, ErrInvalidResponse
	}
	var labels []WorkLabel
	err = c.request(ctx, "GET", path+"/labels?limit=50&page="+strconv.Itoa(page), token, nil, &labels)
	return labels, err
}

func (c *Client) CreateWorkLabel(ctx context.Context, token string, repo Repository, name, color string) (WorkLabel, error) {
	path, err := workPath(repo)
	if err != nil {
		return WorkLabel{}, err
	}
	var label WorkLabel
	err = c.request(ctx, "POST", path+"/labels", token, map[string]string{"name": name, "color": color, "description": "Soda execution outcome; not execution authority"}, &label)
	if err == nil && (label.ID <= 0 || label.Name != name) {
		err = ErrInvalidResponse
	}
	return label, err
}

// AddWorkLabel preserves human-owned labels on the issue or PR.
func (c *Client) AddWorkLabel(ctx context.Context, token string, repo Repository, number, label int64) error {
	path, err := workPath(repo)
	if err != nil || number <= 0 || label <= 0 {
		return ErrInvalidResponse
	}
	return c.request(ctx, "POST", path+"/issues/"+strconv.FormatInt(number, 10)+"/labels", token, map[string]any{"labels": []int64{label}}, nil)
}

func (c *Client) RemoveWorkLabel(ctx context.Context, token string, repo Repository, number, label int64) error {
	path, err := workPath(repo)
	if err != nil || number <= 0 || label <= 0 {
		return ErrInvalidResponse
	}
	return c.request(ctx, "DELETE", path+"/issues/"+strconv.FormatInt(number, 10)+"/labels/"+strconv.FormatInt(label, 10), token, nil, nil)
}

func (c *Client) WorkIssueLabels(ctx context.Context, token string, repo Repository, number int64) ([]WorkLabel, error) {
	path, err := workPath(repo)
	if err != nil || number <= 0 {
		return nil, ErrInvalidResponse
	}
	var labels []WorkLabel
	err = c.request(ctx, "GET", path+"/issues/"+strconv.FormatInt(number, 10)+"/labels", token, nil, &labels)
	return labels, err
}
