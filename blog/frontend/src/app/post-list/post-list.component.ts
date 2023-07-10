import { Component, OnInit } from '@angular/core';
import { MicroService } from '../micro.service';

@Component({
  selector: 'app-post-list',
  templateUrl: './post-list.component.html',
  styleUrls: ['./post-list.component.css']
})
export class PostListComponent implements OnInit {
  m3o: MicroService;
  posts: Object[];

  constructor(m3o: MicroService) {
    this.m3o = m3o
  }

  ngOnInit(): void {
    this.m3o.get("posts", "query", null).then(v => {
      this.posts = v["posts"]
    })
  }

}
