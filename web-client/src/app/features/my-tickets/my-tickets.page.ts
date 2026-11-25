import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { OrderApiService } from '../../services/order-api.service';
import { OrderResponse } from '../../core/models/order.model';
import { PaginationMeta } from '../../core/models/api-response.model';
import { CountdownTimerComponent } from '../../shared/components/countdown-timer/countdown-timer.component';
import { LoadingSkeletonComponent } from '../../shared/components/loading-skeleton/loading-skeleton.component';
import { EmptyStateComponent } from '../../shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '../../shared/components/error-state/error-state.component';

@Component({
  selector: 'app-my-tickets-page',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    CountdownTimerComponent,
    LoadingSkeletonComponent,
    EmptyStateComponent,
    ErrorStateComponent
  ],
  template: `
    <div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
      <!-- Page Header -->
      <div class="mb-8 pb-4 border-b border-[#E1E5EE] flex flex-col sm:flex-row sm:items-end justify-between gap-4">
        <div>
          <h1 class="font-display text-2xl sm:text-3xl font-bold text-[#121622] tracking-tight">Vé &amp; Đơn hàng của tôi</h1>
          <p class="text-sm text-[#5A6275] mt-1">Danh sách tất cả các đơn đặt vé sự kiện bạn đã thực hiện</p>
        </div>

        @if (pagination()?.total_items) {
          <span class="inline-flex items-center px-3 py-1.5 rounded-lg text-xs font-semibold bg-white border border-[#E1E5EE] text-[#121622] tabular-nums">
            Tổng cộng: {{ pagination()?.total_items }} đơn
          </span>
        }
      </div>

      <!-- Content Area -->
      @if (loading()) {
        <app-loading-skeleton type="detail" />
      } @else if (error()) {
        <app-error-state [message]="error()!" (retryClicked)="loadOrders()" />
      } @else if (orders().length === 0) {
        <app-empty-state
          title="Bạn chưa có đơn đặt vé nào"
          description="Khám phá ngay các sự kiện âm nhạc và concert hấp dẫn để đặt vé cho bạn và bạn bè!"
          actionLabel="Khám phá sự kiện"
          (actionClicked)="goToHome()" />
      } @else {
        <!-- Orders List -->
        <div class="space-y-4">
          @for (order of orders(); track order.id) {
            <div class="bg-white rounded-xl p-5 sm:p-6 border border-[#E1E5EE] hover:border-[#C7CEDE] transition-colors">
              <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-dashed border-[#E1E5EE]">
                <div class="flex items-center gap-3">
                  <div class="w-10 h-10 rounded-lg bg-[#121622] text-white flex items-center justify-center font-display font-bold text-sm shrink-0">
                    #
                  </div>
                  <div>
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="font-display font-bold text-[#121622] text-base tabular-nums">{{ order.order_code }}</span>
                      @if (order.status === 'PENDING') {
                        <span class="px-2.5 py-0.5 rounded-md text-xs font-semibold bg-[#FEF3C7] text-[#92400E] border border-[#FDE68A]">
                          Chờ thanh toán (Payment chưa tích hợp)
                        </span>
                      } @else {
                        <span class="px-2.5 py-0.5 rounded-md text-xs font-semibold bg-[#F0FDFA] text-[#115E59] border border-[#99F6E4]">
                          {{ order.status }}
                        </span>
                      }
                    </div>
                    <span class="text-xs text-[#5A6275] block mt-0.5 tabular-nums">Đặt lúc {{ formatDateTime(order.created_at) }}</span>
                  </div>
                </div>

                <!-- Countdown timer if PENDING -->
                @if (order.status === 'PENDING') {
                  <div>
                    <app-countdown-timer [expiresAt]="order.expires_at" />
                  </div>
                }
              </div>

              <!-- Items Breakdown -->
              <div class="py-4 space-y-2">
                @for (item of order.items; track item.id) {
                  <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
                    <div class="flex items-center gap-2">
                      <span class="w-1.5 h-1.5 rounded-full bg-[#1B4BFF]"></span>
                      <span class="font-semibold text-[#121622]">{{ item.ticket_type_name }}</span>
                      <span class="text-xs text-[#5A6275]">({{ item.sku }})</span>
                    </div>
                    <span class="text-xs text-[#5A6275] tabular-nums">
                      {{ formatCurrency(item.unit_price) }} × {{ item.quantity }} = <strong class="text-[#121622]">{{ formatCurrency(item.subtotal) }}</strong>
                    </span>
                  </div>
                }
              </div>

              <!-- Footer Actions -->
              <div class="pt-4 td-stub-divider flex items-center justify-between">
                <div>
                  <span class="text-xs text-[#5A6275] block">Tổng tiền</span>
                  <span class="font-display text-lg font-bold text-[#121622] tabular-nums">{{ formatCurrency(order.total_amount) }}</span>
                </div>

                <a
                  [routerLink]="['/my-tickets', order.id]"
                  class="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-semibold bg-[#121622] text-white hover:bg-[#1B4BFF] transition-colors">
                  <span>Chi tiết đơn</span>
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                  </svg>
                </a>
              </div>
            </div>
          }
        </div>

        <!-- Pagination -->
        @if (pagination() && pagination()!.total_pages > 1) {
          <div class="flex items-center justify-center gap-2 mt-8 pt-6 border-t border-[#E1E5EE]">
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
export class MyTicketsPageComponent implements OnInit {
  private readonly orderApi = inject(OrderApiService);

  readonly orders = signal<OrderResponse[]>([]);
  readonly pagination = signal<PaginationMeta | null>(null);
  readonly loading = signal<boolean>(true);
  readonly error = signal<string | null>(null);
  readonly currentPage = signal<number>(1);

  ngOnInit(): void {
    this.loadOrders();
  }

  loadOrders(page: number = 1): void {
    this.loading.set(true);
    this.error.set(null);
    this.currentPage.set(page);

    this.orderApi.listMyOrders(page, 10).subscribe({
      next: (res) => {
        this.loading.set(false);
        this.orders.set(res.data || []);
        this.pagination.set(res.meta);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.message || 'Không thể tải danh sách đơn hàng.');
      }
    });
  }

  goToPage(page: number): void {
    this.loadOrders(page);
  }

  goToHome(): void {
    window.location.href = '/';
  }

  formatDateTime(iso?: string): string {
    if (!iso) return '';
    return new Date(iso).toLocaleDateString('vi-VN', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  formatCurrency(amount: number): string {
    return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
  }
}
