import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink } from '@angular/router';
import { CheckoutStateService } from '../../core/services/checkout-state.service';
import { OrderApiService } from '../../services/order-api.service';
import { AuthService } from '../../core/services/auth.service';
import { ConfirmDialogComponent } from '../../shared/components/confirm-dialog/confirm-dialog.component';

@Component({
  selector: 'app-checkout-page',
  standalone: true,
  imports: [CommonModule, RouterLink, ConfirmDialogComponent],
  template: `
    <div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
      <!-- Back to Event -->
      <div class="mb-6">
        @if (checkoutState.event()) {
          <a [routerLink]="['/events', checkoutState.event()?.slug]" class="inline-flex items-center gap-2 text-sm font-semibold text-[#5A6275] hover:text-[#1B4BFF] transition-colors">
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            <span>Quay lại trang sự kiện</span>
          </a>
        } @else {
          <a routerLink="/" class="inline-flex items-center gap-2 text-sm font-semibold text-[#5A6275] hover:text-[#1B4BFF] transition-colors">
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            <span>Quay lại trang chủ</span>
          </a>
        }
      </div>

      <!-- Page Header -->
      <div class="mb-8 pb-4 border-b border-[#E1E5EE]">
        <h1 class="font-display text-2xl sm:text-3xl font-bold text-[#121622] tracking-tight">Xác nhận thông tin đặt vé</h1>
        <p class="text-sm text-[#5A6275] mt-1">Vui lòng kiểm tra kỹ danh sách vé và số lượng trước khi xác nhận tạo đơn hàng.</p>
      </div>

      @if (!checkoutState.event() || checkoutState.items().length === 0) {
        <div class="bg-white rounded-2xl p-10 text-center border border-[#E1E5EE] max-w-md mx-auto">
          <div class="w-12 h-12 rounded-xl bg-[#FEF3C7] text-[#D97706] flex items-center justify-center mx-auto mb-4">
            <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h3 class="font-display text-lg font-bold text-[#121622] mb-2">Chưa chọn vé nào</h3>
          <p class="text-sm text-[#5A6275] mb-6 leading-relaxed">Bạn chưa chọn vé hoặc phiên đặt vé đã được làm mới. Vui lòng chọn vé từ trang sự kiện.</p>
          <a routerLink="/" class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-semibold bg-[#1B4BFF] text-white hover:bg-[#143AD6] transition-colors">
            Xem danh sách sự kiện
          </a>
        </div>
      } @else {
        <!-- Error Alert -->
        @if (errorMessage()) {
          <div class="mb-6 p-4 bg-[#FFF1F2] border border-[#FECDD3] rounded-xl text-[#9F1239] text-sm flex items-start gap-3">
            <svg class="w-5 h-5 shrink-0 text-[#E11D48] mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div>
              <span class="font-bold block mb-0.5">Đặt vé không thành công</span>
              <span>{{ errorMessage() }}</span>
            </div>
          </div>
        }

        <div class="grid grid-cols-1 lg:grid-cols-12 gap-8">
          <!-- Order Summary Card -->
          <div class="lg:col-span-8 space-y-5">
            <!-- Event Card Summary -->
            <div class="bg-white rounded-2xl p-5 sm:p-6 border border-[#E1E5EE] flex flex-col sm:flex-row gap-5 items-start">
              @if (checkoutState.event()?.poster_url) {
                <img
                  [src]="checkoutState.event()?.poster_url"
                  [alt]="checkoutState.event()?.name"
                  class="w-full sm:w-28 h-28 rounded-xl object-cover shrink-0 border border-[#E1E5EE]" />
              }
              <div class="flex-1">
                <span class="text-xs font-semibold text-[#1B4BFF] block mb-1">Sự kiện đã chọn</span>
                <h3 class="font-display text-lg font-bold text-[#121622] mb-2">{{ checkoutState.event()?.name }}</h3>
                <div class="space-y-1.5 text-xs text-[#5A6275]">
                  <div class="flex items-center gap-1.5 tabular-nums">
                    <svg class="w-4 h-4 text-[#7C8498] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                    <span>{{ formatDateTime(checkoutState.event()?.start_at) }}</span>
                  </div>
                  <div class="flex items-center gap-1.5">
                    <svg class="w-4 h-4 text-[#7C8498] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                    </svg>
                    <span>{{ checkoutState.event()?.venue?.name }} ({{ checkoutState.event()?.venue?.address }})</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- Selected Tickets Breakdown -->
            <div class="bg-white rounded-2xl p-6 sm:p-7 border border-[#E1E5EE]">
              <h4 class="font-display text-base font-bold text-[#121622] mb-4 pb-3 border-b border-dashed border-[#E1E5EE]">Chi tiết vé đặt</h4>
              <div class="divide-y divide-dashed divide-[#E1E5EE]">
                @for (item of checkoutState.items(); track item.ticketType.id) {
                  <div class="py-4 flex items-center justify-between gap-4">
                    <div>
                      <div class="flex items-center gap-2">
                        <span class="font-semibold text-sm text-[#121622]">{{ item.ticketType.name }}</span>
                        <span class="text-xs px-2 py-0.5 rounded bg-[#F5F6F8] border border-[#E1E5EE] text-[#5A6275]">{{ item.ticketType.sku }}</span>
                      </div>
                      <span class="text-xs text-[#5A6275] block mt-1 tabular-nums">
                        {{ formatCurrency(item.ticketType.price) }} × {{ item.quantity }}
                      </span>
                    </div>

                    <span class="font-bold text-base text-[#121622] tabular-nums">
                      {{ formatCurrency(item.ticketType.price * item.quantity) }}
                    </span>
                  </div>
                }
              </div>
            </div>

            <!-- MVP Notice Banner -->
            <div class="p-4 sm:p-5 rounded-xl bg-[#FEF3C7]/60 border border-[#FDE68A] text-[#78350F] text-xs leading-relaxed flex items-start gap-3">
              <svg class="w-5 h-5 text-[#D97706] shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <span class="font-bold block mb-0.5">Lưu ý về quy trình thanh toán (MVP)</span>
                <span>Hệ thống hiện tại sẽ tạo đơn và giữ vé cho bạn trong vòng 15 phút với trạng thái <strong>Chờ thanh toán</strong>. Tính năng thanh toán online tự động sẽ được bổ sung ở phiên bản tiếp theo.</span>
              </div>
            </div>
          </div>

          <!-- Order Summary Sidebar -->
          <div class="lg:col-span-4">
            <div class="bg-white rounded-2xl p-6 border border-[#E1E5EE] space-y-5 sticky top-24">
              <h4 class="font-display text-lg font-bold text-[#121622] pb-3 border-b border-dashed border-[#E1E5EE]">Tổng kết đơn hàng</h4>

              <div class="space-y-3 text-sm">
                <div class="flex justify-between text-[#5A6275]">
                  <span>Số lượng vé</span>
                  <span class="font-semibold text-[#121622] tabular-nums">{{ checkoutState.totalQuantity() }} vé</span>
                </div>
                <div class="flex justify-between text-[#5A6275]">
                  <span>Người đặt</span>
                  <span class="font-semibold text-[#121622] truncate max-w-[160px]">
                    {{ authService.currentUser()?.profile?.full_name || authService.currentUser()?.email }}
                  </span>
                </div>
                <div class="pt-4 td-stub-divider flex justify-between items-baseline">
                  <span class="font-bold text-[#121622]">Tổng thanh toán</span>
                  <span class="font-display text-2xl font-bold text-[#1B4BFF] tabular-nums">
                    {{ formatCurrency(checkoutState.totalAmount()) }}
                  </span>
                </div>
              </div>

              <!-- Submit Button -->
              <button
                type="button"
                (click)="openConfirmDialog()"
                [disabled]="submitting()"
                class="w-full py-3.5 px-6 rounded-xl font-semibold text-sm bg-[#1B4BFF] hover:bg-[#143AD6] text-white disabled:opacity-40 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2">
                @if (submitting()) {
                  <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                  </svg>
                  <span>Đang xử lý đơn...</span>
                } @else {
                  <span>Xác nhận đặt vé</span>
                }
              </button>
            </div>
          </div>
        </div>
      }

      <!-- Confirm Dialog Modal -->
      <app-confirm-dialog
        [isOpen]="showConfirmDialog()"
        [loading]="submitting()"
        title="Xác nhận tạo đơn đặt vé"
        message="Hệ thống sẽ giữ vé và tạo đơn hàng cho bạn trong 15 phút. Bạn có chắc chắn muốn tiến hành?"
        confirmLabel="Đặt vé ngay"
        (confirmed)="onSubmitOrder()"
        (cancelled)="closeConfirmDialog()" />
    </div>
  `
})
export class CheckoutPageComponent implements OnInit {
  readonly checkoutState = inject(CheckoutStateService);
  readonly authService = inject(AuthService);
  private readonly orderApi = inject(OrderApiService);
  private readonly router = inject(Router);

