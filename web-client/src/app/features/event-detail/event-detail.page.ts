import { Component, inject, OnInit, signal, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { forkJoin, of } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { EventApiService } from '../../services/event-api.service';
import { InventoryApiService } from '../../services/inventory-api.service';
import { Event } from '../../core/models/event.model';
import { StockResponse } from '../../core/models/inventory.model';
import { TicketSelectorComponent } from './ticket-selector.component';
import { LoadingSkeletonComponent } from '../../shared/components/loading-skeleton/loading-skeleton.component';
import { ErrorStateComponent } from '../../shared/components/error-state/error-state.component';

@Component({
  selector: 'app-event-detail-page',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    TicketSelectorComponent,
    LoadingSkeletonComponent,
    ErrorStateComponent
  ],
  template: `
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
      <!-- Back Navigation -->
      <div class="mb-6">
        <a routerLink="/" class="inline-flex items-center gap-2 text-sm font-semibold text-[#5A6275] hover:text-[#1B4BFF] transition-colors">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          <span>Quay lại danh sách sự kiện</span>
        </a>
      </div>

      @if (loading()) {
        <app-loading-skeleton type="detail" />
      } @else if (error()) {
        <app-error-state [message]="error()!" (retryClicked)="loadEventDetail()" />
      } @else if (event()) {
        <!-- Event Hero & Content -->
        <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-10">
          <!-- Left Column: Poster & Details -->
          <div class="lg:col-span-7 space-y-6">
            <!-- Event Hero Poster -->
            <div class="relative w-full h-72 sm:h-96 rounded-2xl overflow-hidden bg-[#121622] border border-[#E1E5EE]">
              @if (event()?.poster_url) {
                <img
                  [src]="event()?.poster_url"
                  [alt]="event()?.name"
                  class="w-full h-full object-cover" />
              } @else {
                <div class="w-full h-full flex flex-col items-center justify-center text-white/50 p-6 text-center td-stage-grid">
                  <svg class="w-16 h-16 mb-2 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
                  </svg>
                  <span class="text-sm font-medium">Ticket Date Concert Event</span>
                </div>
              }

              <!-- Live Status Badge -->
              <div class="absolute top-4 left-4">
                <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-semibold bg-[#121622]/90 backdrop-blur-md text-white border border-white/15">
                  <span class="w-2 h-2 rounded-full bg-teal-400"></span>
                  {{ event()?.status }}
                </span>
              </div>
            </div>

            <!-- Title & Metadata -->
            <div>
              <h1 class="font-display text-2xl sm:text-4xl font-bold text-[#121622] tracking-tight mb-5">
                {{ event()?.name }}
              </h1>

              <!-- Info Cards -->
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <!-- Time Card -->
                <div class="p-4 rounded-xl bg-white border border-[#E1E5EE] flex items-start gap-3">
                  <div class="w-10 h-10 rounded-lg bg-[#1B4BFF]/10 text-[#1B4BFF] flex items-center justify-center shrink-0">
                    <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <div>
                    <span class="text-xs font-medium text-[#5A6275] block">Thời gian tổ chức</span>
                    <span class="text-sm font-semibold text-[#121622] tabular-nums">{{ formatDateTime(event()?.start_at) }}</span>
                    @if (event()?.end_at) {
                      <span class="text-xs text-[#5A6275] block mt-0.5 tabular-nums">Đến {{ formatDateTime(event()?.end_at) }}</span>
                    }
                  </div>
                </div>

                <!-- Venue Card -->
                <div class="p-4 rounded-xl bg-white border border-[#E1E5EE] flex items-start gap-3">
                  <div class="w-10 h-10 rounded-lg bg-[#121622]/6 text-[#121622] flex items-center justify-center shrink-0">
                    <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
                    </svg>
                  </div>
                  <div>
                    <span class="text-xs font-medium text-[#5A6275] block">Địa điểm</span>
                    <span class="text-sm font-semibold text-[#121622]">{{ event()?.venue?.name || 'Đang cập nhật' }}</span>
                    <span class="text-xs text-[#5A6275] block mt-0.5">{{ event()?.venue?.address || '' }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- Organizer Info -->
            @if (event()?.organizer) {
              <div class="p-4 rounded-xl bg-white border border-[#E1E5EE] flex items-center gap-4">
                <div class="w-11 h-11 rounded-lg bg-[#121622] text-white flex items-center justify-center font-display font-bold uppercase text-sm shrink-0">
                  {{ event()?.organizer?.organization_name?.slice(0, 2) }}
                </div>
                <div>
                  <span class="text-xs text-[#5A6275] font-medium block">Đơn vị tổ chức</span>
                  <span class="text-sm font-semibold text-[#121622]">{{ event()?.organizer?.organization_name }}</span>
                </div>
              </div>
            }

            <!-- Description -->
            @if (event()?.description) {
              <div class="bg-white rounded-xl p-6 sm:p-7 border border-[#E1E5EE]">
                <h3 class="font-display text-lg font-bold text-[#121622] mb-3 pb-3 td-stub-divider border-t-0 border-b border-dashed border-[#E1E5EE]">Mô tả sự kiện</h3>
                <div class="text-sm text-[#3F4656] leading-relaxed whitespace-pre-line">
                  {{ event()?.description }}
                </div>
              </div>
            }
          </div>

          <!-- Right Column: Ticket Selector -->
          <div class="lg:col-span-5">
            <div class="sticky top-24">
              <app-ticket-selector
                #ticketSelector
                [event]="event()!"
                [stockMap]="stockMap()" />
            </div>
          </div>
        </div>
      }
    </div>
  `
})
export class EventDetailPageComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly eventApi = inject(EventApiService);
  private readonly inventoryApi = inject(InventoryApiService);

  @ViewChild('ticketSelector') ticketSelector?: TicketSelectorComponent;

  readonly event = signal<Event | null>(null);
  readonly stockMap = signal<Record<string, StockResponse>>({});
  readonly loading = signal<boolean>(true);
  readonly error = signal<string | null>(null);

  ngOnInit(): void {
    this.route.paramMap.subscribe(params => {
      const slug = params.get('slug');
      if (slug) {
        this.loadEventDetail(slug);
      }
    });
  }

  loadEventDetail(slugOverride?: string): void {
    const slug = slugOverride || this.route.snapshot.paramMap.get('slug');
    if (!slug) return;

    this.loading.set(true);
    this.error.set(null);

    this.eventApi.getEventBySlug(slug).subscribe({
      next: (res) => {
        this.loading.set(false);
        const eventData = res.data;
        this.event.set(eventData);

        // Fetch live inventory for each ticket type in parallel
        if (eventData?.ticket_types && eventData.ticket_types.length > 0) {
          const requests = eventData.ticket_types.map(tt =>
            this.inventoryApi.getStock(tt.id).pipe(
              catchError(() => of({ code: '404', message: 'not found', meta: null, data: { ticket_type_id: tt.id, available_quantity: 0, has_stock: false } }))
            )
          );

          forkJoin(requests).subscribe(results => {
            const map: Record<string, StockResponse> = {};
            results.forEach(r => {
              if (r.data) {
                map[r.data.ticket_type_id] = r.data;
              }
            });
            this.stockMap.set(map);
            this.ticketSelector?.updateStock(map);
          });
        }
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.message || 'Không thể tải chi tiết sự kiện.');
      }
    });
  }

  formatDateTime(iso?: string): string {
    if (!iso) return '';
    return new Date(iso).toLocaleDateString('vi-VN', {
      weekday: 'long',
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }
}
