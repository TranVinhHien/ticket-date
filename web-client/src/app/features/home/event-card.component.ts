import { Component, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { Event } from '../../core/models/event.model';

@Component({
  selector: 'app-event-card',
  standalone: true,
  imports: [CommonModule, RouterLink],
  template: `
    <div class="group bg-white rounded-xl overflow-hidden border border-[#E1E5EE] hover:border-[#1B4BFF] transition-colors flex flex-col h-full">
      <!-- Event Poster Image -->
      <div class="relative w-full h-48 bg-[#121622] overflow-hidden">
        @if (event().poster_url) {
          <img
            [src]="event().poster_url"
            [alt]="event().name"
            class="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-300"
            (error)="onImageError($event)" />
        } @else {
          <div class="w-full h-full flex flex-col items-center justify-center text-white/50 p-4 text-center td-stage-grid">
            <svg class="w-10 h-10 mb-2 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
            </svg>
            <span class="text-xs font-medium">Ticket Date Concert</span>
          </div>
        }

        <!-- Status Badge -->
        <div class="absolute top-3 right-3">
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-semibold bg-[#0D9488] text-white">
            <span class="w-1.5 h-1.5 rounded-full bg-white"></span>
            Đang mở bán
          </span>
        </div>
      </div>

      <!-- Card Body -->
      <div class="p-5 flex-1 flex flex-col justify-between">
        <div>
          <!-- Date & Time -->
          <div class="flex items-center gap-1.5 text-xs font-semibold text-[#1B4BFF] mb-2 tabular-nums">
            <svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span>{{ formatDateTime(event().start_at) }}</span>
          </div>

          <!-- Event Name -->
          <h3 class="font-display text-lg font-bold text-[#121622] group-hover:text-[#1B4BFF] transition-colors line-clamp-2 mb-2">
            {{ event().name }}
          </h3>

          <!-- Venue / Location -->
          @if (event().venue) {
            <div class="flex items-start gap-1.5 text-xs text-[#5A6275] mb-4 line-clamp-1">
              <svg class="w-4 h-4 shrink-0 text-[#7C8498] mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <span>{{ event().venue?.name }}{{ event().venue?.city ? ', ' + event().venue?.city : '' }}</span>
            </div>
          }
        </div>

        <!-- Ticket Stub Footer: Price & CTA -->
        <div class="pt-4 td-stub-divider flex items-center justify-between mt-auto">
          <div>
            <span class="text-xs text-[#5A6275] block">Giá từ</span>
            <span class="text-base font-bold text-[#121622] tabular-nums">
              {{ formattedLowestPrice() }}
            </span>
          </div>

          <a
            [routerLink]="['/events', event().slug]"
            class="inline-flex items-center gap-1 px-4 py-2 rounded-lg text-xs font-semibold bg-[#121622] text-white group-hover:bg-[#1B4BFF] transition-colors">
            <span>Chi tiết</span>
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
            </svg>
          </a>
        </div>
      </div>
    </div>
  `
})
export class EventCardComponent {
  readonly event = input.required<Event>();

  readonly formattedLowestPrice = computed(() => {
    const types = this.event().ticket_types;
    if (!types || types.length === 0) return 'Đang cập nhật';
    const minPrice = Math.min(...types.map(t => t.price));
    return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(minPrice);
  });

  formatDateTime(iso: string): string {
    if (!iso) return '';
    const date = new Date(iso);
    return date.toLocaleDateString('vi-VN', {
      weekday: 'short',
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  onImageError(e: any): void {
    e.target.style.display = 'none';
  }
}