  readonly submitting = signal<boolean>(false);
  readonly errorMessage = signal<string>('');
  readonly showConfirmDialog = signal<boolean>(false);

  ngOnInit(): void {
    if (!this.checkoutState.event() || this.checkoutState.items().length === 0) {
      // Direct access without selection
    }
  }

  openConfirmDialog(): void {
    this.showConfirmDialog.set(true);
  }

  closeConfirmDialog(): void {
    this.showConfirmDialog.set(false);
  }

  onSubmitOrder(): void {
    if (this.submitting()) return;

    this.submitting.set(true);
    this.errorMessage.set('');

    const items = this.checkoutState.items().map(i => ({
      ticket_type_id: i.ticketType.id,
      quantity: i.quantity
    }));

    // Use the unique idempotency key generated for this checkout session
    const idempotencyKey = this.checkoutState.idempotencyKey();

    this.orderApi.createOrder({
      idempotency_key: idempotencyKey,
      items
    }).subscribe({
      next: (res) => {
        this.submitting.set(false);
        this.showConfirmDialog.set(false);
        if (res.data) {
          const orderId = res.data.id;
          this.checkoutState.clear();
          this.router.navigate(['/booking-result', orderId]);
        }
      },
      error: (err) => {
        this.submitting.set(false);
        this.showConfirmDialog.set(false);
        this.errorMessage.set(err.message || 'Không thể tạo đơn hàng. Vui lòng thử lại sau.');
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
