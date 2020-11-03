package handler

import (
	"context"
	"time"

	"github.com/micro/micro/v3/service/logger"
	"github.com/micro/micro/v3/service/store"

	"github.com/micro/dev/model"

	proto "posts/proto"

	"github.com/gosimple/slug"
)

type Posts struct {
	db model.Model
}

func NewPosts() *Posts {
	createdIndex := model.ByEquality("created")
	createdIndex.Order.Type = model.OrderTypeDesc

	return &Posts{
		db: model.New(
			store.DefaultStore,
			"posts",
			model.Indexes(model.ByEquality("slug"), createdIndex),
			&model.ModelOptions{
				Debug: false,
			},
		),
	}
}

func (p *Posts) Save(ctx context.Context, req *proto.SaveRequest, rsp *proto.SaveResponse) error {
	logger.Info("Received Posts.Save request")
	post := &proto.Post{
		Id:      req.Id,
		Title:   req.Title,
		Content: req.Content,
		Slug:    req.Slug,
		Created: time.Now().Unix(),
	}
	if req.Slug == "" {
		req.Slug = slug.Make(req.Title)
	}
	return p.db.Save(post)
}
