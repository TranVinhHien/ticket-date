import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { EventApiService } from '../../services/event-api.service';
import { Event } from '../../core/models/event.model';
import { PaginationMeta } from '../../core/models/api-response.model';
import { EventCardComponent } from './event-card.component';
import { LoadingSkeletonComponent } from '../../shared/components/loading-skeleton/loading-skeleton.component';
import { EmptyStateComponent } from '../../shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '../../shared/components/error-state/error-state.component';

@Component({
  selector: 'app-home-page',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    EventCardComponent,
    LoadingSkeletonComponent,
    EmptyStateComponent,
    ErrorStateComponent
  ],
  template: `
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
      <!-- Architectural Stage Banner -->
      <div class="relative rounded-2xl overflow-hidden bg-[#121622] td-stage-grid text-white p-7 sm:p-12 mb-10 border border-[#242938]">
        <div class="relative z-10 max-w-2xl">
          <div class="inline-flex items-center gap-2 px-3 py-1 rounded-md bg-white/8 text-[#9DB4FF] text-xs font-medium mb-4 border border-white/10">
            <span class="w-1.5 h-1.5 rounded-full bg-[#1B4BFF]"></span>
            <span>Vé concert &amp; sự kiện trực tiếp</span>
          </div>
          <h1 class="font-display text-3xl sm:text-5xl font-bold tracking-tight leading-[1.12] mb-4">
            Khám phá &amp; Đặt vé sự kiện âm nhạc đỉnh cao
          </h1>
          <p class="text-base text-[#A9B1C2] mb-8 leading-relaxed max-w-xl">
            Hòa mình vào không khí cuồng nhiệt cùng thần tượng. Đặt vé nhanh chóng, an toàn và dễ dàng với Ticket Date.
          </p>

          <!-- Search Bar -->
          <form (ngSubmit)="onSearch()" class="flex flex-col sm:flex-row gap-2.5 max-w-xl">
            <div class="relative flex-1">
              <svg class="w-5 h-5 text-[#7C8498] absolute left-3.5 top-1/2 -translate-y-1/2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <input
                type="text"
                [(ngModel)]="searchQuery"
                name="search"
                placeholder="Tìm sự kiện theo tên hoặc nghệ sĩ..."
                class="w-full pl-11 pr-4 py-3 rounded-xl bg-white text-[#121622] placeholder-[#7C8498] border border-transparent focus:outline-none focus:ring-2 focus:ring-[#1B4BFF] text-sm transition-colors" />
            </div>
            <button
              type="submit"
              class="px-6 py-3 rounded-xl font-semibold text-sm bg-[#1B4BFF] hover:bg-[#143AD6] text-white transition-colors flex items-center justify-center gap-2 shrink-0">
              Tìm kiếm
            </button>
          </form>
        </div>
      </div>

      <!-- Events List Header -->
      <div class="flex flex-wrap items-end justify-between gap-4 mb-6 pb-4 border-b border-[#E1E5EE]">
        <div>
          <h2 class="font-display text-2xl font-bold text-[#121622] tracking-tight">Sự kiện nổi bật</h2>
          <p class="text-sm text-[#5A6275] mt-0.5">Các sự kiện chính thức đang mở bán vé</p>
        </div>

        @if (pagination()?.total_items) {
          <span class="text-xs font-semibold px-3 py-1.5 rounded-lg bg-white border border-[#E1E5EE] text-[#121622] tabular-nums">
            {{ pagination()?.total_items }} sự kiện
          </span>
        }
      </div>

      <!-- Content Area -->
      @if (loading()) {
        <app-loading-skeleton [count]="6" />
      } @else if (error()) {
        <app-error-state [message]="error()!" (retryClicked)="loadEvents()" />
      } @else if (events().length === 0) {
        <app-empty-state
          title="Không tìm thấy sự kiện nào"
          description="Hiện tại không có sự kiện nào phù hợp với từ khóa của bạn. Thử tìm kiếm với từ khóa khác nhé."
          actionLabel="Xem tất cả sự kiện"
          (actionClicked)="resetSearch()" />
      } @else {
        <!-- Grid of Events -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
          @for (event of events(); track event.id) {
            <app-event-card [event]="event" />
          }
        </div>

        <!-- Pagination Controls -->
        @if (pagination() && pagination()!.total_pages > 1) {
          <div class="flex items-center justify-center gap-2 mt-12 pt-6 border-t border-[#E1E5EE]">
            <button
              (click)="goToPage(currentPage() - 1)"
              [disabled]="!pagination()?.has_previous"
              class="px-4 py-2 rounded-lg text-sm font-semibold bg-white border border-[#E1E5EE] text-[#121622] hover:bg-[#F5F6F8] disabled:opacity-40 disabled:cursor-not-allowed transition-colors">
              Trang trước
            </button>

            <span class="text-xs font-semibold text-[#5A6275] px-4 tabular-nums">
              Trang {{ pagination()?.current_page }} / {{ pagination()?.total_pages }}
            </span>

            <button
              (click)="goToPage(currentPage() + 1)"
              [disabled]="!pagination()?.has_next"
              class="px-4 py-2 rounded-lg text-sm font-semibold bg-white border border-[#E1E5EE] text-[#121622] hover:bg-[#F5F6F8] disabled:opacity-40 disabled:cursor-not-allowed transition-colors">
              Trang sau
            </button>
          </div>
        }
      }
    </div>
  `
})
export class HomePageComponent implements OnInit {
  private readonly eventApi = inject(EventApiService);

  readonly events = signal<Event[]>([]);
  readonly pagination = signal<PaginationMeta | null>(null);
  readonly loading = signal<boolean>(true);
  readonly error = signal<string | null>(null);
  readonly currentPage = signal<number>(1);

  searchQuery = '';

  ngOnInit(): void {
    this.loadEvents();
  }

  loadEvents(page: number = 1): void {
    this.loading.set(true);
    this.error.set(null);
    this.currentPage.set(page);

    this.eventApi.listEvents({
      page,
      limit: 9,
      query: this.searchQuery || undefined,
      status: 'PUBLISHED'
    }).subscribe({
      next: (res) => {
        this.loading.set(false);
        this.events.set(res.data || []);
        this.pagination.set(res.meta);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.message || 'Không thể tải danh sách sự kiện.');
      }
    });
  }

  onSearch(): void {
    this.loadEvents(1);
  }

  resetSearch(): void {
    this.searchQuery = '';
    this.loadEvents(1);
  }

  goToPage(page: number): void {
    this.loadEvents(page);
    window.scrollTo({ top: 400, behavior: 'smooth' });
  }
}
