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
	idIndex      model.Index
	createdIndex model.Index
	slugIndex    model.Index
}

func NewPosts() *Posts {
	createdIndex := model.ByEquality("created")
	createdIndex.Order.Type = model.OrderTypeDesc

	slugIndex := model.ByEquality("slug")

	idIndex := model.ByEquality("id")
	idIndex.Order.Type = model.OrderTypeUnordered

	return &Posts{
		db: model.New(
			store.DefaultStore,
			"posts",
			model.Indexes(slugIndex, createdIndex),
			&model.ModelOptions{
				IdIndex: idIndex,
			},
		),
		createdIndex: createdIndex,
		slugIndex:    slugIndex,
		idIndex:      idIndex,
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
		post.Slug = slug.Make(req.Title)
	}
	return p.db.Save(post)
}

func (p *Posts) Query(ctx context.Context, req *proto.QueryRequest, rsp *proto.QueryResponse) error {
	var q model.Query
	if len(req.Slug) > 0 {
		logger.Infof("Reading post by slug: %v", req.Slug)
		q = p.slugIndex.ToQuery(req.Slug)
	} else if len(req.Id) > 0 {
		logger.Infof("Reading post by id: %v", req.Id)
		q = p.idIndex.ToQuery(req.Id)
	} else {
		q = p.createdIndex.ToQuery(nil)

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
	rsp.Posts = posts
	return nil
}

func (p *Posts) Delete(ctx context.Context, req *proto.DeleteRequest, rsp *proto.DeleteResponse) error {
	logger.Info("Received Post.Delete request")
	return p.db.Delete(model.Equals("id", req.Id))
}
