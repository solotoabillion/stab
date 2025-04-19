package types

import (
	"encoding/json"

	"stab/models"
)

func MapKBModelToType(a *models.KnowledgeBaseArticle) KnowledgeBaseArticle {
	var tags []string
	_ = json.Unmarshal(a.Tags, &tags)
	return KnowledgeBaseArticle{
		ID:        a.ID.String(),
		Title:     a.Title,
		Body:      a.Body,
		Tags:      tags,
		Source:    a.Source,
		IsActive:  a.IsActive,
		CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
