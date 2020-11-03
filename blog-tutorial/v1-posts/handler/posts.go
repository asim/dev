package handler

import (
	"context"
	"time"

	"github.com/micro/dev/model"
	"github.com/micro/go-micro/errors"
	"github.com/micro/micro/v3/service/logger"
	"github.com/micro/micro/v3/service/store"

	proto "posts/proto"

	"github.com/gosimple/slug"
)

type Posts struct {
	db           model.Model
	createdIndex model.Index
	slugIndex    model.Index
}

func NewPosts() *Posts {
	createdIndex := model.ByEquality("created")
	createdIndex.Order.Type = model.OrderTypeDesc

	slugIndex := model.ByEquality("slug")

	return &Posts{
		db: model.New(
			store.DefaultStore,
			"posts",
			model.Indexes(slugIndex, createdIndex),
			nil,
		),
		createdIndex: createdIndex,
		slugIndex:    slugIndex,
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

func (p *Posts) Query(ctx context.Context, req *proto.QueryRequest, rsp *proto.QueryResponse) error {
	var q model.Query
	if len(req.Slug) > 0 {
		logger.Infof("Reading post by slug: %v", req.Slug)
		q = model.Equals("slug", req.Slug)
	} else if len(req.Id) > 0 {
		logger.Infof("Reading post by id: %v", req.Id)
		q = model.Equals("id", req.Id)
		q.Order.Type = model.OrderTypeUnordered
	} else {
		q = model.Equals("created", nil)
		q.Order.Type = model.OrderTypeDesc
		var limit uint
		limit = 20
		if req.Limit > 0 {
			limit = uint(req.Limit)
		}
		q.Limit = int64(limit)
		q.Offset = req.Offset
		logger.Infof("Listing posts, offset: %v, limit: %v", req.Offset, limit)
	}

	posts := []*proto.Post{}
	err := p.db.List(q, &posts)
	if err != nil {
		return errors.BadRequest("proto.query.store-read", "Failed to read from store: %v", err.Error())
	}
	rsp.Posts = make([]*proto.Post, len(posts))
	for i, post := range posts {
		rsp.Posts[i] = &proto.Post{
			Id:      post.Id,
			Title:   post.Title,
			Slug:    post.Slug,
			Content: post.Content,
			Tags:    post.Tags,
		}
	}
	return nil
}

func (p *Posts) Delete(ctx context.Context, req *proto.DeleteRequest, rsp *proto.DeleteResponse) error {
	logger.Info("Received Post.Delete request")
	return p.db.Delete(model.Equals("id", req.Id))
}
