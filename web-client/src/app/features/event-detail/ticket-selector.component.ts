import { Component, input, output, signal, computed, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { Event, TicketType } from '../../core/models/event.model';
import { StockResponse } from '../../core/models/inventory.model';
import { CheckoutStateService, SelectedTicketItem } from '../../core/services/checkout-state.service';
import { AuthService } from '../../core/services/auth.service';

export interface TicketTypeWithStock {
  ticketType: TicketType;
  stock?: StockResponse;
  quantity: number;
}

@Component({
  selector: 'app-ticket-selector',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="bg-white rounded-2xl p-6 sm:p-7 border border-[#E1E5EE]">
      <div class="flex items-center justify-between mb-5 pb-4 border-b border-dashed border-[#E1E5EE]">
        <div>
          <h3 class="font-display text-xl font-bold text-[#121622]">Chọn loại vé</h3>
          <p class="text-xs text-[#5A6275] mt-0.5">Số lượng vé còn lại được kiểm tra theo thời gian thực</p>
        </div>
      </div>

      <!-- Ticket Types List -->
      <div class="space-y-3">
        @for (item of ticketList(); track item.ticketType.id) {
          <div
            class="p-4 rounded-xl border transition-colors"
            [class]="item.quantity > 0 ? 'border-[#1B4BFF] bg-[#1B4BFF]/[0.03]' : 'border-[#E1E5EE] hover:border-[#C7CEDE]'">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <!-- Ticket Info -->
              <div class="flex-1">
                <div class="flex flex-wrap items-center gap-2 mb-1">
                  <h4 class="font-display font-bold text-[#121622] text-base">{{ item.ticketType.name }}</h4>
                  <span class="text-xs px-2 py-0.5 rounded bg-[#F5F6F8] border border-[#E1E5EE] text-[#5A6275]">
                    {{ item.ticketType.sku }}
                  </span>
                </div>

                @if (item.ticketType.description) {
                  <p class="text-xs text-[#5A6275] mb-2">{{ item.ticketType.description }}</p>
                }

                <div class="flex flex-wrap items-center gap-3 text-xs">
                  <!-- Price -->
                  <span class="text-base font-bold text-[#1B4BFF] tabular-nums">
                    {{ formatCurrency(item.ticketType.price) }}
                  </span>

                  <!-- Stock Status -->
                  @if (item.stock) {
                    @if (item.stock.available_quantity > 0) {
                      <span class="text-[#0D9488] font-medium flex items-center gap-1.5 tabular-nums">
                        <span class="w-1.5 h-1.5 rounded-full bg-[#0D9488]"></span>
                        Còn {{ item.stock.available_quantity }} vé (tối đa {{ getMaxAllowed(item) }}/lần)
                      </span>
                    } @else {
                      <span class="text-[#E11D48] font-semibold flex items-center gap-1.5">
                        <span class="w-1.5 h-1.5 rounded-full bg-[#E11D48]"></span>
                        Hết vé
                      </span>
                    }
                  } @else {
                    <span class="text-[#7C8498] animate-pulse">Đang kiểm tra tồn kho...</span>
                  }
                </div>
              </div>

              <!-- Quantity Selector -->
              <div class="flex items-center gap-2.5 shrink-0">
                <button
                  type="button"
                  (click)="decreaseQty(item)"
                  [disabled]="item.quantity === 0"
                  class="w-9 h-9 rounded-lg border border-[#E1E5EE] bg-white flex items-center justify-center font-bold text-[#121622] hover:bg-[#F5F6F8] disabled:opacity-30 disabled:cursor-not-allowed transition-colors">
                  -
                </button>

                <span class="w-8 text-center font-bold text-base text-[#121622] tabular-nums">
                  {{ item.quantity }}
                </span>

                <button
                  type="button"
                  (click)="increaseQty(item)"
                  [disabled]="isIncreaseDisabled(item)"
                  class="w-9 h-9 rounded-lg border border-[#E1E5EE] bg-white flex items-center justify-center font-bold text-[#121622] hover:bg-[#F5F6F8] disabled:opacity-30 disabled:cursor-not-allowed transition-colors">
                  +
                </button>
              </div>
            </div>
          </div>
        }
      </div>

      <!-- Checkout Summary & Action -->
      <div class="mt-6 pt-5 td-stub-divider flex flex-col sm:flex-row items-center justify-between gap-4">
        <div>
          <span class="text-xs text-[#5A6275] block">Tạm tính ({{ totalSelectedQuantity() }} vé)</span>
          <span class="font-display text-2xl font-bold text-[#121622] tabular-nums">
            {{ formatCurrency(totalSelectedPrice()) }}
          </span>
        </div>

        <button
          type="button"
          (click)="onProceedToCheckout()"
          [disabled]="totalSelectedQuantity() === 0"
          class="w-full sm:w-auto px-7 py-3.5 rounded-xl font-semibold text-sm bg-[#1B4BFF] hover:bg-[#143AD6] text-white disabled:opacity-40 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2">
          <span>Tiếp tục đặt vé</span>
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
          </svg>
        </button>
      </div>
    </div>
  `
})
export class TicketSelectorComponent implements OnInit {
  readonly event = input.required<Event>();
  readonly stockMap = input<Record<string, StockResponse>>({});

  private readonly checkoutState = inject(CheckoutStateService);
  private readonly authService = inject(AuthService);
  private readonly router = inject(Router);

  readonly ticketList = signal<TicketTypeWithStock[]>([]);

  readonly totalSelectedQuantity = computed(() => {
    return this.ticketList().reduce((sum, item) => sum + item.quantity, 0);
  });

  readonly totalSelectedPrice = computed(() => {
    return this.ticketList().reduce((sum, item) => sum + (item.ticketType.price * item.quantity), 0);
  });

  ngOnInit(): void {
    const list: TicketTypeWithStock[] = (this.event().ticket_types || []).map(tt => ({
      ticketType: tt,
      stock: this.stockMap()[tt.id],
      quantity: 0
    }));
    this.ticketList.set(list);
  }

  // Update stock when parent supplies stockMap
  updateStock(stockMap: Record<string, StockResponse>): void {
    this.ticketList.update(list =>
      list.map(item => ({
        ...item,
        stock: stockMap[item.ticketType.id]
      }))
    );
  }

  getMaxAllowed(item: TicketTypeWithStock): number {
    const maxPerOrder = item.ticketType.max_per_order || 4;
    const available = item.stock?.available_quantity ?? 0;
    return Math.min(maxPerOrder, available);
  }

  isIncreaseDisabled(item: TicketTypeWithStock): boolean {
    const max = this.getMaxAllowed(item);
    return max <= 0 || item.quantity >= max;
  }

  increaseQty(item: TicketTypeWithStock): void {
    if (this.isIncreaseDisabled(item)) return;
    this.ticketList.update(list =>
      list.map(i => i.ticketType.id === item.ticketType.id ? { ...i, quantity: i.quantity + 1 } : i)
    );
  }

  decreaseQty(item: TicketTypeWithStock): void {
    if (item.quantity <= 0) return;
    this.ticketList.update(list =>
      list.map(i => i.ticketType.id === item.ticketType.id ? { ...i, quantity: i.quantity - 1 } : i)
    );
  }

  formatCurrency(amount: number): string {
    return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
  }

  onProceedToCheckout(): void {
    const selected: SelectedTicketItem[] = this.ticketList()
      .filter(i => i.quantity > 0)
      .map(i => ({ ticketType: i.ticketType, quantity: i.quantity }));

    if (selected.length === 0) return;

    // Store in checkout state
    this.checkoutState.setCheckoutSelection(this.event(), selected);

    // If not logged in, route to login with returnUrl
    if (!this.authService.isAuthenticated()) {
      this.router.navigate(['/login'], { queryParams: { returnUrl: '/checkout' } });
    } else {
      this.router.navigate(['/checkout']);
    }
  }
}
