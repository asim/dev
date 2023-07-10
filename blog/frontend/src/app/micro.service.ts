import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { environment } from '../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class MicroService {
  public address: string = environment.microAddress;
  public namespace: string = environment.microNamespace;

  constructor(private http: HttpClient) { }

  get(service: string, endpoint: string, params: HttpParams): Promise<Object> {
    return this.http.get(this.address + "/" + service + "/" + endpoint, {
      headers: {
        "Micro-Namespace": [this.namespace]
      },
      params: params,
    }).toPromise()
  }
}
