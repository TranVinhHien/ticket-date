import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { API_CONFIG } from '../core/config/api.config';
import { ApiResponse } from '../core/models/api-response.model';
import { Event, EventQuery } from '../core/models/event.model';

@Injectable({
  providedIn: 'root'
})
export class EventApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${API_CONFIG.baseUrl}/events`;

  listEvents(query?: EventQuery): Observable<ApiResponse<Event[]>> {
    let params = new HttpParams();
    // Default status to PUBLISHED per requirements
    params = params.set('status', query?.status || 'PUBLISHED');

    if (query?.page) params = params.set('page', query.page.toString());
    if (query?.limit) params = params.set('limit', query.limit.toString());
    if (query?.query) params = params.set('query', query.query);
    if (query?.organizer_id) params = params.set('organizer_id', query.organizer_id);
    if (query?.venue_id) params = params.set('venue_id', query.venue_id);
    if (query?.sort) params = params.set('sort', query.sort);
    if (query?.order) params = params.set('order', query.order);

    return this.http.get<ApiResponse<Event[]>>(this.baseUrl, { params });
  }

  getEventBySlug(slug: string): Observable<ApiResponse<Event>> {
    return this.http.get<ApiResponse<Event>>(`${this.baseUrl}/slug/${slug}`);
  }

  getEventById(id: string): Observable<ApiResponse<Event>> {
    return this.http.get<ApiResponse<Event>>(`${this.baseUrl}/${id}`);
  }
}
