import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';
import { API_CONFIG } from '../core/config/api.config';
import { ApiResponse } from '../core/models/api-response.model';
import { CreateOrderRequest, OrderResponse } from '../core/models/order.model';

@Injectable({
  providedIn: 'root'
})
export class OrderApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${API_CONFIG.baseUrl}/orders`;

  createOrder(req: CreateOrderRequest): Observable<ApiResponse<OrderResponse>> {
    const headers = new HttpHeaders({
      'Idempotency-Key': req.idempotency_key
    });
    return this.http.post<ApiResponse<OrderResponse>>(this.baseUrl, req, { headers });
  }

  getOrderById(id: string): Observable<ApiResponse<OrderResponse>> {
    return this.http.get<ApiResponse<OrderResponse>>(`${this.baseUrl}/${id}`);
  }

  listMyOrders(page: number = 1, limit: number = 10): Observable<ApiResponse<OrderResponse[]>> {
    let params = new HttpParams()
      .set('page', page.toString())
      .set('limit', limit.toString());
    return this.http.get<ApiResponse<OrderResponse[]>>(this.baseUrl, { params });
  }
}
