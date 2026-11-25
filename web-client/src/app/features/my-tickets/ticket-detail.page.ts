import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { OrderApiService } from '../../services/order-api.service';
import { OrderResponse } from '../../core/models/order.model';
import { CountdownTimerComponent } from '../../shared/components/countdown-timer/countdown-timer.component';
import { LoadingSkeletonComponent } from '../../shared/components/loading-skeleton/loading-skeleton.component';
import { ErrorStateComponent } from '../../shared/components/error-state/error-state.component';

@Component({
  selector: 'app-ticket-detail-page',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    CountdownTimerComponent,
    LoadingSkeletonComponent,
    ErrorStateComponent
  ],
  template: `
    <div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
      <!-- Back Navigation -->
      <div class="mb-6">
        <a routerLink="/my-tickets" class="inline-flex items-center gap-2 text-sm font-semibold text-[#5A6275] hover:text-[#1B4BFF] transition-colors">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          <span>Quay lại vé của tôi</span>
        </a>
      </div>

      @if (loading()) {
        <app-loading-skeleton type="detail" />
      } @else if (error()) {
        <app-error-state [message]="error()!" (retryClicked)="loadOrder()" />
      } @else if (order()) {
        <div class="bg-white rounded-2xl p-6 sm:p-8 border border-[#E1E5EE] space-y-7">
          <!-- Order Header -->
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-dashed border-[#E1E5EE]">
            <div>
              <span class="text-xs font-semibold text-[#1B4BFF] block mb-1">Chi tiết đơn hàng</span>
              <div class="flex flex-wrap items-center gap-3">
                <h1 class="font-display text-2xl sm:text-3xl font-bold text-[#121622] tracking-tight tabular-nums">
                  {{ order()?.order_code }}
                </h1>
                @if (order()?.status === 'PENDING') {
                  <span class="px-3 py-1 rounded-lg text-xs font-semibold bg-[#FEF3C7] text-[#92400E] border border-[#FDE68A]">
                    Chờ thanh toán (Payment chưa tích hợp)
                  </span>
                } @else {
                  <span class="px-3 py-1 rounded-lg text-xs font-semibold bg-[#F0FDFA] text-[#115E59] border border-[#99F6E4]">
                    {{ order()?.status }}
                  </span>
                }
              </div>
              <span class="text-xs text-[#5A6275] block mt-1 tabular-nums">Khởi tạo lúc {{ formatDateTime(order()?.created_at) }}</span>
            </div>

            <!-- Countdown Timer if PENDING -->
            @if (order()?.status === 'PENDING') {
              <div>
                <app-countdown-timer [expiresAt]="order()?.expires_at" />
              </div>
            }
          </div>

          <!-- Order Items Table -->
          <div>
            <h3 class="font-display text-base font-bold text-[#121622] mb-4">Danh sách vé</h3>
            <div class="rounded-xl border border-[#E1E5EE] overflow-x-auto">
              <table class="w-full text-left text-sm">
                <thead class="bg-[#F5F6F8] text-xs font-semibold text-[#5A6275] border-b border-[#E1E5EE]">
                  <tr>
                    <th class="py-3 px-4">Loại vé / SKU</th>
                    <th class="py-3 px-4 text-center">Số lượng</th>
                    <th class="py-3 px-4 text-right">Đơn giá</th>
                    <th class="py-3 px-4 text-right">Thành tiền</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-dashed divide-[#E1E5EE]">
                  @for (item of order()?.items; track item.id) {
                    <tr>
                      <td class="py-4 px-4">
                        <span class="font-semibold text-[#121622] block">{{ item.ticket_type_name }}</span>
                        <span class="text-xs text-[#5A6275]">{{ item.sku }}</span>
                      </td>
                      <td class="py-4 px-4 text-center font-bold text-[#121622] tabular-nums">{{ item.quantity }}</td>
                      <td class="py-4 px-4 text-right font-medium text-[#5A6275] tabular-nums">{{ formatCurrency(item.unit_price) }}</td>
                      <td class="py-4 px-4 text-right font-bold text-[#121622] tabular-nums">{{ formatCurrency(item.subtotal) }}</td>
                    </tr>
                  }
                </tbody>
                <tfoot class="bg-[#F5F6F8] font-bold border-t border-[#E1E5EE]">
                  <tr>
                    <td colspan="3" class="py-4 px-4 text-[#121622]">Tổng thanh toán</td>
                    <td class="py-4 px-4 text-right font-display text-lg text-[#1B4BFF] font-bold tabular-nums">
                      {{ formatCurrency(order()?.total_amount || 0) }}
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>

          <!-- E-ticket Information Banner -->
          <div class="p-4 sm:p-5 rounded-xl bg-[#F5F6F8] border border-[#E1E5EE] text-[#121622] text-xs flex items-start gap-3">
            <svg class="w-5 h-5 text-[#1B4BFF] shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
            </svg>
            <div>
              <span class="font-bold block mb-0.5">Thông tin vé điện tử</span>
              <span class="text-[#5A6275] leading-relaxed">Vé điện tử chính thức và mã QR check-in sẽ được hệ thống cung cấp sau khi đơn hàng được xác nhận thanh toán thành công qua cổng thanh toán (dự kiến tích hợp ở giai đoạn tiếp theo).</span>
            </div>
          </div>
        </div>
      }
    </div>
  `
})
export class TicketDetailPageComponent implements OnInit {
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
        this.error.set(err.message || 'Không thể tìm thấy thông tin chi tiết đơn hàng.');
      }
    });
  }

  formatDateTime(iso?: string): string {
    if (!iso) return '';
    return new Date(iso).toLocaleDateString('vi-VN', {
      weekday: 'short',
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
