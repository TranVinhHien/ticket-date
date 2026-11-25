import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { API_CONFIG } from '../core/config/api.config';
import { ApiResponse } from '../core/models/api-response.model';
import { StockResponse } from '../core/models/inventory.model';

@Injectable({
  providedIn: 'root'
})
export class InventoryApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${API_CONFIG.baseUrl}/inventories/stock`;

  getStock(ticketTypeId: string): Observable<ApiResponse<StockResponse>> {
    return this.http.get<ApiResponse<StockResponse>>(`${this.baseUrl}/${ticketTypeId}`);
  }
}
