import { Component, input } from '@angular/core';

@Component({
  selector: 'app-loading-skeleton',
  standalone: true,
  template: `
    @if (type() === 'cards') {
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        @for (i of countArray; track $index) {
          <div class="bg-white rounded-xl p-4 border border-[#E1E5EE] animate-pulse flex flex-col gap-3">
            <div class="w-full h-48 bg-[#E5E8EF] rounded-lg"></div>
            <div class="h-5 bg-[#E5E8EF] rounded w-3/4 mt-2"></div>
            <div class="h-4 bg-[#EEF0F5] rounded w-1/2"></div>
            <div class="flex items-center justify-between mt-4 pt-3 td-stub-divider">
              <div class="h-5 bg-[#E5E8EF] rounded w-1/3"></div>
              <div class="h-8 bg-[#E5E8EF] rounded-lg w-20"></div>
            </div>
          </div>
        }
      </div>
    } @else {
      <div class="bg-white rounded-xl p-6 border border-[#E1E5EE] animate-pulse space-y-4">
        <div class="h-7 bg-[#E5E8EF] rounded w-1/3"></div>
        <div class="h-4 bg-[#EEF0F5] rounded w-2/3"></div>
        <div class="h-32 bg-[#EEF0F5] rounded-lg"></div>
      </div>
    }
  `
})
export class LoadingSkeletonComponent {
  readonly type = input<'cards' | 'detail'>('cards');
  readonly count = input<number>(6);

  get countArray(): number[] {
    return Array.from({ length: this.count() }, (_, i) => i);
  }
}
