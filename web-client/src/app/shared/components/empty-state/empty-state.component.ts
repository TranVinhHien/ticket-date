import { Component, input, output } from '@angular/core';

@Component({
  selector: 'app-empty-state',
  standalone: true,
  template: `
    <div class="bg-white rounded-xl border border-dashed border-[#D0D7E3] p-10 sm:p-12 text-center flex flex-col items-center justify-center max-w-md mx-auto my-8">
      <div class="w-14 h-14 rounded-xl bg-[#F5F6F8] border border-[#E1E5EE] text-[#1B4BFF] flex items-center justify-center mb-4">
        <svg class="w-7 h-7" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
        </svg>
      </div>
      <h3 class="font-display text-lg font-bold text-[#121622] mb-1.5">{{ title() }}</h3>
      <p class="text-sm text-[#5A6275] mb-6 leading-relaxed">{{ description() }}</p>
      @if (actionLabel()) {
        <button
          (click)="actionClicked.emit()"
          class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg text-sm font-semibold bg-[#1B4BFF] text-white hover:bg-[#143AD6] transition-colors">
          {{ actionLabel() }}
        </button>
      }
    </div>
  `
})
export class EmptyStateComponent {
  readonly title = input<string>('Không có dữ liệu');
  readonly description = input<string>('Hiện tại chưa có thông tin nào để hiển thị.');
  readonly actionLabel = input<string>('');
  readonly actionClicked = output<void>();
}
