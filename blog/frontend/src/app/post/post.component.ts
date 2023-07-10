import { Component, OnInit } from '@angular/core';
import { MicroService } from '../micro.service';
import { HttpParams } from '@angular/common/http';
import { ActivatedRoute } from '@angular/router';

@Component({
  selector: 'app-post',
  templateUrl: './post.component.html',
  styleUrls: ['./post.component.css'],
})
export class PostComponent implements OnInit {
  post: Object;
  comments: Object[];

  constructor(private m3o: MicroService, private route: ActivatedRoute) {}

  ngOnInit(): void {
    const that = this
    this.route.paramMap.subscribe((paramMap) => {
      if (paramMap.get('slug').length == 0) {
        throw("Slug should not be empty")
      }
      that.m3o
        .get(
          'posts',
          'query',
          new HttpParams({
            fromObject: {
              slug: paramMap.get('slug'),
            },
          })
        )
        .then((v) => {
          that.post = v['posts'][0];
          that.m3o.get(
            'comments',
            'list',
            new HttpParams({
              fromObject: {
                post: that.post["id"],
              },
            })
          ) .then((v) => {
            that.comments = v["comments"]
          })
        });
    });
  }
}
