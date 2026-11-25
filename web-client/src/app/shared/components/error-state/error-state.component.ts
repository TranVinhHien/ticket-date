import { Component, input, output } from '@angular/core';

@Component({
  selector: 'app-error-state',
  standalone: true,
  template: `
    <div class="bg-[#FFF1F2] border border-[#FECDD3] rounded-xl p-8 text-center max-w-lg mx-auto my-8">
      <div class="w-12 h-12 rounded-xl bg-white border border-[#FECDD3] text-[#E11D48] flex items-center justify-center mx-auto mb-3">
        <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
      <h3 class="font-display text-base font-bold text-[#881337] mb-1">Không thể tải dữ liệu</h3>
      <p class="text-sm text-[#9F1239] mb-5">{{ message() }}</p>
      @if (showRetry()) {
        <button
          (click)="retryClicked.emit()"
          class="inline-flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-semibold bg-[#E11D48] text-white hover:bg-[#BE123C] transition-colors">
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Thử lại
        </button>
      }
    </div>
  `
})
export class ErrorStateComponent {
  readonly message = input<string>('Đã xảy ra lỗi khi kết nối tới máy chủ.');
  readonly showRetry = input<boolean>(true);
  readonly retryClicked = output<void>();
}
