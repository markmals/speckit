package github

import (
	"context"
	"fmt"
	"strings"
)

// Project is a GitHub Projects v2 board plus its fields (node IDs cached on the
// struct — the API takes IDs, not names, so resolve once and reuse).
type Project struct {
	ID     string
	Title  string
	Number int
	Fields []Field
}

// Field is a project field; Options is populated for single-select fields (e.g.
// the Status column set).
type Field struct {
	ID      string
	Name    string
	Options []FieldOption
}

// FieldOption is one choice of a single-select field (e.g. a Status column).
type FieldOption struct {
	ID   string
	Name string
}

// Field returns the named field (case-insensitive).
func (p Project) Field(name string) (Field, bool) {
	for _, f := range p.Fields {
		if strings.EqualFold(f.Name, name) {
			return f, true
		}
	}
	return Field{}, false
}

// Option returns the named option of a single-select field (case-insensitive).
func (f Field) Option(name string) (FieldOption, bool) {
	for _, o := range f.Options {
		if strings.EqualFold(o.Name, name) {
			return o, true
		}
	}
	return FieldOption{}, false
}

// ResolveProject loads a project and its fields by owner (user OR org) + number.
// One query resolves every node ID the board loop needs. The repositoryOwner +
// ProjectV2Owner inline fragment handles user- and org-owned projects alike, so no
// config block is needed to know the owner type.
func (c *Client) ResolveProject(ctx context.Context, owner string, number int) (Project, error) {
	const q = `
query($owner:String!, $number:Int!) {
  repositoryOwner(login: $owner) {
    ... on ProjectV2Owner {
      projectV2(number: $number) {
        id
        title
        number
        fields(first: 50) {
          nodes {
            ... on ProjectV2FieldCommon { id name }
            ... on ProjectV2SingleSelectField { id name options { id name } }
          }
        }
      }
    }
  }
}`
	var resp struct {
		RepositoryOwner struct {
			ProjectV2 *struct {
				ID     string `json:"id"`
				Title  string `json:"title"`
				Number int    `json:"number"`
				Fields struct {
					Nodes []struct {
						ID      string `json:"id"`
						Name    string `json:"name"`
						Options []struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"options"`
					} `json:"nodes"`
				} `json:"fields"`
			} `json:"projectV2"`
		} `json:"repositoryOwner"`
	}
	if err := c.GraphQL(ctx, q, map[string]any{"owner": owner, "number": number}, &resp); err != nil {
		return Project{}, err
	}
	pv := resp.RepositoryOwner.ProjectV2
	if pv == nil {
		return Project{}, fmt.Errorf("no project #%d for %q (check the number and that the token has read:project)", number, owner)
	}
	p := Project{ID: pv.ID, Title: pv.Title, Number: pv.Number}
	for _, n := range pv.Fields.Nodes {
		if n.ID == "" {
			continue
		}
		f := Field{ID: n.ID, Name: n.Name}
		for _, o := range n.Options {
			f.Options = append(f.Options, FieldOption{ID: o.ID, Name: o.Name})
		}
		p.Fields = append(p.Fields, f)
	}
	return p, nil
}

// Item is a project board item: its project-item id, the linked issue/PR, and its
// current value in the status (column) field.
type Item struct {
	ItemID string
	Number int
	Title  string
	URL    string
	State  string
	Status string
}

// ListItems returns all of a project's items with their value in the named status
// field (the kanban column). statusField defaults to "Status". Pages through the
// whole board. The "ready queue" is just the items whose Status is the actionable
// column — a filter the caller applies, not a computed field.
func (c *Client) ListItems(ctx context.Context, projectID, statusField string) ([]Item, error) {
	if statusField == "" {
		statusField = "Status"
	}
	const q = `
query($project:ID!, $statusField:String!, $after:String) {
  node(id: $project) {
    ... on ProjectV2 {
      items(first: 50, after: $after) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          status: fieldValueByName(name: $statusField) {
            ... on ProjectV2ItemFieldSingleSelectValue { name }
          }
          content {
            __typename
            ... on Issue { number title url state }
            ... on PullRequest { number title url state }
          }
        }
      }
    }
  }
}`
	var items []Item
	after := ""
	first := true
	for {
		vars := map[string]any{"project": projectID, "statusField": statusField}
		if after != "" {
			vars["after"] = after
		}
		var resp struct {
			Node *struct {
				Items struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						ID     string `json:"id"`
						Status *struct {
							Name string `json:"name"`
						} `json:"status"`
						Content *struct {
							TypeName string `json:"__typename"`
							Number   int    `json:"number"`
							Title    string `json:"title"`
							URL      string `json:"url"`
							State    string `json:"state"`
						} `json:"content"`
					} `json:"nodes"`
				} `json:"items"`
			} `json:"node"`
		}
		if err := c.GraphQL(ctx, q, vars, &resp); err != nil {
			return nil, err
		}
		// node:null is a valid 200 (the project no longer resolves — deleted, made
		// private, or read:project lost since ResolveProject). Surface it on the
		// first page rather than silently returning an empty board.
		if resp.Node == nil {
			if first {
				return nil, fmt.Errorf("project %q no longer resolves (deleted, made private, or token lost read:project)", projectID)
			}
			break
		}
		first = false
		for _, n := range resp.Node.Items.Nodes {
			// Skip draft items and redacted/no-access content (otherwise they decode
			// to a phantom issue #0).
			if n.Content == nil || (n.Content.TypeName != "Issue" && n.Content.TypeName != "PullRequest") {
				continue
			}
			it := Item{ItemID: n.ID, Number: n.Content.Number, Title: n.Content.Title, URL: n.Content.URL, State: n.Content.State}
			if n.Status != nil {
				it.Status = n.Status.Name
			}
			items = append(items, it)
		}
		if !resp.Node.Items.PageInfo.HasNextPage {
			break
		}
		after = resp.Node.Items.PageInfo.EndCursor
	}
	return items, nil
}

// AddItem adds an issue/PR (by its content node ID) to a project, returning the
// project item ID. Idempotent: re-adding an existing item returns it.
func (c *Client) AddItem(ctx context.Context, projectID, contentNodeID string) (string, error) {
	const m = `
mutation($project:ID!, $content:ID!) {
  addProjectV2ItemById(input: {projectId: $project, contentId: $content}) {
    item { id }
  }
}`
	var resp struct {
		Add struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		} `json:"addProjectV2ItemById"`
	}
	if err := c.GraphQL(ctx, m, map[string]any{"project": projectID, "content": contentNodeID}, &resp); err != nil {
		return "", err
	}
	return resp.Add.Item.ID, nil
}

// SetSingleSelect sets a single-select field (e.g. Status) on a project item to an
// option. Field writes are last-write-wins. This is the move-a-card primitive.
func (c *Client) SetSingleSelect(ctx context.Context, projectID, itemID, fieldID, optionID string) error {
	const m = `
mutation($project:ID!, $item:ID!, $field:ID!, $option:String!) {
  updateProjectV2ItemFieldValue(input: {
    projectId: $project, itemId: $item, fieldId: $field,
    value: { singleSelectOptionId: $option }
  }) {
    projectV2Item { id }
  }
}`
	return c.GraphQL(ctx, m, map[string]any{
		"project": projectID, "item": itemID, "field": fieldID, "option": optionID,
	}, nil)
}
