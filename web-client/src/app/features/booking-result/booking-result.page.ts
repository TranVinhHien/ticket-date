import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { OrderApiService } from '../../services/order-api.service';
import { OrderResponse } from '../../core/models/order.model';
import { CountdownTimerComponent } from '../../shared/components/countdown-timer/countdown-timer.component';
import { LoadingSkeletonComponent } from '../../shared/components/loading-skeleton/loading-skeleton.component';
import { ErrorStateComponent } from '../../shared/components/error-state/error-state.component';

@Component({
  selector: 'app-booking-result-page',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    CountdownTimerComponent,
    LoadingSkeletonComponent,
    ErrorStateComponent
  ],
  template: `
    <div class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-12">
      @if (loading()) {
        <app-loading-skeleton type="detail" />
      } @else if (error()) {
        <app-error-state [message]="error()!" (retryClicked)="loadOrder()" />
      } @else if (order()) {
        <div class="bg-white rounded-2xl p-6 sm:p-10 border border-[#E1E5EE] text-center">
          <!-- Success Icon -->
          <div class="w-16 h-16 rounded-2xl bg-[#F0FDFA] border border-[#99F6E4] text-[#0D9488] flex items-center justify-center mx-auto mb-5">
            <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
            </svg>
          </div>

          <!-- Main Status Header -->
          <h1 class="font-display text-2xl sm:text-3xl font-bold text-[#121622] tracking-tight mb-2">
            Đặt vé thành công!
          </h1>
          <p class="text-sm text-[#5A6275] max-w-md mx-auto mb-6">
            Đơn hàng của bạn đã được ghi nhận vào hệ thống. Vé đã được giữ chỗ tạm thời.
          </p>

          <!-- Order Code & Status Badges -->
          <div class="inline-flex flex-wrap items-center justify-center gap-2.5 p-2.5 bg-[#F5F6F8] rounded-xl mb-6 border border-[#E1E5EE]">
            <div class="flex items-center gap-1.5 px-3 py-1 bg-white rounded-lg border border-[#E1E5EE] text-xs font-bold text-[#121622] tabular-nums">
              <span class="text-[#5A6275] font-normal">Mã đơn:</span>
              <span>{{ order()?.order_code }}</span>
            </div>

            <!-- Status Badge -->
            <span class="px-3 py-1 rounded-lg text-xs font-semibold bg-[#FEF3C7] text-[#92400E] border border-[#FDE68A]">
              Chờ thanh toán (Payment chưa tích hợp)
            </span>
          </div>

          <!-- Countdown Timer for PENDING orders -->
          @if (order()?.status === 'PENDING') {
            <div class="mb-7">
              <app-countdown-timer [expiresAt]="order()?.expires_at" />
            </div>
          }

          <!-- Order Summary Details -->
          <div class="bg-[#F5F6F8] rounded-xl p-5 sm:p-6 text-left mb-7 border border-[#E1E5EE]">
            <h3 class="font-display text-sm font-bold text-[#121622] mb-3 pb-3 border-b border-dashed border-[#D0D7E3]">Chi tiết các vé đã đặt</h3>
            <div class="divide-y divide-dashed divide-[#D0D7E3] text-sm">
              @for (item of order()?.items; track item.id) {
                <div class="py-3 flex items-center justify-between gap-4">
                  <div>
                    <span class="font-semibold text-[#121622] block">{{ item.ticket_type_name }}</span>
                    <span class="text-xs text-[#5A6275] tabular-nums">{{ item.sku }} — {{ formatCurrency(item.unit_price) }} × {{ item.quantity }}</span>
                  </div>
                  <span class="font-bold text-[#121622] tabular-nums">{{ formatCurrency(item.subtotal) }}</span>
                </div>
              }
            </div>

            <div class="pt-4 mt-2 border-t border-dashed border-[#D0D7E3] flex items-center justify-between">
              <span class="font-bold text-[#121622]">Tổng thanh toán</span>
              <span class="font-display text-xl font-bold text-[#1B4BFF] tabular-nums">{{ formatCurrency(order()?.total_amount || 0) }}</span>
            </div>
          </div>

          <!-- Action Buttons -->
          <div class="flex flex-col sm:flex-row items-center justify-center gap-3">
            <a
              [routerLink]="['/my-tickets', order()?.id]"
              class="w-full sm:w-auto px-6 py-3 rounded-xl text-sm font-semibold bg-[#1B4BFF] text-white hover:bg-[#143AD6] transition-colors">
              Xem chi tiết đơn hàng
            </a>
            <a
              routerLink="/"
              class="w-full sm:w-auto px-6 py-3 rounded-xl text-sm font-semibold border border-[#E1E5EE] text-[#121622] hover:bg-[#F5F6F8] transition-colors">
              Quay lại trang chủ
            </a>
          </div>
        </div>
      }
    </div>
  `
})
export class BookingResultPageComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly orderApi = inject(OrderApiService);

  readonly order = signal<OrderResponse | null>(null);
  readonly loading = signal<boolean>(true);
  readonly error = signal<string | null>(null);

  ngOnInit(): void {
    this.route.paramMap.subscribe(params => {
      const orderId = params.get('orderId');
      if (orderId) {
        this.loadOrder(orderId);
      }
    });
  }

  loadOrder(idOverride?: string): void {
    const id = idOverride || this.route.snapshot.paramMap.get('orderId');
    if (!id) return;

    this.loading.set(true);
    this.error.set(null);

    this.orderApi.getOrderById(id).subscribe({
      next: (res) => {
        this.loading.set(false);
        this.order.set(res.data);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.message || 'Không thể tìm thấy thông tin đơn hàng.');
      }
    });
  }

  formatCurrency(amount: number): string {
    return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
  }
}
