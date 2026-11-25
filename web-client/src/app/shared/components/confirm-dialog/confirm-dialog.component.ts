import { Component, input, output } from '@angular/core';

@Component({
  selector: 'app-confirm-dialog',
  standalone: true,
  template: `
    @if (isOpen()) {
      <div class="fixed inset-0 z-50 overflow-y-auto flex items-center justify-center p-4">
        <!-- Backdrop -->
        <div class="fixed inset-0 bg-[#121622]/70 backdrop-blur-xs transition-opacity" (click)="onCancel()"></div>

        <!-- Dialog Box -->
        <div class="relative bg-white rounded-2xl max-w-md w-full p-6 sm:p-8 shadow-2xl border border-[#E1E5EE] z-10">
          <div class="w-12 h-12 rounded-xl bg-[#1B4BFF]/10 text-[#1B4BFF] flex items-center justify-center mx-auto mb-4">
            <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>

          <h3 class="font-display text-xl font-bold text-[#121622] text-center mb-2">{{ title() }}</h3>
          <p class="text-sm text-[#5A6275] text-center mb-6 leading-relaxed">{{ message() }}</p>

          <div class="flex items-center gap-3 pt-4 td-stub-divider">
            <button
              type="button"
              (click)="onCancel()"
              [disabled]="loading()"
              class="flex-1 py-3 px-4 rounded-xl text-sm font-semibold border border-[#E1E5EE] text-[#121622] hover:bg-[#F5F6F8] disabled:opacity-40 transition-colors">
              Hủy
            </button>
            <button
              type="button"
              (click)="onConfirm()"
              [disabled]="loading()"
              class="flex-1 py-3 px-4 rounded-xl text-sm font-semibold bg-[#1B4BFF] hover:bg-[#143AD6] text-white disabled:opacity-40 transition-colors flex items-center justify-center gap-2">
              @if (loading()) {
                <svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Đang xử lý...</span>
              } @else {
                <span>{{ confirmLabel() }}</span>
              }
            </button>
          </div>
        </div>
      </div>
    }
  `
})
export class ConfirmDialogComponent {
  readonly isOpen = input<boolean>(false);
  readonly title = input<string>('Xác nhận đặt vé');
  readonly message = input<string>('Bạn có chắc chắn muốn tiến hành đặt các vé đã chọn không?');
  readonly confirmLabel = input<string>('Xác nhận');
  readonly loading = input<boolean>(false);

  readonly confirmed = output<void>();
  readonly cancelled = output<void>();

  onConfirm(): void {
    this.confirmed.emit();
  }

  onCancel(): void {
    this.cancelled.emit();
  }
}
